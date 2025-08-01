package v2

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

// StrategyParameters represents the parameters tracked by each strategy
type StrategyParameters struct {
	// Common parameters across all strategies
	InLongTrade      bool `json:"in_long_trade" db:"in_long_trade"`
	InShortTrade     bool `json:"in_short_trade" db:"in_short_trade"`
	CanGenerateLong  bool `json:"can_generate_long" db:"can_generate_long"`
	CanGenerateShort bool `json:"can_generate_short" db:"can_generate_short"`

	// 1ST_ENTRY strategy parameters
	MRHigh           *float64 `json:"mr_high,omitempty" db:"mr_high"`
	MRLow            *float64 `json:"mr_low,omitempty" db:"mr_low"`
	MRHighWithBuffer *float64 `json:"mr_high_with_buffer,omitempty" db:"mr_high_with_buffer"`
	MRLowWithBuffer  *float64 `json:"mr_low_with_buffer,omitempty" db:"mr_low_with_buffer"`
	BufferPercentage *float64 `json:"buffer_percentage,omitempty" db:"buffer_percentage"`

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

// StrategyParametersRepository defines the interface for strategy parameters persistence
type StrategyParametersRepository interface {
	// Save parameters for a specific stock, strategy, and candle
	SaveParameters(stockID, strategyName string, timestamp time.Time, params *StrategyParameters) error

	// Get parameters for a specific stock, strategy, and candle
	GetParameters(stockID, strategyName string, timestamp time.Time) (*StrategyParameters, error)

	// Get parameters for a stock and strategy within a time range
	GetParametersRange(stockID, strategyName string, startTime, endTime time.Time) ([]*StrategyParametersRecord, error)

	// Get the latest parameters for a stock and strategy
	GetLatestParameters(stockID, strategyName string) (*StrategyParameters, error)

	// Update parameters for a specific record
	UpdateParameters(id int64, params *StrategyParameters) error

	// Delete parameters for a specific stock, strategy, and candle
	DeleteParameters(stockID, strategyName string, timestamp time.Time) error

	// Clean up old parameters (data retention)
	CleanupOldParameters(beforeTime time.Time) error
}

// StrategyParametersManager manages strategy parameters with caching and state persistence
type StrategyParametersManager struct {
	repo  StrategyParametersRepository
	cache map[string]*StrategyParameters // Cache key: stockID:strategyName:timestamp
}

// NewStrategyParametersManager creates a new strategy parameters manager
func NewStrategyParametersManager(repo StrategyParametersRepository) *StrategyParametersManager {
	return &StrategyParametersManager{
		repo:  repo,
		cache: make(map[string]*StrategyParameters),
	}
}

// GetCacheKey generates a cache key for strategy parameters
func (m *StrategyParametersManager) GetCacheKey(stockID, strategyName string, timestamp time.Time) string {
	return fmt.Sprintf("%s:%s:%d", stockID, strategyName, timestamp.Unix())
}

// GetParameters retrieves parameters from cache or database
func (m *StrategyParametersManager) GetParameters(stockID, strategyName string, timestamp time.Time) (*StrategyParameters, error) {
	cacheKey := m.GetCacheKey(stockID, strategyName, timestamp)

	// Check cache first
	if params, exists := m.cache[cacheKey]; exists {
		return params, nil
	}

	// Get from database
	params, err := m.repo.GetParameters(stockID, strategyName, timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default parameters if not found
			return &StrategyParameters{
				CanGenerateLong:  true,
				CanGenerateShort: true,
				StrategyState:    make(map[string]interface{}),
				Metadata:         make(map[string]interface{}),
			}, nil
		}
		return nil, err
	}

	// Cache the result
	m.cache[cacheKey] = params
	return params, nil
}

// SaveParameters saves parameters to database and cache
func (m *StrategyParametersManager) SaveParameters(stockID, strategyName string, timestamp time.Time, params *StrategyParameters) error {
	// Save to database
	err := m.repo.SaveParameters(stockID, strategyName, timestamp, params)
	if err != nil {
		return err
	}

	// Update cache
	cacheKey := m.GetCacheKey(stockID, strategyName, timestamp)
	m.cache[cacheKey] = params

	return nil
}

// UpdateParameters updates existing parameters
func (m *StrategyParametersManager) UpdateParameters(stockID, strategyName string, timestamp time.Time, params *StrategyParameters) error {
	// Get existing record
	existing, err := m.repo.GetParameters(stockID, strategyName, timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create new record if doesn't exist
			return m.SaveParameters(stockID, strategyName, timestamp, params)
		}
		return err
	}

	// Merge with existing parameters
	merged := m.mergeParameters(existing, params)

	// Save merged parameters
	return m.SaveParameters(stockID, strategyName, timestamp, merged)
}

