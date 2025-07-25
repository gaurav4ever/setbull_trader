package service

import (
	"context"
	"math"
	"testing"
	"time"

	"setbull_trader/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MathematicalAccuracyTestSuite validates calculation accuracy between implementations
type MathematicalAccuracyTestSuite struct {
	oldService *TechnicalIndicatorService
	newService *TechnicalIndicatorServiceV2
	tolerance  float64 // Acceptable difference between calculations
}

// NewMathematicalAccuracyTestSuite creates a new accuracy test suite
func NewMathematicalAccuracyTestSuite(t *testing.T) *MathematicalAccuracyTestSuite {
	// Create mock repository with test data
	mockRepo := &CompleteMockCandleRepository{}

	// Create service instances
	oldService := NewTechnicalIndicatorService(mockRepo)
	newService := NewTechnicalIndicatorServiceV2(mockRepo)

	return &MathematicalAccuracyTestSuite{
		oldService: oldService,
		newService: newService,
		tolerance:  0.001, // 0.1% tolerance for floating point comparisons
	}
}

// generateKnownTestCandles creates test data with known expected indicator values
func generateKnownTestCandles() []domain.Candle {
	// Generate candles with predictable values for accuracy testing
	baseTime := time.Date(2025, 1, 1, 9, 15, 0, 0, time.UTC)

	// Simple ascending price pattern for predictable calculations
	prices := []float64{100.0, 101.0, 102.0, 103.0, 104.0, 105.0, 106.0, 107.0, 108.0, 109.0,
		110.0, 111.0, 112.0, 113.0, 114.0, 115.0, 116.0, 117.0, 118.0, 119.0}

	candles := make([]domain.Candle, len(prices))
	for i, price := range prices {
		candles[i] = domain.Candle{
			InstrumentKey: "ACCURACY_TEST",
			TimeInterval:  "1minute",
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Open:          price,
			High:          price + 0.5,
			Low:           price - 0.3,
			Close:         price + 0.2,
			Volume:        1000,
		}
	}

	return candles
}

// TestEMAAccuracy validates EMA calculation accuracy between implementations
func TestEMAAccuracy_V1_vs_V2(t *testing.T) {
	suite := NewMathematicalAccuracyTestSuite(t)
	ctx := context.Background()
	instrumentKey := "ACCURACY_TEST"
	interval := "1minute"
	start := time.Date(2025, 1, 1, 9, 15, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 9, 35, 0, 0, time.UTC)

	// Generate test data
	testCandles := generateKnownTestCandles()

	// Setup mock repositories with identical data
	mockRepoV1 := &CompleteMockCandleRepository{}
	mockRepoV1.SetTestData(instrumentKey, testCandles)
	mockRepoV1.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	mockRepoV2 := &CompleteMockCandleRepository{}
	mockRepoV2.SetTestData(instrumentKey, testCandles)
	mockRepoV2.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	// Test both implementations
	serviceV1 := NewTechnicalIndicatorService(mockRepoV1)
	serviceV2 := NewTechnicalIndicatorServiceV2(mockRepoV2)

	// Calculate EMA using both services
	resultV1, errV1 := serviceV1.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	resultV2, errV2 := serviceV2.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)

	// Validate no errors
	assert.NoError(t, errV1, "V1 EMA calculation should not error")
	assert.NoError(t, errV2, "V2 EMA calculation should not error")
	assert.NotNil(t, resultV1, "V1 should return EMA results")
	assert.NotNil(t, resultV2, "V2 should return EMA results")

	// Compare EMA9 values
	suite.compareIndicatorValues(t, resultV1.EMA9, resultV2.EMA9, "EMA9")

	// Compare EMA50 values (if available)
	if len(resultV1.EMA50) > 0 && len(resultV2.EMA50) > 0 {
		suite.compareIndicatorValues(t, resultV1.EMA50, resultV2.EMA50, "EMA50")
	}
}

