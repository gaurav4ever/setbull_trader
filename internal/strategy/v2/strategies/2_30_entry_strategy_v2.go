package strategies

import (
	"fmt"
	"math"
	"time"

	v2 "setbull_trader/internal/strategy/v2"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

// TwoThirtyEntryStrategyV2 implements the 2_30_ENTRY strategy
// It calculates range from 2:00-2:30 PM and looks for breakouts at 2:30 PM
type TwoThirtyEntryStrategyV2 struct {
	*v2.BaseStrategy

	// Strategy parameters
	entryTime        time.Time // 2:30 PM
	rangeStartTime   time.Time // 2:00 PM
	rangeEndTime     time.Time // 2:30 PM
	minPriceMovement float64   // Minimum price movement required

	// Strategy state
	canGenerateLong  bool
	canGenerateShort bool

	// Range values (calculated from 2:00-2:30 PM)
	rangeHigh           float64
	rangeLow            float64
	rangeHighEntryPrice float64
	rangeLowEntryPrice  float64
	rangeCalculated     bool
	direction           string // "BULLISH", "BEARISH", or "NEUTRAL"
}

// NewTwoThirtyEntryStrategyV2 creates a new 2_30_ENTRY strategy
func NewTwoThirtyEntryStrategyV2() *TwoThirtyEntryStrategyV2 {
	metadata := v2.StrategyMetadata{
		Name:        "2_30_ENTRY_V2",
		Version:     "1.0.0",
		Description: "2:30 PM entry strategy - calculates range from 2:00-2:30 PM and looks for breakouts at 2:30 PM",
		Author:      "Setbull Trader",
		Tags:        []string{"entry_strategy", "afternoon_range", "breakout", "2_30_entry"},
	}

	base := v2.NewBaseStrategy(metadata)
	strategy := &TwoThirtyEntryStrategyV2{
		BaseStrategy:     base,
		entryTime:        time.Date(2000, 1, 1, 14, 30, 0, 0, time.UTC), // 2:30 PM
		rangeStartTime:   time.Date(2000, 1, 1, 14, 0, 0, 0, time.UTC),  // 2:00 PM
		rangeEndTime:     time.Date(2000, 1, 1, 14, 30, 0, 0, time.UTC), // 2:30 PM
		minPriceMovement: 0.5,                                           // 0.5% minimum movement
		canGenerateLong:  true,
		canGenerateShort: true,
		rangeCalculated:  false,
		direction:        "NEUTRAL",
	}

	// Set default configuration
	strategy.Configure(map[string]interface{}{
		"enabled":     true,
		"timeout":     10 * time.Second,
		"max_retries": 3,
		"parameters": map[string]interface{}{
			"entry_time":         "14:30",
			"range_start_time":   "14:00",
			"range_end_time":     "14:30",
			"min_price_movement": 0.5,
		},
	})

	return strategy
}

// Process implements the 2_30_ENTRY strategy processing logic
func (s *TwoThirtyEntryStrategyV2) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
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

		// Check if this is the range calculation period (2:00-2:30)
		if s.isRangeCalculationPeriod(timestamp) {
			s.calculateRange(high, low, close, timestamp)
		}

		// Check for entry conditions at 2:30 PM
		if s.isEntryTime(timestamp) && s.rangeCalculated {
			signal := s.checkEntryConditions(high, low, timestamp)
			if signal != nil {
				// Add signal to DataFrame
				result = s.addSignalToDataFrame(result, i, signal)
			}
		}
	}

	return &result, nil
}

// isRangeCalculationPeriod checks if the current time is during range calculation
func (s *TwoThirtyEntryStrategyV2) isRangeCalculationPeriod(candleTime time.Time) bool {
	candleHour := candleTime.Hour()
	candleMinute := candleTime.Minute()

	// Check if it's between 2:00 and 2:30
	return (candleHour == 14 && candleMinute >= 0 && candleMinute < 30)
}

// isEntryTime checks if the current time is the entry time (2:30 PM)
func (s *TwoThirtyEntryStrategyV2) isEntryTime(candleTime time.Time) bool {
	candleHour := candleTime.Hour()
	candleMinute := candleTime.Minute()

	// Check if it's exactly 2:30 PM
	return (candleHour == 14 && candleMinute == 30)
}

// calculateRange calculates the range from 2:00-2:30 PM
func (s *TwoThirtyEntryStrategyV2) calculateRange(high, low, close float64, candleTime time.Time) {
	if !s.rangeCalculated {
		// Initialize range values
		if s.rangeHigh == 0 {
			s.rangeHigh = high
			s.rangeLow = low
		} else {
			// Update range values
			if high > s.rangeHigh {
				s.rangeHigh = high
			}
			if low < s.rangeLow {
				s.rangeLow = low
			}
		}

		// If this is the last candle of the range period (2:30), finalize the calculation
		if s.isEntryTime(candleTime) {
			s.finalizeRangeCalculation(close)
		}
	}
}

