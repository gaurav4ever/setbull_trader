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

// BBWidthEntryStrategyV2 implements the BB_WIDTH_ENTRY strategy
// It detects Bollinger Bands squeeze and looks for breakout entries
type BBWidthEntryStrategyV2 struct {
	*v2.BaseStrategy

	// Strategy parameters
	bbWidthThreshold   float64
	bbPeriod           int
	bbStdDev           float64
	squeezeDurationMin int
	squeezeDurationMax int

	// Strategy state
	canGenerateLong  bool
	canGenerateShort bool

	// BB Width values
	currentBBWidth     float64
	lowestBBWidth      float64
	squeezeDetected    bool
	squeezeStartTime   time.Time
	squeezeCandleCount int
	bbUpper            float64
	bbLower            float64
	bbMiddle           float64
}

// NewBBWidthEntryStrategyV2 creates a new BB_WIDTH_ENTRY strategy
func NewBBWidthEntryStrategyV2() *BBWidthEntryStrategyV2 {
	metadata := v2.StrategyMetadata{
		Name:        "BB_WIDTH_ENTRY_V2",
		Version:     "1.0.0",
		Description: "Bollinger Bands Width entry strategy - detects squeeze and looks for breakout entries",
		Author:      "Setbull Trader",
		Tags:        []string{"entry_strategy", "bollinger_bands", "squeeze", "breakout"},
	}

	base := v2.NewBaseStrategy(metadata)
	strategy := &BBWidthEntryStrategyV2{
		BaseStrategy:       base,
		bbWidthThreshold:   0.2, // 20% default
		bbPeriod:           20,
		bbStdDev:           2.0,
		squeezeDurationMin: 3,
		squeezeDurationMax: 5,
		canGenerateLong:    true,
		canGenerateShort:   true,
		squeezeDetected:    false,
		lowestBBWidth:      math.Inf(1),
	}

	// Set default configuration
	strategy.Configure(map[string]interface{}{
		"enabled":     true,
		"timeout":     10 * time.Second,
		"max_retries": 3,
		"parameters": map[string]interface{}{
			"bb_width_threshold":   0.2,
			"bb_period":            20,
			"bb_std_dev":           2.0,
			"squeeze_duration_min": 3,
			"squeeze_duration_max": 5,
		},
	})

	return strategy
}

