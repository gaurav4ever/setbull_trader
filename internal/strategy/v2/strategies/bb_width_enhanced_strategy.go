package strategies

import (
	"math"
	"time"

	v2 "setbull_trader/internal/strategy/v2"
	"setbull_trader/pkg/log"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

// BBWidthEnhancedStrategyV2 implements a comprehensive BB Width strategy matching Python implementation
type BBWidthEnhancedStrategyV2 struct {
	*v2.BaseStrategy

	// Strategy parameters
	bbWidthThreshold   float64
	bbPeriod           int
	bbStdDev           float64
	squeezeDurationMin int
	squeezeDurationMax int

	// Trading hours
	marketOpen  time.Time
	marketClose time.Time

	// Strategy state
	squeezeDetected    bool
	squeezeStartTime   time.Time
	squeezeCandleCount int
	lowestBBWidth      float64
	currentBBWidth     float64
	bbUpper            float64
	bbLower            float64
	bbMiddle           float64
}

// NewBBWidthEnhancedStrategyV2 creates a new enhanced BB Width strategy
func NewBBWidthEnhancedStrategyV2() *BBWidthEnhancedStrategyV2 {
	metadata := v2.StrategyMetadata{
		Name:        "BB_Width_Enhanced_V2",
		Version:     "1.0.0",
		Description: "Enhanced Bollinger Bands Width strategy with squeeze detection and entry conditions",
		Author:      "Setbull Trader",
		Tags:        []string{"technical_analysis", "bollinger_bands", "volatility", "squeeze"},
	}

	base := v2.NewBaseStrategy(metadata)
	strategy := &BBWidthEnhancedStrategyV2{
		BaseStrategy:       base,
		bbWidthThreshold:   0.2, // 20% default
		bbPeriod:           20,
		bbStdDev:           2.0,
		squeezeDurationMin: 3,
		squeezeDurationMax: 5,
		marketOpen:         time.Date(2000, 1, 1, 9, 15, 0, 0, time.UTC),
		marketClose:        time.Date(2000, 1, 1, 15, 30, 0, 0, time.UTC),
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

// Process implements the enhanced strategy processing logic
func (s *BBWidthEnhancedStrategyV2) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
	startTime := time.Now()

	// Log strategy processing start
	log.Info("BB_WIDTH_ENHANCED_V2: Starting strategy processing - strategy=%s version=%s data_rows=%d bb_width_threshold=%.4f bb_period=%d",
		s.GetName(),
		s.GetVersion(),
		df.Nrow(),
		s.bbWidthThreshold,
		s.bbPeriod)

	if df.Nrow() == 0 {
		log.Error("BB_WIDTH_ENHANCED_V2: Invalid dataframe - no rows - strategy=%s error=%s",
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
		log.Error("BB_WIDTH_ENHANCED_V2: Failed to get required series - strategy=%s close_error=%v timestamp_error=%v",
			s.GetName(),
			closeSeries.Err,
			timestampSeries.Err)
		return nil, closeSeries.Err
	}

	log.Debug("BB_WIDTH_ENHANCED_V2: Successfully extracted required series - strategy=%s series_count=%d",
		s.GetName(),
		2)

	// Calculate Bollinger Bands
	bbUpper, bbMiddle, bbLower, bbWidth := s.calculateBollingerBands(closeSeries)

	log.Debug("BB_WIDTH_ENHANCED_V2: Bollinger Bands calculated - strategy=%s bb_upper_count=%d bb_width_count=%d",
		s.GetName(),
		len(bbUpper),
		len(bbWidth))

	// Calculate squeeze detection and entry signals
	squeezeDetected, entrySignal, squeezeMetrics := s.calculateSqueezeAndEntry(
		bbWidth, timestampSeries, closeSeries)

	log.Debug("BB_WIDTH_ENHANCED_V2: Squeeze and entry signals calculated - strategy=%s squeeze_detected_count=%d entry_signal_count=%d",
		s.GetName(),
		len(squeezeDetected),
		len(entrySignal))

	// Add all new columns
	result = result.Mutate(series.New(bbUpper, series.Float, "bb_upper_enhanced"))
	result = result.Mutate(series.New(bbMiddle, series.Float, "bb_middle_enhanced"))
	result = result.Mutate(series.New(bbLower, series.Float, "bb_lower_enhanced"))
	result = result.Mutate(series.New(bbWidth, series.Float, "bb_width_enhanced"))
	result = result.Mutate(series.New(squeezeDetected, series.Bool, "squeeze_detected"))
	result = result.Mutate(series.New(entrySignal, series.Int, "entry_signal"))
	result = result.Mutate(series.New(squeezeMetrics.lowestBBWidth, series.Float, "lowest_bb_width_enhanced"))
	result = result.Mutate(series.New(squeezeMetrics.distanceFromLowest, series.Float, "distance_from_lowest_enhanced"))
	result = result.Mutate(series.New(squeezeMetrics.squeezeDuration, series.Int, "squeeze_duration"))

	processingTime := time.Since(startTime)

	// Log processing completion
	log.Info("BB_WIDTH_ENHANCED_V2: Strategy processing completed - strategy=%s processing_time_ms=%d candles_processed=%d columns_added=%d",
		s.GetName(),
		processingTime.Milliseconds(),
		df.Nrow(),
		9)

	return &result, nil
}

// calculateBollingerBands calculates Bollinger Bands and width
func (s *BBWidthEnhancedStrategyV2) calculateBollingerBands(closeSeries series.Series) ([]float64, []float64, []float64, []float64) {
	closeValues := closeSeries.Float()
	n := len(closeValues)

	bbUpper := make([]float64, n)
	bbMiddle := make([]float64, n)
	bbLower := make([]float64, n)
	bbWidth := make([]float64, n)

	for i := 0; i < n; i++ {
		if i < s.bbPeriod-1 {
			// Not enough data for calculation
			bbUpper[i] = math.NaN()
			bbMiddle[i] = math.NaN()
			bbLower[i] = math.NaN()
			bbWidth[i] = math.NaN()
			continue
		}

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
			diff := closeValues[j] - sma
			variance += diff * diff
		}
		stdDev := math.Sqrt(variance / float64(s.bbPeriod))

		// Calculate upper and lower bands
		bbUpper[i] = sma + (s.bbStdDev * stdDev)
		bbLower[i] = sma - (s.bbStdDev * stdDev)

		// Calculate BB width as percentage
		if sma > 0 {
			bbWidth[i] = (bbUpper[i] - bbLower[i]) / sma * 100
		} else {
			bbWidth[i] = math.NaN()
		}
	}

	return bbUpper, bbMiddle, bbLower, bbWidth
}

// squeezeMetrics holds squeeze-related calculations
type squeezeMetrics struct {
	lowestBBWidth      []float64
	distanceFromLowest []float64
	squeezeDuration    []int
}

// calculateSqueezeAndEntry calculates squeeze detection and entry signals
func (s *BBWidthEnhancedStrategyV2) calculateSqueezeAndEntry(
	bbWidth []float64,
	timestampSeries series.Series,
	closeSeries series.Series,
) ([]bool, []int, squeezeMetrics) {
	n := len(bbWidth)

	log.Debug("BB_WIDTH_ENHANCED_V2: Starting squeeze and entry calculation - strategy=%s bb_width_count=%d bb_width_threshold=%.4f",
		s.GetName(),
		n,
		s.bbWidthThreshold)

	squeezeDetected := make([]bool, n)
	entrySignal := make([]int, n) // 0: no signal, 1: long, -1: short
	lowestBBWidth := make([]float64, n)
	distanceFromLowest := make([]float64, n)
	squeezeDuration := make([]int, n)

	// Calculate lowest BB width in lookback period (50 candles like Python)
	lookbackPeriod := 50
	for i := 0; i < n; i++ {
		if i < lookbackPeriod-1 {
			lowestBBWidth[i] = math.NaN()
			distanceFromLowest[i] = math.NaN()
			squeezeDuration[i] = 0
			continue
		}

		// Find lowest BB width in lookback period
		minWidth := math.Inf(1)
		for j := i - lookbackPeriod + 1; j <= i; j++ {
			if !math.IsNaN(bbWidth[j]) && bbWidth[j] < minWidth {
				minWidth = bbWidth[j]
			}
		}

		if math.IsInf(minWidth, 1) {
			lowestBBWidth[i] = math.NaN()
		} else {
			lowestBBWidth[i] = minWidth
		}

		// Calculate distance from lowest BB width
		if math.IsNaN(bbWidth[i]) || math.IsNaN(lowestBBWidth[i]) || lowestBBWidth[i] == 0 {
			distanceFromLowest[i] = math.NaN()
		} else {
			distanceFromLowest[i] = ((bbWidth[i] - lowestBBWidth[i]) / lowestBBWidth[i]) * 100
		}

		// Check squeeze conditions
		if !math.IsNaN(bbWidth[i]) && bbWidth[i] <= s.bbWidthThreshold {
			// BB width is below threshold - potential squeeze
			if !s.squeezeDetected {
				// Start new squeeze
				s.squeezeDetected = true
				s.squeezeStartTime = time.Now() // In real implementation, use actual timestamp
				s.squeezeCandleCount = 1

				log.Debug("BB_WIDTH_ENHANCED_V2: Squeeze detection started - strategy=%s candle_index=%d bb_width=%.4f threshold=%.4f",
					s.GetName(),
					i,
					bbWidth[i],
					s.bbWidthThreshold)
			} else {
				// Continue existing squeeze
				s.squeezeCandleCount++
			}
			squeezeDetected[i] = true
			squeezeDuration[i] = s.squeezeCandleCount
		} else {
			// BB width above threshold - no squeeze
			if s.squeezeDetected {
				// Check if squeeze duration is within valid range for entry
				if s.squeezeCandleCount >= s.squeezeDurationMin &&
					s.squeezeCandleCount <= s.squeezeDurationMax {
					// Valid squeeze completed - generate entry signal
					entrySignal[i] = s.generateEntrySignal(closeSeries, i)

					if entrySignal[i] != 0 {
						log.Info("BB_WIDTH_ENHANCED_V2: Entry signal generated - strategy=%s candle_index=%d signal=%d squeeze_duration=%d bb_width=%.4f",
							s.GetName(),
							i,
							entrySignal[i],
							s.squeezeCandleCount,
							bbWidth[i])
					}
				} else {
					log.Debug("BB_WIDTH_ENHANCED_V2: Squeeze duration not in range - strategy=%s candle_index=%d duration=%d min=%d max=%d",
						s.GetName(),
						i,
						s.squeezeCandleCount,
						s.squeezeDurationMin,
						s.squeezeDurationMax)
				}
			}
			// Reset squeeze state
			s.squeezeDetected = false
			s.squeezeCandleCount = 0
			squeezeDetected[i] = false
			squeezeDuration[i] = 0
		}
	}

	return squeezeDetected, entrySignal, squeezeMetrics{
		lowestBBWidth:      lowestBBWidth,
		distanceFromLowest: distanceFromLowest,
		squeezeDuration:    squeezeDuration,
	}
}

// generateEntrySignal generates entry signal based on price action
func (s *BBWidthEnhancedStrategyV2) generateEntrySignal(closeSeries series.Series, currentIndex int) int {
	if currentIndex < 2 {
		return 0 // Need at least 3 candles for signal
	}

	closeValues := closeSeries.Float()
	currentClose := closeValues[currentIndex]
	prevClose := closeValues[currentIndex-1]
	prevPrevClose := closeValues[currentIndex-2]

	// Simple breakout logic: if price breaks above previous high, go long
	// if price breaks below previous low, go short
	prevHigh := math.Max(prevClose, prevPrevClose)
	prevLow := math.Min(prevClose, prevPrevClose)

	if currentClose > prevHigh {
		return 1 // Long signal
	} else if currentClose < prevLow {
		return -1 // Short signal
	}

	return 0 // No signal
}

// GetRequiredHistory returns the number of historical candles required
func (s *BBWidthEnhancedStrategyV2) GetRequiredHistory() int {
	return s.bbPeriod + 50 // BB period + lookback period
}

// Configure overrides the base configuration to handle strategy-specific parameters
func (s *BBWidthEnhancedStrategyV2) Configure(config map[string]interface{}) error {
	log.Info("BB_WIDTH_ENHANCED_V2: Starting configuration - strategy=%s config_keys=%d",
		s.GetName(),
		len(config))

	// Call base configuration first
	if err := s.BaseStrategy.Configure(config); err != nil {
		log.Error("BB_WIDTH_ENHANCED_V2: Base configuration failed - strategy=%s error=%s",
			s.GetName(),
			err.Error())
		return err
	}

	// Handle strategy-specific parameters
	if params, ok := config["parameters"].(map[string]interface{}); ok {
		log.Debug("BB_WIDTH_ENHANCED_V2: Configuring strategy-specific parameters - strategy=%s parameter_count=%d",
			s.GetName(),
			len(params))

		if threshold, ok := params["bb_width_threshold"].(float64); ok {
			oldThreshold := s.bbWidthThreshold
			s.bbWidthThreshold = threshold
			log.Info("BB_WIDTH_ENHANCED_V2: BB width threshold updated - strategy=%s old_value=%.4f new_value=%.4f",
				s.GetName(),
				oldThreshold,
				threshold)
		}
		if period, ok := params["bb_period"].(int); ok {
			oldPeriod := s.bbPeriod
			s.bbPeriod = period
			log.Info("BB_WIDTH_ENHANCED_V2: BB period updated - strategy=%s old_value=%d new_value=%d",
				s.GetName(),
				oldPeriod,
				period)
		}
		if stdDev, ok := params["bb_std_dev"].(float64); ok {
			oldStdDev := s.bbStdDev
			s.bbStdDev = stdDev
			log.Info("BB_WIDTH_ENHANCED_V2: BB std dev updated - strategy=%s old_value=%.2f new_value=%.2f",
				s.GetName(),
				oldStdDev,
				stdDev)
		}
		if minDuration, ok := params["squeeze_duration_min"].(int); ok {
			oldMin := s.squeezeDurationMin
			s.squeezeDurationMin = minDuration
			log.Info("BB_WIDTH_ENHANCED_V2: Squeeze duration min updated - strategy=%s old_value=%d new_value=%d",
				s.GetName(),
				oldMin,
				minDuration)
		}
		if maxDuration, ok := params["squeeze_duration_max"].(int); ok {
			oldMax := s.squeezeDurationMax
			s.squeezeDurationMax = maxDuration
			log.Info("BB_WIDTH_ENHANCED_V2: Squeeze duration max updated - strategy=%s old_value=%d new_value=%d",
				s.GetName(),
				oldMax,
				maxDuration)
		}
	}

	log.Info("BB_WIDTH_ENHANCED_V2: Configuration completed successfully - strategy=%s bb_width_threshold=%.4f bb_period=%d bb_std_dev=%.2f squeeze_duration_min=%d squeeze_duration_max=%d",
		s.GetName(),
		s.bbWidthThreshold,
		s.bbPeriod,
		s.bbStdDev,
		s.squeezeDurationMin,
		s.squeezeDurationMax)

	return nil
}

// ValidateConfiguration validates strategy-specific configuration
func (s *BBWidthEnhancedStrategyV2) ValidateConfiguration() error {
	log.Debug("BB_WIDTH_ENHANCED_V2: Starting configuration validation - strategy=%s",
		s.GetName())

	// Call base validation first
	if err := s.BaseStrategy.ValidateConfiguration(); err != nil {
		log.Error("BB_WIDTH_ENHANCED_V2: Base configuration validation failed - strategy=%s error=%s",
			s.GetName(),
			err.Error())
		return err
	}

	// Validate strategy-specific parameters
	if s.bbWidthThreshold <= 0 {
		log.Error("BB_WIDTH_ENHANCED_V2: Invalid BB width threshold - strategy=%s bb_width_threshold=%.4f",
			s.GetName(),
			s.bbWidthThreshold)
		return v2.ErrInvalidTimeout // Reusing error for simplicity
	}
	if s.bbPeriod < 2 {
		log.Error("BB_WIDTH_ENHANCED_V2: Invalid BB period - strategy=%s bb_period=%d",
			s.GetName(),
			s.bbPeriod)
		return v2.ErrInvalidTimeout
	}
	if s.bbStdDev <= 0 {
		log.Error("BB_WIDTH_ENHANCED_V2: Invalid BB std dev - strategy=%s bb_std_dev=%.2f",
			s.GetName(),
			s.bbStdDev)
		return v2.ErrInvalidMaxRetries
	}
	if s.squeezeDurationMin < 1 {
		log.Error("BB_WIDTH_ENHANCED_V2: Invalid squeeze duration min - strategy=%s squeeze_duration_min=%d",
			s.GetName(),
			s.squeezeDurationMin)
		return v2.ErrInvalidTimeout
	}
	if s.squeezeDurationMax < s.squeezeDurationMin {
		log.Error("BB_WIDTH_ENHANCED_V2: Invalid squeeze duration max - strategy=%s squeeze_duration_max=%d squeeze_duration_min=%d",
			s.GetName(),
			s.squeezeDurationMax,
			s.squeezeDurationMin)
		return v2.ErrInvalidMaxRetries
	}

	log.Info("BB_WIDTH_ENHANCED_V2: Configuration validation completed successfully - strategy=%s bb_width_threshold=%.4f bb_period=%d bb_std_dev=%.2f squeeze_duration_min=%d squeeze_duration_max=%d",
		s.GetName(),
		s.bbWidthThreshold,
		s.bbPeriod,
		s.bbStdDev,
		s.squeezeDurationMin,
		s.squeezeDurationMax)

	return nil
}

// GetEstimatedProcessingTime returns estimated processing time
func (s *BBWidthEnhancedStrategyV2) GetEstimatedProcessingTime() time.Duration {
	return 100 * time.Millisecond // Enhanced strategy takes more time
}

// GetMemoryRequirements returns estimated memory requirements
func (s *BBWidthEnhancedStrategyV2) GetMemoryRequirements() int64 {
	return 3 * 1024 * 1024 // 3MB for enhanced calculations
}

// ResetState resets the strategy state (useful for backtesting)
func (s *BBWidthEnhancedStrategyV2) ResetState() {
	log.Info("BB_WIDTH_ENHANCED_V2: Resetting strategy state - strategy=%s previous_squeeze_detected=%t previous_squeeze_candle_count=%d",
		s.GetName(),
		s.squeezeDetected,
		s.squeezeCandleCount)

	s.squeezeDetected = false
	s.squeezeCandleCount = 0
	s.lowestBBWidth = math.Inf(1)
	s.currentBBWidth = 0
	s.bbUpper = 0
	s.bbLower = 0
	s.bbMiddle = 0

	log.Debug("BB_WIDTH_ENHANCED_V2: Strategy state reset completed - strategy=%s squeeze_detected=%t squeeze_candle_count=%d",
		s.GetName(),
		s.squeezeDetected,
		s.squeezeCandleCount)
}
