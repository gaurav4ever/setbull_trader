package strategies

import (
	"fmt"
	"math"
	"time"

	v2 "setbull_trader/internal/strategy/v2"
	"setbull_trader/pkg/log"

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
	startTime := time.Now()

	// Log strategy processing start
	log.Info("2_30_ENTRY_V2: Starting strategy processing - strategy=%s version=%s data_rows=%d range_calculated=%t can_generate_long=%t can_generate_short=%t direction=%s",
		s.GetName(),
		s.GetVersion(),
		df.Nrow(),
		s.rangeCalculated,
		s.canGenerateLong,
		s.canGenerateShort,
		s.direction)

	if df.Nrow() == 0 {
		log.Error("2_30_ENTRY_V2: Invalid dataframe - no rows - strategy=%s error=%s",
			s.GetName(),
			v2.ErrInvalidDataFrame.Error())
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
		log.Error("2_30_ENTRY_V2: Failed to get required series - strategy=%s high_error=%v low_error=%v close_error=%v timestamp_error=%v",
			s.GetName(),
			highSeries.Err,
			lowSeries.Err,
			closeSeries.Err,
			timestampSeries.Err)
		return nil, fmt.Errorf("failed to get required series: %v", highSeries.Err)
	}

	log.Debug("2_30_ENTRY_V2: Successfully extracted required series - strategy=%s series_count=%d",
		s.GetName(),
		4)

	// Process each candle
	signalsGenerated := 0
	rangeProcessed := false

	for i := 0; i < df.Nrow(); i++ {
		// Get current candle data
		high := highSeries.Float()[i]
		low := lowSeries.Float()[i]
		close := closeSeries.Float()[i]
		timestampStr := timestampSeries.Elem(i).String()

		// Parse timestamp
		timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr)
		if err != nil {
			log.Warn("2_30_ENTRY_V2: Skipping invalid timestamp - strategy=%s candle_index=%d timestamp_str=%s error=%s",
				s.GetName(),
				i,
				timestampStr,
				err.Error())
			continue // Skip invalid timestamps
		}

		// Check if this is the range calculation period (2:00-2:30)
		if s.isRangeCalculationPeriod(timestamp) {
			wasCalculated := s.rangeCalculated
			s.calculateRange(high, low, close, timestamp)
			if !wasCalculated && s.rangeCalculated {
				rangeProcessed = true
			}
		}

		// Check for entry conditions at 2:30 PM
		if s.isEntryTime(timestamp) && s.rangeCalculated {
			signal := s.checkEntryConditions(high, low, timestamp)
			if signal != nil {
				// Add signal to DataFrame
				result = s.addSignalToDataFrame(result, i, signal)
				signalsGenerated++

				log.Info("2_30_ENTRY_V2: Signal generated - strategy=%s signal_type=%s direction=%s price=%.2f timestamp=%s candle_index=%d high=%.2f low=%.2f",
					s.GetName(),
					signal.Type,
					signal.Direction,
					signal.Price,
					timestamp.Format("15:04:05"),
					i,
					high,
					low)
			}
		}
	}

	processingTime := time.Since(startTime)

	// Log processing completion
	log.Info("2_30_ENTRY_V2: Strategy processing completed - strategy=%s processing_time_ms=%d candles_processed=%d signals_generated=%d range_processed=%t range_calculated=%t can_generate_long=%t can_generate_short=%t direction=%s",
		s.GetName(),
		processingTime.Milliseconds(),
		df.Nrow(),
		signalsGenerated,
		rangeProcessed,
		s.rangeCalculated,
		s.canGenerateLong,
		s.canGenerateShort,
		s.direction)

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
			log.Debug("2_30_ENTRY_V2: Range calculation initialized - strategy=%s timestamp=%s high=%.2f low=%.2f",
				s.GetName(),
				candleTime.Format("15:04:05"),
				high,
				low)
		} else {
			// Update range values
			oldHigh := s.rangeHigh
			oldLow := s.rangeLow
			if high > s.rangeHigh {
				s.rangeHigh = high
			}
			if low < s.rangeLow {
				s.rangeLow = low
			}

			if oldHigh != s.rangeHigh || oldLow != s.rangeLow {
				log.Debug("2_30_ENTRY_V2: Range values updated - strategy=%s timestamp=%s high=%.2f->%.2f low=%.2f->%.2f",
					s.GetName(),
					candleTime.Format("15:04:05"),
					oldHigh, s.rangeHigh,
					oldLow, s.rangeLow)
			}
		}

		// If this is the last candle of the range period (2:30), finalize the calculation
		if s.isEntryTime(candleTime) {
			s.finalizeRangeCalculation(close)
		}
	} else {
		log.Debug("2_30_ENTRY_V2: Range already calculated, skipping - strategy=%s timestamp=%s existing_range_high=%.2f existing_range_low=%.2f",
			s.GetName(),
			candleTime.Format("15:04:05"),
			s.rangeHigh,
			s.rangeLow)
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

	log.Info("2_30_ENTRY_V2: Range calculation finalized - strategy=%s range_high=%.2f range_low=%.2f range_size=%.2f close=%.2f range_mid=%.2f direction=%s range_high_entry_price=%.2f range_low_entry_price=%.2f buffer=%.2f",
		s.GetName(),
		s.rangeHigh,
		s.rangeLow,
		rangeSize,
		close,
		rangeMid,
		s.direction,
		s.rangeHighEntryPrice,
		s.rangeLowEntryPrice,
		buffer)
}

