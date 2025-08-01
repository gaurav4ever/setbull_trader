package strategies

import (
	"fmt"
	"math"
	"time"

	v2 "setbull_trader/internal/strategy/v2"

	"github.com/go-gota/gota/dataframe"
)

// FirstEntryStrategyV2Simple implements the 1ST_ENTRY strategy correctly
// It calculates MR from the first 5-minute candle (9:15-9:20) and checks for breakouts from 9:20 onwards
// Focus: Signal generation and parameter tracking only (no position sizing, risk management, or trade management)
type FirstEntryStrategyV2Simple struct {
	*v2.BaseStrategy

	// Strategy parameters
	bufferPercentage float64
	marketOpen       time.Time
	morningRangeEnd  time.Time // 9:20 AM for 5MR

	// Strategy state (for signal generation only)
	canGenerateLong  bool
	canGenerateShort bool

	// Morning range values (calculated from 9:15-9:20 AM candle)
	mrHigh           float64
	mrLow            float64
	mrHighWithBuffer float64
	mrLowWithBuffer  float64
	mrCalculated     bool // Flag to track if MR has been calculated

	// Parameters manager for state persistence
	paramsManager *v2.StrategyParametersManager
}

// NewFirstEntryStrategyV2Simple creates a new 1ST_ENTRY strategy
func NewFirstEntryStrategyV2Simple(paramsManager *v2.StrategyParametersManager) *FirstEntryStrategyV2Simple {
	metadata := v2.StrategyMetadata{
		Name:        "1ST_ENTRY_V2",
		Version:     "1.0.0",
		Description: "First entry strategy - calculates MR from 9:15-9:20 AM and checks for immediate breakouts from 9:20 AM onwards. Focus: Signal generation and parameter tracking only.",
		Author:      "Setbull Trader",
		Tags:        []string{"entry_strategy", "morning_range", "breakout", "immediate", "5MR", "signal_generation"},
	}

	base := v2.NewBaseStrategy(metadata)
	strategy := &FirstEntryStrategyV2Simple{
		BaseStrategy:     base,
		bufferPercentage: 0.0007, // 0.07% buffer
		marketOpen:       time.Date(2000, 1, 1, 9, 15, 0, 0, time.UTC),
		morningRangeEnd:  time.Date(2000, 1, 1, 9, 20, 0, 0, time.UTC), // 5MR end time
		canGenerateLong:  true,
		canGenerateShort: true,
		mrCalculated:     false,
		paramsManager:    paramsManager,
	}

	// Set default configuration
	strategy.Configure(map[string]interface{}{
		"enabled":     true,
		"timeout":     10 * time.Second,
		"max_retries": 3,
		"parameters": map[string]interface{}{
			"buffer_percentage": 0.0007,
			"market_open":       "09:15",
			"morning_range_end": "09:20",
		},
	})

	return strategy
}

// Process implements the 1ST_ENTRY strategy processing logic
func (s *FirstEntryStrategyV2Simple) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
	if df.Nrow() == 0 {
		return df, v2.ErrInvalidDataFrame
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

		// Check if this is the morning range calculation period (9:15-9:20 AM)
		if s.isMorningRangeCalculationPeriod(candleTime) {
			s.calculateMorningRange(high, low, candleTime, params)
			continue // Skip signal generation for MR calculation period
		}

		// Check if this is after morning range calculation (9:20 AM onwards)
		if s.isAfterMorningRangeCalculation(candleTime) {
			// Check entry conditions only if MR has been calculated
			if s.mrCalculated {
				signal := s.checkEntryConditions(high, low, candleTime, params)

				// Update parameters with new state
				s.updateParametersWithState(params, signal)

				// Save parameters
				s.saveParameters(candleTime, params)

				// Log signal if generated (simplified approach)
				if signal != nil {
					fmt.Printf("1ST_ENTRY Signal: %s %s at %.2f (MR: %.2f-%.2f)\n",
						signal.Type, signal.Direction, signal.Price, s.mrLow, s.mrHigh)
				}
			}
		}
	}

	return &result, nil
}

