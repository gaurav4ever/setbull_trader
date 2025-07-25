package postgres

import (
	"context"
	"errors"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Candle5MinRepository implements repository.Candle5MinRepository using PostgreSQL
type Candle5MinRepository struct {
	db *gorm.DB
}

// NewCandle5MinRepository creates a new Candle5MinRepository
func NewCandle5MinRepository(db *gorm.DB) repository.Candle5MinRepository {
	return &Candle5MinRepository{db: db}
}

// Store stores a single 5-minute candle record
func (r *Candle5MinRepository) Store(ctx context.Context, candle *domain.Candle5Min) error {
	// Set the time interval to 5minute for 5-minute candles
	candle.TimeInterval = "5minute"

	result := r.db.WithContext(ctx).
		Table("stock_candle_data_5min").
		Omit("id").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "instrument_key"}, {Name: "timestamp"}, {Name: "time_interval"}},
			DoUpdates: clause.AssignmentColumns([]string{"open", "high", "low", "close", "volume", "open_interest", "bb_upper", "bb_middle", "bb_lower", "bb_width", "bb_width_normalized", "bb_width_normalized_percentage", "ema_5", "ema_9", "ema_20", "ema_50", "atr", "rsi", "vwap", "ma_9", "lowest_bb_width"}),
		}).
		Create(candle)

	if result.Error != nil {
		return fmt.Errorf("failed to store 5-minute candle: %w", result.Error)
	}

	return nil
}

// StoreBatch stores multiple 5-minute candle records in a batch operation
func (r *Candle5MinRepository) StoreBatch(ctx context.Context, candles []domain.Candle5Min) (int, error) {
	if len(candles) == 0 {
		return 0, nil
	}

	// Set time interval for all candles
	for i := range candles {
		candles[i].TimeInterval = "5minute"
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
		Table("stock_candle_data_5min").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "instrument_key"}, {Name: "timestamp"}, {Name: "time_interval"}},
			DoUpdates: clause.AssignmentColumns([]string{"open", "high", "low", "close", "volume", "open_interest", "bb_upper", "bb_middle", "bb_lower", "bb_width", "bb_width_normalized", "bb_width_normalized_percentage", "ema_5", "ema_9", "ema_20", "ema_50", "atr", "rsi", "vwap", "ma_9", "lowest_bb_width"}),
		}).
		CreateInBatches(candles, 1000)

	if result.Error != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to store 5-minute candles: %w", result.Error)
	}

	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return len(candles), nil
}

// FindByInstrumentKey retrieves all 5-minute candles for a specific instrument
func (r *Candle5MinRepository) FindByInstrumentKey(ctx context.Context, instrumentKey string) ([]domain.Candle5Min, error) {
	var candles []domain.Candle5Min

	result := r.db.WithContext(ctx).
		Table("stock_candle_data_5min").
		Where("instrument_key = ? AND active = ?", instrumentKey, true).
		Order("timestamp").
		Find(&candles)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to find 5-minute candles by instrument key: %w", result.Error)
	}

	return candles, nil
}

// FindByInstrumentAndTimeRange retrieves 5-minute candles for an instrument within a time range
func (r *Candle5MinRepository) FindByInstrumentAndTimeRange(
	ctx context.Context,
	instrumentKey string,
	fromTime,
	toTime time.Time,
) ([]domain.Candle5Min, error) {
	var candles []domain.Candle5Min

	result := r.db.WithContext(ctx).
		Table("stock_candle_data_5min").
		Where("instrument_key = ? AND timestamp >= ? AND timestamp <= ? AND active = ?",
			instrumentKey, fromTime, toTime, true).
		Order("timestamp").
		Find(&candles)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to find 5-minute candles by instrument and time range: %w", result.Error)
	}

	return candles, nil
}

// DeleteByInstrumentAndTimeRange deletes 5-minute candles for an instrument within a time range
func (r *Candle5MinRepository) DeleteByInstrumentAndTimeRange(
	ctx context.Context,
	instrumentKey string,
	fromTime,
	toTime time.Time,
) (int, error) {
	result := r.db.WithContext(ctx).
		Table("stock_candle_data_5min").
		Where("instrument_key = ? AND timestamp >= ? AND timestamp <= ?",
			instrumentKey, fromTime, toTime).
		Update("active", false)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete 5-minute candles by instrument and time range: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}

