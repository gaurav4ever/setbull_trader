// internal/core/adapters/client/upstox/token_repository.go

package upstox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"setbull_trader/pkg/cache"
	"setbull_trader/pkg/log"

	"github.com/pkg/errors"
)

// TokenRepository implements the TokenStore interface using a cache backend
type TokenRepository struct {
	cacheManager cache.API
	keyPrefix    string
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(cacheManager cache.API) *TokenRepository {
	return &TokenRepository{
		cacheManager: cacheManager,
		keyPrefix:    "upstox:token:",
	}
}

// SaveToken saves a token for a user
func (r *TokenRepository) SaveToken(ctx context.Context, userID string, token *UpstoxToken) error {
	if token == nil {
		return errors.New("cannot save nil token")
	}

	// Calculate TTL based on token expiration
	ttl := token.ExpirationTime.Sub(time.Now())
	if ttl <= 0 {
		return errors.New("cannot save expired token")
	}

	// Convert token to JSON
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return errors.Wrap(err, "failed to marshal token")
	}

	// Save to cache with expiry time
	key := r.generateKey(userID)
	r.cacheManager.SetWithDuration(ctx, key, string(tokenJSON), ttl)

	log.Info("Upstox token saved for user: %s, expires in: %v", userID, ttl)
	return nil
}

// GetToken retrieves a token for a user
func (r *TokenRepository) GetToken(ctx context.Context, userID string) (*UpstoxToken, error) {
	key := r.generateKey(userID)
	tokenJSON, exists := r.cacheManager.Get(ctx, key)
	if !exists {
		return nil, nil // No token found, but not an error
	}

	var token UpstoxToken
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal token")
	}

	// Check if token is expired
	if token.IsExpired() {
		// Token is expired, delete it
		r.DeleteToken(ctx, userID)
		return nil, nil
	}

	return &token, nil
}

// DeleteToken deletes a token for a user
func (r *TokenRepository) DeleteToken(ctx context.Context, userID string) error {
	key := r.generateKey(userID)
	// Use a short expiry to effectively delete the token
	r.cacheManager.SetWithDuration(ctx, key, "", time.Millisecond)
	log.Info("Upstox token deleted for user: %s", userID)
	return nil
}

// generateKey creates a cache key for a user
func (r *TokenRepository) generateKey(userID string) string {
	return fmt.Sprintf("%s%s", r.keyPrefix, userID)
}

// DatabaseTokenRepository is an alternative implementation that uses a database
type DatabaseTokenRepository struct {
	// Add your database connection/ORM here
	// e.g., db *gorm.DB
}

// NewDatabaseTokenRepository creates a new database token repository
/*
func NewDatabaseTokenRepository(db *gorm.DB) *DatabaseTokenRepository {
	return &DatabaseTokenRepository{
		db: db,
	}
}

// SaveToken saves a token to the database
func (r *DatabaseTokenRepository) SaveToken(ctx context.Context, userID string, token *UpstoxToken) error {
	// Create a database model from the token
	tokenModel := &TokenModel{
		UserID:         userID,
		AccessToken:    token.TokenResponse.AccessToken,
		RefreshToken:   token.TokenResponse.RefreshToken,
		ExtendedToken:  token.TokenResponse.ExtendedToken,
		CreatedAt:      token.CreatedAt,
		ExpirationTime: token.ExpirationTime,
	}

	// Use a transaction to ensure atomicity
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Delete any existing tokens for this user
	if err := tx.Where("user_id = ?", userID).Delete(&TokenModel{}).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to delete existing tokens")
	}

	// Save the new token
	if err := tx.Create(tokenModel).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to save token")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	log.Info("Upstox token saved to database for user: %s", userID)
	return nil
}

// GetToken retrieves a token from the database
func (r *DatabaseTokenRepository) GetToken(ctx context.Context, userID string) (*UpstoxToken, error) {
	var tokenModel TokenModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&tokenModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No token found, but not an error
		}
		return nil, errors.Wrap(err, "failed to retrieve token")
	}

	// Check if token is expired
	if time.Now().After(tokenModel.ExpirationTime) {
		// Token is expired, delete it
		r.DeleteToken(ctx, userID)
		return nil, nil
	}

	// Convert database model to token
	tokenResponse := &swagger.TokenResponse{
		AccessToken:   tokenModel.AccessToken,
		RefreshToken:  tokenModel.RefreshToken,
		ExtendedToken: tokenModel.ExtendedToken,
	}

	token := &UpstoxToken{
		TokenResponse:  tokenResponse,
		CreatedAt:      tokenModel.CreatedAt,
		ExpirationTime: tokenModel.ExpirationTime,
		UserID:         userID,
	}

	return token, nil
}

// DeleteToken deletes a token from the database
func (r *DatabaseTokenRepository) DeleteToken(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&TokenModel{}).Error; err != nil {
		return errors.Wrap(err, "failed to delete token")
	}

	log.Info("Upstox token deleted from database for user: %s", userID)
	return nil
}

// TokenModel represents the database model for storing tokens
type TokenModel struct {
	ID             string    `gorm:"column:id;primaryKey"`
	UserID         string    `gorm:"column:user_id;index"`
	AccessToken    string    `gorm:"column:access_token"`
	RefreshToken   string    `gorm:"column:refresh_token"`
	ExtendedToken  string    `gorm:"column:extended_token"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	ExpirationTime time.Time `gorm:"column:expiration_time"`
}

// TableName returns the table name for the token model
func (TokenModel) TableName() string {
	return "upstox_tokens"
}
*/