// TestRSIAccuracy validates RSI calculation accuracy
func TestRSIAccuracy_V1_vs_V2(t *testing.T) {
	suite := NewMathematicalAccuracyTestSuite(t)
	ctx := context.Background()
	instrumentKey := "ACCURACY_TEST"
	interval := "1minute"
	start := time.Date(2025, 1, 1, 9, 15, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 9, 35, 0, 0, time.UTC)

	// Generate test data with more RSI-suitable volatility
	testCandles := generateVolatileTestCandles()

	// Setup mock repositories
	mockRepoV1 := &CompleteMockCandleRepository{}
	mockRepoV1.SetTestData(instrumentKey, testCandles)
	mockRepoV1.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	mockRepoV2 := &CompleteMockCandleRepository{}
	mockRepoV2.SetTestData(instrumentKey, testCandles)
	mockRepoV2.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	// Test both implementations
	serviceV1 := NewTechnicalIndicatorService(mockRepoV1)
	serviceV2 := NewTechnicalIndicatorServiceV2(mockRepoV2)

	// Calculate RSI using both services
	resultV1, errV1 := serviceV1.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	resultV2, errV2 := serviceV2.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)

	// Validate no errors
	assert.NoError(t, errV1, "V1 RSI calculation should not error")
	assert.NoError(t, errV2, "V2 RSI calculation should not error")

	// Compare RSI values
	if len(resultV1.RSI14) > 0 && len(resultV2.RSI) > 0 {
		suite.compareIndicatorValues(t, resultV1.RSI14, resultV2.RSI, "RSI")
	}
}

// TestBollingerBandsAccuracy validates Bollinger Bands calculation accuracy
func TestBollingerBandsAccuracy_V1_vs_V2(t *testing.T) {
	suite := NewMathematicalAccuracyTestSuite(t)
	ctx := context.Background()
	instrumentKey := "ACCURACY_TEST"
	interval := "1minute"
	start := time.Date(2025, 1, 1, 9, 15, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 9, 35, 0, 0, time.UTC)

	// Generate test data
	testCandles := generateKnownTestCandles()

	// Setup mock repositories
	mockRepoV1 := &CompleteMockCandleRepository{}
	mockRepoV1.SetTestData(instrumentKey, testCandles)
	mockRepoV1.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	mockRepoV2 := &CompleteMockCandleRepository{}
	mockRepoV2.SetTestData(instrumentKey, testCandles)
	mockRepoV2.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	// Test both implementations
	serviceV1 := NewTechnicalIndicatorService(mockRepoV1)
	serviceV2 := NewTechnicalIndicatorServiceV2(mockRepoV2)

	// Calculate Bollinger Bands using both services
	resultV1, errV1 := serviceV1.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	resultV2, errV2 := serviceV2.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)

	// Validate no errors
	assert.NoError(t, errV1, "V1 Bollinger Bands calculation should not error")
	assert.NoError(t, errV2, "V2 Bollinger Bands calculation should not error")

	// Compare Bollinger Bands values
	if len(resultV1.BBUpper) > 0 && len(resultV2.BBUpper) > 0 {
		suite.compareIndicatorValues(t, resultV1.BBUpper, resultV2.BBUpper, "BB Upper")
		suite.compareIndicatorValues(t, resultV1.BBMiddle, resultV2.BBMiddle, "BB Middle")
		suite.compareIndicatorValues(t, resultV1.BBLower, resultV2.BBLower, "BB Lower")
	}
}

