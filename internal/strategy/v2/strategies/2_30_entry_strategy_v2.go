package strategies

import (
	"fmt"
	"math"
	"time"

	"github.com/go-gota/gota/dataframe"

	v2 "setbull_trader/internal/strategy/v2"
)

// TwoThirtyEntryStrategyV2 implements the 2:30 PM entry strategy
// This strategy waits for 2:30 PM to establish a range and then looks for breakouts
// Focus: Signal generation and parameter tracking only (no position sizing, risk management, or trade management)
type TwoThirtyEntryStrategyV2 struct {
	*v2.BaseStrategy

	// Strategy parameters
	entryTime        time.Time
	bufferPercentage float64
	direction        string
	minPriceMovement float64

	// Strategy state (for signal generation only)
	canGenerateLong  bool
	canGenerateShort bool

	// Range values (calculated at 2:30 PM)
	rangeHigh           float64
	rangeLow            float64
	rangeHighEntryPrice float64
	rangeLowEntryPrice  float64
	rangeCalculated     bool

	// Parameters manager for state persistence
	paramsManager *v2.StrategyParametersManager
}

// NewTwoThirtyEntryStrategyV2 creates a new 2:30 entry strategy instance
func NewTwoThirtyEntryStrategyV2(paramsManager *v2.StrategyParametersManager) *TwoThirtyEntryStrategyV2 {
	return &TwoThirtyEntryStrategyV2{
		BaseStrategy: v2.NewBaseStrategy(v2.StrategyMetadata{
			Name:        "2_30_ENTRY",
			Version:     "1.0.0",
			Description: "2:30 PM entry strategy with range-based breakout detection",
			Author:      "Setbull Trader",
			Tags:        []string{"time-based", "range", "breakout"},
		}),

		// Default parameters
		entryTime:        time.Date(2000, 1, 1, 14, 30, 0, 0, time.UTC), // 2:30 PM
		bufferPercentage: 0.0003,                                        // 0.03%
		direction:        "BULLISH",
		minPriceMovement: 0.001, // 0.1%

		// Strategy state
		canGenerateLong:  true,
		canGenerateShort: true,

		// Range values
		rangeCalculated: false,

		// Parameters manager
		paramsManager: paramsManager,
	}
}

// Process implements the StrategyV2 interface
// Processes the DataFrame and generates signals based on 2:30 PM entry logic
func (s *TwoThirtyEntryStrategyV2) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
	if df.Nrow() == 0 {
		return df, nil
	}

	// Clone the DataFrame to avoid modifying the original
	result := df.Copy()

	// Get required series
	highSeries := result.Col("high")
	lowSeries := result.Col("low")
	timestampSeries := result.Col("timestamp")
	if highSeries.Err != nil || lowSeries.Err != nil || timestampSeries.Err != nil {
		return nil, fmt.Errorf("failed to get required series: %v", highSeries.Err)
	}

	// Load previous state from parameters
	s.loadStateFromParameters()

	// Process each candle
	for i := 0; i < df.Nrow(); i++ {
		// Get current candle data
		high := highSeries.Float()[i]
		low := lowSeries.Float()[i]
		timestamp := timestampSeries.Elem(i).String()

		// Parse timestamp
		candleTime, err := time.Parse("2006-01-02 15:04:05", timestamp)
		if err != nil {
			continue // Skip invalid timestamps
		}

		// Get or create strategy parameters for this candle
		params, err := s.getOrCreateParameters(candleTime)
		if err != nil {
			continue // Skip if we can't get parameters
		}

		// Update strategy state from parameters
		s.updateStateFromParameters(params)

		// Check if this is the entry time (2:30 PM)
		if s.isEntryTime(candleTime) {
			s.calculateEntryRange(high, low, candleTime, params)
			continue // Skip signal generation for entry time
		}

		// Check entry conditions after 2:30 PM
		if s.isAfterEntryTime(candleTime) && s.rangeCalculated {
			signal := s.checkEntryConditions(high, low, candleTime, params)

			// Update parameters with new state
			s.updateParametersWithState(params, signal)

			// Save parameters
			s.saveParameters(candleTime, params)

			// Log signal if generated (for data tracking only)
			if signal != nil {
				fmt.Printf("2_30_ENTRY Signal Generated at %s: %s %s at %.2f (Range: %.2f-%.2f)\n",
					candleTime.Format("15:04"), signal.Type, signal.Direction, signal.Price, s.rangeLow, s.rangeHigh)
			}
		}
	}

	return &result, nil
}

