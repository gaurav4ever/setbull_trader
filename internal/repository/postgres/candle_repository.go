package postgres

import (
	"context"
	"errors"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CandleRepository implements repository.CandleRepository using PostgreSQL
type CandleRepository struct {
	db *gorm.DB
}

// NewCandleRepository creates a new CandleRepository
func NewCandleRepository(db *gorm.DB) repository.CandleRepository {
	return &CandleRepository{db: db}
}

// Store stores a single candle record
func (r *CandleRepository) Store(ctx context.Context, candle *domain.Candle) error {
	result := r.db.WithContext(ctx).
		Omit("id").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "instrument_key"}, {Name: "timestamp"}, {Name: "interval"}},
			DoUpdates: clause.AssignmentColumns([]string{"open", "high", "low", "close", "volume", "open_interest"}),
		}).
		Create(candle)

	if result.Error != nil {
		return fmt.Errorf("failed to store candle: %w", result.Error)
	}

	return nil
}

// StoreBatch stores multiple candle records in a batch operation
func (r *CandleRepository) StoreBatch(ctx context.Context, candles []domain.Candle) (int, error) {
	if len(candles) == 0 {
		return 0, nil
	}

	// Use a transaction for batch operations
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Error("Panic in StoreBatch: %v", r)
		}
	}()

	// Standard GORM batch insert with conflict handling
	result := tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "instrument_key"}, {Name: "timestamp"}, {Name: "time_interval"}},
			DoUpdates: clause.AssignmentColumns([]string{"open", "high", "low", "close", "volume", "open_interest"}),
		}).
		CreateInBatches(candles, 1000)

	if result.Error != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to store candles: %w", result.Error)
	}

	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return len(candles), nil
}

// FindByInstrumentKey retrieves all candles for a specific instrument
func (r *CandleRepository) FindByInstrumentKey(ctx context.Context, instrumentKey string) ([]domain.Candle, error) {
	var candles []domain.Candle

	result := r.db.WithContext(ctx).
		Where("instrument_key = ?", instrumentKey).
		Order("timestamp").
		Find(&candles)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to find candles by instrument key: %w", result.Error)
	}

	return candles, nil
}

// FindByInstrumentAndInterval retrieves candles for an instrument with a specific interval
func (r *CandleRepository) FindByInstrumentAndInterval(ctx context.Context, instrumentKey, interval string) ([]domain.Candle, error) {
	var candles []domain.Candle

	result := r.db.WithContext(ctx).
		Where("instrument_key = ? AND interval = ?", instrumentKey, interval).
		Order("timestamp").
		Find(&candles)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to find candles by instrument and interval: %w", result.Error)
	}

	return candles, nil
}

// FindByInstrumentAndTimeRange retrieves candles for an instrument within a time range
func (r *CandleRepository) FindByInstrumentAndTimeRange(
	ctx context.Context,
	instrumentKey string,
	interval string,
	fromTime,
	toTime time.Time,
) ([]domain.Candle, error) {
	var candles []domain.Candle

	result := r.db.WithContext(ctx).
		Where("instrument_key = ? AND interval = ? AND timestamp BETWEEN ? AND ?",
			instrumentKey, interval, fromTime, toTime).
		Order("timestamp").
		Find(&candles)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to find candles by time range: %w", result.Error)
	}

	return candles, nil
}

// DeleteByInstrumentAndTimeRange deletes candles for an instrument within a time range
func (r *CandleRepository) DeleteByInstrumentAndTimeRange(
	ctx context.Context,
	instrumentKey string,
	interval string,
	fromTime,
	toTime time.Time,
) (int, error) {
	result := r.db.WithContext(ctx).
		Where("instrument_key = ? AND interval = ? AND timestamp BETWEEN ? AND ?",
			instrumentKey, interval, fromTime, toTime).
		Delete(&domain.Candle{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete candles by time range: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}

// CountByInstrumentAndTimeRange counts candles for an instrument within a time range
func (r *CandleRepository) CountByInstrumentAndTimeRange(
	ctx context.Context,
	instrumentKey string,
	interval string,
	fromTime,
	toTime time.Time,
) (int, error) {
	var count int64

	result := r.db.WithContext(ctx).Model(&domain.Candle{}).
		Where("instrument_key = ? AND interval = ? AND timestamp BETWEEN ? AND ?",
			instrumentKey, interval, fromTime, toTime).
		Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to count candles by time range: %w", result.Error)
	}

	return int(count), nil
}

// DeleteOlderThan deletes candles older than a specified time
func (r *CandleRepository) DeleteOlderThan(ctx context.Context, olderThan time.Time) (int, error) {
	result := r.db.WithContext(ctx).
		Where("timestamp < ?", olderThan).
		Delete(&domain.Candle{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old candles: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}

// bulkInsertWithCopy performs a bulk insert using PostgreSQL COPY command
// This is much faster than individual inserts for large batches
func (r *CandleRepository) bulkInsertWithCopy(ctx context.Context, tx *gorm.DB, candles []domain.Candle) (int, error) {
	// Get the underlying SQL DB
	sqlDB, err := tx.DB()
	if err != nil {
		return 0, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Start a transaction if we're not already in one
	txn, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a statement for preparing the COPY
	stmt, err := txn.Prepare(pq.CopyIn("stock_candle_data",
		"id", "instrument_key", "timestamp", "open", "high", "low", "close",
		"volume", "open_interest", "time_interval", "created_at"))
	if err != nil {
		txn.Rollback()
		return 0, fmt.Errorf("failed to prepare copy statement: %w", err)
	}

	// Execute batch insert
	for _, candle := range candles {
		_, err = stmt.Exec(
			candle.ID,
			candle.InstrumentKey,
			candle.Timestamp,
			candle.Open,
			candle.High,
			candle.Low,
			candle.Close,
			candle.Volume,
			candle.OpenInterest,
			candle.TimeInterval,
			time.Now(),
		)
		if err != nil {
			stmt.Close()
			txn.Rollback()
			return 0, fmt.Errorf("failed to execute copy statement: %w", err)
		}
	}

	// Close the statement to complete the COPY
	if err = stmt.Close(); err != nil {
		txn.Rollback()
		return 0, fmt.Errorf("failed to close copy statement: %w", err)
	}

	// Commit the transaction
	if err = txn.Commit(); err != nil {
		// Check for unique constraint violations
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			// If we have duplicates, fall back to the normal insert method which handles conflicts
			return 0, errors.New("duplicate keys detected, falling back to normal insert")
		}
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return len(candles), nil
}
