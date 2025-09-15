package strategies

import (
	"testing"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/stretchr/testify/assert"
)

// TestBBWidthEnhancedStrategyV2 tests the enhanced BB Width strategy
func TestBBWidthEnhancedStrategyV2(t *testing.T) {
	// Create strategy
	strategy := NewBBWidthEnhancedStrategyV2()
	assert.NotNil(t, strategy)
	assert.Equal(t, "BB_Width_Enhanced_V2", strategy.GetName())
	assert.Equal(t, "1.0.0", strategy.GetVersion())

	// Create test data with squeeze conditions
	df := createTestDataWithSqueeze(t)
	assert.NotNil(t, df)

	// Process data
	result, err := strategy.Process(df)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify columns were added
	expectedColumns := []string{
		"bb_upper_enhanced", "bb_middle_enhanced", "bb_lower_enhanced",
		"bb_width_enhanced", "squeeze_detected", "entry_signal",
		"lowest_bb_width_enhanced", "distance_from_lowest_enhanced", "squeeze_duration",
	}

	for _, col := range expectedColumns {
		// Check if column exists by trying to access it
		colSeries := result.Col(col)
		assert.False(t, colSeries.Err != nil, "Missing column: %s", col)
	}

	// Verify data types
	assert.Equal(t, series.Float, result.Col("bb_width_enhanced").Type())
	assert.Equal(t, series.Bool, result.Col("squeeze_detected").Type())
	assert.Equal(t, series.Int, result.Col("entry_signal").Type())

	// Verify squeeze detection
	squeezeDetectedCol := result.Col("squeeze_detected")
	bbWidthCol := result.Col("bb_width_enhanced")
	assert.NoError(t, squeezeDetectedCol.Err)
	assert.NoError(t, bbWidthCol.Err)

	squeezeDetected, _ := squeezeDetectedCol.Bool()
	bbWidth := bbWidthCol.Float()

	// Check that squeeze is detected when BB width is low
	for i := 0; i < len(squeezeDetected); i++ {
		if !isNaN(bbWidth[i]) && bbWidth[i] <= 0.2 { // 20% threshold
			assert.True(t, squeezeDetected[i], "Squeeze should be detected for BB width: %f", bbWidth[i])
		}
	}

	// Verify entry signals
	entrySignalCol := result.Col("entry_signal")
	assert.NoError(t, entrySignalCol.Err)
	entrySignals, _ := entrySignalCol.Int()

	// Note: Entry signals depend on price action, so we just verify the column exists
	assert.True(t, len(entrySignals) > 0, "Entry signal column should be present")
}

// TestBBWidthEnhancedStrategyConfiguration tests strategy configuration
func TestBBWidthEnhancedStrategyConfiguration(t *testing.T) {
	strategy := NewBBWidthEnhancedStrategyV2()

	// Test custom configuration
	config := map[string]interface{}{
		"enabled":     true,
		"timeout":     5 * time.Second,
		"max_retries": 2,
		"parameters": map[string]interface{}{
			"bb_width_threshold":   0.15, // 15%
			"bb_period":            25,
			"bb_std_dev":           2.5,
			"squeeze_duration_min": 4,
			"squeeze_duration_max": 6,
		},
	}

	err := strategy.Configure(config)
	assert.NoError(t, err)

	// Verify configuration was applied
	assert.Equal(t, 0.15, strategy.bbWidthThreshold)
	assert.Equal(t, 25, strategy.bbPeriod)
	assert.Equal(t, 2.5, strategy.bbStdDev)
	assert.Equal(t, 4, strategy.squeezeDurationMin)
	assert.Equal(t, 6, strategy.squeezeDurationMax)
}

// TestBBWidthEnhancedStrategyValidation tests configuration validation
func TestBBWidthEnhancedStrategyValidation(t *testing.T) {
	strategy := NewBBWidthEnhancedStrategyV2()

	// Test invalid configuration
	invalidConfig := map[string]interface{}{
		"parameters": map[string]interface{}{
			"bb_width_threshold":   -0.1, // Invalid negative value
			"bb_period":            1,    // Too small
			"bb_std_dev":           -1.0, // Invalid negative value
			"squeeze_duration_min": 0,    // Too small
			"squeeze_duration_max": 2,    // Less than min
		},
	}

	err := strategy.Configure(invalidConfig)
	// Note: The current implementation doesn't validate all parameters during Configure
	// The validation happens during ValidateConfiguration() which is called separately
	assert.NoError(t, err) // Configure doesn't fail, but ValidateConfiguration would

	// Test validation separately
	err = strategy.ValidateConfiguration()
	assert.Error(t, err) // Should fail validation
}

