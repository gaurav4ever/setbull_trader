package strategies

import (
	v2 "setbull_trader/internal/strategy/v2"
	"testing"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBBWidthEntryStrategyV2(t *testing.T) {
	strategy := NewBBWidthEntryStrategyV2(nil)

	assert.NotNil(t, strategy)
	assert.Equal(t, "BB_WIDTH_ENTRY", strategy.GetName())
	assert.Equal(t, 30, strategy.GetRequiredHistory()) // bbPeriod + 10
	assert.Equal(t, 50*time.Millisecond, strategy.GetEstimatedProcessingTime())
	assert.Equal(t, int64(512*1024), strategy.GetMemoryRequirements())

	// Check default parameters
	assert.Equal(t, 0.2, strategy.bbWidthThreshold)
	assert.Equal(t, 20, strategy.bbPeriod)
	assert.Equal(t, 2.0, strategy.bbStdDev)
	assert.Equal(t, 3, strategy.squeezeDurationMin)
	assert.Equal(t, 5, strategy.squeezeDurationMax)

	// Check initial state
	assert.True(t, strategy.canGenerateLong)
	assert.True(t, strategy.canGenerateShort)
	assert.False(t, strategy.squeezeDetected)
	assert.Equal(t, 0, strategy.squeezeCandleCount)
}

func TestBBWidthEntryStrategyValidation(t *testing.T) {
	strategy := NewBBWidthEntryStrategyV2(nil)

	// Test valid configuration
	err := strategy.ValidateConfiguration()
	assert.NoError(t, err)

	// Test invalid BB width threshold
	strategy.bbWidthThreshold = -0.1
	err = strategy.ValidateConfiguration()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bb_width_threshold")

	// Test invalid BB period
	strategy.bbWidthThreshold = 0.2
	strategy.bbPeriod = 0
	err = strategy.ValidateConfiguration()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bb_period")

	// Test invalid BB std dev
	strategy.bbPeriod = 20
	strategy.bbStdDev = 0
	err = strategy.ValidateConfiguration()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bb_std_dev")

	// Test invalid squeeze duration
	strategy.bbStdDev = 2.0
	strategy.squeezeDurationMin = 10
	strategy.squeezeDurationMax = 5
	err = strategy.ValidateConfiguration()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "squeeze_duration_min")
}

func TestBBWidthEntryStrategyConfiguration(t *testing.T) {
	strategy := NewBBWidthEntryStrategyV2(nil)

	config := map[string]interface{}{
		"parameters": map[string]interface{}{
			"bb_width_threshold":   0.3,
			"bb_period":            30,
			"bb_std_dev":           2.5,
			"squeeze_duration_min": 4,
			"squeeze_duration_max": 6,
		},
	}

	err := strategy.Configure(config)
	assert.NoError(t, err)

	assert.Equal(t, 0.3, strategy.bbWidthThreshold)
	assert.Equal(t, 30, strategy.bbPeriod)
	assert.Equal(t, 2.5, strategy.bbStdDev)
	assert.Equal(t, 4, strategy.squeezeDurationMin)
	assert.Equal(t, 6, strategy.squeezeDurationMax)
}

func TestBBWidthEntryStrategyBBDataValidation(t *testing.T) {
	strategy := NewBBWidthEntryStrategyV2(nil)

	// Test valid BB data
	assert.True(t, strategy.validateBBData(100.0, 90.0, 95.0))

	// Test invalid values (zero or negative)
	assert.False(t, strategy.validateBBData(0.0, 90.0, 95.0))
	assert.False(t, strategy.validateBBData(100.0, 0.0, 95.0))
	assert.False(t, strategy.validateBBData(100.0, 90.0, 0.0))

	// Test invalid relationships (upper < lower)
	assert.False(t, strategy.validateBBData(90.0, 100.0, 95.0))

	// Test invalid relationships (middle not between upper and lower)
	assert.False(t, strategy.validateBBData(100.0, 90.0, 110.0))
	assert.False(t, strategy.validateBBData(100.0, 90.0, 80.0))
}

func TestBBWidthEntryStrategyBBWidthHistory(t *testing.T) {
	strategy := NewBBWidthEntryStrategyV2(nil)

	// Test empty history
	lowest := strategy.getLowestBBWidth()
	assert.Equal(t, 0.001, lowest)

	// Test with history
	strategy.updateBBWidthHistory(0.5)
	strategy.updateBBWidthHistory(0.3)
	strategy.updateBBWidthHistory(0.7)

	lowest = strategy.getLowestBBWidth()
	assert.Equal(t, 0.3, lowest)

	// Test history length limit
	for i := 0; i < 60; i++ {
		strategy.updateBBWidthHistory(float64(i))
	}

	assert.Len(t, strategy.bbWidthHistory, 50) // maxHistoryLength
}