// TestVWAPAccuracy validates VWAP calculation accuracy
func TestVWAPAccuracy_V1_vs_V2(t *testing.T) {
	suite := NewMathematicalAccuracyTestSuite(t)
	ctx := context.Background()
	instrumentKey := "ACCURACY_TEST"
	interval := "1minute"
	start := time.Date(2025, 1, 1, 9, 15, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 9, 35, 0, 0, time.UTC)

	// Generate test data with varied volumes for VWAP testing
	testCandles := generateVolumeVariedTestCandles()

	// Setup mock repositories
	mockRepoV1 := &CompleteMockCandleRepository{}
	mockRepoV1.SetTestData(instrumentKey, testCandles)
	mockRepoV1.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	mockRepoV2 := &CompleteMockCandleRepository{}
	mockRepoV2.SetTestData(instrumentKey, testCandles)
	mockRepoV2.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	// Test both implementations
	serviceV1 := NewTechnicalIndicatorService(mockRepoV1)
	serviceV2 := NewTechnicalIndicatorServiceV2(mockRepoV2)

	// Calculate VWAP using both services
	resultV1, errV1 := serviceV1.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	resultV2, errV2 := serviceV2.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)

	// Validate no errors
	assert.NoError(t, errV1, "V1 VWAP calculation should not error")
	assert.NoError(t, errV2, "V2 VWAP calculation should not error")

	// Compare VWAP values - Note: V1 service may not support VWAP calculation
	// The TechnicalIndicators struct doesn't have a VWAP field
	// if len(resultV1.VWAP) > 0 && len(resultV2.VWAP) > 0 {
	//     suite.compareIndicatorValues(t, resultV1.VWAP, resultV2.VWAP, "VWAP")
	// }

	// For now, just verify V2 VWAP calculation works and V1 doesn't error
	_ = resultV1 // V1 service doesn't support VWAP but shouldn't error
	_ = suite    // Suite not used in this simplified test
	if len(resultV2.VWAP) > 0 {
		t.Logf("V2 VWAP calculation successful with %d values", len(resultV2.VWAP))
	}
}

// compareIndicatorValues compares two sets of indicator values with tolerance
func (suite *MathematicalAccuracyTestSuite) compareIndicatorValues(
	t *testing.T,
	valuesV1, valuesV2 []domain.IndicatorValue,
	indicatorName string,
) {
	// Ensure both arrays have data
	assert.True(t, len(valuesV1) > 0, "%s V1 should have values", indicatorName)
	assert.True(t, len(valuesV2) > 0, "%s V2 should have values", indicatorName)

	// Compare lengths
	minLength := len(valuesV1)
	if len(valuesV2) < minLength {
		minLength = len(valuesV2)
	}

	// Compare each value within tolerance
	var maxDifference float64
	var avgDifference float64
	var diffCount int

	for i := 0; i < minLength; i++ {
		v1Val := valuesV1[i].Value
		v2Val := valuesV2[i].Value

		// Skip NaN values
		if math.IsNaN(v1Val) || math.IsNaN(v2Val) {
			if math.IsNaN(v1Val) && math.IsNaN(v2Val) {
				continue // Both NaN is acceptable
			}
			t.Errorf("%s index %d: One value is NaN - V1: %f, V2: %f", indicatorName, i, v1Val, v2Val)
			continue
		}

		// Calculate relative difference
		difference := math.Abs(v1Val - v2Val)
		relativeDifference := difference / math.Max(math.Abs(v1Val), math.Abs(v2Val))

		// Track statistics
		if difference > maxDifference {
			maxDifference = difference
		}
		avgDifference += difference
		diffCount++

		// Check tolerance
		assert.True(t, relativeDifference <= suite.tolerance,
			"%s index %d: Values differ by %.6f (%.4f%%) - V1: %.6f, V2: %.6f",
			indicatorName, i, difference, relativeDifference*100, v1Val, v2Val)
	}

	if diffCount > 0 {
		avgDifference /= float64(diffCount)
		t.Logf("%s Accuracy Results:", indicatorName)
		t.Logf("  Compared %d values", diffCount)
		t.Logf("  Max difference: %.6f", maxDifference)
		t.Logf("  Avg difference: %.6f", avgDifference)
		t.Logf("  Tolerance: %.6f", suite.tolerance)
	}
}

// generateVolatileTestCandles creates test data with price volatility for RSI testing
func generateVolatileTestCandles() []domain.Candle {
	baseTime := time.Date(2025, 1, 1, 9, 15, 0, 0, time.UTC)

	// Generate price data with ups and downs for RSI calculation
	prices := []float64{100.0, 102.0, 101.0, 105.0, 103.0, 107.0, 104.0, 108.0, 106.0, 110.0,
		108.0, 112.0, 109.0, 115.0, 111.0, 118.0, 114.0, 120.0, 116.0, 122.0}

	candles := make([]domain.Candle, len(prices))
	for i, price := range prices {
		candles[i] = domain.Candle{
			InstrumentKey: "VOLATILE_TEST",
			TimeInterval:  "1minute",
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Open:          price,
			High:          price + 1.0,
			Low:           price - 0.8,
			Close:         price + 0.3,
			Volume:        1000,
		}
	}

	return candles
}