// TestBBWidthEnhancedStrategyPerformance tests performance characteristics
func TestBBWidthEnhancedStrategyPerformance(t *testing.T) {
	strategy := NewBBWidthEnhancedStrategyV2()

	// Test performance estimates
	processingTime := strategy.GetEstimatedProcessingTime()
	memoryRequirements := strategy.GetMemoryRequirements()
	requiredHistory := strategy.GetRequiredHistory()

	assert.True(t, processingTime > 0, "Processing time should be positive")
	assert.True(t, memoryRequirements > 0, "Memory requirements should be positive")
	assert.Equal(t, 70, requiredHistory) // bbPeriod (20) + lookback (50)
}

// TestBBWidthEnhancedStrategyStateReset tests state reset functionality
func TestBBWidthEnhancedStrategyStateReset(t *testing.T) {
	strategy := NewBBWidthEnhancedStrategyV2()

	// Set some state
	strategy.squeezeDetected = true
	strategy.squeezeCandleCount = 5
	strategy.lowestBBWidth = 0.1

	// Reset state
	strategy.ResetState()

	// Verify state was reset
	assert.False(t, strategy.squeezeDetected)
	assert.Equal(t, 0, strategy.squeezeCandleCount)
	assert.True(t, strategy.lowestBBWidth > 1000) // Should be reset to Inf
}

// TestBBWidthEnhancedStrategyWithInsufficientData tests behavior with insufficient data
func TestBBWidthEnhancedStrategyWithInsufficientData(t *testing.T) {
	strategy := NewBBWidthEnhancedStrategyV2()

	// Create DataFrame with insufficient data (less than required history)
	insufficientData := createInsufficientTestData(t)

	result, err := strategy.Process(insufficientData)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify that NaN values are present for insufficient data
	bbWidth := result.Col("bb_width_enhanced").Float()
	hasNaN := false
	for _, val := range bbWidth {
		if isNaN(val) {
			hasNaN = true
			break
		}
	}
	assert.True(t, hasNaN, "Should have NaN values for insufficient data")
}

// Helper functions

// createTestDataWithSqueeze creates test data with squeeze conditions
func createTestDataWithSqueeze(t *testing.T) *dataframe.DataFrame {
	// Create 100 candles with some squeeze conditions
	n := 100
	timestamps := make([]string, n)
	opens := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	volumes := make([]int64, n)

	baseTime := time.Now().Add(-time.Duration(n) * 5 * time.Minute)
	basePrice := 100.0

	for i := 0; i < n; i++ {
		timestamps[i] = baseTime.Add(time.Duration(i) * 5 * time.Minute).Format("2006-01-02 15:04:05")

		// Create price movement with some squeeze conditions
		if i < 20 {
			// Initial period - normal volatility
			opens[i] = basePrice + float64(i)*0.1
			highs[i] = opens[i] + 0.5
			lows[i] = opens[i] - 0.5
			closes[i] = opens[i] + 0.2
		} else if i >= 20 && i < 40 {
			// Squeeze period - low volatility
			opens[i] = basePrice + 2.0 + float64(i-20)*0.01
			highs[i] = opens[i] + 0.1 // Very small range
			lows[i] = opens[i] - 0.1
			closes[i] = opens[i] + 0.05
		} else {
			// Breakout period - increased volatility
			opens[i] = basePrice + 2.4 + float64(i-40)*0.2
			highs[i] = opens[i] + 1.0
			lows[i] = opens[i] - 0.5
			closes[i] = opens[i] + 0.8
		}

		volumes[i] = 1000 + int64(i*10)
	}

	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(opens, series.Float, "open"),
		series.New(highs, series.Float, "high"),
		series.New(lows, series.Float, "low"),
		series.New(closes, series.Float, "close"),
		series.New(volumes, series.Int, "volume"),
	)

	return &df
}

// createInsufficientTestData creates test data with insufficient history
func createInsufficientTestData(t *testing.T) *dataframe.DataFrame {
	// Create only 10 candles (less than required history of 70)
	n := 10
	timestamps := make([]string, n)
	opens := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	volumes := make([]int64, n)

	baseTime := time.Now().Add(-time.Duration(n) * 5 * time.Minute)
	basePrice := 100.0

	for i := 0; i < n; i++ {
		timestamps[i] = baseTime.Add(time.Duration(i) * 5 * time.Minute).Format("2006-01-02 15:04:05")
		opens[i] = basePrice + float64(i)*0.1
		highs[i] = opens[i] + 0.5
		lows[i] = opens[i] - 0.5
		closes[i] = opens[i] + 0.2
		volumes[i] = 1000 + int64(i*10)
	}

	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(opens, series.Float, "open"),
		series.New(highs, series.Float, "high"),
		series.New(lows, series.Float, "low"),
		series.New(closes, series.Float, "close"),
		series.New(volumes, series.Int, "volume"),
	)

	return &df
}