func TestBBWidthEntryStrategySqueezeDetection(t *testing.T) {
	strategy := NewBBWidthEntryStrategyV2(nil)

	// Set up lowest BB width
	strategy.lowestBBWidth = 0.1

	// Test squeeze detection (current width <= threshold)
	strategy.currentBBWidth = 0.12 // <= 0.1 * (1 + 0.2) = 0.12
	candleTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	params := &v2.StrategyParameters{}

	strategy.checkSqueezeCondition(candleTime, params)

	assert.True(t, strategy.squeezeDetected)
	assert.Equal(t, 1, strategy.squeezeCandleCount)
	assert.Equal(t, candleTime, *strategy.squeezeStartTime)

	// Test squeeze continuation
	strategy.checkSqueezeCondition(candleTime, params)
	assert.Equal(t, 2, strategy.squeezeCandleCount)

	// Test squeeze end (current width > threshold)
	strategy.currentBBWidth = 0.15 // > 0.12
	strategy.checkSqueezeCondition(candleTime, params)

	assert.False(t, strategy.squeezeDetected)
	assert.Equal(t, 0, strategy.squeezeCandleCount)
	assert.Nil(t, strategy.squeezeStartTime)
}

func TestBBWidthEntryStrategyEntryConditions(t *testing.T) {
	strategy := NewBBWidthEntryStrategyV2(nil)

	// Set up BB values
	strategy.bbUpper = 100.0
	strategy.bbLower = 90.0
	strategy.bbMiddle = 95.0
	strategy.currentBBWidth = 10.0
	strategy.lowestBBWidth = 5.0
	strategy.squeezeDetected = true
	strategy.squeezeCandleCount = 3
	squeezeTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	strategy.squeezeStartTime = &squeezeTime

	candleTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	params := &v2.StrategyParameters{}

	// Test long entry (price above BB upper)
	high := 105.0
	low := 95.0

	signal := strategy.checkEntryConditions(high, low, candleTime, params)

	assert.NotNil(t, signal)
	assert.Equal(t, "BB_WIDTH_ENTRY", signal.Type)
	assert.Equal(t, "LONG", signal.Direction)
	assert.Equal(t, 100.0, signal.Price)
	assert.Equal(t, candleTime, signal.Timestamp)

	// Check metadata
	assert.Equal(t, "data_tracking", signal.Metadata["signal_purpose"])
	assert.Equal(t, 100.0, signal.Metadata["bb_upper"])
	assert.Equal(t, 90.0, signal.Metadata["bb_lower"])
	assert.Equal(t, 95.0, signal.Metadata["bb_middle"])
	assert.Equal(t, 10.0, signal.Metadata["bb_width"])
	assert.Equal(t, 5.0, signal.Metadata["lowest_bb_width"])
	assert.Equal(t, 3, signal.Metadata["squeeze_duration"])

	// Test that long signal generation is disabled after signal
	assert.False(t, strategy.canGenerateLong)

	// Test short entry (price below BB lower)
	strategy.canGenerateShort = true
	high = 95.0
	low = 85.0

	signal = strategy.checkEntryConditions(high, low, candleTime, params)

	assert.NotNil(t, signal)
	assert.Equal(t, "SHORT", signal.Direction)
	assert.Equal(t, 90.0, signal.Price)

	// Test no entry when already generated signals
	signal = strategy.checkEntryConditions(high, low, candleTime, params)
	assert.Nil(t, signal)
}

func TestBBWidthEntryStrategyProcessing(t *testing.T) {
	strategy := NewBBWidthEntryStrategyV2(nil)

	// Create test DataFrame
	timestamps := []string{
		"2024-01-01 09:15:00",
		"2024-01-01 09:20:00",
		"2024-01-01 09:25:00",
		"2024-01-01 09:30:00",
		"2024-01-01 09:35:00",
	}

	highs := []float64{100.0, 101.0, 102.0, 103.0, 105.0}
	lows := []float64{99.0, 100.0, 101.0, 102.0, 103.0}
	bbUppers := []float64{102.0, 102.5, 103.0, 103.5, 104.0}
	bbLowers := []float64{98.0, 98.5, 99.0, 99.5, 100.0}
	bbMiddles := []float64{100.0, 100.5, 101.0, 101.5, 102.0}

	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(highs, series.Float, "high"),
		series.New(lows, series.Float, "low"),
		series.New(bbUppers, series.Float, "bb_upper_x"),
		series.New(bbLowers, series.Float, "bb_lower_x"),
		series.New(bbMiddles, series.Float, "bb_middle_x"),
	)

	// Process the DataFrame
	result, err := strategy.Process(&df)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, df.Nrow(), result.Nrow())

	// Verify that BB width history was updated
	assert.Len(t, strategy.bbWidthHistory, 5)

	// Verify that lowest BB width was calculated
	assert.Greater(t, strategy.lowestBBWidth, 0.0)
}

