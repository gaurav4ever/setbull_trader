package domain

import (
	"time"
)

// StrategyParameters represents the parameters tracked by each strategy
type StrategyParameters struct {
	// Common parameters across all strategies (signal generation only)
	CanGenerateLong  bool `json:"can_generate_long" db:"can_generate_long"`
	CanGenerateShort bool `json:"can_generate_short" db:"can_generate_short"`

	// 1ST_ENTRY strategy parameters
	MRHigh           *float64 `json:"mr_high,omitempty" db:"mr_high"`
	MRLow            *float64 `json:"mr_low,omitempty" db:"mr_low"`
	MRHighWithBuffer *float64 `json:"mr_high_with_buffer,omitempty" db:"mr_high_with_buffer"`
	MRLowWithBuffer  *float64 `json:"mr_low_with_buffer,omitempty" db:"mr_low_with_buffer"`
	BufferPercentage *float64 `json:"buffer_percentage,omitempty" db:"buffer_percentage"`
	MRCalculated     *bool    `json:"mr_calculated,omitempty" db:"mr_calculated"`

	// 2_30_ENTRY strategy parameters
	EntryTime           *time.Time `json:"entry_time,omitempty" db:"entry_time"`
	RangeHigh           *float64   `json:"range_high,omitempty" db:"range_high"`
	RangeLow            *float64   `json:"range_low,omitempty" db:"range_low"`
	RangeHighEntryPrice *float64   `json:"range_high_entry_price,omitempty" db:"range_high_entry_price"`
	RangeLowEntryPrice  *float64   `json:"range_low_entry_price,omitempty" db:"range_low_entry_price"`
	Direction           *string    `json:"direction,omitempty" db:"direction"`

	// BB_WIDTH_ENTRY strategy parameters
	BBWidthThreshold   *float64   `json:"bb_width_threshold,omitempty" db:"bb_width_threshold"`
	BBPeriod           *int       `json:"bb_period,omitempty" db:"bb_period"`
	BBStdDev           *float64   `json:"bb_std_dev,omitempty" db:"bb_std_dev"`
	CurrentBBWidth     *float64   `json:"current_bb_width,omitempty" db:"current_bb_width"`
	LowestBBWidth      *float64   `json:"lowest_bb_width,omitempty" db:"lowest_bb_width"`
	SqueezeDetected    bool       `json:"squeeze_detected" db:"squeeze_detected"`
	SqueezeStartTime   *time.Time `json:"squeeze_start_time,omitempty" db:"squeeze_start_time"`
	SqueezeCandleCount *int       `json:"squeeze_candle_count,omitempty" db:"squeeze_candle_count"`
	BBUpper            *float64   `json:"bb_upper,omitempty" db:"bb_upper"`
	BBLower            *float64   `json:"bb_lower,omitempty" db:"bb_lower"`
	BBMiddle           *float64   `json:"bb_middle,omitempty" db:"bb_middle"`

	// Strategy state and metadata
	StrategyState map[string]interface{} `json:"strategy_state,omitempty" db:"strategy_state"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// StrategyParametersRecord represents a database record for strategy parameters
type StrategyParametersRecord struct {
	ID              int64     `db:"id"`
	StockID         string    `db:"stock_id"`
	StrategyName    string    `db:"strategy_name"`
	CandleTimestamp time.Time `db:"candle_timestamp"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
	Active          bool      `db:"active"`
	StrategyParameters
}
