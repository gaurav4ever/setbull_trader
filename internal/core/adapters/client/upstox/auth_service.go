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
	"strings"
	"time"

	"setbull_trader/pkg/cache"
	"setbull_trader/pkg/log"
	swagger "setbull_trader/upstox/go_api_client"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// AuthService handles Upstox authentication operations
type AuthService struct {
	config       *AuthConfig
	tokenStore   TokenStore
	cacheManager cache.API
	statePrefix  string
	client       *http.Client
}

// NewAuthService creates a new authentication service
func NewAuthService(config *AuthConfig, tokenStore TokenStore, cacheManager cache.API) *AuthService {
	return &AuthService{
		config:       config,
		tokenStore:   tokenStore,
		cacheManager: cacheManager,
		statePrefix:  "upstox:state:",
		client:       &http.Client{Timeout: 30 * time.Second},
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
	sessionID := uuid.New().String()

	// Store the state in the cache with expiry (15 minutes is typical for OAuth flows)
	stateKey := s.getStateKey(sessionID)
	s.cacheManager.SetWithDuration(ctx, stateKey, state, 15*time.Minute)

	// Generate the login URL
	loginURL := s.config.GenerateLoginURL(state)

	log.Info("Upstox login initiated with session ID: %s", sessionID)
	return loginURL, sessionID, nil
}

// HandleCallback processes the authorization callback from Upstox
func (s *AuthService) HandleCallback(ctx context.Context, code string, state string, sessionID string) (*UpstoxToken, error) {
	// Verify state to prevent CSRF
	if err := s.verifyState(ctx, sessionID, state); err != nil {
		return nil, err
	}

	// Exchange the authorization code for a token
	token, err := s.exchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err, "failed to exchange code for token")
	}

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
		// Log detailed error information
		if httpResp != nil {
			log.Error("HTTP Status: %d, Error: %v", httpResp.StatusCode, err)
		}
		return nil, errors.Wrap(err, "failed to get historical candle data")
	}

	return &response, nil
}

// GetHistoricalCandleDataWithDateRange is a helper method to get historical candle data with date range
func (s *AuthService) GetHistoricalCandleDataWithDateRange(ctx context.Context, userID string, instrumentKey string, interval string, toDate string, fromDate string) (*swagger.GetHistoricalCandleResponse, error) {
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
	if err != nil {
		// Log detailed error information
		if httpResp != nil {
			log.Error("HTTP Status: %d, Error: %v", httpResp.StatusCode, err)
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
		// Log detailed error information
		if httpResp != nil {
			log.Error("HTTP Status: %d, Error: %v", httpResp.StatusCode, err)
		}
		return nil, errors.Wrap(err, "failed to get intra-day candle data")
	}

	return &response, nil
}

// exchangeCodeForToken exchanges an authorization code for an access token
func (s *AuthService) exchangeCodeForToken(ctx context.Context, code string) (*UpstoxToken, error) {
	// Prepare the token request
	tokenURL := "https://api.upstox.com/v2/login/authorization/token"
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
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("token request failed: %s - %s", errResp.Error, errResp.Description)
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
func (s *AuthService) verifyState(ctx context.Context, sessionID string, state string) error {
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