func TestBBWidthEntryStrategyResetState(t *testing.T) {
	strategy := NewBBWidthEntryStrategyV2(nil)

	// Set some state
	strategy.canGenerateLong = false
	strategy.canGenerateShort = false
	strategy.squeezeDetected = true
	strategy.squeezeCandleCount = 5
	strategy.bbWidthHistory = []float64{0.1, 0.2, 0.3}

	// Reset state
	strategy.ResetState()

	// Verify reset
	assert.True(t, strategy.canGenerateLong)
	assert.True(t, strategy.canGenerateShort)
	assert.False(t, strategy.squeezeDetected)
	assert.Equal(t, 0, strategy.squeezeCandleCount)
	assert.Nil(t, strategy.squeezeStartTime)
	assert.Equal(t, float64(0), strategy.lowestBBWidth)
	assert.Empty(t, strategy.bbWidthHistory)
}

func TestBBWidthEntryStrategyParameterHandling(t *testing.T) {
	strategy := NewBBWidthEntryStrategyV2(nil)

	// Test getOrCreateParameters with nil manager
	params, err := strategy.getOrCreateParameters(time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, params)
	assert.True(t, params.CanGenerateLong)
	assert.True(t, params.CanGenerateShort)

	// Test updateStateFromParameters
	params.CanGenerateLong = false
	params.CanGenerateShort = false
	strategy.updateStateFromParameters(params)

	assert.False(t, strategy.canGenerateLong)
	assert.False(t, strategy.canGenerateShort)

	// Test updateParametersWithState
	strategy.canGenerateLong = true
	strategy.canGenerateShort = true
	strategy.currentBBWidth = 0.5
	strategy.lowestBBWidth = 0.1
	strategy.squeezeDetected = true
	strategy.squeezeCandleCount = 3

	newParams := &v2.StrategyParameters{}
	strategy.updateParametersWithState(newParams, nil)

	assert.True(t, newParams.CanGenerateLong)
	assert.True(t, newParams.CanGenerateShort)
	assert.Equal(t, 0.5, *newParams.CurrentBBWidth)
	assert.Equal(t, 0.1, *newParams.LowestBBWidth)
	assert.True(t, newParams.SqueezeDetected)
	assert.Equal(t, 3, *newParams.SqueezeCandleCount)
}

func TestBBWidthEntryStrategyPerformance(t *testing.T) {
	strategy := NewBBWidthEntryStrategyV2(nil)

	// Create large dataset
	timestamps := make([]string, 1000)
	highs := make([]float64, 1000)
	lows := make([]float64, 1000)
	bbUppers := make([]float64, 1000)
	bbLowers := make([]float64, 1000)
	bbMiddles := make([]float64, 1000)

	for i := 0; i < 1000; i++ {
		timestamps[i] = time.Date(2024, 1, 1, 9, 15+i, 0, 0, time.UTC).Format("2006-01-02 15:04:05")
		highs[i] = 100.0 + float64(i)*0.1
		lows[i] = 99.0 + float64(i)*0.1
		bbUppers[i] = 102.0 + float64(i)*0.1
		bbLowers[i] = 98.0 + float64(i)*0.1
		bbMiddles[i] = 100.0 + float64(i)*0.1
	}

	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(highs, series.Float, "high"),
		series.New(lows, series.Float, "low"),
		series.New(bbUppers, series.Float, "bb_upper_x"),
		series.New(bbLowers, series.Float, "bb_lower_x"),
		series.New(bbMiddles, series.Float, "bb_middle_x"),
	)

	// Measure processing time
	start := time.Now()
	result, err := strategy.Process(&df)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify performance (should be much faster than 50ms for 1000 candles)
	assert.Less(t, duration, 100*time.Millisecond)

	// Verify memory usage is reasonable
	assert.Len(t, strategy.bbWidthHistory, 50) // Should not exceed maxHistoryLength
}