// Process implements the BB_WIDTH_ENTRY strategy processing logic
func (s *BBWidthEntryStrategyV2) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
	startTime := time.Now()

	// Log strategy processing start
	log.Info("BB_WIDTH_ENTRY_V2: Starting strategy processing - strategy=%s version=%s data_rows=%d squeeze_detected=%t can_generate_long=%t can_generate_short=%t bb_width_threshold=%.2f",
		s.GetName(),
		s.GetVersion(),
		df.Nrow(),
		s.squeezeDetected,
		s.canGenerateLong,
		s.canGenerateShort,
		s.bbWidthThreshold)

	if df.Nrow() == 0 {
		log.Error("BB_WIDTH_ENTRY_V2: Invalid dataframe - no rows - strategy=%s error=%s",
			s.GetName(),
			v2.ErrInvalidDataFrame.Error())
		return df, v2.ErrInvalidDataFrame
	}

	// Clone the DataFrame to avoid modifying the original
	result := df.Copy()

	// Get required series
	closeSeries := result.Col("close")
	timestampSeries := result.Col("timestamp")
	if closeSeries.Err != nil || timestampSeries.Err != nil {
		log.Error("BB_WIDTH_ENTRY_V2: Failed to get required series - strategy=%s close_error=%v timestamp_error=%v",
			s.GetName(),
			closeSeries.Err,
			timestampSeries.Err)
		return nil, fmt.Errorf("failed to get required series: %v", closeSeries.Err)
	}

	log.Debug("BB_WIDTH_ENTRY_V2: Successfully extracted required series - strategy=%s series_count=%d",
		s.GetName(),
		2)

	// Calculate Bollinger Bands
	bbUpper, bbMiddle, bbLower, bbWidth := s.calculateBollingerBands(closeSeries)

	// Process each candle
	signalsGenerated := 0
	squeezeDetected := false

	for i := 0; i < df.Nrow(); i++ {
		// Get current candle data
		close := closeSeries.Float()[i]
		timestampStr := timestampSeries.Elem(i).String()

		// Parse timestamp
		timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr)
		if err != nil {
			log.Warn("BB_WIDTH_ENTRY_V2: Skipping invalid timestamp - strategy=%s candle_index=%d timestamp_str=%s error=%s",
				s.GetName(),
				i,
				timestampStr,
				err.Error())
			continue // Skip invalid timestamps
		}

		// Update BB values
		if i < len(bbUpper) {
			s.bbUpper = bbUpper[i]
			s.bbMiddle = bbMiddle[i]
			s.bbLower = bbLower[i]
			s.currentBBWidth = bbWidth[i]
		}

		// Check for squeeze detection
		wasSqueezeDetected := s.squeezeDetected
		s.checkSqueezeDetection(timestamp)
		if !wasSqueezeDetected && s.squeezeDetected {
			squeezeDetected = true
		}

		// Check for entry conditions
		if s.squeezeDetected {
			signal := s.checkEntryConditions(close, timestamp)
			if signal != nil {
				// Add signal to DataFrame
				result = s.addSignalToDataFrame(result, i, signal)
				signalsGenerated++

				log.Info("BB_WIDTH_ENTRY_V2: Signal generated - strategy=%s signal_type=%s direction=%s price=%.2f timestamp=%s candle_index=%d close=%.2f bb_width=%.4f",
					s.GetName(),
					signal.Type,
					signal.Direction,
					signal.Price,
					timestamp.Format("15:04:05"),
					i,
					close,
					s.currentBBWidth)
			}
		}
	}

	processingTime := time.Since(startTime)

	// Log processing completion
	log.Info("BB_WIDTH_ENTRY_V2: Strategy processing completed - strategy=%s processing_time_ms=%d candles_processed=%d signals_generated=%d squeeze_detected=%t current_bb_width=%.4f bb_width_threshold=%.2f",
		s.GetName(),
		processingTime.Milliseconds(),
		df.Nrow(),
		signalsGenerated,
		squeezeDetected,
		s.currentBBWidth,
		s.bbWidthThreshold)

	return &result, nil
}

// calculateBollingerBands calculates Bollinger Bands and width
func (s *BBWidthEntryStrategyV2) calculateBollingerBands(closeSeries series.Series) ([]float64, []float64, []float64, []float64) {
	closeValues := closeSeries.Float()
	n := len(closeValues)

	bbUpper := make([]float64, n)
	bbMiddle := make([]float64, n)
	bbLower := make([]float64, n)
	bbWidth := make([]float64, n)

	for i := s.bbPeriod - 1; i < n; i++ {
		// Calculate SMA (middle band)
		sum := 0.0
		for j := i - s.bbPeriod + 1; j <= i; j++ {
			sum += closeValues[j]
		}
		sma := sum / float64(s.bbPeriod)
		bbMiddle[i] = sma

		// Calculate standard deviation
		variance := 0.0
		for j := i - s.bbPeriod + 1; j <= i; j++ {
			variance += math.Pow(closeValues[j]-sma, 2)
		}
		stdDev := math.Sqrt(variance / float64(s.bbPeriod))

		// Calculate upper and lower bands
		bbUpper[i] = sma + (s.bbStdDev * stdDev)
		bbLower[i] = sma - (s.bbStdDev * stdDev)

		// Calculate BB width
		bbWidth[i] = (bbUpper[i] - bbLower[i]) / sma
	}

	return bbUpper, bbMiddle, bbLower, bbWidth
}

