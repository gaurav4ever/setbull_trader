package strategies

import (
	"math"
	"time"

	v2 "setbull_trader/internal/strategy/v2"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

// BBWidthStrategyV2 implements a Bollinger Bands Width strategy using Gota DataFrames
type BBWidthStrategyV2 struct {
	*v2.BaseStrategy
	period   int
	stdDev   float64
	lookback int
}

// NewBBWidthStrategyV2 creates a new BB Width strategy
func NewBBWidthStrategyV2() *BBWidthStrategyV2 {
	metadata := v2.StrategyMetadata{
		Name:        "BB_Width_V2",
		Version:     "1.0.0",
		Description: "Bollinger Bands Width calculation strategy using Gota DataFrames",
		Author:      "Setbull Trader",
		Tags:        []string{"technical_analysis", "bollinger_bands", "volatility"},
	}

	base := v2.NewBaseStrategy(metadata)
	strategy := &BBWidthStrategyV2{
		BaseStrategy: base,
		period:       20,
		stdDev:       2.0,
		lookback:     5,
	}

	// Set default configuration
	strategy.Configure(map[string]interface{}{
		"enabled":     true,
		"timeout":     5 * time.Second,
		"max_retries": 3,
		"parameters": map[string]interface{}{
			"period":   20,
			"std_dev":  2.0,
			"lookback": 5,
		},
	})

	return strategy
}

// Process implements the strategy processing logic
func (s *BBWidthStrategyV2) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
	if df.Nrow() == 0 {
		return df, v2.ErrInvalidDataFrame
	}

	// Clone the DataFrame to avoid modifying the original
	result := df.Copy()

	// Get the close price series
	closeSeries := result.Col("close")
	if closeSeries.Err != nil {
		return nil, closeSeries.Err
	}

	// Calculate Bollinger Bands
	bbUpper, bbMiddle, bbLower, bbWidth := s.calculateBollingerBands(closeSeries)

	// Add new columns to the DataFrame one by one
	result = result.Mutate(series.New(bbUpper, series.Float, "bb_upper_v2"))
	result = result.Mutate(series.New(bbMiddle, series.Float, "bb_middle_v2"))
	result = result.Mutate(series.New(bbLower, series.Float, "bb_lower_v2"))
	result = result.Mutate(series.New(bbWidth, series.Float, "bb_width_v2"))

	// Calculate additional metrics
	s.calculateAdditionalMetrics(&result)

	return &result, nil
}

// calculateBollingerBands calculates Bollinger Bands and width
func (s *BBWidthStrategyV2) calculateBollingerBands(closeSeries series.Series) ([]float64, []float64, []float64, []float64) {
	closeValues := closeSeries.Float()
	n := len(closeValues)

	bbUpper := make([]float64, n)
	bbMiddle := make([]float64, n)
	bbLower := make([]float64, n)
	bbWidth := make([]float64, n)

	for i := 0; i < n; i++ {
		if i < s.period-1 {
			// Not enough data for calculation
			bbUpper[i] = math.NaN()
			bbMiddle[i] = math.NaN()
			bbLower[i] = math.NaN()
			bbWidth[i] = math.NaN()
			continue
		}

		// Calculate SMA (middle band)
		sum := 0.0
		for j := i - s.period + 1; j <= i; j++ {
			sum += closeValues[j]
		}
		sma := sum / float64(s.period)
		bbMiddle[i] = sma

		// Calculate standard deviation
		variance := 0.0
		for j := i - s.period + 1; j <= i; j++ {
			diff := closeValues[j] - sma
			variance += diff * diff
		}
		stdDev := math.Sqrt(variance / float64(s.period))

		// Calculate upper and lower bands
		bbUpper[i] = sma + (s.stdDev * stdDev)
		bbLower[i] = sma - (s.stdDev * stdDev)

		// Calculate BB width as percentage
		if sma > 0 {
			bbWidth[i] = (bbUpper[i] - bbLower[i]) / sma * 100
		} else {
			bbWidth[i] = math.NaN()
		}
	}

	return bbUpper, bbMiddle, bbLower, bbWidth
}