// isEntryTime checks if the current time is 2:30 PM (entry time)
func (s *TwoThirtyEntryStrategyV2) isEntryTime(candleTime time.Time) bool {
	candleTimeOnly := time.Date(2000, 1, 1, candleTime.Hour(), candleTime.Minute(), 0, 0, time.UTC)
	return candleTimeOnly.Equal(s.entryTime)
}

// isAfterEntryTime checks if this is after entry time (2:30 PM onwards)
func (s *TwoThirtyEntryStrategyV2) isAfterEntryTime(candleTime time.Time) bool {
	candleTimeOnly := time.Date(2000, 1, 1, candleTime.Hour(), candleTime.Minute(), 0, 0, time.UTC)
	return candleTimeOnly.After(s.entryTime)
}

// calculateEntryRange calculates the range at 2:30 PM entry time
func (s *TwoThirtyEntryStrategyV2) calculateEntryRange(high, low float64, candleTime time.Time, params *v2.StrategyParameters) {
	s.rangeHigh = high
	s.rangeLow = low
	s.rangeCalculated = true

	// Calculate buffer values
	s.calculateBufferValues()

	// Update parameters with range values
	s.updateParametersWithRangeValues(params)

	// Log range calculation
	fmt.Printf("2_30_ENTRY Range Calculation at %s: High=%.2f, Low=%.2f, Size=%.2f\n",
		candleTime.Format("15:04"), s.rangeHigh, s.rangeLow, s.rangeHigh-s.rangeLow)
}

// calculateBufferValues calculates entry prices with buffer
func (s *TwoThirtyEntryStrategyV2) calculateBufferValues() {
	s.rangeHighEntryPrice = s.rangeHigh * (1 + s.bufferPercentage)
	s.rangeLowEntryPrice = s.rangeLow * (1 - s.bufferPercentage)

	// Round to 2 decimal places
	s.rangeHighEntryPrice = math.Round(s.rangeHighEntryPrice*100) / 100
	s.rangeLowEntryPrice = math.Round(s.rangeLowEntryPrice*100) / 100
}

// checkEntryConditions checks for entry conditions based on direction bias
func (s *TwoThirtyEntryStrategyV2) checkEntryConditions(high, low float64, timestamp time.Time, params *v2.StrategyParameters) *EntrySignal {
	if !s.rangeCalculated {
		return nil
	}

	// Check based on direction bias
	switch s.direction {
	case "BULLISH":
		return s.checkBullishEntry(high, low, timestamp, params)
	case "BEARISH":
		return s.checkBearishEntry(high, low, timestamp, params)
	default:
		return s.checkNeutralEntry(high, low, timestamp, params)
	}
}

// checkBullishEntry checks for bullish entry conditions
func (s *TwoThirtyEntryStrategyV2) checkBullishEntry(high, low float64, timestamp time.Time, params *v2.StrategyParameters) *EntrySignal {
	// Check if price is above range high entry price for long entry
	if high > s.rangeHighEntryPrice && s.canGenerateLong {
		s.canGenerateLong = false
		return &EntrySignal{
			Type:      "IMMEDIATE_BREAKOUT",
			Direction: "LONG",
			Price:     s.rangeHighEntryPrice,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"entry_type":             "14:30",
				"entry_time":             "14:30",
				"signal_purpose":         "data_tracking",
				"range_high":             s.rangeHigh,
				"range_low":              s.rangeLow,
				"range_high_entry_price": s.rangeHighEntryPrice,
				"range_low_entry_price":  s.rangeLowEntryPrice,
			},
		}
	}
	return nil
}

