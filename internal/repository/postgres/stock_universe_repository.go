package postgres

import (
	"context"
	"errors"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StockUniverseRepository implements repository.StockUniverseRepository interface using PostgreSQL
type StockUniverseRepository struct {
	db *gorm.DB
}

// NewStockUniverseRepository creates a new StockUniverseRepository
func NewStockUniverseRepository(db *gorm.DB) repository.StockUniverseRepository {
	return &StockUniverseRepository{
		db: db,
	}
}

// Create inserts a new stock into the database
func (r *StockUniverseRepository) Create(ctx context.Context, stock *domain.StockUniverse) (*domain.StockUniverse, error) {
	if err := r.db.WithContext(ctx).Create(stock).Error; err != nil {
		return nil, fmt.Errorf("failed to create stock: %w", err)
	}
	return stock, nil
}

// BulkUpsert inserts or updates multiple stocks in a single transaction
func (r *StockUniverseRepository) BulkUpsert(ctx context.Context, stocks []domain.StockUniverse) (int, int, error) {
	if len(stocks) == 0 {
		return 0, 0, nil
	}

	// Start a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Use a defer to ensure the transaction is rolled back if an error occurs
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Prepare for counting created vs updated records
	created := 0
	updated := 0

	// Process in batches of 100 to avoid overwhelming the database
	batchSize := 100
	for i := 0; i < len(stocks); i += batchSize {
		end := i + batchSize
		if end > len(stocks) {
			end = len(stocks)
		}

		batch := stocks[i:end]

		// Use Upsert operation (insert or update if exists)
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "symbol"}}, // Conflict on symbol
			UpdateAll: true,                              // Update all fields if conflict
		}).Create(&batch)

		if result.Error != nil {
			tx.Rollback()
			return 0, 0, fmt.Errorf("failed to upsert stocks batch: %w", result.Error)
		}

		// We can't directly know how many were created vs updated from GORM
		// So we'll query to check which ones existed before
		for _, stock := range batch {
			var count int64
			tx.Model(&domain.StockUniverse{}).
				Where("symbol = ? AND created_at != updated_at", stock.Symbol).
				Count(&count)

			if count > 0 {
				updated++
			} else {
				created++
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return created, updated, nil
}

// GetBySymbol retrieves a stock by its symbol
func (r *StockUniverseRepository) GetBySymbol(ctx context.Context, symbol string) (*domain.StockUniverse, error) {
	var stock domain.StockUniverse
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&stock).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("stock with symbol %s not found", symbol)
		}
		return nil, fmt.Errorf("failed to get stock by symbol: %w", err)
	}
	return &stock, nil
}

// GetAll retrieves all stocks with optional filtering
func (r *StockUniverseRepository) GetAll(
	ctx context.Context,
	onlySelected bool,
	limit, offset int,
) ([]domain.StockUniverse, int64, error) {
	var stocks []domain.StockUniverse
	var count int64

	// Build the query
	query := r.db.WithContext(ctx).Model(&domain.StockUniverse{})

	// Apply filters
	if onlySelected {
		query = query.Where("is_selected = ?", true)
	}

	// Get total count
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count stocks: %w", err)
	}

	// Apply pagination and get results
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Execute the query
	if err := query.Find(&stocks).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get stocks: %w", err)
	}

	return stocks, count, nil
}

// ToggleSelection toggles the is_selected flag for a stock
func (r *StockUniverseRepository) ToggleSelection(ctx context.Context, symbol string, isSelected bool) (*domain.StockUniverse, error) {
	// First get the current stock
	stock, err := r.GetBySymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// Set the selection status
	stock.IsSelected = isSelected

	// Update in the database
	if err := r.db.WithContext(ctx).Save(stock).Error; err != nil {
		return nil, fmt.Errorf("failed to toggle selection: %w", err)
	}

	return stock, nil
}

// DeleteBySymbol deletes a stock by its symbol
func (r *StockUniverseRepository) DeleteBySymbol(ctx context.Context, symbol string) error {
	result := r.db.WithContext(ctx).Where("symbol = ?", symbol).Delete(&domain.StockUniverse{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete stock: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("stock with symbol %s not found", symbol)
	}
	return nil
}
