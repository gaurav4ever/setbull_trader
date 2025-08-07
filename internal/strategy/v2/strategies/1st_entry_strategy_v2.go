package strategies

import (
	"fmt"
	"math"
	"time"

	v2 "setbull_trader/internal/strategy/v2"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

// FirstEntryStrategyV2 implements the 1ST_ENTRY strategy correctly
// It calculates MR from the first 5-minute candle (9:15-9:20) and checks for breakouts from 9:20 onwards
type FirstEntryStrategyV2 struct {
	*v2.BaseStrategy

	// Strategy parameters
	bufferPercentage float64
	marketOpen       time.Time
	morningRangeEnd  time.Time // 9:20 AM for 5MR

	// Strategy state
	canGenerateLong  bool
	canGenerateShort bool

	// Morning range values (calculated from 9:15-9:20 AM candle)
	mrHigh           float64
	mrLow            float64
	mrHighWithBuffer float64
	mrLowWithBuffer  float64
	mrCalculated     bool // Flag to track if MR has been calculated
}

// NewFirstEntryStrategyV2 creates a new 1ST_ENTRY strategy
func NewFirstEntryStrategyV2() *FirstEntryStrategyV2 {
	metadata := v2.StrategyMetadata{
		Name:        "1ST_ENTRY_V2",
		Version:     "1.0.0",
		Description: "First entry strategy - calculates MR from 9:15-9:20 AM and checks for immediate breakouts from 9:20 AM onwards",
		Author:      "Setbull Trader",
		Tags:        []string{"entry_strategy", "morning_range", "breakout", "immediate", "5MR"},
	}

	base := v2.NewBaseStrategy(metadata)
	strategy := &FirstEntryStrategyV2{
		BaseStrategy:     base,
		bufferPercentage: 0.0007, // 0.07% buffer
		marketOpen:       time.Date(2000, 1, 1, 9, 15, 0, 0, time.UTC),
		morningRangeEnd:  time.Date(2000, 1, 1, 9, 20, 0, 0, time.UTC), // 5MR end time
		canGenerateLong:  true,
		canGenerateShort: true,
		mrCalculated:     false,
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
func (s *FirstEntryStrategyV2) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
	if df.Nrow() == 0 {
		return df, v2.ErrInvalidDataFrame
	}

	// Clone the DataFrame to avoid modifying the original
	result := df.Copy()

	// Get required series
	highSeries := result.Col("high")
	lowSeries := result.Col("low")
	closeSeries := result.Col("close")
	timestampSeries := result.Col("timestamp")
	if highSeries.Err != nil || lowSeries.Err != nil || closeSeries.Err != nil || timestampSeries.Err != nil {
		return nil, fmt.Errorf("failed to get required series: %v", highSeries.Err)
	}

	// Process each candle
	for i := 0; i < df.Nrow(); i++ {
		// Get current candle data
		high := highSeries.Float()[i]
		low := lowSeries.Float()[i]
		close := closeSeries.Float()[i]
		timestampStr := timestampSeries.Elem(i).String()

		// Parse timestamp
		timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr)
		if err != nil {
			continue // Skip invalid timestamps
		}

		// Check if this is the morning range calculation period (9:15-9:20)
		if s.isMorningRangeCalculationPeriod(timestamp) {
			s.calculateMorningRange(high, low, close, timestamp)
		}

		// Check for entry conditions after morning range is calculated
		if s.isAfterMorningRangeCalculation(timestamp) && s.mrCalculated {
			signal := s.checkEntryConditions(high, low, timestamp)
			if signal != nil {
				// Add signal to DataFrame
				result = s.addSignalToDataFrame(result, i, signal)
			}
		}
	}

	return &result, nil
}

// isMorningRangeCalculationPeriod checks if the current time is during MR calculation
func (s *FirstEntryStrategyV2) isMorningRangeCalculationPeriod(candleTime time.Time) bool {
	candleHour := candleTime.Hour()
	candleMinute := candleTime.Minute()

	// Check if it's between 9:15 and 9:20
	return (candleHour == 9 && candleMinute >= 15 && candleMinute < 20)
}

// isAfterMorningRangeCalculation checks if the current time is after MR calculation
func (s *FirstEntryStrategyV2) isAfterMorningRangeCalculation(candleTime time.Time) bool {
	candleHour := candleTime.Hour()
	candleMinute := candleTime.Minute()

	// Check if it's after 9:20
	return (candleHour > 9) || (candleHour == 9 && candleMinute >= 20)
}

// calculateMorningRange calculates the morning range from the first 5-minute candle
func (s *FirstEntryStrategyV2) calculateMorningRange(high, low, close float64, candleTime time.Time) {
	if !s.mrCalculated {
		s.mrHigh = high
		s.mrLow = low
		s.calculateBufferValues()
		s.mrCalculated = true

		fmt.Printf("1ST_ENTRY: Morning Range calculated - High: %.2f, Low: %.2f, Buffer High: %.2f, Buffer Low: %.2f\n",
			s.mrHigh, s.mrLow, s.mrHighWithBuffer, s.mrLowWithBuffer)
	}
}

// checkEntryConditions checks for entry conditions based on MR breakout
func (s *FirstEntryStrategyV2) checkEntryConditions(high, low float64, timestamp time.Time) *EntrySignal {
	// Check for long entry (breakout above MR high with buffer)
	if s.canGenerateLong && high > s.mrHighWithBuffer {
		s.canGenerateLong = false // Prevent multiple signals
		return &EntrySignal{
			Type:      "LONG_ENTRY",
			Direction: "LONG",
			Price:     s.mrHighWithBuffer,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"mr_high":             s.mrHigh,
				"mr_high_with_buffer": s.mrHighWithBuffer,
				"breakout_price":      high,
			},
		}
	}

	// Check for short entry (breakout below MR low with buffer)
	if s.canGenerateShort && low < s.mrLowWithBuffer {
		s.canGenerateShort = false // Prevent multiple signals
		return &EntrySignal{
			Type:      "SHORT_ENTRY",
			Direction: "SHORT",
			Price:     s.mrLowWithBuffer,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"mr_low":             s.mrLow,
				"mr_low_with_buffer": s.mrLowWithBuffer,
				"breakout_price":     low,
			},
		}
	}

	return nil
}

