package v2

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// StrategyParametersRepositoryImpl implements StrategyParametersRepository
type StrategyParametersRepositoryImpl struct {
	db *sqlx.DB
}

// NewStrategyParametersRepository creates a new strategy parameters repository
func NewStrategyParametersRepository(db *sqlx.DB) StrategyParametersRepository {
	return &StrategyParametersRepositoryImpl{
		db: db,
	}
}

// SaveParameters saves parameters for a specific stock, strategy, and candle
func (r *StrategyParametersRepositoryImpl) SaveParameters(stockID, strategyName string, timestamp time.Time, params *StrategyParameters) error {
	query := `
		INSERT INTO strategy_parameters_v2 (
			stock_id, strategy_name, candle_timestamp,
			can_generate_long, can_generate_short,
			mr_high, mr_low, mr_high_with_buffer, mr_low_with_buffer, buffer_percentage,
			entry_time, range_high, range_low, range_high_entry_price, range_low_entry_price, direction,
			bb_width_threshold, bb_period, bb_std_dev, current_bb_width, lowest_bb_width,
			squeeze_detected, squeeze_start_time, squeeze_candle_count,
			bb_upper, bb_lower, bb_middle,
			strategy_state, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29
		)
		ON CONFLICT (stock_id, strategy_name, candle_timestamp) 
		DO UPDATE SET
			can_generate_long = EXCLUDED.can_generate_long,
			can_generate_short = EXCLUDED.can_generate_short,
			mr_high = EXCLUDED.mr_high,
			mr_low = EXCLUDED.mr_low,
			mr_high_with_buffer = EXCLUDED.mr_high_with_buffer,
			mr_low_with_buffer = EXCLUDED.mr_low_with_buffer,
			buffer_percentage = EXCLUDED.buffer_percentage,
			entry_time = EXCLUDED.entry_time,
			range_high = EXCLUDED.range_high,
			range_low = EXCLUDED.range_low,
			range_high_entry_price = EXCLUDED.range_high_entry_price,
			range_low_entry_price = EXCLUDED.range_low_entry_price,
			direction = EXCLUDED.direction,
			bb_width_threshold = EXCLUDED.bb_width_threshold,
			bb_period = EXCLUDED.bb_period,
			bb_std_dev = EXCLUDED.bb_std_dev,
			current_bb_width = EXCLUDED.current_bb_width,
			lowest_bb_width = EXCLUDED.lowest_bb_width,
			squeeze_detected = EXCLUDED.squeeze_detected,
			squeeze_start_time = EXCLUDED.squeeze_start_time,
			squeeze_candle_count = EXCLUDED.squeeze_candle_count,
			bb_upper = EXCLUDED.bb_upper,
			bb_lower = EXCLUDED.bb_lower,
			bb_middle = EXCLUDED.bb_middle,
			strategy_state = EXCLUDED.strategy_state,
			metadata = EXCLUDED.metadata,
			updated_at = CURRENT_TIMESTAMP
	`

	// Convert strategy state and metadata to JSON
	strategyStateJSON, err := json.Marshal(params.StrategyState)
	if err != nil {
		return fmt.Errorf("failed to marshal strategy state: %w", err)
	}

	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.Exec(query,
		stockID, strategyName, timestamp,
		params.CanGenerateLong, params.CanGenerateShort,
		params.MRHigh, params.MRLow, params.MRHighWithBuffer, params.MRLowWithBuffer, params.BufferPercentage,
		params.EntryTime, params.RangeHigh, params.RangeLow, params.RangeHighEntryPrice, params.RangeLowEntryPrice, params.Direction,
		params.BBWidthThreshold, params.BBPeriod, params.BBStdDev, params.CurrentBBWidth, params.LowestBBWidth,
		params.SqueezeDetected, params.SqueezeStartTime, params.SqueezeCandleCount,
		params.BBUpper, params.BBLower, params.BBMiddle,
		strategyStateJSON, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to save strategy parameters: %w", err)
	}

	return nil
}

// GetParameters gets parameters for a specific stock, strategy, and candle
func (r *StrategyParametersRepositoryImpl) GetParameters(stockID, strategyName string, timestamp time.Time) (*StrategyParameters, error) {
	query := `
		SELECT 
			id, stock_id, strategy_name, candle_timestamp, created_at, updated_at, active,
			can_generate_long, can_generate_short,
			mr_high, mr_low, mr_high_with_buffer, mr_low_with_buffer, buffer_percentage,
			entry_time, range_high, range_low, range_high_entry_price, range_low_entry_price, direction,
			bb_width_threshold, bb_period, bb_std_dev, current_bb_width, lowest_bb_width,
			squeeze_detected, squeeze_start_time, squeeze_candle_count,
			bb_upper, bb_lower, bb_middle,
			strategy_state, metadata
		FROM strategy_parameters_v2
		WHERE stock_id = $1 AND strategy_name = $2 AND candle_timestamp = $3 AND active = true
	`

	var record StrategyParametersRecord
	err := r.db.Get(&record, query, stockID, strategyName, timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get strategy parameters: %w", err)
	}

	// Unmarshal JSON fields
	if record.StrategyState == nil {
		record.StrategyState = make(map[string]interface{})
	}
	if record.Metadata == nil {
		record.Metadata = make(map[string]interface{})
	}

	return &record.StrategyParameters, nil
}