// CountByInstrumentAndTimeRange counts 5-minute candles for an instrument within a time range
func (r *Candle5MinRepository) CountByInstrumentAndTimeRange(
	ctx context.Context,
	instrumentKey string,
	fromTime,
	toTime time.Time,
) (int, error) {
	var count int64

	result := r.db.WithContext(ctx).
		Table("stock_candle_data_5min").
		Where("instrument_key = ? AND timestamp >= ? AND timestamp <= ? AND active = ?",
			instrumentKey, fromTime, toTime, true).
		Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to count 5-minute candles by instrument and time range: %w", result.Error)
	}

	return int(count), nil
}

// DeleteOlderThan deletes 5-minute candles older than a specified time
func (r *Candle5MinRepository) DeleteOlderThan(ctx context.Context, olderThan time.Time) (int, error) {
	result := r.db.WithContext(ctx).
		Table("stock_candle_data_5min").
		Where("timestamp < ?", olderThan).
		Update("active", false)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete 5-minute candles older than %v: %w", olderThan, result.Error)
	}

	return int(result.RowsAffected), nil
}

// GetLatestCandle retrieves the most recent 5-minute candle for a specific instrument
func (r *Candle5MinRepository) GetLatestCandle(ctx context.Context, instrumentKey string) (*domain.Candle5Min, error) {
	var candle domain.Candle5Min

	result := r.db.WithContext(ctx).
		Table("stock_candle_data_5min").
		Where("instrument_key = ? AND active = ?", instrumentKey, true).
		Order("timestamp DESC").
		First(&candle)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest 5-minute candle: %w", result.Error)
	}

	return &candle, nil
}

// GetEarliestCandle retrieves the oldest 5-minute candle for a specific instrument
func (r *Candle5MinRepository) GetEarliestCandle(ctx context.Context, instrumentKey string) (*domain.Candle5Min, error) {
	var candle domain.Candle5Min

	result := r.db.WithContext(ctx).
		Table("stock_candle_data_5min").
		Where("instrument_key = ? AND active = ?", instrumentKey, true).
		Order("timestamp ASC").
		First(&candle)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get earliest 5-minute candle: %w", result.Error)
	}

	return &candle, nil
}

// GetCandleDateRange retrieves the earliest and latest timestamps for 5-minute candles of a specific instrument
func (r *Candle5MinRepository) GetCandleDateRange(ctx context.Context, instrumentKey string) (earliest, latest time.Time, exists bool, err error) {
	type DateRange struct {
		EarliestDate time.Time `gorm:"column:earliest_date"`
		LatestDate   time.Time `gorm:"column:latest_date"`
	}

	var dateRange DateRange

	result := r.db.WithContext(ctx).
		Table("stock_candle_data_5min").
		Select("MIN(timestamp) as earliest_date, MAX(timestamp) as latest_date").
		Where("instrument_key = ? AND active = ?", instrumentKey, true).
		Scan(&dateRange)

	if result.Error != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("failed to get 5-minute candle date range: %w", result.Error)
	}

	// Check if any records exist
	if dateRange.EarliestDate.IsZero() || dateRange.LatestDate.IsZero() {
		return time.Time{}, time.Time{}, false, nil
	}

	return dateRange.EarliestDate, dateRange.LatestDate, true, nil
}

// GetNLatestCandles retrieves the N most recent 5-minute candles for a specific instrument
func (r *Candle5MinRepository) GetNLatestCandles(ctx context.Context, instrumentKey string, n int) ([]domain.Candle5Min, error) {
	var candles []domain.Candle5Min

	result := r.db.WithContext(ctx).
		Table("stock_candle_data_5min").
		Where("instrument_key = ? AND active = ?", instrumentKey, true).
		Order("timestamp DESC").
		Limit(n).
		Find(&candles)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get N latest 5-minute candles: %w", result.Error)
	}

	// Reverse the order to get chronological order
	for i, j := 0, len(candles)-1; i < j; i, j = i+1, j-1 {
		candles[i], candles[j] = candles[j], candles[i]
	}

	return candles, nil
}

// UpdateCandlesInRangeCount updates the candles_in_range_count for the latest candle of a specific instrument
func (r *Candle5MinRepository) UpdateCandlesInRangeCount(ctx context.Context, instrumentKey string, count int) error {
	result := r.db.WithContext(ctx).
		Table("stock_candle_data_5min").
		Where("instrument_key = ? AND timestamp = (SELECT MAX(timestamp) FROM stock_candle_data_5min WHERE instrument_key = ? AND active = ?)",
			instrumentKey, instrumentKey, true).
		Update("candles_in_range_count", count)

	if result.Error != nil {
		return fmt.Errorf("failed to update candles_in_range_count: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no rows updated for instrument_key: %s", instrumentKey)
	}

	return nil
}
