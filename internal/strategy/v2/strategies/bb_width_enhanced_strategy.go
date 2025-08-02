package strategies

import (
	"math"
	"time"

	v2 "setbull_trader/internal/strategy/v2"

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
	if df.Nrow() == 0 {
		return df, v2.ErrInvalidDataFrame
	}

	// Clone the DataFrame to avoid modifying the original
	result := df.Copy()

	// Get required series
	closeSeries := result.Col("close")
	timestampSeries := result.Col("timestamp")
	if closeSeries.Err != nil || timestampSeries.Err != nil {
		return nil, closeSeries.Err
	}

	// Calculate Bollinger Bands
	bbUpper, bbMiddle, bbLower, bbWidth := s.calculateBollingerBands(closeSeries)

	// Calculate squeeze detection and entry signals
	squeezeDetected, entrySignal, squeezeMetrics := s.calculateSqueezeAndEntry(
		bbWidth, timestampSeries, closeSeries)

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
	// Call base configuration first
	if err := s.BaseStrategy.Configure(config); err != nil {
		return err
	}

	// Handle strategy-specific parameters
	if params, ok := config["parameters"].(map[string]interface{}); ok {
		if threshold, ok := params["bb_width_threshold"].(float64); ok {
			s.bbWidthThreshold = threshold
		}
		if period, ok := params["bb_period"].(int); ok {
			s.bbPeriod = period
		}
		if stdDev, ok := params["bb_std_dev"].(float64); ok {
			s.bbStdDev = stdDev
		}
		if minDuration, ok := params["squeeze_duration_min"].(int); ok {
			s.squeezeDurationMin = minDuration
		}
		if maxDuration, ok := params["squeeze_duration_max"].(int); ok {
			s.squeezeDurationMax = maxDuration
		}
	}

	return nil
}

// ValidateConfiguration validates strategy-specific configuration
func (s *BBWidthEnhancedStrategyV2) ValidateConfiguration() error {
	// Call base validation first
	if err := s.BaseStrategy.ValidateConfiguration(); err != nil {
		return err
	}

	// Validate strategy-specific parameters
	if s.bbWidthThreshold <= 0 {
		return v2.ErrInvalidTimeout // Reusing error for simplicity
	}
	if s.bbPeriod < 2 {
		return v2.ErrInvalidTimeout
	}
	if s.bbStdDev <= 0 {
		return v2.ErrInvalidMaxRetries
	}
	if s.squeezeDurationMin < 1 {
		return v2.ErrInvalidTimeout
	}
	if s.squeezeDurationMax < s.squeezeDurationMin {
		return v2.ErrInvalidMaxRetries
	}

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
	s.squeezeDetected = false
	s.squeezeCandleCount = 0
	s.lowestBBWidth = math.Inf(1)
	s.currentBBWidth = 0
	s.bbUpper = 0
	s.bbLower = 0
	s.bbMiddle = 0
}