// calculateAdditionalMetrics calculates additional BB width metrics
func (s *BBWidthStrategyV2) calculateAdditionalMetrics(df *dataframe.DataFrame) {
	bbWidthSeries := df.Col("bb_width_v2")
	if bbWidthSeries.Err != nil {
		return
	}

	bbWidthValues := bbWidthSeries.Float()
	n := len(bbWidthValues)

	// Calculate lowest BB width in lookback period
	lowestBBWidth := make([]float64, n)
	for i := 0; i < n; i++ {
		if i < s.lookback-1 {
			lowestBBWidth[i] = math.NaN()
			continue
		}

		minWidth := math.Inf(1)
		for j := i - s.lookback + 1; j <= i; j++ {
			if !math.IsNaN(bbWidthValues[j]) && bbWidthValues[j] < minWidth {
				minWidth = bbWidthValues[j]
			}
		}

		if math.IsInf(minWidth, 1) {
			lowestBBWidth[i] = math.NaN()
		} else {
			lowestBBWidth[i] = minWidth
		}
	}

	// Calculate distance from lowest BB width
	distanceFromLowest := make([]float64, n)
	for i := 0; i < n; i++ {
		if math.IsNaN(bbWidthValues[i]) || math.IsNaN(lowestBBWidth[i]) || lowestBBWidth[i] == 0 {
			distanceFromLowest[i] = math.NaN()
		} else {
			distanceFromLowest[i] = ((bbWidthValues[i] - lowestBBWidth[i]) / lowestBBWidth[i]) * 100
		}
	}

	// Add new columns one by one
	*df = df.Mutate(series.New(lowestBBWidth, series.Float, "lowest_bb_width_v2"))
	*df = df.Mutate(series.New(distanceFromLowest, series.Float, "distance_from_lowest_v2"))
}

// GetRequiredHistory returns the number of historical candles required
func (s *BBWidthStrategyV2) GetRequiredHistory() int {
	return s.period + s.lookback
}

// Configure overrides the base configuration to handle strategy-specific parameters
func (s *BBWidthStrategyV2) Configure(config map[string]interface{}) error {
	// Call base configuration first
	if err := s.BaseStrategy.Configure(config); err != nil {
		return err
	}

	// Handle strategy-specific parameters
	if params, ok := config["parameters"].(map[string]interface{}); ok {
		if period, ok := params["period"].(int); ok {
			s.period = period
		}
		if stdDev, ok := params["std_dev"].(float64); ok {
			s.stdDev = stdDev
		}
		if lookback, ok := params["lookback"].(int); ok {
			s.lookback = lookback
		}
	}

	return nil
}

// ValidateConfiguration validates strategy-specific configuration
func (s *BBWidthStrategyV2) ValidateConfiguration() error {
	// Call base validation first
	if err := s.BaseStrategy.ValidateConfiguration(); err != nil {
		return err
	}

	// Validate strategy-specific parameters
	if s.period < 2 {
		return v2.ErrInvalidTimeout // Reusing error for simplicity
	}
	if s.stdDev <= 0 {
		return v2.ErrInvalidMaxRetries // Reusing error for simplicity
	}
	if s.lookback < 1 {
		return v2.ErrInvalidTimeout // Reusing error for simplicity
	}

	return nil
}

// GetEstimatedProcessingTime returns estimated processing time
func (s *BBWidthStrategyV2) GetEstimatedProcessingTime() time.Duration {
	return 50 * time.Millisecond
}

// GetMemoryRequirements returns estimated memory requirements
func (s *BBWidthStrategyV2) GetMemoryRequirements() int64 {
	return 2 * 1024 * 1024 // 2MB
}