// isMorningRangeCalculationPeriod checks if the candle is in the MR calculation period (9:15-9:20 AM)
func (s *FirstEntryStrategyV2Simple) isMorningRangeCalculationPeriod(candleTime time.Time) bool {
	candleTimeOnly := time.Date(2000, 1, 1, candleTime.Hour(), candleTime.Minute(), 0, 0, time.UTC)
	return candleTimeOnly.Equal(s.marketOpen) || (candleTimeOnly.After(s.marketOpen) && candleTimeOnly.Before(s.morningRangeEnd))
}

// isAfterMorningRangeCalculation checks if the candle is after MR calculation (9:20 AM onwards)
func (s *FirstEntryStrategyV2Simple) isAfterMorningRangeCalculation(candleTime time.Time) bool {
	candleTimeOnly := time.Date(2000, 1, 1, candleTime.Hour(), candleTime.Minute(), 0, 0, time.UTC)
	return candleTimeOnly.Equal(s.morningRangeEnd) || candleTimeOnly.After(s.morningRangeEnd)
}

// calculateMorningRange calculates MR from the first 5-minute candle (9:15-9:20 AM)
func (s *FirstEntryStrategyV2Simple) calculateMorningRange(high, low float64, candleTime time.Time, params *v2.StrategyParameters) {
	// For the first candle in the MR period, initialize MR values
	if !s.mrCalculated {
		s.mrHigh = high
		s.mrLow = low
		s.mrCalculated = true
	} else {
		// Update MR values with the highest high and lowest low
		if high > s.mrHigh {
			s.mrHigh = high
		}
		if low < s.mrLow {
			s.mrLow = low
		}
	}

	// Calculate buffer values
	s.calculateBufferValues()

	// Update parameters with MR values
	s.updateParametersWithMRValues(params)

	// Log MR calculation
	fmt.Printf("1ST_ENTRY MR Calculation at %s: High=%.2f, Low=%.2f, Size=%.2f\n",
		candleTime.Format("15:04"), s.mrHigh, s.mrLow, s.mrHigh-s.mrLow)
}

// getOrCreateParameters retrieves or creates strategy parameters for a timestamp
func (s *FirstEntryStrategyV2Simple) getOrCreateParameters(timestamp time.Time) (*v2.StrategyParameters, error) {
	// Try to get existing parameters
	params, err := s.paramsManager.GetParameters("", s.GetName(), timestamp)
	if err != nil {
		// Create new parameters if not found
		params = &v2.StrategyParameters{
			CanGenerateLong:  s.canGenerateLong,
			CanGenerateShort: s.canGenerateShort,
			StrategyState:    make(map[string]interface{}),
			Metadata:         make(map[string]interface{}),
		}
	}

	return params, nil
}

// updateStateFromParameters updates strategy state from stored parameters
func (s *FirstEntryStrategyV2Simple) updateStateFromParameters(params *v2.StrategyParameters) {
	s.canGenerateLong = params.CanGenerateLong
	s.canGenerateShort = params.CanGenerateShort

	// Update morning range values if available
	if params.MRHigh != nil {
		s.mrHigh = *params.MRHigh
	}
	if params.MRLow != nil {
		s.mrLow = *params.MRLow
	}
	if params.MRHighWithBuffer != nil {
		s.mrHighWithBuffer = *params.MRHighWithBuffer
	}
	if params.MRLowWithBuffer != nil {
		s.mrLowWithBuffer = *params.MRLowWithBuffer
	}
	if params.MRCalculated != nil {
		s.mrCalculated = *params.MRCalculated
	}
}

