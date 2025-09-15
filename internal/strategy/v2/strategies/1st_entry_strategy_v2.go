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
	startTime := time.Now()

	// Log strategy processing start
	log.Info("1ST_ENTRY_V2: Starting strategy processing - strategy=%s version=%s data_rows=%d mr_calculated=%t can_generate_long=%t can_generate_short=%t",
		s.GetName(),
		s.GetVersion(),
		df.Nrow(),
		s.mrCalculated,
		s.canGenerateLong,
		s.canGenerateShort)

	if df.Nrow() == 0 {
		log.Error("1ST_ENTRY_V2: Invalid dataframe - no rows - strategy=%s error=%s",
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
		log.Error("1ST_ENTRY_V2: Failed to get required series - strategy=%s high_error=%v low_error=%v close_error=%v timestamp_error=%v",
			s.GetName(),
			highSeries.Err,
			lowSeries.Err,
			closeSeries.Err,
			timestampSeries.Err)
		return nil, fmt.Errorf("failed to get required series: %v", highSeries.Err)
	}

	log.Debug("1ST_ENTRY_V2: Successfully extracted required series - strategy=%s series_count=%d",
		s.GetName(),
		4)

	// Process each candle
	signalsGenerated := 0
	morningRangeProcessed := false

	for i := 0; i < df.Nrow(); i++ {
		// Get current candle data
		high := highSeries.Float()[i]
		low := lowSeries.Float()[i]
		close := closeSeries.Float()[i]
		timestampStr := timestampSeries.Elem(i).String()

		// Parse timestamp
		timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr)
		if err != nil {
			log.Warn("1ST_ENTRY_V2: Skipping invalid timestamp - strategy=%s candle_index=%d timestamp_str=%s error=%s",
				s.GetName(),
				i,
				timestampStr,
				err.Error())
			continue // Skip invalid timestamps
		}

		// Check if this is the morning range calculation period (9:15-9:20)
		if s.isMorningRangeCalculationPeriod(timestamp) {
			wasCalculated := s.mrCalculated
			s.calculateMorningRange(high, low, close, timestamp)
			if !wasCalculated && s.mrCalculated {
				morningRangeProcessed = true
			}
		}

		// Check for entry conditions after morning range is calculated
		if s.isAfterMorningRangeCalculation(timestamp) && s.mrCalculated {
			signal := s.checkEntryConditions(high, low, timestamp)
			if signal != nil {
				// Add signal to DataFrame
				result = s.addSignalToDataFrame(result, i, signal)
				signalsGenerated++

				log.Info("1ST_ENTRY_V2: Signal generated - strategy=%s signal_type=%s direction=%s price=%.2f timestamp=%s candle_index=%d high=%.2f low=%.2f",
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
	log.Info("1ST_ENTRY_V2: Strategy processing completed - strategy=%s processing_time_ms=%d candles_processed=%d signals_generated=%d morning_range_processed=%t mr_calculated=%t can_generate_long=%t can_generate_short=%t",
		s.GetName(),
		processingTime.Milliseconds(),
		df.Nrow(),
		signalsGenerated,
		morningRangeProcessed,
		s.mrCalculated,
		s.canGenerateLong,
		s.canGenerateShort)

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

		log.Info("1ST_ENTRY_V2: Morning Range calculated - strategy=%s timestamp=%s mr_high=%.2f mr_low=%.2f mr_high_with_buffer=%.2f mr_low_with_buffer=%.2f buffer_percentage=%.4f close_price=%.2f",
			s.GetName(),
			candleTime.Format("15:04:05"),
			s.mrHigh,
			s.mrLow,
			s.mrHighWithBuffer,
			s.mrLowWithBuffer,
			s.bufferPercentage,
			close)
	} else {
		log.Debug("1ST_ENTRY_V2: Morning Range already calculated, skipping - strategy=%s timestamp=%s existing_mr_high=%.2f existing_mr_low=%.2f",
			s.GetName(),
			candleTime.Format("15:04:05"),
			s.mrHigh,
			s.mrLow)
	}
}

// checkEntryConditions checks for entry conditions based on MR breakout
func (s *FirstEntryStrategyV2) checkEntryConditions(high, low float64, timestamp time.Time) *EntrySignal {
	// Check for long entry (breakout above MR high with buffer)
	if s.canGenerateLong && high > s.mrHighWithBuffer {
		s.canGenerateLong = false // Prevent multiple signals

		log.Info("1ST_ENTRY_V2: Long entry condition met - strategy=%s timestamp=%s high=%.2f mr_high_with_buffer=%.2f breakout_amount=%.2f mr_high=%.2f",
			s.GetName(),
			timestamp.Format("15:04:05"),
			high,
			s.mrHighWithBuffer,
			high-s.mrHighWithBuffer,
			s.mrHigh)

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

		log.Info("1ST_ENTRY_V2: Short entry condition met - strategy=%s timestamp=%s low=%.2f mr_low_with_buffer=%.2f breakout_amount=%.2f mr_low=%.2f",
			s.GetName(),
			timestamp.Format("15:04:05"),
			low,
			s.mrLowWithBuffer,
			s.mrLowWithBuffer-low,
			s.mrLow)

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

	// Log when conditions are not met for debugging
	log.Debug("1ST_ENTRY_V2: Entry conditions not met - strategy=%s timestamp=%s high=%.2f low=%.2f mr_high_with_buffer=%.2f mr_low_with_buffer=%.2f can_generate_long=%t can_generate_short=%t",
		s.GetName(),
		timestamp.Format("15:04:05"),
		high,
		low,
		s.mrHighWithBuffer,
		s.mrLowWithBuffer,
		s.canGenerateLong,
		s.canGenerateShort)

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

	log.Debug("1ST_ENTRY_V2: Buffer values calculated - strategy=%s buffer_percentage=%.4f buffer_ticks=%.4f mr_high=%.2f mr_low=%.2f mr_high_with_buffer=%.2f mr_low_with_buffer=%.2f",
		s.GetName(),
		s.bufferPercentage,
		bufferTicks,
		s.mrHigh,
		s.mrLow,
		s.mrHighWithBuffer,
		s.mrLowWithBuffer)
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
	log.Info("1ST_ENTRY_V2: Resetting strategy state - strategy=%s previous_mr_calculated=%t previous_can_generate_long=%t previous_can_generate_short=%t",
		s.GetName(),
		s.mrCalculated,
		s.canGenerateLong,
		s.canGenerateShort)

	s.canGenerateLong = true
	s.canGenerateShort = true
	s.mrHigh = 0
	s.mrLow = 0
	s.mrHighWithBuffer = 0
	s.mrLowWithBuffer = 0
	s.mrCalculated = false

	log.Debug("1ST_ENTRY_V2: Strategy state reset completed - strategy=%s mr_calculated=%t can_generate_long=%t can_generate_short=%t",
		s.GetName(),
		s.mrCalculated,
		s.canGenerateLong,
		s.canGenerateShort)
}

// GetRequiredHistory returns the number of historical candles required
func (s *FirstEntryStrategyV2) GetRequiredHistory() int {
	return 5 // Need at least 5 candles for morning range calculation
}

// Configure configures the strategy with new parameters
func (s *FirstEntryStrategyV2) Configure(config map[string]interface{}) error {
	log.Info("1ST_ENTRY_V2: Starting configuration - strategy=%s config_keys=%d",
		s.GetName(),
		len(config))

	// Call base configuration
	err := s.BaseStrategy.Configure(config)
	if err != nil {
		log.Error("1ST_ENTRY_V2: Base configuration failed - strategy=%s error=%s",
			s.GetName(),
			err.Error())
		return err
	}

	// Configure strategy-specific parameters
	if parameters, ok := config["parameters"].(map[string]interface{}); ok {
		log.Debug("1ST_ENTRY_V2: Configuring strategy-specific parameters - strategy=%s parameter_count=%d",
			s.GetName(),
			len(parameters))

		if bufferPercentage, ok := parameters["buffer_percentage"].(float64); ok {
			oldBuffer := s.bufferPercentage
			s.bufferPercentage = bufferPercentage
			log.Info("1ST_ENTRY_V2: Buffer percentage updated - strategy=%s old_value=%.4f new_value=%.4f",
				s.GetName(),
				oldBuffer,
				bufferPercentage)
		}
		if marketOpen, ok := parameters["market_open"].(string); ok {
			// Parse market open time
			if t, err := time.Parse("15:04", marketOpen); err == nil {
				oldTime := s.marketOpen
				s.marketOpen = time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
				log.Info("1ST_ENTRY_V2: Market open time updated - strategy=%s old_time=%s new_time=%s",
					s.GetName(),
					oldTime.Format("15:04"),
					marketOpen)
			} else {
				log.Warn("1ST_ENTRY_V2: Invalid market open time format - strategy=%s time_string=%s error=%s",
					s.GetName(),
					marketOpen,
					err.Error())
			}
		}
		if morningRangeEnd, ok := parameters["morning_range_end"].(string); ok {
			// Parse morning range end time
			if t, err := time.Parse("15:04", morningRangeEnd); err == nil {
				oldTime := s.morningRangeEnd
				s.morningRangeEnd = time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
				log.Info("1ST_ENTRY_V2: Morning range end time updated - strategy=%s old_time=%s new_time=%s",
					s.GetName(),
					oldTime.Format("15:04"),
					morningRangeEnd)
			} else {
				log.Warn("1ST_ENTRY_V2: Invalid morning range end time format - strategy=%s time_string=%s error=%s",
					s.GetName(),
					morningRangeEnd,
					err.Error())
			}
		}
	}

	log.Info("1ST_ENTRY_V2: Configuration completed successfully - strategy=%s buffer_percentage=%.4f market_open=%s morning_range_end=%s",
		s.GetName(),
		s.bufferPercentage,
		s.marketOpen.Format("15:04"),
		s.morningRangeEnd.Format("15:04"))

	return nil
}

// ValidateConfiguration validates the strategy configuration
func (s *FirstEntryStrategyV2) ValidateConfiguration() error {
	log.Debug("1ST_ENTRY_V2: Starting configuration validation - strategy=%s",
		s.GetName())

	// Call base validation
	err := s.BaseStrategy.ValidateConfiguration()
	if err != nil {
		log.Error("1ST_ENTRY_V2: Base configuration validation failed - strategy=%s error=%s",
			s.GetName(),
			err.Error())
		return err
	}

	// Validate strategy-specific parameters
	if s.bufferPercentage <= 0 || s.bufferPercentage > 0.1 {
		err := fmt.Errorf("invalid buffer percentage: %f (must be between 0 and 0.1)", s.bufferPercentage)
		log.Error("1ST_ENTRY_V2: Invalid buffer percentage - strategy=%s buffer_percentage=%.4f error=%s",
			s.GetName(),
			s.bufferPercentage,
			err.Error())
		return err
	}

	log.Info("1ST_ENTRY_V2: Configuration validation completed successfully - strategy=%s buffer_percentage=%.4f market_open=%s morning_range_end=%s",
		s.GetName(),
		s.bufferPercentage,
		s.marketOpen.Format("15:04"),
		s.morningRangeEnd.Format("15:04"))

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