// mergeParameters merges two parameter sets, with new params taking precedence
func (m *StrategyParametersManager) mergeParameters(existing, new *StrategyParameters) *StrategyParameters {
	merged := &StrategyParameters{
		InLongTrade:      new.InLongTrade,
		InShortTrade:     new.InShortTrade,
		CanGenerateLong:  new.CanGenerateLong,
		CanGenerateShort: new.CanGenerateShort,
		SqueezeDetected:  new.SqueezeDetected,
	}

	// Merge 1ST_ENTRY parameters
	if new.MRHigh != nil {
		merged.MRHigh = new.MRHigh
	} else {
		merged.MRHigh = existing.MRHigh
	}
	if new.MRLow != nil {
		merged.MRLow = new.MRLow
	} else {
		merged.MRLow = existing.MRLow
	}
	// ... continue for all parameters

	// Merge strategy state
	merged.StrategyState = make(map[string]interface{})
	for k, v := range existing.StrategyState {
		merged.StrategyState[k] = v
	}
	for k, v := range new.StrategyState {
		merged.StrategyState[k] = v
	}

	// Merge metadata
	merged.Metadata = make(map[string]interface{})
	for k, v := range existing.Metadata {
		merged.Metadata[k] = v
	}
	for k, v := range new.Metadata {
		merged.Metadata[k] = v
	}

	return merged
}

// ConvertDataFrameToParameters converts a DataFrame row to StrategyParameters
func (m *StrategyParametersManager) ConvertDataFrameToParameters(df *dataframe.DataFrame, rowIndex int) (*StrategyParameters, error) {
	if df.Nrow() <= rowIndex {
		return nil, fmt.Errorf("row index %d out of bounds for DataFrame with %d rows", rowIndex, df.Nrow())
	}

	params := &StrategyParameters{
		StrategyState: make(map[string]interface{}),
		Metadata:      make(map[string]interface{}),
	}

	// Extract common parameters
	if col := df.Col("in_long_trade"); col.Err == nil {
		if bools, err := col.Bool(); err == nil && rowIndex < len(bools) {
			params.InLongTrade = bools[rowIndex]
		}
	}
	if col := df.Col("in_short_trade"); col.Err == nil {
		if bools, err := col.Bool(); err == nil && rowIndex < len(bools) {
			params.InShortTrade = bools[rowIndex]
		}
	}
	if col := df.Col("can_generate_long"); col.Err == nil {
		if bools, err := col.Bool(); err == nil && rowIndex < len(bools) {
			params.CanGenerateLong = bools[rowIndex]
		}
	}
	if col := df.Col("can_generate_short"); col.Err == nil {
		if bools, err := col.Bool(); err == nil && rowIndex < len(bools) {
			params.CanGenerateShort = bools[rowIndex]
		}
	}

	// Extract 1ST_ENTRY parameters
	if col := df.Col("mr_high"); col.Err == nil && rowIndex < len(col.Float()) {
		value := col.Float()[rowIndex]
		params.MRHigh = &value
	}
	if col := df.Col("mr_low"); col.Err == nil && rowIndex < len(col.Float()) {
		value := col.Float()[rowIndex]
		params.MRLow = &value
	}
	// ... continue for all parameters

	return params, nil
}

// ConvertParametersToDataFrame converts StrategyParameters to DataFrame columns
func (m *StrategyParametersManager) ConvertParametersToDataFrame(params *StrategyParameters) map[string]series.Series {
	result := make(map[string]series.Series)

	// Common parameters
	result["in_long_trade"] = series.New([]bool{params.InLongTrade}, series.Bool, "in_long_trade")
	result["in_short_trade"] = series.New([]bool{params.InShortTrade}, series.Bool, "in_short_trade")
	result["can_generate_long"] = series.New([]bool{params.CanGenerateLong}, series.Bool, "can_generate_long")
	result["can_generate_short"] = series.New([]bool{params.CanGenerateShort}, series.Bool, "can_generate_short")
	result["squeeze_detected"] = series.New([]bool{params.SqueezeDetected}, series.Bool, "squeeze_detected")

	// 1ST_ENTRY parameters
	if params.MRHigh != nil {
		result["mr_high"] = series.New([]float64{*params.MRHigh}, series.Float, "mr_high")
	}
	if params.MRLow != nil {
		result["mr_low"] = series.New([]float64{*params.MRLow}, series.Float, "mr_low")
	}
	// ... continue for all parameters

	return result
}

// CleanupCache removes old entries from cache
func (m *StrategyParametersManager) CleanupCache() {
	// Simple cache cleanup - in production, implement LRU or TTL
	if len(m.cache) > 1000 {
		// Clear cache if it gets too large
		m.cache = make(map[string]*StrategyParameters)
	}
}
