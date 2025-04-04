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

// GetLatestCandle retrieves the most recent candle for a specific instrument and interval
func (r *CandleRepository) GetLatestCandle(
	ctx context.Context,
	instrumentKey string,
	interval string,
) (*domain.CandleData, error) {
	var candle domain.CandleData

	result := r.db.WithContext(ctx).
		Where("instrument_key = ? AND time_interval = ?", instrumentKey, interval).
		Order("timestamp DESC").
		First(&candle)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // No candle found
		}
		return nil, fmt.Errorf("failed to get latest candle: %w", result.Error)
	}

	return &candle, nil
}

// GetAggregated5MinCandles retrieves 5-minute candles aggregated from 1-minute candles
func (r *CandleRepository) GetAggregated5MinCandles(
	ctx context.Context,
	instrumentKey string,
	start, end time.Time,
) ([]domain.AggregatedCandle, error) {
	// Use raw SQL for complex aggregation
	var aggregatedCandles []domain.AggregatedCandle

	// Create temporary table with candles grouped by 5-minute intervals
	if err := r.db.WithContext(ctx).Exec(`
		CREATE TEMPORARY TABLE IF NOT EXISTS temp_5min_candles AS
		SELECT 
			instrument_key,
			FROM_UNIXTIME(FLOOR(UNIX_TIMESTAMP(timestamp) / 300) * 300) AS interval_timestamp,
			MIN(timestamp) AS first_timestamp,
			MAX(timestamp) AS last_timestamp,
			MAX(high) AS high_price,
			MIN(low) AS low_price,
			SUM(volume) AS total_volume
		FROM 
			stock_candle_data
		WHERE 
			time_interval = '1minute'
			AND instrument_key = ?
			AND timestamp BETWEEN ? AND ?
		GROUP BY 
			instrument_key, interval_timestamp
	`, instrumentKey, start, end).Error; err != nil {
		return nil, fmt.Errorf("failed to create temp_5min_candles: %w", err)
	}
	defer r.db.Exec("DROP TEMPORARY TABLE IF EXISTS temp_5min_candles")

	// Get open prices from first candles
	if err := r.db.WithContext(ctx).Exec(`
		CREATE TEMPORARY TABLE IF NOT EXISTS temp_5min_open_prices AS
		SELECT 
			t.instrument_key,
			t.interval_timestamp,
			scd.open AS open_price
		FROM 
			temp_5min_candles t
		JOIN 
			stock_candle_data scd ON scd.instrument_key = t.instrument_key 
				AND scd.timestamp = t.first_timestamp
	`).Error; err != nil {
		return nil, fmt.Errorf("failed to create temp_5min_open_prices: %w", err)
	}
	defer r.db.Exec("DROP TEMPORARY TABLE IF EXISTS temp_5min_open_prices")

	// Get close prices from last candles
	if err := r.db.WithContext(ctx).Exec(`
		CREATE TEMPORARY TABLE IF NOT EXISTS temp_5min_close_prices AS
		SELECT 
			t.instrument_key,
			t.interval_timestamp,
			scd.close AS close_price,
			scd.open_interest AS open_interest
		FROM 
			temp_5min_candles t
		JOIN 
			stock_candle_data scd ON scd.instrument_key = t.instrument_key 
				AND scd.timestamp = t.last_timestamp
	`).Error; err != nil {
		return nil, fmt.Errorf("failed to create temp_5min_close_prices: %w", err)
	}
	defer r.db.Exec("DROP TEMPORARY TABLE IF EXISTS temp_5min_close_prices")

	// Query the final result
	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			t.instrument_key,
			t.interval_timestamp AS timestamp,
			o.open_price AS open,
			t.high_price AS high,
			t.low_price AS low,
			c.close_price AS close,
			t.total_volume AS volume,
			c.open_interest,
			'5minute' AS time_interval
		FROM 
			temp_5min_candles t
		JOIN 
			temp_5min_open_prices o ON t.instrument_key = o.instrument_key 
				AND t.interval_timestamp = o.interval_timestamp
		JOIN 
			temp_5min_close_prices c ON t.instrument_key = c.instrument_key 
				AND t.interval_timestamp = c.interval_timestamp
		ORDER BY 
			t.instrument_key, t.interval_timestamp
	`).Scan(&aggregatedCandles).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query aggregated 5-minute candles: %w", err)
	}

	log.Info("Retrieved %d aggregated 5-minute candles for instrument %s", len(aggregatedCandles), instrumentKey)
	return aggregatedCandles, nil
}

// GetAggregatedDailyCandles retrieves daily candles aggregated from 1-minute candles
func (r *CandleRepository) GetAggregatedDailyCandles(
	ctx context.Context,
	instrumentKey string,
	start, end time.Time,
) ([]domain.AggregatedCandle, error) {
	// Use raw SQL for complex aggregation
	var aggregatedCandles []domain.AggregatedCandle

	// Create temporary table with candles grouped by day
	if err := r.db.WithContext(ctx).Exec(`
		CREATE TEMPORARY TABLE IF NOT EXISTS temp_daily_candles AS
		SELECT 
			instrument_key,
			DATE(timestamp) AS interval_date,
			MIN(timestamp) AS first_timestamp,
			MAX(timestamp) AS last_timestamp,
			MAX(high) AS high_price,
			MIN(low) AS low_price,
			SUM(volume) AS total_volume
		FROM 
			stock_candle_data
		WHERE 
			time_interval = '1minute'
			AND instrument_key = ?
			AND timestamp BETWEEN ? AND ?
		GROUP BY 
			instrument_key, DATE(timestamp)
	`, instrumentKey, start, end).Error; err != nil {
		return nil, fmt.Errorf("failed to create temp_daily_candles: %w", err)
	}
	defer r.db.Exec("DROP TEMPORARY TABLE IF EXISTS temp_daily_candles")

	// Get open prices from first candles
	if err := r.db.WithContext(ctx).Exec(`
		CREATE TEMPORARY TABLE IF NOT EXISTS temp_daily_open_prices AS
		SELECT 
			t.instrument_key,
			t.interval_date,
			scd.open AS open_price
		FROM 
			temp_daily_candles t
		JOIN 
			stock_candle_data scd ON scd.instrument_key = t.instrument_key 
				AND scd.timestamp = t.first_timestamp
	`).Error; err != nil {
		return nil, fmt.Errorf("failed to create temp_daily_open_prices: %w", err)
	}
	defer r.db.Exec("DROP TEMPORARY TABLE IF EXISTS temp_daily_open_prices")

	// Get close prices from last candles
	if err := r.db.WithContext(ctx).Exec(`
		CREATE TEMPORARY TABLE IF NOT EXISTS temp_daily_close_prices AS
		SELECT 
			t.instrument_key,
			t.interval_date,
			scd.close AS close_price,
			scd.open_interest AS open_interest
		FROM 
			temp_daily_candles t
		JOIN 
			stock_candle_data scd ON scd.instrument_key = t.instrument_key 
				AND scd.timestamp = t.last_timestamp
	`).Error; err != nil {
		return nil, fmt.Errorf("failed to create temp_daily_close_prices: %w", err)
	}
	defer r.db.Exec("DROP TEMPORARY TABLE IF EXISTS temp_daily_close_prices")

	// Query the final result
	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			t.instrument_key,
			TIMESTAMP(t.interval_date) AS timestamp,
			o.open_price AS open,
			t.high_price AS high,
			t.low_price AS low,
			c.close_price AS close,
			t.total_volume AS volume,
			c.open_interest,
			'day' AS time_interval
		FROM 
			temp_daily_candles t
		JOIN 
			temp_daily_open_prices o ON t.instrument_key = o.instrument_key 
				AND t.interval_date = o.interval_date
		JOIN 
			temp_daily_close_prices c ON t.instrument_key = c.instrument_key 
				AND t.interval_date = c.interval_date
		ORDER BY 
			t.instrument_key, t.interval_date
	`).Scan(&aggregatedCandles).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query aggregated daily candles: %w", err)
	}

	log.Info("Retrieved %d aggregated daily candles for instrument %s", len(aggregatedCandles), instrumentKey)
	return aggregatedCandles, nil
}