// checkEntryConditions checks for entry conditions at 2:30 PM
func (s *TwoThirtyEntryStrategyV2) checkEntryConditions(high, low float64, timestamp time.Time) *EntrySignal {
	// Check for long entry (breakout above range high)
	if s.canGenerateLong && s.direction == "BULLISH" && high > s.rangeHighEntryPrice {
		s.canGenerateLong = false // Prevent multiple signals

		log.Info("2_30_ENTRY_V2: Long entry condition met - strategy=%s timestamp=%s high=%.2f range_high_entry_price=%.2f breakout_amount=%.2f direction=%s",
			s.GetName(),
			timestamp.Format("15:04:05"),
			high,
			s.rangeHighEntryPrice,
			high-s.rangeHighEntryPrice,
			s.direction)

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

		log.Info("2_30_ENTRY_V2: Short entry condition met - strategy=%s timestamp=%s low=%.2f range_low_entry_price=%.2f breakout_amount=%.2f direction=%s",
			s.GetName(),
			timestamp.Format("15:04:05"),
			low,
			s.rangeLowEntryPrice,
			s.rangeLowEntryPrice-low,
			s.direction)

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

	// Log when conditions are not met for debugging
	log.Debug("2_30_ENTRY_V2: Entry conditions not met - strategy=%s timestamp=%s high=%.2f low=%.2f range_high_entry_price=%.2f range_low_entry_price=%.2f can_generate_long=%t can_generate_short=%t direction=%s",
		s.GetName(),
		timestamp.Format("15:04:05"),
		high,
		low,
		s.rangeHighEntryPrice,
		s.rangeLowEntryPrice,
		s.canGenerateLong,
		s.canGenerateShort,
		s.direction)

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
	log.Info("2_30_ENTRY_V2: Resetting strategy state - strategy=%s previous_range_calculated=%t previous_can_generate_long=%t previous_can_generate_short=%t previous_direction=%s",
		s.GetName(),
		s.rangeCalculated,
		s.canGenerateLong,
		s.canGenerateShort,
		s.direction)

	s.canGenerateLong = true
	s.canGenerateShort = true
	s.rangeHigh = 0
	s.rangeLow = 0
	s.rangeHighEntryPrice = 0
	s.rangeLowEntryPrice = 0
	s.rangeCalculated = false
	s.direction = "NEUTRAL"

	log.Debug("2_30_ENTRY_V2: Strategy state reset completed - strategy=%s range_calculated=%t can_generate_long=%t can_generate_short=%t direction=%s",
		s.GetName(),
		s.rangeCalculated,
		s.canGenerateLong,
		s.canGenerateShort,
		s.direction)
}

// GetRequiredHistory returns the number of historical candles required
func (s *TwoThirtyEntryStrategyV2) GetRequiredHistory() int {
	return 30 // Need at least 30 candles for range calculation (2:00-2:30)
}

// Configure configures the strategy with new parameters
func (s *TwoThirtyEntryStrategyV2) Configure(config map[string]interface{}) error {
	log.Info("2_30_ENTRY_V2: Starting configuration - strategy=%s config_keys=%d",
		s.GetName(),
		len(config))

	// Call base configuration
	err := s.BaseStrategy.Configure(config)
	if err != nil {
		log.Error("2_30_ENTRY_V2: Base configuration failed - strategy=%s error=%s",
			s.GetName(),
			err.Error())
		return err
	}

	// Configure strategy-specific parameters
	if parameters, ok := config["parameters"].(map[string]interface{}); ok {
		log.Debug("2_30_ENTRY_V2: Configuring strategy-specific parameters - strategy=%s parameter_count=%d",
			s.GetName(),
			len(parameters))

		if entryTime, ok := parameters["entry_time"].(string); ok {
			if t, err := time.Parse("15:04", entryTime); err == nil {
				oldTime := s.entryTime
				s.entryTime = time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
				log.Info("2_30_ENTRY_V2: Entry time updated - strategy=%s old_time=%s new_time=%s",
					s.GetName(),
					oldTime.Format("15:04"),
					entryTime)
			} else {
				log.Warn("2_30_ENTRY_V2: Invalid entry time format - strategy=%s time_string=%s error=%s",
					s.GetName(),
					entryTime,
					err.Error())
			}
		}
		if rangeStartTime, ok := parameters["range_start_time"].(string); ok {
			if t, err := time.Parse("15:04", rangeStartTime); err == nil {
				oldTime := s.rangeStartTime
				s.rangeStartTime = time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
				log.Info("2_30_ENTRY_V2: Range start time updated - strategy=%s old_time=%s new_time=%s",
					s.GetName(),
					oldTime.Format("15:04"),
					rangeStartTime)
			} else {
				log.Warn("2_30_ENTRY_V2: Invalid range start time format - strategy=%s time_string=%s error=%s",
					s.GetName(),
					rangeStartTime,
					err.Error())
			}
		}
		if rangeEndTime, ok := parameters["range_end_time"].(string); ok {
			if t, err := time.Parse("15:04", rangeEndTime); err == nil {
				oldTime := s.rangeEndTime
				s.rangeEndTime = time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
				log.Info("2_30_ENTRY_V2: Range end time updated - strategy=%s old_time=%s new_time=%s",
					s.GetName(),
					oldTime.Format("15:04"),
					rangeEndTime)
			} else {
				log.Warn("2_30_ENTRY_V2: Invalid range end time format - strategy=%s time_string=%s error=%s",
					s.GetName(),
					rangeEndTime,
					err.Error())
			}
		}
		if minPriceMovement, ok := parameters["min_price_movement"].(float64); ok {
			oldMovement := s.minPriceMovement
			s.minPriceMovement = minPriceMovement
			log.Info("2_30_ENTRY_V2: Min price movement updated - strategy=%s old_value=%.2f new_value=%.2f",
				s.GetName(),
				oldMovement,
				minPriceMovement)
		}
	}

	log.Info("2_30_ENTRY_V2: Configuration completed successfully - strategy=%s entry_time=%s range_start_time=%s range_end_time=%s min_price_movement=%.2f",
		s.GetName(),
		s.entryTime.Format("15:04"),
		s.rangeStartTime.Format("15:04"),
		s.rangeEndTime.Format("15:04"),
		s.minPriceMovement)

	return nil
}

// ValidateConfiguration validates the strategy configuration
func (s *TwoThirtyEntryStrategyV2) ValidateConfiguration() error {
	log.Debug("2_30_ENTRY_V2: Starting configuration validation - strategy=%s",
		s.GetName())

	// Call base validation
	err := s.BaseStrategy.ValidateConfiguration()
	if err != nil {
		log.Error("2_30_ENTRY_V2: Base configuration validation failed - strategy=%s error=%s",
			s.GetName(),
			err.Error())
		return err
	}

	// Validate strategy-specific parameters
	if s.minPriceMovement <= 0 || s.minPriceMovement > 10 {
		err := fmt.Errorf("invalid min price movement: %f (must be between 0 and 10)", s.minPriceMovement)
		log.Error("2_30_ENTRY_V2: Invalid min price movement - strategy=%s min_price_movement=%.2f error=%s",
			s.GetName(),
			s.minPriceMovement,
			err.Error())
		return err
	}

	log.Info("2_30_ENTRY_V2: Configuration validation completed successfully - strategy=%s entry_time=%s range_start_time=%s range_end_time=%s min_price_movement=%.2f",
		s.GetName(),
		s.entryTime.Format("15:04"),
		s.rangeStartTime.Format("15:04"),
		s.rangeEndTime.Format("15:04"),
		s.minPriceMovement)

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