// checkSqueezeDetection checks for Bollinger Bands squeeze
func (s *BBWidthEntryStrategyV2) checkSqueezeDetection(timestamp time.Time) {
	// Check if current BB width is below threshold
	if s.currentBBWidth < s.bbWidthThreshold {
		if !s.squeezeDetected {
			// Start squeeze detection
			s.squeezeDetected = true
			s.squeezeStartTime = timestamp
			s.squeezeCandleCount = 1
			s.lowestBBWidth = s.currentBBWidth

			log.Info("BB_WIDTH_ENTRY_V2: Squeeze detection started - strategy=%s timestamp=%s current_bb_width=%.4f bb_width_threshold=%.2f",
				s.GetName(),
				timestamp.Format("15:04:05"),
				s.currentBBWidth,
				s.bbWidthThreshold)
		} else {
			// Continue squeeze
			s.squeezeCandleCount++
			if s.currentBBWidth < s.lowestBBWidth {
				s.lowestBBWidth = s.currentBBWidth
			}

			log.Debug("BB_WIDTH_ENTRY_V2: Squeeze continuing - strategy=%s timestamp=%s candle_count=%d current_bb_width=%.4f lowest_bb_width=%.4f",
				s.GetName(),
				timestamp.Format("15:04:05"),
				s.squeezeCandleCount,
				s.currentBBWidth,
				s.lowestBBWidth)
		}
	} else {
		// Reset squeeze if width is above threshold
		if s.squeezeDetected {
			log.Info("BB_WIDTH_ENTRY_V2: Squeeze detection ended - strategy=%s timestamp=%s current_bb_width=%.4f bb_width_threshold=%.2f candle_count=%d",
				s.GetName(),
				timestamp.Format("15:04:05"),
				s.currentBBWidth,
				s.bbWidthThreshold,
				s.squeezeCandleCount)

			s.squeezeDetected = false
			s.squeezeCandleCount = 0
			s.lowestBBWidth = math.Inf(1)
		}
	}
}

// checkEntryConditions checks for entry conditions during squeeze
func (s *BBWidthEntryStrategyV2) checkEntryConditions(close float64, timestamp time.Time) *EntrySignal {
	// Check if squeeze duration is within range
	if s.squeezeCandleCount < s.squeezeDurationMin || s.squeezeCandleCount > s.squeezeDurationMax {
		log.Debug("BB_WIDTH_ENTRY_V2: Squeeze duration not in range - strategy=%s timestamp=%s candle_count=%d min_duration=%d max_duration=%d",
			s.GetName(),
			timestamp.Format("15:04:05"),
			s.squeezeCandleCount,
			s.squeezeDurationMin,
			s.squeezeDurationMax)
		return nil
	}

	// Check for long entry (breakout above upper band)
	if s.canGenerateLong && close > s.bbUpper {
		s.canGenerateLong = false // Prevent multiple signals

		log.Info("BB_WIDTH_ENTRY_V2: Long entry condition met - strategy=%s timestamp=%s close=%.2f bb_upper=%.2f breakout_amount=%.2f squeeze_candle_count=%d",
			s.GetName(),
			timestamp.Format("15:04:05"),
			close,
			s.bbUpper,
			close-s.bbUpper,
			s.squeezeCandleCount)

		return &EntrySignal{
			Type:      "LONG_ENTRY",
			Direction: "LONG",
			Price:     s.bbUpper,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"bb_upper":             s.bbUpper,
				"bb_middle":            s.bbMiddle,
				"bb_lower":             s.bbLower,
				"current_bb_width":     s.currentBBWidth,
				"lowest_bb_width":      s.lowestBBWidth,
				"squeeze_candle_count": s.squeezeCandleCount,
				"breakout_price":       close,
			},
		}
	}

	// Check for short entry (breakout below lower band)
	if s.canGenerateShort && close < s.bbLower {
		s.canGenerateShort = false // Prevent multiple signals

		log.Info("BB_WIDTH_ENTRY_V2: Short entry condition met - strategy=%s timestamp=%s close=%.2f bb_lower=%.2f breakout_amount=%.2f squeeze_candle_count=%d",
			s.GetName(),
			timestamp.Format("15:04:05"),
			close,
			s.bbLower,
			s.bbLower-close,
			s.squeezeCandleCount)

		return &EntrySignal{
			Type:      "SHORT_ENTRY",
			Direction: "SHORT",
			Price:     s.bbLower,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"bb_upper":             s.bbUpper,
				"bb_middle":            s.bbMiddle,
				"bb_lower":             s.bbLower,
				"current_bb_width":     s.currentBBWidth,
				"lowest_bb_width":      s.lowestBBWidth,
				"squeeze_candle_count": s.squeezeCandleCount,
				"breakout_price":       close,
			},
		}
	}

	// Log when conditions are not met for debugging
	log.Debug("BB_WIDTH_ENTRY_V2: Entry conditions not met - strategy=%s timestamp=%s close=%.2f bb_upper=%.2f bb_lower=%.2f can_generate_long=%t can_generate_short=%t squeeze_candle_count=%d",
		s.GetName(),
		timestamp.Format("15:04:05"),
		close,
		s.bbUpper,
		s.bbLower,
		s.canGenerateLong,
		s.canGenerateShort,
		s.squeezeCandleCount)

	return nil
}

