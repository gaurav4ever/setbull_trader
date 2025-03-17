package postgres

import (
	"context"
	"errors"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StockRepository implements repository.StockRepository interface using PostgreSQL
type StockRepository struct {
	db *gorm.DB
}

// NewStockRepository creates a new StockRepository
func NewStockRepository(db *gorm.DB) repository.StockRepository {
	return &StockRepository{db: db}
}

// Create creates a new stock
func (r *StockRepository) Create(ctx context.Context, stock *domain.Stock) error {
	if stock.ID == "" {
		stock.ID = uuid.New().String()
	}

	stock.CreatedAt = time.Now()
	stock.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).Create(stock).Error
}

// GetByID retrieves a stock by its ID
func (r *StockRepository) GetByID(ctx context.Context, id string) (*domain.Stock, error) {
	var stock domain.Stock
	err := r.db.WithContext(ctx).First(&stock, "id = ? AND active = 1", id).Error // Use GORM's First method
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if stock not found
		}
		return nil, err
	}
	return &stock, nil
}

// GetBySymbol retrieves a stock by its symbol
func (r *StockRepository) GetBySymbol(ctx context.Context, symbol string) (*domain.Stock, error) {
	var stock domain.Stock
	err := r.db.WithContext(ctx).Where("symbol = ? AND active = 1", symbol).First(&stock).Error // Use GORM's Where and First methods
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if stock not found
		}
		return nil, err
	}
	return &stock, nil
}

// GetBySecurityID retrieves a stock by its security ID
func (r *StockRepository) GetBySecurityID(ctx context.Context, securityID string) (*domain.Stock, error) {
	var stock domain.Stock
	err := r.db.WithContext(ctx).Where("security_id = ? AND active = 1", securityID).First(&stock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if stock not found
		}
		return nil, err
	}
	return &stock, nil
}

// GetAll retrieves all stocks
func (r *StockRepository) GetAll(ctx context.Context) ([]*domain.Stock, error) {
	var stocks []*domain.Stock
	err := r.db.WithContext(ctx).Order("symbol").Find(&stocks).Error // Use GORM's Find method
	if err != nil {
		return nil, err
	}
	return stocks, nil
}

// GetSelected retrieves all selected stocks
func (r *StockRepository) GetSelected(ctx context.Context) ([]*domain.Stock, error) {
	var stocks []*domain.Stock
	err := r.db.WithContext(ctx).Where("is_selected = ? AND active = 1", true).Order("symbol").Find(&stocks).Error // Use GORM's Where and Find methods
	if err != nil {
		return nil, err
	}
	return stocks, nil
}

// Update updates a stock
func (r *StockRepository) Update(ctx context.Context, stock *domain.Stock) error {
	return r.db.WithContext(ctx).Save(stock).Error // Use GORM's Save method
}

// ToggleSelection toggles the selection status of a stock
func (r *StockRepository) ToggleSelection(ctx context.Context, id string, isSelected bool) error {
	return r.db.WithContext(ctx).Model(&domain.Stock{}).Where("id = ? AND active = 1", id).
		Update("is_selected", isSelected).Error // Use GORM's Update method
}

// Delete deletes a stock
func (r *StockRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.Stock{}, "id = ? AND active = 1", id).Error
}
