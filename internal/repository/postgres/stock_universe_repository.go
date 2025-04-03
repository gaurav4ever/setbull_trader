package postgres

import (
	"context"
	"errors"
	"fmt"
	"setbull_trader/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StockUniverseRepository handles database operations for the StockUniverse entity
type StockUniverseRepository struct {
	db *gorm.DB
}

// NewStockUniverseRepository creates a new instance of StockUniverseRepository
// db: Database connection
func NewStockUniverseRepository(db *gorm.DB) *StockUniverseRepository {
	return &StockUniverseRepository{
		db: db,
	}
}

// Create inserts a new stock into the database
// Returns:
// - The created stock with its ID
// - Error if any occurred during creation
func (r *StockUniverseRepository) Create(ctx context.Context, stock *domain.StockUniverse) (*domain.StockUniverse, error) {
	if err := r.db.WithContext(ctx).Create(stock).Error; err != nil {
		return nil, fmt.Errorf("failed to create stock: %w", err)
	}
	return stock, nil
}

// BulkUpsert inserts or updates multiple stocks in a single transaction
// This is more efficient than individual inserts when processing many stocks
// Returns:
// - Number of stocks created
// - Number of stocks updated
// - Error if any occurred during the operation
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

		// Since we can't directly know how many were created vs updated from GORM,
		// we'll query to check which ones existed before to determine the count
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
// Returns:
// - The stock if found
// - Error if any occurred or if the stock was not found
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
// Parameters:
// - onlySelected: If true, only returns stocks that have is_selected=true
// - limit: Maximum number of stocks to return (0 means no limit)
// - offset: Number of stocks to skip (for pagination)
// Returns:
// - Slice of stocks
// - Total count of stocks matching the filter (before limit/offset)
// - Error if any occurred
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
// Returns:
// - The updated stock
// - Error if any occurred
func (r *StockUniverseRepository) ToggleSelection(ctx context.Context, symbol string) (*domain.StockUniverse, error) {
	// First get the current stock to check its current selection status
	stock, err := r.GetBySymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// Toggle the selection status
	stock.IsSelected = !stock.IsSelected

	// Update in the database
	if err := r.db.WithContext(ctx).Save(stock).Error; err != nil {
		return nil, fmt.Errorf("failed to toggle selection: %w", err)
	}

	return stock, nil
}

// DeleteBySymbol deletes a stock by its symbol
// Returns:
// - Error if any occurred
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