// checkBearishEntry checks for bearish entry conditions
func (s *TwoThirtyEntryStrategyV2) checkBearishEntry(high, low float64, timestamp time.Time, params *v2.StrategyParameters) *EntrySignal {
	// Check if price is below range low entry price for short entry
	if low < s.rangeLowEntryPrice && s.canGenerateShort {
		s.canGenerateShort = false
		return &EntrySignal{
			Type:      "IMMEDIATE_BREAKOUT",
			Direction: "SHORT",
			Price:     s.rangeLowEntryPrice,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"entry_type":             "14:30",
				"entry_time":             "14:30",
				"signal_purpose":         "data_tracking",
				"range_high":             s.rangeHigh,
				"range_low":              s.rangeLow,
				"range_high_entry_price": s.rangeHighEntryPrice,
				"range_low_entry_price":  s.rangeLowEntryPrice,
			},
		}
	}
	return nil
}

// checkNeutralEntry checks for neutral entry conditions (both directions)
func (s *TwoThirtyEntryStrategyV2) checkNeutralEntry(high, low float64, timestamp time.Time, params *v2.StrategyParameters) *EntrySignal {
	// Check for long entry
	if high > s.rangeHighEntryPrice && s.canGenerateLong {
		s.canGenerateLong = false
		return &EntrySignal{
			Type:      "IMMEDIATE_BREAKOUT",
			Direction: "LONG",
			Price:     s.rangeHighEntryPrice,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"entry_type":             "14:30",
				"entry_time":             "14:30",
				"signal_purpose":         "data_tracking",
				"range_high":             s.rangeHigh,
				"range_low":              s.rangeLow,
				"range_high_entry_price": s.rangeHighEntryPrice,
				"range_low_entry_price":  s.rangeLowEntryPrice,
			},
		}
	}

	// Check for short entry
	if low < s.rangeLowEntryPrice && s.canGenerateShort {
		s.canGenerateShort = false
		return &EntrySignal{
			Type:      "IMMEDIATE_BREAKOUT",
			Direction: "SHORT",
			Price:     s.rangeLowEntryPrice,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"entry_type":             "14:30",
				"entry_time":             "14:30",
				"signal_purpose":         "data_tracking",
				"range_high":             s.rangeHigh,
				"range_low":              s.rangeLow,
				"range_high_entry_price": s.rangeHighEntryPrice,
				"range_low_entry_price":  s.rangeLowEntryPrice,
			},
		}
	}

	return nil
}

// getOrCreateParameters gets or creates parameters for a specific timestamp
func (s *TwoThirtyEntryStrategyV2) getOrCreateParameters(timestamp time.Time) (*v2.StrategyParameters, error) {
	if s.paramsManager == nil {
		// Return default parameters if no manager is available
		return &v2.StrategyParameters{
			CanGenerateLong:  s.canGenerateLong,
			CanGenerateShort: s.canGenerateShort,
		}, nil
	}

	// Try to get existing parameters
	params, err := s.paramsManager.GetParameters("", "2_30_ENTRY", timestamp)
	if err != nil {
		// Create new parameters if none exist
		params = &v2.StrategyParameters{
			CanGenerateLong:  s.canGenerateLong,
			CanGenerateShort: s.canGenerateShort,
		}
	}

	return params, nil
}

// updateStateFromParameters updates strategy state from parameters
func (s *TwoThirtyEntryStrategyV2) updateStateFromParameters(params *v2.StrategyParameters) {
	s.canGenerateLong = params.CanGenerateLong
	s.canGenerateShort = params.CanGenerateShort
}

// updateParametersWithRangeValues updates parameters with range values
func (s *TwoThirtyEntryStrategyV2) updateParametersWithRangeValues(params *v2.StrategyParameters) {
	params.RangeHigh = &s.rangeHigh
	params.RangeLow = &s.rangeLow
	params.RangeHighEntryPrice = &s.rangeHighEntryPrice
	params.RangeLowEntryPrice = &s.rangeLowEntryPrice
}