// addSignalToDataFrame adds signal information to the DataFrame
func (s *BBWidthEntryStrategyV2) addSignalToDataFrame(df dataframe.DataFrame, index int, signal *EntrySignal) dataframe.DataFrame {
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
func (s *BBWidthEntryStrategyV2) ResetState() {
	log.Info("BB_WIDTH_ENTRY_V2: Resetting strategy state - strategy=%s previous_squeeze_detected=%t previous_can_generate_long=%t previous_can_generate_short=%t previous_squeeze_candle_count=%d",
		s.GetName(),
		s.squeezeDetected,
		s.canGenerateLong,
		s.canGenerateShort,
		s.squeezeCandleCount)

	s.canGenerateLong = true
	s.canGenerateShort = true
	s.currentBBWidth = 0
	s.lowestBBWidth = math.Inf(1)
	s.squeezeDetected = false
	s.squeezeCandleCount = 0
	s.bbUpper = 0
	s.bbLower = 0
	s.bbMiddle = 0

	log.Debug("BB_WIDTH_ENTRY_V2: Strategy state reset completed - strategy=%s squeeze_detected=%t can_generate_long=%t can_generate_short=%t squeeze_candle_count=%d",
		s.GetName(),
		s.squeezeDetected,
		s.canGenerateLong,
		s.canGenerateShort,
		s.squeezeCandleCount)
}

// GetRequiredHistory returns the number of historical candles required
func (s *BBWidthEntryStrategyV2) GetRequiredHistory() int {
	return s.bbPeriod + 10 // Need BB period plus some extra for calculations
}

// Configure configures the strategy with new parameters
func (s *BBWidthEntryStrategyV2) Configure(config map[string]interface{}) error {
	log.Info("BB_WIDTH_ENTRY_V2: Starting configuration - strategy=%s config_keys=%d",
		s.GetName(),
		len(config))

	// Call base configuration
	err := s.BaseStrategy.Configure(config)
	if err != nil {
		log.Error("BB_WIDTH_ENTRY_V2: Base configuration failed - strategy=%s error=%s",
			s.GetName(),
			err.Error())
		return err
	}

	// Configure strategy-specific parameters
	if parameters, ok := config["parameters"].(map[string]interface{}); ok {
		log.Debug("BB_WIDTH_ENTRY_V2: Configuring strategy-specific parameters - strategy=%s parameter_count=%d",
			s.GetName(),
			len(parameters))

		if bbWidthThreshold, ok := parameters["bb_width_threshold"].(float64); ok {
			oldThreshold := s.bbWidthThreshold
			s.bbWidthThreshold = bbWidthThreshold
			log.Info("BB_WIDTH_ENTRY_V2: BB width threshold updated - strategy=%s old_value=%.4f new_value=%.4f",
				s.GetName(),
				oldThreshold,
				bbWidthThreshold)
		}
		if bbPeriod, ok := parameters["bb_period"].(int); ok {
			oldPeriod := s.bbPeriod
			s.bbPeriod = bbPeriod
			log.Info("BB_WIDTH_ENTRY_V2: BB period updated - strategy=%s old_value=%d new_value=%d",
				s.GetName(),
				oldPeriod,
				bbPeriod)
		}
		if bbStdDev, ok := parameters["bb_std_dev"].(float64); ok {
			oldStdDev := s.bbStdDev
			s.bbStdDev = bbStdDev
			log.Info("BB_WIDTH_ENTRY_V2: BB std dev updated - strategy=%s old_value=%.2f new_value=%.2f",
				s.GetName(),
				oldStdDev,
				bbStdDev)
		}
		if squeezeDurationMin, ok := parameters["squeeze_duration_min"].(int); ok {
			oldMin := s.squeezeDurationMin
			s.squeezeDurationMin = squeezeDurationMin
			log.Info("BB_WIDTH_ENTRY_V2: Squeeze duration min updated - strategy=%s old_value=%d new_value=%d",
				s.GetName(),
				oldMin,
				squeezeDurationMin)
		}
		if squeezeDurationMax, ok := parameters["squeeze_duration_max"].(int); ok {
			oldMax := s.squeezeDurationMax
			s.squeezeDurationMax = squeezeDurationMax
			log.Info("BB_WIDTH_ENTRY_V2: Squeeze duration max updated - strategy=%s old_value=%d new_value=%d",
				s.GetName(),
				oldMax,
				squeezeDurationMax)
		}
	}

	log.Info("BB_WIDTH_ENTRY_V2: Configuration completed successfully - strategy=%s bb_width_threshold=%.4f bb_period=%d bb_std_dev=%.2f squeeze_duration_min=%d squeeze_duration_max=%d",
		s.GetName(),
		s.bbWidthThreshold,
		s.bbPeriod,
		s.bbStdDev,
		s.squeezeDurationMin,
		s.squeezeDurationMax)

	return nil
}

// ValidateConfiguration validates the strategy configuration
func (s *BBWidthEntryStrategyV2) ValidateConfiguration() error {
	log.Debug("BB_WIDTH_ENTRY_V2: Starting configuration validation - strategy=%s",
		s.GetName())

	// Call base validation
	err := s.BaseStrategy.ValidateConfiguration()
	if err != nil {
		log.Error("BB_WIDTH_ENTRY_V2: Base configuration validation failed - strategy=%s error=%s",
			s.GetName(),
			err.Error())
		return err
	}

	// Validate strategy-specific parameters
	if s.bbWidthThreshold <= 0 || s.bbWidthThreshold > 1 {
		err := fmt.Errorf("invalid BB width threshold: %f (must be between 0 and 1)", s.bbWidthThreshold)
		log.Error("BB_WIDTH_ENTRY_V2: Invalid BB width threshold - strategy=%s bb_width_threshold=%.4f error=%s",
			s.GetName(),
			s.bbWidthThreshold,
			err.Error())
		return err
	}
	if s.bbPeriod <= 0 || s.bbPeriod > 100 {
		err := fmt.Errorf("invalid BB period: %d (must be between 1 and 100)", s.bbPeriod)
		log.Error("BB_WIDTH_ENTRY_V2: Invalid BB period - strategy=%s bb_period=%d error=%s",
			s.GetName(),
			s.bbPeriod,
			err.Error())
		return err
	}
	if s.bbStdDev <= 0 || s.bbStdDev > 5 {
		err := fmt.Errorf("invalid BB std dev: %f (must be between 0 and 5)", s.bbStdDev)
		log.Error("BB_WIDTH_ENTRY_V2: Invalid BB std dev - strategy=%s bb_std_dev=%.2f error=%s",
			s.GetName(),
			s.bbStdDev,
			err.Error())
		return err
	}

	log.Info("BB_WIDTH_ENTRY_V2: Configuration validation completed successfully - strategy=%s bb_width_threshold=%.4f bb_period=%d bb_std_dev=%.2f squeeze_duration_min=%d squeeze_duration_max=%d",
		s.GetName(),
		s.bbWidthThreshold,
		s.bbPeriod,
		s.bbStdDev,
		s.squeezeDurationMin,
		s.squeezeDurationMax)

	return nil
}

// GetEstimatedProcessingTime returns estimated processing time
func (s *BBWidthEntryStrategyV2) GetEstimatedProcessingTime() time.Duration {
	return 100 * time.Millisecond // Longer processing for BB calculations
}

// GetMemoryRequirements returns estimated memory requirements in bytes
func (s *BBWidthEntryStrategyV2) GetMemoryRequirements() int64 {
	return 1024 * 1024 // 1MB for BB_WIDTH_ENTRY strategy
}
