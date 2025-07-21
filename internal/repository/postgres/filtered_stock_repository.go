package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FilteredStockRepository implements repository.FilteredStockRepository using PostgreSQL
type FilteredStockRepository struct {
	db *gorm.DB
}

// NewFilteredStockRepository creates a new FilteredStockRepository
func NewFilteredStockRepository(db *gorm.DB) repository.FilteredStockRepository {
	return &FilteredStockRepository{db: db}
}

// Store stores a single filtered stock record
func (r *FilteredStockRepository) Store(ctx context.Context, record *domain.FilteredStockRecord) error {
	// Convert slices to JSON
	mambaSeriesJSON, err := json.Marshal(record.MambaSeries)
	if err != nil {
		return fmt.Errorf("failed to marshal mamba series: %w", err)
	}

	nonMambaSeriesJSON, err := json.Marshal(record.NonMambaSeries)
	if err != nil {
		return fmt.Errorf("failed to marshal non-mamba series: %w", err)
	}

	// Create record map with JSON fields
	recordMap := map[string]interface{}{
		"symbol":              record.Symbol,
		"instrument_key":      record.InstrumentKey,
		"exchange_token":      record.ExchangeToken,
		"current_price":       record.CurrentPrice,
		"mamba_count":         record.MambaCount,
		"bullish_mamba_count": record.BullishMambaCount,
		"bearish_mamba_count": record.BearishMambaCount,
		"avg_mamba_move":      record.AvgMambaMove,
		"avg_non_mamba_move":  record.AvgNonMambaMove,
		"mamba_series":        string(mambaSeriesJSON),
		"non_mamba_series":    string(nonMambaSeriesJSON),
		"filter_date":         record.FilterDate,
	}

	result := r.db.WithContext(ctx).
		Table("filtered_stocks").
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "symbol"}, {Name: "filter_date"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"current_price",
				"mamba_count",
				"bullish_mamba_count",
				"bearish_mamba_count",
				"avg_mamba_move",
				"avg_non_mamba_move",
				"mamba_series",
				"non_mamba_series",
			}),
		}).
		Create(recordMap)

	if result.Error != nil {
		return fmt.Errorf("failed to store filtered stock record: %w", result.Error)
	}

	return nil
}

// StoreBatch stores multiple filtered stock records in a batch operation
func (r *FilteredStockRepository) StoreBatch(ctx context.Context, records []domain.FilteredStockRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Use a transaction for batch operations
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Error("Panic in StoreBatch: %v", r)
		}
	}()

	// Convert records to maps with JSON fields
	recordMaps := make([]map[string]interface{}, len(records))
	for i, record := range records {
		mambaSeriesJSON, err := json.Marshal(record.MambaSeries)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to marshal mamba series: %w", err)
		}

		nonMambaSeriesJSON, err := json.Marshal(record.NonMambaSeries)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to marshal non-mamba series: %w", err)
		}

		recordMaps[i] = map[string]interface{}{
			"symbol":              record.Symbol,
			"instrument_key":      record.InstrumentKey,
			"exchange_token":      record.ExchangeToken,
			"trend":               record.Trend,
			"current_price":       record.CurrentPrice,
			"mamba_count":         record.MambaCount,
			"bullish_mamba_count": record.BullishMambaCount,
			"bearish_mamba_count": record.BearishMambaCount,
			"avg_mamba_move":      record.AvgMambaMove,
			"avg_non_mamba_move":  record.AvgNonMambaMove,
			"mamba_series":        string(mambaSeriesJSON),
			"non_mamba_series":    string(nonMambaSeriesJSON),
			"filter_date":         record.FilterDate,
		}
	}

	// Batch insert with proper table name
	result := tx.Table("filtered_stocks").
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "symbol"}, {Name: "filter_date"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"trend",
				"current_price",
				"mamba_count",
				"bullish_mamba_count",
				"bearish_mamba_count",
				"avg_mamba_move",
				"avg_non_mamba_move",
				"mamba_series",
				"non_mamba_series",
			}),
		}).
		CreateInBatches(recordMaps, 100)

	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to store filtered stock records: %w", result.Error)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetBySymbol retrieves filtered stock records for a specific symbol