// updateParametersWithState updates parameters with current state
func (s *TwoThirtyEntryStrategyV2) updateParametersWithState(params *v2.StrategyParameters, signal *EntrySignal) {
	params.CanGenerateLong = s.canGenerateLong
	params.CanGenerateShort = s.canGenerateShort

	// Add strategy-specific parameters
	entryTimeStr := s.entryTime.Format("15:04")
	params.EntryTime = &s.entryTime
	params.BufferPercentage = &s.bufferPercentage
	params.Direction = &s.direction
	params.MinPriceMovement = &s.minPriceMovement

	// Add signal metadata if signal was generated
	if signal != nil {
		if params.Metadata == nil {
			params.Metadata = make(map[string]interface{})
		}
		params.Metadata["last_signal"] = signal
	}
}

// saveParameters saves parameters to the database
func (s *TwoThirtyEntryStrategyV2) saveParameters(timestamp time.Time, params *v2.StrategyParameters) {
	if s.paramsManager != nil {
		s.paramsManager.SaveParameters("", "2_30_ENTRY", timestamp, params)
	}
}

// loadStateFromParameters loads strategy state from parameters
func (s *TwoThirtyEntryStrategyV2) loadStateFromParameters() {
	if s.paramsManager == nil {
		return
	}

	// Load the most recent parameters for this strategy
	params, err := s.paramsManager.GetLatestParameters("", "2_30_ENTRY")
	if err == nil && params != nil {
		s.updateStateFromParameters(params)
	}
}

// GetRequiredHistory returns the number of candles required for this strategy
func (s *TwoThirtyEntryStrategyV2) GetRequiredHistory() int {
	return 1 // Only need current candle for range calculation
}

// GetName returns the strategy name
func (s *TwoThirtyEntryStrategyV2) GetName() string {
	return "2_30_ENTRY"
}

// Configure configures the strategy with parameters
func (s *TwoThirtyEntryStrategyV2) Configure(config map[string]interface{}) error {
	if params, ok := config["parameters"].(map[string]interface{}); ok {
		// Parse entry time
		if entryTimeStr, ok := params["entry_time"].(string); ok {
			if t, err := time.Parse("15:04", entryTimeStr); err == nil {
				s.entryTime = time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
			}
		}

		// Parse buffer percentage
		if bufferPct, ok := params["buffer_percentage"].(float64); ok {
			s.bufferPercentage = bufferPct
		}

		// Parse direction
		if direction, ok := params["direction"].(string); ok {
			s.direction = direction
		}

		// Parse minimum price movement
		if minMovement, ok := params["min_price_movement"].(float64); ok {
			s.minPriceMovement = minMovement
		}
	}

	return nil
}

// ValidateConfiguration validates the strategy configuration
func (s *TwoThirtyEntryStrategyV2) ValidateConfiguration() error {
	if s.bufferPercentage <= 0 || s.bufferPercentage > 0.1 {
		return fmt.Errorf("buffer_percentage must be between 0 and 0.1")
	}

	if s.direction != "BULLISH" && s.direction != "BEARISH" && s.direction != "NEUTRAL" {
		return fmt.Errorf("direction must be BULLISH, BEARISH, or NEUTRAL")
	}

	if s.minPriceMovement <= 0 || s.minPriceMovement > 0.1 {
		return fmt.Errorf("min_price_movement must be between 0 and 0.1")
	}

	return nil
}

// GetEstimatedProcessingTime returns the estimated processing time
func (s *TwoThirtyEntryStrategyV2) GetEstimatedProcessingTime() time.Duration {
	return 50 * time.Millisecond // < 50ms per stock group
}

// GetMemoryRequirements returns the memory requirements
func (s *TwoThirtyEntryStrategyV2) GetMemoryRequirements() int64 {
	return 512 * 1024 // 512KB per strategy instance
}

// ResetState resets the strategy state
func (s *TwoThirtyEntryStrategyV2) ResetState() {
	s.canGenerateLong = true
	s.canGenerateShort = true
	s.rangeCalculated = false
	s.rangeHigh = 0
	s.rangeLow = 0
	s.rangeHighEntryPrice = 0
	s.rangeLowEntryPrice = 0
}