// calculateBufferValues calculates the buffer values for MR high and low
func (s *FirstEntryStrategyV2) calculateBufferValues() {
	bufferTicks := s.bufferPercentage * s.mrHigh // Use MR high as base for buffer calculation
	s.mrHighWithBuffer = s.mrHigh + bufferTicks
	s.mrLowWithBuffer = s.mrLow - bufferTicks

	// Round to 2 decimal places (matching Python implementation)
	s.mrHighWithBuffer = math.Round(s.mrHighWithBuffer*100) / 100
	s.mrLowWithBuffer = math.Round(s.mrLowWithBuffer*100) / 100
}

// addSignalToDataFrame adds signal information to the DataFrame
func (s *FirstEntryStrategyV2) addSignalToDataFrame(df dataframe.DataFrame, index int, signal *EntrySignal) dataframe.DataFrame {
	// Create signal columns if they don't exist
	if df.Col("signal_type").Err != nil {
		signalType := make([]string, df.Nrow())
		signalDirection := make([]string, df.Nrow())
		signalPrice := make([]float64, df.Nrow())
		signalGenerated := make([]bool, df.Nrow())

		// Set signal values
		signalType[index] = signal.Type
		signalDirection[index] = signal.Direction
		signalPrice[index] = signal.Price
		signalGenerated[index] = true

		// Add columns to DataFrame one by one
		df = df.Mutate(series.New(signalType, series.String, "signal_type"))
		df = df.Mutate(series.New(signalDirection, series.String, "signal_direction"))
		df = df.Mutate(series.New(signalPrice, series.Float, "signal_price"))
		df = df.Mutate(series.New(signalGenerated, series.Bool, "signal_generated"))
	} else {
		// Update existing signal columns
		signalType := df.Col("signal_type")
		signalDirection := df.Col("signal_direction")
		signalPrice := df.Col("signal_price")
		signalGenerated := df.Col("signal_generated")

		if signalType.Err == nil {
			signalType.Set(index, series.Strings([]string{signal.Type}))
		}
		if signalDirection.Err == nil {
			signalDirection.Set(index, series.Strings([]string{signal.Direction}))
		}
		if signalPrice.Err == nil {
			signalPrice.Set(index, series.Floats([]float64{signal.Price}))
		}
		if signalGenerated.Err == nil {
			signalGenerated.Set(index, series.Bools([]bool{true}))
		}
	}

	return df
}

// ResetState resets the strategy state
func (s *FirstEntryStrategyV2) ResetState() {
	s.canGenerateLong = true
	s.canGenerateShort = true
	s.mrHigh = 0
	s.mrLow = 0
	s.mrHighWithBuffer = 0
	s.mrLowWithBuffer = 0
	s.mrCalculated = false
}

// GetRequiredHistory returns the number of historical candles required
func (s *FirstEntryStrategyV2) GetRequiredHistory() int {
	return 5 // Need at least 5 candles for morning range calculation
}

// Configure configures the strategy with new parameters
func (s *FirstEntryStrategyV2) Configure(config map[string]interface{}) error {
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
func (s *FirstEntryStrategyV2) ValidateConfiguration() error {
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
func (s *FirstEntryStrategyV2) GetEstimatedProcessingTime() time.Duration {
	return 50 * time.Millisecond // Fast processing for real-time requirements
}

// GetMemoryRequirements returns estimated memory requirements in bytes
func (s *FirstEntryStrategyV2) GetMemoryRequirements() int64 {
	return 512 * 1024 // 512KB for 1ST_ENTRY strategy
}