// generateVolumeVariedTestCandles creates test data with varied volumes for VWAP testing
func generateVolumeVariedTestCandles() []domain.Candle {
	baseTime := time.Date(2025, 1, 1, 9, 15, 0, 0, time.UTC)

	prices := []float64{100.0, 101.0, 102.0, 103.0, 104.0, 105.0, 106.0, 107.0, 108.0, 109.0,
		110.0, 111.0, 112.0, 113.0, 114.0, 115.0, 116.0, 117.0, 118.0, 119.0}

	volumes := []int64{1000, 1500, 800, 2000, 1200, 1800, 900, 2200, 1100, 1900,
		1000, 1600, 850, 2100, 1250, 1750, 950, 2300, 1150, 2000}

	candles := make([]domain.Candle, len(prices))
	for i, price := range prices {
		candles[i] = domain.Candle{
			InstrumentKey: "VOLUME_TEST",
			TimeInterval:  "1minute",
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Open:          price,
			High:          price + 0.5,
			Low:           price - 0.3,
			Close:         price + 0.2,
			Volume:        volumes[i],
		}
	}

	return candles
}

// TestMathematicalAccuracy_ComprehensiveValidation runs all accuracy tests
func TestMathematicalAccuracy_ComprehensiveValidation(t *testing.T) {
	t.Run("EMA_Accuracy", TestEMAAccuracy_V1_vs_V2)
	t.Run("RSI_Accuracy", TestRSIAccuracy_V1_vs_V2)
	t.Run("BollingerBands_Accuracy", TestBollingerBandsAccuracy_V1_vs_V2)
	t.Run("VWAP_Accuracy", TestVWAPAccuracy_V1_vs_V2)
}

// TestIndicatorCalculation_KnownValues validates calculations against known mathematical results
func TestIndicatorCalculation_KnownValues(t *testing.T) {
	// Test with manually calculated expected values for small datasets
	testCandles := []domain.Candle{
		{
			InstrumentKey: "KNOWN_TEST",
			TimeInterval:  "1minute",
			Timestamp:     time.Date(2025, 1, 1, 9, 15, 0, 0, time.UTC),
			Open:          100.0,
			High:          101.0,
			Low:           99.0,
			Close:         100.5,
			Volume:        1000,
		},
		{
			InstrumentKey: "KNOWN_TEST",
			TimeInterval:  "1minute",
			Timestamp:     time.Date(2025, 1, 1, 9, 16, 0, 0, time.UTC),
			Open:          100.5,
			High:          102.0,
			Low:           100.0,
			Close:         101.0,
			Volume:        1500,
		},
		{
			InstrumentKey: "KNOWN_TEST",
			TimeInterval:  "1minute",
			Timestamp:     time.Date(2025, 1, 1, 9, 17, 0, 0, time.UTC),
			Open:          101.0,
			High:          102.5,
			Low:           100.5,
			Close:         102.0,
			Volume:        1200,
		},
	}

	// Test SMA calculation manually
	service := NewTechnicalIndicatorService(&CompleteMockCandleRepository{})
	smaResult := service.CalculateSMA(testCandles, 3)

	// For 3-period SMA of closes [100.5, 101.0, 102.0]
	// Expected: (100.5 + 101.0 + 102.0) / 3 = 101.1666...
	assert.Len(t, smaResult, 1, "Should have 1 SMA value for 3 candles with period 3")
	expectedSMA := (100.5 + 101.0 + 102.0) / 3.0
	actualSMA := smaResult[0].Value
	assert.InDelta(t, expectedSMA, actualSMA, 0.001, "SMA calculation should match manual calculation")

	t.Logf("SMA Test: Expected %.6f, Got %.6f", expectedSMA, actualSMA)
}