// updateParametersWithMRValues updates parameters with MR calculation values
func (s *FirstEntryStrategyV2Simple) updateParametersWithMRValues(params *v2.StrategyParameters) {
	if s.mrHigh > 0 {
		params.MRHigh = &s.mrHigh
	}
	if s.mrLow > 0 {
		params.MRLow = &s.mrLow
	}
	if s.mrHighWithBuffer > 0 {
		params.MRHighWithBuffer = &s.mrHighWithBuffer
	}
	if s.mrLowWithBuffer > 0 {
		params.MRLowWithBuffer = &s.mrLowWithBuffer
	}

	params.BufferPercentage = &s.bufferPercentage
	params.MRCalculated = &s.mrCalculated

	// Update strategy state
	params.StrategyState["mr_calculated"] = s.mrCalculated
	params.StrategyState["mr_high"] = s.mrHigh
	params.StrategyState["mr_low"] = s.mrLow
	params.StrategyState["mr_high_with_buffer"] = s.mrHighWithBuffer
	params.StrategyState["mr_low_with_buffer"] = s.mrLowWithBuffer
}

// checkEntryConditions checks for entry conditions based on Python logic
// Focus: Signal generation only (no position sizing or risk management)
func (s *FirstEntryStrategyV2Simple) checkEntryConditions(high, low float64, timestamp time.Time, params *v2.StrategyParameters) *EntrySignal {
	// Skip if morning range values are not available
	if !s.mrCalculated || s.mrHigh == 0 || s.mrLow == 0 {
		return nil
	}

	// Calculate buffer values if not already calculated
	if s.mrHighWithBuffer == 0 || s.mrLowWithBuffer == 0 {
		s.calculateBufferValues()
	}

	// Check long breakout (signal generation only)
	if high >= s.mrHighWithBuffer && s.canGenerateLong {
		s.canGenerateLong = false
		return &EntrySignal{
			Type:      "IMMEDIATE_BREAKOUT",
			Direction: "LONG",
			Price:     s.mrHighWithBuffer,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"breakout_type":  "immediate",
				"entry_type":     "1st_entry",
				"entry_time":     timestamp.Format("15:04"),
				"mr_high":        s.mrHigh,
				"mr_low":         s.mrLow,
				"mr_size":        s.mrHigh - s.mrLow,
				"signal_purpose": "data_tracking", // Indicate this is for data tracking only
			},
		}
	}

	// Check short breakout (signal generation only)
	if low <= s.mrLowWithBuffer && s.canGenerateShort {
		s.canGenerateShort = false
		return &EntrySignal{
			Type:      "IMMEDIATE_BREAKOUT",
			Direction: "SHORT",
			Price:     s.mrLowWithBuffer,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"breakout_type":  "immediate",
				"entry_type":     "1st_entry",
				"entry_time":     timestamp.Format("15:04"),
				"mr_high":        s.mrHigh,
				"mr_low":         s.mrLow,
				"mr_size":        s.mrHigh - s.mrLow,
				"signal_purpose": "data_tracking", // Indicate this is for data tracking only
			},
		}
	}

	return nil
}

// calculateBufferValues calculates buffer values for morning range
func (s *FirstEntryStrategyV2Simple) calculateBufferValues() {
	s.mrHighWithBuffer = s.mrHigh * (1 + s.bufferPercentage)
	s.mrLowWithBuffer = s.mrLow * (1 - s.bufferPercentage)

	// Round to 2 decimal places (matching Python implementation)
	s.mrHighWithBuffer = math.Round(s.mrHighWithBuffer*100) / 100
	s.mrLowWithBuffer = math.Round(s.mrLowWithBuffer*100) / 100
}