// GetParametersRange gets parameters for a stock and strategy within a time range
func (r *StrategyParametersRepositoryImpl) GetParametersRange(stockID, strategyName string, startTime, endTime time.Time) ([]*StrategyParametersRecord, error) {
	query := `
		SELECT 
			id, stock_id, strategy_name, candle_timestamp, created_at, updated_at, active,
			can_generate_long, can_generate_short,
			mr_high, mr_low, mr_high_with_buffer, mr_low_with_buffer, buffer_percentage,
			entry_time, range_high, range_low, range_high_entry_price, range_low_entry_price, direction,
			bb_width_threshold, bb_period, bb_std_dev, current_bb_width, lowest_bb_width,
			squeeze_detected, squeeze_start_time, squeeze_candle_count,
			bb_upper, bb_lower, bb_middle,
			strategy_state, metadata
		FROM strategy_parameters_v2
		WHERE stock_id = $1 AND strategy_name = $2 AND candle_timestamp BETWEEN $3 AND $4 AND active = true
		ORDER BY candle_timestamp ASC
	`

	var records []*StrategyParametersRecord
	err := r.db.Select(&records, query, stockID, strategyName, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy parameters range: %w", err)
	}

	// Initialize JSON fields for each record
	for _, record := range records {
		if record.StrategyState == nil {
			record.StrategyState = make(map[string]interface{})
		}
		if record.Metadata == nil {
			record.Metadata = make(map[string]interface{})
		}
	}

	return records, nil
}

// GetLatestParameters gets the latest parameters for a stock and strategy
func (r *StrategyParametersRepositoryImpl) GetLatestParameters(stockID, strategyName string) (*StrategyParameters, error) {
	query := `
		SELECT 
			can_generate_long, can_generate_short,
			mr_high, mr_low, mr_high_with_buffer, mr_low_with_buffer, buffer_percentage,
			entry_time, range_high, range_low, range_high_entry_price, range_low_entry_price, direction,
			bb_width_threshold, bb_period, bb_std_dev, current_bb_width, lowest_bb_width,
			squeeze_detected, squeeze_start_time, squeeze_candle_count,
			bb_upper, bb_lower, bb_middle,
			strategy_state, metadata
		FROM strategy_parameters_v2
		WHERE stock_id = $1 AND strategy_name = $2 AND active = true
		ORDER BY candle_timestamp DESC
		LIMIT 1
	`

	var params StrategyParameters
	err := r.db.Get(&params, query, stockID, strategyName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get latest strategy parameters: %w", err)
	}

	// Initialize JSON fields
	if params.StrategyState == nil {
		params.StrategyState = make(map[string]interface{})
	}
	if params.Metadata == nil {
		params.Metadata = make(map[string]interface{})
	}

	return &params, nil
}

// UpdateParameters updates parameters for a specific record
func (r *StrategyParametersRepositoryImpl) UpdateParameters(id int64, params *StrategyParameters) error {
	query := `
		UPDATE strategy_parameters_v2 SET
			can_generate_long = $2, can_generate_short = $3,
			mr_high = $6, mr_low = $7, mr_high_with_buffer = $8, mr_low_with_buffer = $9, buffer_percentage = $10,
			entry_time = $11, range_high = $12, range_low = $13, range_high_entry_price = $14, range_low_entry_price = $15, direction = $16,
			bb_width_threshold = $17, bb_period = $18, bb_std_dev = $19, current_bb_width = $20, lowest_bb_width = $21,
			squeeze_detected = $22, squeeze_start_time = $23, squeeze_candle_count = $24,
			bb_upper = $25, bb_lower = $26, bb_middle = $27,
			strategy_state = $28, metadata = $29, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND active = true
	`

	// Convert strategy state and metadata to JSON
	strategyStateJSON, err := json.Marshal(params.StrategyState)
	if err != nil {
		return fmt.Errorf("failed to marshal strategy state: %w", err)
	}

	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := r.db.Exec(query,
		id,
		params.CanGenerateLong, params.CanGenerateShort,
		params.MRHigh, params.MRLow, params.MRHighWithBuffer, params.MRLowWithBuffer, params.BufferPercentage,
		params.EntryTime, params.RangeHigh, params.RangeLow, params.RangeHighEntryPrice, params.RangeLowEntryPrice, params.Direction,
		params.BBWidthThreshold, params.BBPeriod, params.BBStdDev, params.CurrentBBWidth, params.LowestBBWidth,
		params.SqueezeDetected, params.SqueezeStartTime, params.SqueezeCandleCount,
		params.BBUpper, params.BBLower, params.BBMiddle,
		strategyStateJSON, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update strategy parameters: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// DeleteParameters deletes parameters for a specific stock, strategy, and candle
func (r *StrategyParametersRepositoryImpl) DeleteParameters(stockID, strategyName string, timestamp time.Time) error {
	query := `
		UPDATE strategy_parameters_v2 
		SET active = false, updated_at = CURRENT_TIMESTAMP
		WHERE stock_id = $1 AND strategy_name = $2 AND candle_timestamp = $3 AND active = true
	`

	result, err := r.db.Exec(query, stockID, strategyName, timestamp)
	if err != nil {
		return fmt.Errorf("failed to delete strategy parameters: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// CleanupOldParameters removes old parameters based on data retention policy
func (r *StrategyParametersRepositoryImpl) CleanupOldParameters(beforeTime time.Time) error {
	query := `
		UPDATE strategy_parameters_v2 
		SET active = false, updated_at = CURRENT_TIMESTAMP
		WHERE candle_timestamp < $1 AND active = true
	`

	result, err := r.db.Exec(query, beforeTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old strategy parameters: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log cleanup results
	fmt.Printf("Cleaned up %d old strategy parameter records before %s\n", rowsAffected, beforeTime.Format("2006-01-02 15:04:05"))

	return nil
}
