// internal/core/adapters/client/upstox/config.go

package upstox

import (
	"context"
	"fmt"
	"time"

	swagger "setbull_trader/upstox/go_api_client"

	"github.com/pkg/errors"
)

// AuthConfig represents the Upstox authentication configuration
type AuthConfig struct {
	ClientID     string `json:"client_id" yaml:"client_id"`         // API key
	ClientSecret string `json:"client_secret" yaml:"client_secret"` // API secret
	RedirectURI  string `json:"redirect_uri" yaml:"redirect_uri"`   // Registered callback URL
	BasePath     string `json:"base_path" yaml:"base_path"`         // API base path
}

// NewUpstoxConfig creates a new configuration for Upstox API
func NewUpstoxConfig() *AuthConfig {
	return &AuthConfig{
		BasePath: "https://api-v2.upstox.com", // Default base path matching the client
	}
}

// CreateSwaggerConfig creates a swagger configuration from the auth config
func (c *AuthConfig) CreateSwaggerConfig() *swagger.Configuration {
	config := swagger.NewConfiguration()
	config.BasePath = c.BasePath
	config.AddDefaultHeader("Content-Type", "application/json")
	config.AddDefaultHeader("Accept", "application/json")
	return config
}

// TokenStore represents an interface for storing Upstox tokens
type TokenStore interface {
	// SaveToken saves a token for a user
	SaveToken(ctx context.Context, userID string, token *UpstoxToken) error

	// GetToken retrieves a token for a user
	GetToken(ctx context.Context, userID string) (*UpstoxToken, error)

	// DeleteToken deletes a token for a user
	DeleteToken(ctx context.Context, userID string) error
}

// UpstoxToken represents an authentication token from Upstox with additional metadata
type UpstoxToken struct {
	// Original response from Upstox
	TokenResponse *swagger.TokenResponse

	// Additional metadata
	CreatedAt      time.Time `json:"created_at"`
	ExpirationTime time.Time `json:"expiration_time"`
	UserID         string    `json:"user_id"`
}

// NewUpstoxToken creates a new token with metadata
func NewUpstoxToken(tokenResponse *swagger.TokenResponse, userID string) *UpstoxToken {
	createdAt := time.Now()
	// Calculate expiration time - default to 24 hours if not provided
	expirySeconds := int64(86400) // 24 hours default

	return &UpstoxToken{
		TokenResponse:  tokenResponse,
		CreatedAt:      createdAt,
		ExpirationTime: createdAt.Add(time.Duration(expirySeconds) * time.Second),
		UserID:         userID,
	}
}

// IsExpired checks if the token has expired
func (t *UpstoxToken) IsExpired() bool {
	// Add a 5-minute buffer to ensure we don't use tokens that are about to expire
	return time.Now().Add(5 * time.Minute).After(t.ExpirationTime)
}

// GetAccessToken returns the access token string
func (t *UpstoxToken) GetAccessToken() string {
	if t.TokenResponse == nil {
		return ""
	}
	return t.TokenResponse.AccessToken
}

// GenerateLoginURL generates the Upstox login URL
func (c *AuthConfig) GenerateLoginURL(state string) string {
	return fmt.Sprintf("https://api.upstox.com/v2/login/authorization/dialog?response_type=code&client_id=%s&redirect_uri=%s&state=%s",
		c.ClientID, c.RedirectURI, state)
}

// AuthError represents an authentication error
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e AuthError) Error() string {
	return fmt.Sprintf("Upstox auth error: %s - %s", e.Code, e.Message)
}

// UpstoxClientWrapper creates a wrapper around the Upstox API client
type UpstoxClientWrapper struct {
	apiClient  *swagger.APIClient
	authConfig *AuthConfig
	tokenStore TokenStore
}

// NewUpstoxClientWrapper creates a new Upstox client wrapper
func NewUpstoxClientWrapper(authConfig *AuthConfig, tokenStore TokenStore) *UpstoxClientWrapper {
	config := authConfig.CreateSwaggerConfig()
	apiClient := swagger.NewAPIClient(config)

	return &UpstoxClientWrapper{
		apiClient:  apiClient,
		authConfig: authConfig,
		tokenStore: tokenStore,
	}
}

// GetAuthenticatedContext creates a context with authentication for API calls
func (w *UpstoxClientWrapper) GetAuthenticatedContext(ctx context.Context, userID string) (context.Context, error) {
	token, err := w.tokenStore.GetToken(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token from store")
	}

	if token == nil || token.IsExpired() {
		return nil, errors.New("token not found or expired")
	}

	// Add token to context for Upstox client
	return context.WithValue(ctx, swagger.ContextAccessToken, token.GetAccessToken()), nil
}