// finalizeRangeCalculation finalizes the range calculation and determines direction
func (s *TwoThirtyEntryStrategyV2) finalizeRangeCalculation(close float64) {
	// Calculate range percentage
	rangeSize := s.rangeHigh - s.rangeLow

	// Determine direction based on close price relative to range
	rangeMid := (s.rangeHigh + s.rangeLow) / 2

	if close > rangeMid {
		s.direction = "BULLISH"
	} else if close < rangeMid {
		s.direction = "BEARISH"
	} else {
		s.direction = "NEUTRAL"
	}

	// Calculate entry prices with buffer
	buffer := rangeSize * 0.1 // 10% of range as buffer
	s.rangeHighEntryPrice = s.rangeHigh + buffer
	s.rangeLowEntryPrice = s.rangeLow - buffer

	// Round to 2 decimal places
	s.rangeHighEntryPrice = math.Round(s.rangeHighEntryPrice*100) / 100
	s.rangeLowEntryPrice = math.Round(s.rangeLowEntryPrice*100) / 100

	s.rangeCalculated = true

	fmt.Printf("2_30_ENTRY: Range calculated - High: %.2f, Low: %.2f, Direction: %s, Entry High: %.2f, Entry Low: %.2f\n",
		s.rangeHigh, s.rangeLow, s.direction, s.rangeHighEntryPrice, s.rangeLowEntryPrice)
}

// checkEntryConditions checks for entry conditions at 2:30 PM
func (s *TwoThirtyEntryStrategyV2) checkEntryConditions(high, low float64, timestamp time.Time) *EntrySignal {
	// Check for long entry (breakout above range high)
	if s.canGenerateLong && s.direction == "BULLISH" && high > s.rangeHighEntryPrice {
		s.canGenerateLong = false // Prevent multiple signals
		return &EntrySignal{
			Type:      "LONG_ENTRY",
			Direction: "LONG",
			Price:     s.rangeHighEntryPrice,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"range_high":             s.rangeHigh,
				"range_high_entry_price": s.rangeHighEntryPrice,
				"breakout_price":         high,
				"direction":              s.direction,
			},
		}
	}

	// Check for short entry (breakout below range low)
	if s.canGenerateShort && s.direction == "BEARISH" && low < s.rangeLowEntryPrice {
		s.canGenerateShort = false // Prevent multiple signals
		return &EntrySignal{
			Type:      "SHORT_ENTRY",
			Direction: "SHORT",
			Price:     s.rangeLowEntryPrice,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"range_low":             s.rangeLow,
				"range_low_entry_price": s.rangeLowEntryPrice,
				"breakout_price":        low,
				"direction":             s.direction,
			},
		}
	}

	return nil
}

// addSignalToDataFrame adds signal information to the DataFrame
func (s *TwoThirtyEntryStrategyV2) addSignalToDataFrame(df dataframe.DataFrame, index int, signal *EntrySignal) dataframe.DataFrame {
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
func (s *TwoThirtyEntryStrategyV2) ResetState() {
	s.canGenerateLong = true
	s.canGenerateShort = true
	s.rangeHigh = 0
	s.rangeLow = 0
	s.rangeHighEntryPrice = 0
	s.rangeLowEntryPrice = 0
	s.rangeCalculated = false
	s.direction = "NEUTRAL"
}

// GetRequiredHistory returns the number of historical candles required
func (s *TwoThirtyEntryStrategyV2) GetRequiredHistory() int {
	return 30 // Need at least 30 candles for range calculation (2:00-2:30)
}

// Configure configures the strategy with new parameters
func (s *TwoThirtyEntryStrategyV2) Configure(config map[string]interface{}) error {
	// Call base configuration
	err := s.BaseStrategy.Configure(config)
	if err != nil {
		return err
	}

	// Configure strategy-specific parameters
	if parameters, ok := config["parameters"].(map[string]interface{}); ok {
		if entryTime, ok := parameters["entry_time"].(string); ok {
			if t, err := time.Parse("15:04", entryTime); err == nil {
				s.entryTime = time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
			}
		}
		if rangeStartTime, ok := parameters["range_start_time"].(string); ok {
			if t, err := time.Parse("15:04", rangeStartTime); err == nil {
				s.rangeStartTime = time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
			}
		}
		if rangeEndTime, ok := parameters["range_end_time"].(string); ok {
			if t, err := time.Parse("15:04", rangeEndTime); err == nil {
				s.rangeEndTime = time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
			}
		}
		if minPriceMovement, ok := parameters["min_price_movement"].(float64); ok {
			s.minPriceMovement = minPriceMovement
		}
	}

	return nil
}

// ValidateConfiguration validates the strategy configuration
func (s *TwoThirtyEntryStrategyV2) ValidateConfiguration() error {
	// Call base validation
	err := s.BaseStrategy.ValidateConfiguration()
	if err != nil {
		return err
	}

	// Validate strategy-specific parameters
	if s.minPriceMovement <= 0 || s.minPriceMovement > 10 {
		return fmt.Errorf("invalid min price movement: %f (must be between 0 and 10)", s.minPriceMovement)
	}

	return nil
}

// GetEstimatedProcessingTime returns estimated processing time
func (s *TwoThirtyEntryStrategyV2) GetEstimatedProcessingTime() time.Duration {
	return 75 * time.Millisecond // Slightly longer processing for range calculation
}

// GetMemoryRequirements returns estimated memory requirements in bytes
func (s *TwoThirtyEntryStrategyV2) GetMemoryRequirements() int64 {
	return 768 * 1024 // 768KB for 2_30_ENTRY strategy
}
