// internal/core/adapters/client/upstox/auth_service.go

package upstox

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"setbull_trader/pkg/cache"
	"setbull_trader/pkg/log"
	swagger "setbull_trader/upstox/go_api_client"

	"github.com/antihax/optional"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type UpstoxErrorResponse struct {
	Status string `json:"status"`
	Errors []struct {
		ErrorCode    string      `json:"error_code"`
		Message      string      `json:"message"`
		PropertyPath interface{} `json:"property_path"`
		InvalidValue interface{} `json:"invalid_value"`
	} `json:"errors"`
}

// AuthService handles Upstox authentication operations
type AuthService struct {
	config       *AuthConfig
	tokenStore   TokenStore
	cacheManager cache.API
	statePrefix  string
	client       *http.Client
	// Rate limiters
	secondLimiter    *rate.Limiter
	minuteLimiter    *rate.Limiter
	thirtyMinLimiter *rate.Limiter
	limiterMutex     sync.Mutex
}

// NewAuthService creates a new authentication service
func NewAuthService(config *AuthConfig, tokenStore TokenStore, cacheManager cache.API) *AuthService {
	return &AuthService{
		config:       config,
		tokenStore:   tokenStore,
		cacheManager: cacheManager,
		statePrefix:  "upstox:state:",
		client:       &http.Client{Timeout: 30 * time.Second},
		// Initialize rate limiters according to Upstox API limits
		secondLimiter:    rate.NewLimiter(rate.Limit(50), 50),          // 50 requests per second
		minuteLimiter:    rate.NewLimiter(rate.Limit(500/60), 500),     // 500 requests per minute
		thirtyMinLimiter: rate.NewLimiter(rate.Limit(2000/1800), 2000), // 2000 requests per 30 minutes
	}
}

// InitiateLogin starts the login flow and returns the authorization URL
func (s *AuthService) InitiateLogin(ctx context.Context) (string, string, error) {
	// Generate a secure random state to prevent CSRF
	state, err := s.generateSecureState()
	if err != nil {
		return "", "", errors.Wrap(err, "failed to generate secure state")
	}

	// Generate a unique session ID to associate with this login flow
	sessionID := "upstox_session"

	// Store the state in the cache with expiry (15 minutes is typical for OAuth flows)
	stateKey := s.getStateKey(sessionID)
	s.cacheManager.SetWithDuration(ctx, stateKey, state, 15*time.Minute)

	// Generate the login URL
	loginURL := s.config.GenerateLoginURL(state)

	log.Info("Upstox login initiated with session ID: %s", sessionID)
	return loginURL, sessionID, nil
}

// HandleCallback processes the authorization callback from Upstox
func (s *AuthService) HandleCallback(ctx context.Context, code string, state string) (*UpstoxToken, error) {
	// Verify state to prevent CSRF
	if err := s.verifyState(ctx, state); err != nil {
		return nil, err
	}

	// Exchange the authorization code for a token
	token, err := s.exchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err, "failed to exchange code for token")
	}

	sessionID := "upstox_session"
	// Store the token
	if err := s.tokenStore.SaveToken(ctx, sessionID, token); err != nil {
		return nil, errors.Wrap(err, "failed to save token")
	}

	log.Info("Successfully authenticated with Upstox for session: %s", sessionID)
	return token, nil
}

// GetAuthenticatedContext creates a context with authentication token
func (s *AuthService) GetAuthenticatedContext(ctx context.Context, userID string) (context.Context, error) {
	// Get token from store
	token, err := s.tokenStore.GetToken(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token")
	}

	if token == nil || token.IsExpired() {
		return nil, errors.New("no valid token found for user")
	}

	// Create authentication context
	authCtx := context.WithValue(ctx, swagger.ContextAccessToken, token.GetAccessToken())
	return authCtx, nil
}

