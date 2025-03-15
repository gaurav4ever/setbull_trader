package postgres

import (
	"context"
	"errors"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TradeParametersRepository implements repository.TradeParametersRepository interface using GORM
type TradeParametersRepository struct {
	db *gorm.DB
}

// NewTradeParametersRepository creates a new TradeParametersRepository
func NewTradeParametersRepository(db *gorm.DB) repository.TradeParametersRepository {
	return &TradeParametersRepository{db: db}
}

// Create creates new trade parameters
func (r *TradeParametersRepository) Create(ctx context.Context, params *domain.TradeParameters) error {
	if params.ID == "" {
		params.ID = uuid.New().String()
	}

	return r.db.WithContext(ctx).Create(params).Error
}

// GetByID retrieves trade parameters by their ID
func (r *TradeParametersRepository) GetByID(ctx context.Context, id string) (*domain.TradeParameters, error) {
	var params domain.TradeParameters
	err := r.db.WithContext(ctx).First(&params, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &params, nil
}

// GetByStockID retrieves trade parameters for a specific stock
func (r *TradeParametersRepository) GetByStockID(ctx context.Context, stockID string) (*domain.TradeParameters, error) {
	var params domain.TradeParameters
	err := r.db.WithContext(ctx).Where("stock_id = ?", stockID).First(&params).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &params, nil
}

// Update updates trade parameters
func (r *TradeParametersRepository) Update(ctx context.Context, params *domain.TradeParameters) error {
	return r.db.WithContext(ctx).Save(params).Error
}

// Delete deletes trade parameters
func (r *TradeParametersRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.TradeParameters{}, id).Error
}