func (r *FilteredStockRepository) GetBySymbol(ctx context.Context, symbol string) ([]domain.FilteredStockRecord, error) {
	var records []domain.FilteredStockRecord

	result := r.db.WithContext(ctx).
		Table("filtered_stocks").
		Where("symbol = ?", symbol).
		Order("filter_date DESC").
		Find(&records)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get filtered stock records by symbol: %w", result.Error)
	}

	// Unmarshal JSON arrays
	for i := range records {
		if err := json.Unmarshal([]byte(records[i].MambaSeries.(string)), &records[i].MambaSeries); err != nil {
			return nil, fmt.Errorf("failed to unmarshal mamba series: %w", err)
		}
		if err := json.Unmarshal([]byte(records[i].NonMambaSeries.(string)), &records[i].NonMambaSeries); err != nil {
			return nil, fmt.Errorf("failed to unmarshal non-mamba series: %w", err)
		}
	}

	return records, nil
}

// GetByDate retrieves all filtered stocks for a specific date
func (r *FilteredStockRepository) GetByDate(ctx context.Context, date time.Time) ([]domain.FilteredStockRecord, error) {
	var records []domain.FilteredStockRecord

	result := r.db.WithContext(ctx).
		Table("filtered_stocks").
		Where("DATE(filter_date) = DATE(?)", date).
		Order("mamba_count DESC").
		Find(&records)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get filtered stock records by date: %w", result.Error)
	}

	return records, nil
}

// GetByDateRange retrieves filtered stocks within a date range
func (r *FilteredStockRepository) GetByDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.FilteredStockRecord, error) {
	var records []domain.FilteredStockRecord

	result := r.db.WithContext(ctx).
		Table("filtered_stocks").
		Where("filter_date BETWEEN ? AND ?", startDate, endDate).
		Order("filter_date DESC, mamba_count DESC").
		Find(&records)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get filtered stock records by date range: %w", result.Error)
	}

	return records, nil
}

// DeleteOlderThan deletes filtered stock records older than the specified date
func (r *FilteredStockRepository) DeleteOlderThan(ctx context.Context, date time.Time) (int, error) {
	result := r.db.WithContext(ctx).
		Table("filtered_stocks").
		Where("filter_date < ?", date).
		Delete(&domain.FilteredStockRecord{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old filtered stock records: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}

// GetLatestBySymbol retrieves the most recent filtered stock record for a symbol
func (r *FilteredStockRepository) GetLatestBySymbol(ctx context.Context, symbol string) (*domain.FilteredStockRecord, error) {
	var record domain.FilteredStockRecord

	result := r.db.WithContext(ctx).
		Table("filtered_stocks").
		Where("symbol = ?", symbol).
		Order("filter_date DESC").
		First(&record)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest filtered stock record: %w", result.Error)
	}

	return &record, nil
}

// GetTop10FilteredStocks retrieves all filtered stocks for the latest filter date, ordered by filter_date DESC, mamba_count DESC
func (r *FilteredStockRepository) GetTop10FilteredStocks(ctx context.Context) ([]domain.FilteredStockRecord, error) {
	var records []domain.FilteredStockRecord

	query := `
		SELECT fs.*
		FROM filtered_stocks fs
		WHERE fs.filter_date = (
			SELECT MAX(filter_date) 
			FROM filtered_stocks
		)
		ORDER BY fs.filter_date DESC, fs.mamba_count DESC
	`

	result := r.db.WithContext(ctx).Raw(query).Scan(&records)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get filtered stocks for latest date: %w", result.Error)
	}

	return records, nil
}