// GetHistoricalCandleData is a helper method to get historical candle data
func (s *AuthService) GetHistoricalCandleData(ctx context.Context, userID string, instrumentKey string, interval string, toDate string) (*swagger.GetHistoricalCandleResponse, error) {
	// Get authenticated context
	authCtx, err := s.GetAuthenticatedContext(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create API client
	config := s.config.CreateSwaggerConfig()
	client := swagger.NewAPIClient(config)

	// Call the historical candle data API with the authenticated context
	response, httpResp, err := client.HistoryApi.GetHistoricalCandleData(authCtx, instrumentKey, interval, toDate)
	if err != nil {
		// Parse the error response if available
		if httpResp != nil && httpResp.Body != nil {
			defer httpResp.Body.Close()

			// Try to read the response body
			body, readErr := ioutil.ReadAll(httpResp.Body)
			if readErr == nil {
				// Try to parse as ApiGatewayErrorResponse
				var apiError swagger.ApiGatewayErrorResponse
				if jsonErr := json.Unmarshal(body, &apiError); jsonErr == nil {
					// Log the detailed API error
					log.Error("Upstox API error: Status: %d, Error Status: %s",
						httpResp.StatusCode, apiError.Status)

					for i, problem := range apiError.Errors {
						log.Error("  Error %d: %v", i+1, problem)
					}

					return nil, fmt.Errorf("upstox API error: %s", apiError.Status)
				}

				// If not an API error, log the raw response
				log.Error("HTTP Status: %d, Raw Response: %s", httpResp.StatusCode, string(body))
			} else {
				log.Error("HTTP Status: %d, Error: %v, Failed to read response body: %v",
					httpResp.StatusCode, err, readErr)
			}
		} else {
			log.Error("Error with no HTTP response: %v", err)
		}

		return nil, errors.Wrap(err, "failed to get historical candle data")
	}

	return &response, nil
}

// GetHistoricalCandleDataWithDateRange is a helper method to get historical candle data with date range
func (s *AuthService) GetHistoricalCandleDataWithDateRange(ctx context.Context, userID string, instrumentKey string, interval string, toDate string, fromDate string) (*swagger.GetHistoricalCandleResponse, error) {
	// Wait for rate limit allowance before proceeding
	if err := s.waitForRateLimit(ctx); err != nil {
		return nil, errors.Wrap(err, "rate limit wait error")
	}

	// Get authenticated context
	authCtx, err := s.GetAuthenticatedContext(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create API client
	config := s.config.CreateSwaggerConfig()
	client := swagger.NewAPIClient(config)

	// Call the historical candle data API with date range using the authenticated context
	response, httpResp, err := client.HistoryApi.GetHistoricalCandleData1(authCtx, instrumentKey, interval, toDate, fromDate)

	// Handle rate limit errors specifically
	if err != nil && httpResp != nil && httpResp.StatusCode == 429 {
		log.Warn("Rate limit exceeded, retrying after backoff")

		// Exponential backoff - wait longer and retry
		retryAfter := 2 * time.Second
		if retryAfterHeader := httpResp.Header.Get("Retry-After"); retryAfterHeader != "" {
			if seconds, parseErr := strconv.Atoi(retryAfterHeader); parseErr == nil {
				retryAfter = time.Duration(seconds) * time.Second
			}
		}

		// Wait for the specified time
		select {
		case <-time.After(retryAfter):
			// Try again recursively with the same parameters
			return s.GetHistoricalCandleDataWithDateRange(ctx, userID, instrumentKey, interval, toDate, fromDate)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Handle other errors
	if err != nil {
		// Parse the error response if available
		if httpResp != nil && httpResp.Body != nil {
			defer httpResp.Body.Close()

			// Try to read the response body
			body, readErr := ioutil.ReadAll(httpResp.Body)
			if readErr == nil {
				// Try to parse as ApiGatewayErrorResponse
				var apiError swagger.ApiGatewayErrorResponse
				if jsonErr := json.Unmarshal(body, &apiError); jsonErr == nil {
					// Log the detailed API error
					log.Error("Upstox API error for %s from %s to %s: Status: %d, Error Status: %s",
						instrumentKey, fromDate, toDate, httpResp.StatusCode, apiError.Status)

					for i, problem := range apiError.Errors {
						log.Error("  Error %d: %v", i+1, problem)
					}

					return nil, fmt.Errorf("upstox API error: %s", apiError.Status)
				}

				// If not an API error, log the raw response
				log.Error("HTTP Status: %d, Raw Response: %s", httpResp.StatusCode, string(body))
			} else {
				log.Error("HTTP Status: %d, Error: %v, Failed to read response body: %v",
					httpResp.StatusCode, err, readErr)
			}
		} else {
			log.Error("Error with no HTTP response: %v", err)
		}

		return nil, errors.Wrap(err, "failed to get historical candle data with date range")
	}

	return &response, nil
}

// GetIntraDayCandleData is a helper method to get intra-day candle data
func (s *AuthService) GetIntraDayCandleData(ctx context.Context, userID string, instrumentKey string, interval string) (*swagger.GetIntraDayCandleResponse, error) {
	// Get authenticated context
	authCtx, err := s.GetAuthenticatedContext(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create API client
	config := s.config.CreateSwaggerConfig()
	client := swagger.NewAPIClient(config)

	// Call the intra-day candle data API using the authenticated context
	response, httpResp, err := client.HistoryApi.GetIntraDayCandleData(authCtx, instrumentKey, interval)
	if err != nil {
		// Parse the error response if available
		if httpResp != nil && httpResp.Body != nil {
			defer httpResp.Body.Close()

			// Try to read the response body
			body, readErr := ioutil.ReadAll(httpResp.Body)
			if readErr == nil {
				// Try to parse as ApiGatewayErrorResponse
				var apiError swagger.ApiGatewayErrorResponse
				if jsonErr := json.Unmarshal(body, &apiError); jsonErr == nil {
					// Log the detailed API error
					log.Error("Upstox API error for intraday data for %s: Status: %d, Error Status: %s",
						instrumentKey, httpResp.StatusCode, apiError.Status)

					for i, problem := range apiError.Errors {
						log.Error("  Error %d: %v", i+1, problem)
					}

					return nil, fmt.Errorf("upstox API error: %s", apiError.Status)
				}

				// If not an API error, log the raw response
				log.Error("HTTP Status: %d, Raw Response: %s", httpResp.StatusCode, string(body))
			} else {
				log.Error("HTTP Status: %d, Error: %v, Failed to read response body: %v",
					httpResp.StatusCode, err, readErr)
			}
		} else {
			log.Error("Error with no HTTP response: %v", err)
		}

		return nil, errors.Wrap(err, "failed to get intra-day candle data")
	}

	return &response, nil
}

// exchangeCodeForToken exchanges an authorization code for an access token
func (s *AuthService) exchangeCodeForToken(ctx context.Context, code string) (*UpstoxToken, error) {
	// Prepare the token request
	tokenURL := fmt.Sprintf("%s/v2/login/authorization/token", s.config.BasePath)
	formData := url.Values{
		"code":          {code},
		"client_id":     {s.config.ClientID},
		"client_secret": {s.config.ClientSecret},
		"redirect_uri":  {s.config.RedirectURI},
		"grant_type":    {"authorization_code"},
	}

	// Make the request
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create token request")
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute token request")
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read token response")
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errResp UpstoxErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			if len(errResp.Errors) > 0 {
				return nil, fmt.Errorf("upstox error: %s - %s",
					errResp.Errors[0].ErrorCode,
					errResp.Errors[0].Message)
			}
		}
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse token response
	var tokenResp swagger.TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal token response")
	}

	// Create a session ID for this token
	sessionID := uuid.New().String()

	// Create token with metadata
	token := NewUpstoxToken(&tokenResp, sessionID)

	return token, nil
}

// getStateKey generates a cache key for a state
func (s *AuthService) getStateKey(sessionID string) string {
	return fmt.Sprintf("%s%s", s.statePrefix, sessionID)
}

// generateSecureState generates a secure random state string
func (s *AuthService) generateSecureState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// verifyState verifies that the state parameter matches the expected state
func (s *AuthService) verifyState(ctx context.Context, state string) error {
	sessionID := "upstox_session"
	stateKey := s.getStateKey(sessionID)
	expectedState, exists := s.cacheManager.Get(ctx, stateKey)
	if !exists {
		return errors.New("state not found or expired")
	}

	if expectedState != state {
		return errors.New("state mismatch, possible CSRF attack")
	}

	// Clean up the state after use
	s.cacheManager.SetWithDuration(ctx, stateKey, "", time.Millisecond)
	return nil
}

// Add this method to waitForRateLimit
func (s *AuthService) waitForRateLimit(ctx context.Context) error {
	s.limiterMutex.Lock()
	defer s.limiterMutex.Unlock()

	// Log current rate limiter states before waiting
	log.Info("Rate limit check - Second: %v/%v, Minute: %v/%v, 30Min: %v/%v",
		s.secondLimiter.Tokens(), s.secondLimiter.Burst(),
		s.minuteLimiter.Tokens(), s.minuteLimiter.Burst(),
		s.thirtyMinLimiter.Tokens(), s.thirtyMinLimiter.Burst())

	// Wait for all three rate limiters
	if err := s.secondLimiter.Wait(ctx); err != nil {
		return err
	}

	if err := s.minuteLimiter.Wait(ctx); err != nil {
		return err
	}

	if err := s.thirtyMinLimiter.Wait(ctx); err != nil {
		return err
	}

	return nil
}

// GetMarketQuote fetches OHLC market quotes for the given instrumentKeys from Upstox.
// Returns a map of instrumentKey to Ohlc, a map of instrumentKey to error string (for failures), and an error for fatal issues.
func (s *AuthService) GetMarketQuote(ctx context.Context, userID string, instrumentKeys []string, interval string) (map[string]Ohlc, map[string]string, error) {
	if interval == "" {
		interval = "1min"
	}

	// Step 1: Authenticate user
	authCtx, err := s.GetAuthenticatedContext(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	// Step 2: Create Upstox API client
	config := s.config.CreateSwaggerConfig()
	client := swagger.NewAPIClient(config)

	// Step 3: Prepare request to Upstox GetFullMarketQuote
	joinedKeys := strings.Join(instrumentKeys, ",")
	opts := &swagger.MarketQuoteApiGetFullMarketQuoteOpts{
		InstrumentKey: optional.NewString(joinedKeys),
	}
	resp, httpResp, err := client.MarketQuoteApi.GetFullMarketQuote(authCtx, opts)
	if err != nil {
		// If the error is a 200 with partial data, Upstox may still return some data. Try to parse resp.Data if possible.
		if httpResp != nil && resp.Data != nil && len(resp.Data) > 0 {
			// Continue to process partial data
			log.Warn("Partial data received from Upstox: %v", err)
		} else {
			return nil, nil, errors.Wrap(err, "failed to fetch market quotes from Upstox")
		}
	}

	// Step 4: Map results
	data := make(map[string]Ohlc)
	errorsMap := make(map[string]string)
	for _, key := range instrumentKeys {
		symbol, ok := resp.Data[key]
		if !ok || symbol.Ohlc == nil {
			errorsMap[key] = "No OHLC data returned"
			continue
		}
		data[key] = Ohlc{
			Open:  symbol.Ohlc.Open,
			High:  symbol.Ohlc.High,
			Low:   symbol.Ohlc.Low,
			Close: symbol.Ohlc.Close,
		}
	}

	// Step 5: For any keys not present in resp.Data, add to errors
	for _, key := range instrumentKeys {
		if _, ok := data[key]; !ok {
			if _, already := errorsMap[key]; !already {
				errorsMap[key] = "Instrument not found in Upstox response"
			}
		}
	}

	return data, errorsMap, nil
}

// Ohlc struct for adapter layer mapping (matches response DTO)
type Ohlc struct {
	Open  float64 `json:"open,omitempty"`
	High  float64 `json:"high,omitempty"`
	Low   float64 `json:"low,omitempty"`
	Close float64 `json:"close,omitempty"`
}