// updateParametersWithState updates parameters with current strategy state
func (s *FirstEntryStrategyV2Simple) updateParametersWithState(params *v2.StrategyParameters, signal *EntrySignal) {
	params.CanGenerateLong = s.canGenerateLong
	params.CanGenerateShort = s.canGenerateShort

	// Update morning range values
	if s.mrHigh > 0 {
		params.MRHigh = &s.mrHigh
	}
	if s.mrLow > 0 {
		params.MRLow = &s.mrLow
	}
	if s.mrHighWithBuffer > 0 {
		params.MRHighWithBuffer = &s.mrHighWithBuffer
	}
	if s.mrLowWithBuffer > 0 {
		params.MRLowWithBuffer = &s.mrLowWithBuffer
	}

	params.BufferPercentage = &s.bufferPercentage
	params.MRCalculated = &s.mrCalculated

	// Update strategy state
	params.StrategyState["can_generate_long"] = s.canGenerateLong
	params.StrategyState["can_generate_short"] = s.canGenerateShort
	params.StrategyState["mr_calculated"] = s.mrCalculated
	params.StrategyState["mr_high"] = s.mrHigh
	params.StrategyState["mr_low"] = s.mrLow
	params.StrategyState["mr_high_with_buffer"] = s.mrHighWithBuffer
	params.StrategyState["mr_low_with_buffer"] = s.mrLowWithBuffer

	// Add signal to metadata if generated
	if signal != nil {
		params.Metadata["last_signal"] = signal
		params.Metadata["signal_generated"] = true
		params.Metadata["signal_purpose"] = "data_tracking"
	} else {
		params.Metadata["signal_generated"] = false
	}
}

// saveParameters saves parameters to the database
func (s *FirstEntryStrategyV2Simple) saveParameters(timestamp time.Time, params *v2.StrategyParameters) {
	err := s.paramsManager.SaveParameters("", s.GetName(), timestamp, params)
	if err != nil {
		// Log error but don't fail the strategy
		fmt.Printf("Failed to save parameters for %s: %v\n", s.GetName(), err)
	}
}

// ResetState resets the strategy state
func (s *FirstEntryStrategyV2Simple) ResetState() {
	s.canGenerateLong = true
	s.canGenerateShort = true
	s.mrHigh = 0
	s.mrLow = 0
	s.mrHighWithBuffer = 0
	s.mrLowWithBuffer = 0
	s.mrCalculated = false
}

// GetRequiredHistory returns the number of historical candles required
func (s *FirstEntryStrategyV2Simple) GetRequiredHistory() int {
	return 5 // Need at least 5 candles for morning range calculation
}

// Configure configures the strategy with new parameters
func (s *FirstEntryStrategyV2Simple) Configure(config map[string]interface{}) error {
	// Call base configuration
	err := s.BaseStrategy.Configure(config)
	if err != nil {
		return err
	}

	// Configure strategy-specific parameters
	if parameters, ok := config["parameters"].(map[string]interface{}); ok {
		if bufferPercentage, ok := parameters["buffer_percentage"].(float64); ok {
			s.bufferPercentage = bufferPercentage
		}
		if marketOpen, ok := parameters["market_open"].(string); ok {
			// Parse market open time
			if t, err := time.Parse("15:04", marketOpen); err == nil {
				s.marketOpen = time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
			}
		}
		if morningRangeEnd, ok := parameters["morning_range_end"].(string); ok {
			// Parse morning range end time
			if t, err := time.Parse("15:04", morningRangeEnd); err == nil {
				s.morningRangeEnd = time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
			}
		}
	}

	return nil
}

// ValidateConfiguration validates the strategy configuration
func (s *FirstEntryStrategyV2Simple) ValidateConfiguration() error {
	// Call base validation
	err := s.BaseStrategy.ValidateConfiguration()
	if err != nil {
		return err
	}

	// Validate strategy-specific parameters
	if s.bufferPercentage <= 0 || s.bufferPercentage > 0.1 {
		return fmt.Errorf("invalid buffer percentage: %f (must be between 0 and 0.1)", s.bufferPercentage)
	}

	return nil
}

// GetEstimatedProcessingTime returns estimated processing time
func (s *FirstEntryStrategyV2Simple) GetEstimatedProcessingTime() time.Duration {
	return 50 * time.Millisecond // Fast processing for real-time requirements
}

// GetMemoryRequirements returns estimated memory requirements in bytes
func (s *FirstEntryStrategyV2Simple) GetMemoryRequirements() int64 {
	return 512 * 1024 // 512KB for 1ST_ENTRY strategy
}