func (r *CandleRepository) GetDailyCandlesByTimeframe(
	ctx context.Context,
	instrumentKey string,
	startTime time.Time,
) ([]domain.Candle, error) {
	var candles []domain.Candle

	result := r.db.WithContext(ctx).
		Where("instrument_key = ? AND time_interval = ? AND timestamp >= ?",
			instrumentKey,
			"day",
			startTime,
		).
		Order("timestamp ASC").
		Find(&candles)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get daily candles: %w", result.Error)
	}

	log.Info("Retrieved %d daily candles for instrument %s from %s",
		len(candles),
		instrumentKey,
		startTime.Format("2006-01-02 15:04:05"))

	return candles, nil
}

// StoreAggregatedCandles stores aggregated candles (useful for caching common timeframes)
func (r *CandleRepository) StoreAggregatedCandles(ctx context.Context, candles []domain.CandleData) error {
	if len(candles) == 0 {
		return nil
	}

	// Use Create with batch size for better performance
	const batchSize = 100
	result := r.db.WithContext(ctx).CreateInBatches(candles, batchSize)
	if result.Error != nil {
		return fmt.Errorf("failed to store aggregated candles: %w", result.Error)
	}

	log.Info("Stored %d aggregated candles", result.RowsAffected)
	return nil
}

// GetStocksWithExistingDailyCandles returns a list of instrument keys that already have daily candle data
// for the specified date range
func (r *CandleRepository) GetStocksWithExistingDailyCandles(
	ctx context.Context,
	startDate, endDate time.Time,
) ([]string, error) {
	var instrumentKeys []string

	// Query for distinct instrument keys that have data in the date range
	result := r.db.WithContext(ctx).
		Model(&domain.Candle{}).
		Where("time_interval = ? AND timestamp BETWEEN ? AND ?", "day", startDate, endDate).
		Distinct().
		Pluck("instrument_key", &instrumentKeys)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get stocks with existing candles: %w", result.Error)
	}

	return instrumentKeys, nil
}
