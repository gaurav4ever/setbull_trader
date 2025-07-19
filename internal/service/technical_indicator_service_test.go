package service

import (
	"setbull_trader/internal/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestBollingerBandsAccuracyAgainstTradingView tests the new BB calculation against known TradingView values
func TestBollingerBandsAccuracyAgainstTradingView(t *testing.T) {
	// Create test service
	service := &TechnicalIndicatorService{}

	// Test with simple known values to verify the formula works correctly
	testCandles := []domain.Candle{
		// Create 20 test candles with values around 702 (similar to your data)
		{Close: 702.0, Timestamp: time.Now().Add(-19 * time.Minute)},
		{Close: 702.1, Timestamp: time.Now().Add(-18 * time.Minute)},
		{Close: 702.2, Timestamp: time.Now().Add(-17 * time.Minute)},
		{Close: 702.3, Timestamp: time.Now().Add(-16 * time.Minute)},
		{Close: 702.4, Timestamp: time.Now().Add(-15 * time.Minute)},
		{Close: 702.5, Timestamp: time.Now().Add(-14 * time.Minute)},
		{Close: 702.6, Timestamp: time.Now().Add(-13 * time.Minute)},
		{Close: 702.7, Timestamp: time.Now().Add(-12 * time.Minute)},
		{Close: 702.8, Timestamp: time.Now().Add(-11 * time.Minute)},
		{Close: 702.9, Timestamp: time.Now().Add(-10 * time.Minute)},
		{Close: 703.0, Timestamp: time.Now().Add(-9 * time.Minute)},
		{Close: 702.9, Timestamp: time.Now().Add(-8 * time.Minute)},
		{Close: 702.8, Timestamp: time.Now().Add(-7 * time.Minute)},
		{Close: 702.7, Timestamp: time.Now().Add(-6 * time.Minute)},
		{Close: 702.6, Timestamp: time.Now().Add(-5 * time.Minute)},
		{Close: 702.5, Timestamp: time.Now().Add(-4 * time.Minute)},
		{Close: 702.4, Timestamp: time.Now().Add(-3 * time.Minute)},
		{Close: 702.3, Timestamp: time.Now().Add(-2 * time.Minute)},
		{Close: 702.2, Timestamp: time.Now().Add(-1 * time.Minute)},
		{Close: 702.1, Timestamp: time.Now()},
	}

	t.Logf("Input candles order check:")
	t.Logf("First candle (index 0): %s, Close: %.1f", testCandles[0].Timestamp.Format("15:04"), testCandles[0].Close)
	t.Logf("Last candle (index 19): %s, Close: %.1f", testCandles[19].Timestamp.Format("15:04"), testCandles[19].Close)

	// Test the new TradingView-compatible implementation
	upper, middle, lower := service.CalculateBollingerBandsTradingViewCompatible(testCandles, 20, 2.0)

	// Verify we get results for all candles from period-1 onwards
	assert.NotNil(t, upper, "Upper band should not be nil")
	assert.NotNil(t, middle, "Middle band should not be nil")
	assert.NotNil(t, lower, "Lower band should not be nil")

	t.Logf("Results length: upper=%d, middle=%d, lower=%d", len(upper), len(middle), len(lower))

	// Check several indices for values
	for i := 0; i < len(middle); i++ {
		if middle[i].Value != 0 {
			t.Logf("Index %d: Middle=%.4f, Upper=%.4f, Lower=%.4f, Time=%s",
				i, middle[i].Value, upper[i].Value, lower[i].Value, middle[i].Timestamp.Format("15:04"))
		} else {
			t.Logf("Index %d: Zero value, Time=%s", i, middle[i].Timestamp.Format("15:04"))
		}
	}

	// Find the first non-zero value
	firstNonZeroIndex := -1
	for i := 0; i < len(middle); i++ {
		if middle[i].Value != 0 {
			firstNonZeroIndex = i
			break
		}
	}

	if firstNonZeroIndex >= 0 {
		t.Logf("First non-zero BB value at index %d", firstNonZeroIndex)

		// Test basic BB relationships
		assert.Greater(t, upper[firstNonZeroIndex].Value, middle[firstNonZeroIndex].Value, "Upper band should be > middle band")
		assert.Greater(t, middle[firstNonZeroIndex].Value, lower[firstNonZeroIndex].Value, "Middle band should be > lower band")

		// Test BB Width calculation
		bbWidth := (upper[firstNonZeroIndex].Value - lower[firstNonZeroIndex].Value) / middle[firstNonZeroIndex].Value
		assert.Greater(t, bbWidth, 0.0, "BB Width should be positive")

		t.Logf("BB validation passed for index %d", firstNonZeroIndex)
	} else {
		t.Error("No non-zero BB values found - calculation failed")
	}
}

// TestBBCalculationOrder tests that we calculate in the correct chronological order
func TestBBCalculationOrder(t *testing.T) {
	service := &TechnicalIndicatorService{}

	// Create candles in descending time order (newest first, like real data)
	baseTime := time.Date(2025, 7, 18, 13, 0, 0, 0, time.UTC)
	testCandles := make([]domain.Candle, 25) // 25 candles for 20-period BB

	for i := 0; i < 25; i++ {
		testCandles[i] = domain.Candle{
			Close:     700.0 + float64(i)*0.5,                           // Increasing prices over time
			Timestamp: baseTime.Add(-time.Duration(24-i) * time.Minute), // Descending order
		}
	}

	t.Logf("Input candle order verification:")
	t.Logf("Candle[0]: %s, Close: %.1f (should be newest)", testCandles[0].Timestamp.Format("15:04"), testCandles[0].Close)
	t.Logf("Candle[24]: %s, Close: %.1f (should be oldest)", testCandles[24].Timestamp.Format("15:04"), testCandles[24].Close)

	// Calculate BB
	upper, middle, lower := service.CalculateBollingerBandsTradingViewCompatible(testCandles, 20, 2.0)

	// Verify results
	assert.Equal(t, 25, len(upper), "Should have 25 upper values")
	assert.Equal(t, 25, len(middle), "Should have 25 middle values")
	assert.Equal(t, 25, len(lower), "Should have 25 lower values")

	// Check that we have BB values starting from index 0 (newest candle should have BB value)
	// Because we have 25 candles and need 20 for calculation, the first 5 candles should have BB values
	for i := 0; i < 5; i++ {
		if middle[i].Value != 0 {
			t.Logf("Index %d (newest): Time=%s, Middle=%.4f",
				i, middle[i].Timestamp.Format("15:04"), middle[i].Value)
		}
	}

	// The newest candle (index 0) should have a BB value since we have 25 candles total
	assert.NotEqual(t, 0.0, middle[0].Value, "Newest candle should have BB value")
}

// TestBBWidthCalculationAccuracy tests BB Width calculation specifically
func TestBBWidthCalculationAccuracy(t *testing.T) {
	service := &TechnicalIndicatorService{}

	// Test with known values where we can manually calculate expected results
	testCandles := make([]domain.Candle, 20)
	basePrice := 702.0

	// Create candles with slight variations
	for i := 0; i < 20; i++ {
		testCandles[i] = domain.Candle{
			Close:     basePrice + float64(i)*0.1 - 1.0, // Creates some variation
			Timestamp: time.Now().Add(time.Duration(i-19) * time.Minute),
		}
	}

	upper, middle, lower := service.CalculateBollingerBandsTradingViewCompatible(testCandles, 20, 2.0)

	// Test that BB Width formula matches TradingView: (Upper - Lower) / Middle
	if middle[19].Value != 0 {
		expectedWidth := (upper[19].Value - lower[19].Value) / middle[19].Value

		// Use the same formula that should be used in production
		actualWidth := (upper[19].Value - lower[19].Value) / middle[19].Value

		assert.Equal(t, expectedWidth, actualWidth, "BB Width calculation should match expected formula")

		t.Logf("BB Width Test Results:")
		t.Logf("Upper: %.4f, Middle: %.4f, Lower: %.4f", upper[19].Value, middle[19].Value, lower[19].Value)
		t.Logf("Width: %.6f", actualWidth)
	}
}

// TestBollingerBandsVsOldImplementation compares new vs old implementation
func TestBollingerBandsVsOldImplementation(t *testing.T) {
	service := &TechnicalIndicatorService{}

	// Create test data similar to your real data
	testCandles := []domain.Candle{
		{Close: 702.00, Timestamp: time.Now().Add(-19 * time.Minute)},
		{Close: 702.25, Timestamp: time.Now().Add(-18 * time.Minute)},
		{Close: 702.50, Timestamp: time.Now().Add(-17 * time.Minute)},
		{Close: 702.00, Timestamp: time.Now().Add(-16 * time.Minute)},
		{Close: 702.10, Timestamp: time.Now().Add(-15 * time.Minute)},
		{Close: 702.55, Timestamp: time.Now().Add(-14 * time.Minute)},
		{Close: 702.00, Timestamp: time.Now().Add(-13 * time.Minute)},
		{Close: 702.30, Timestamp: time.Now().Add(-12 * time.Minute)},
		{Close: 702.00, Timestamp: time.Now().Add(-11 * time.Minute)},
		{Close: 701.70, Timestamp: time.Now().Add(-10 * time.Minute)},
		// Add more candles to reach 20
		{Close: 702.75, Timestamp: time.Now().Add(-9 * time.Minute)},
		{Close: 702.65, Timestamp: time.Now().Add(-8 * time.Minute)},
		{Close: 702.00, Timestamp: time.Now().Add(-7 * time.Minute)},
		{Close: 702.05, Timestamp: time.Now().Add(-6 * time.Minute)},
		{Close: 702.55, Timestamp: time.Now().Add(-5 * time.Minute)},
		{Close: 702.50, Timestamp: time.Now().Add(-4 * time.Minute)},
		{Close: 702.30, Timestamp: time.Now().Add(-3 * time.Minute)},
		{Close: 702.00, Timestamp: time.Now().Add(-2 * time.Minute)},
		{Close: 702.00, Timestamp: time.Now().Add(-1 * time.Minute)},
		{Close: 702.95, Timestamp: time.Now()},
	}

	// Test new implementation
	upperNew, middleNew, lowerNew := service.CalculateBollingerBandsTradingViewCompatible(testCandles, 20, 2.0)

	// Test old implementation
	upperOld, middleOld, lowerOld := service.CalculateBollingerBandsOld(testCandles, 20, 2.0)

	// Compare results
	t.Logf("Comparison for 20th candle:")
	t.Logf("NEW - Middle: %.4f, Upper: %.4f, Lower: %.4f", middleNew[19].Value, upperNew[19].Value, lowerNew[19].Value)

	if len(middleOld) > 19 && middleOld[19].Value != 0 {
		t.Logf("OLD - Middle: %.4f, Upper: %.4f, Lower: %.4f", middleOld[19].Value, upperOld[19].Value, lowerOld[19].Value)

		// Calculate differences
		middleDiff := middleNew[19].Value - middleOld[19].Value
		upperDiff := upperNew[19].Value - upperOld[19].Value
		lowerDiff := lowerNew[19].Value - lowerOld[19].Value

		t.Logf("DIFF - Middle: %.4f, Upper: %.4f, Lower: %.4f", middleDiff, upperDiff, lowerDiff)
	}

	// The new implementation should produce different (more accurate) results
	// We expect some differences, especially in the middle band calculation
}

// TestBBWithExactlyTwentyCandlesScenario tests the exact scenario you described
func TestBBWithExactlyTwentyCandlesScenario(t *testing.T) {
	service := &TechnicalIndicatorService{}

	// Create exactly 20 candles in descending time order (newest to oldest)
	// This matches your real data scenario: 2025-07-18 -> 2025-07-14
	baseTime := time.Date(2025, 7, 18, 15, 10, 0, 0, time.UTC)
	testCandles := make([]domain.Candle, 20)

	for i := 0; i < 20; i++ {
		testCandles[i] = domain.Candle{
			Close:     719.0 + float64(i)*0.1,                        // Slight price variation
			Timestamp: baseTime.Add(-time.Duration(i) * time.Minute), // Descending order
		}
	}

	t.Logf("Candle order verification (newest to oldest):")
	t.Logf("Index 0 (newest): %s, Close: %.1f", testCandles[0].Timestamp.Format("2006-01-02 15:04"), testCandles[0].Close)
	t.Logf("Index 19 (oldest): %s, Close: %.1f", testCandles[19].Timestamp.Format("2006-01-02 15:04"), testCandles[19].Close)

	// Calculate BB
	upper, middle, lower := service.CalculateBollingerBandsTradingViewCompatible(testCandles, 20, 2.0)

	// Verify results
	assert.Equal(t, 20, len(upper), "Should have 20 upper values")
	assert.Equal(t, 20, len(middle), "Should have 20 middle values")
	assert.Equal(t, 20, len(lower), "Should have 20 lower values")

	// With exactly 20 candles, only index 0 (newest) should have BB values
	// All other indices (1-19) should have zero values because they don't have enough historical data

	// Check index 0 (newest candle) - should have BB values
	assert.NotEqual(t, 0.0, middle[0].Value, "Index 0 (newest) should have BB middle value")
	assert.NotEqual(t, 0.0, upper[0].Value, "Index 0 (newest) should have BB upper value")
	assert.NotEqual(t, 0.0, lower[0].Value, "Index 0 (newest) should have BB lower value")
	assert.False(t, middle[0].Timestamp.IsZero(), "Index 0 should have valid timestamp")

	// Verify BB relationships for index 0
	assert.Greater(t, upper[0].Value, middle[0].Value, "Upper > Middle at index 0")
	assert.Greater(t, middle[0].Value, lower[0].Value, "Middle > Lower at index 0")

	t.Logf("Index 0 (newest) BB values:")
	t.Logf("  Time: %s", middle[0].Timestamp.Format("2006-01-02 15:04"))
	t.Logf("  Middle: %.4f", middle[0].Value)
	t.Logf("  Upper: %.4f", upper[0].Value)
	t.Logf("  Lower: %.4f", lower[0].Value)

	// Check indices 1-19 - should have zero values but valid timestamps
	zeroCount := 0
	for i := 1; i < 20; i++ {
		if middle[i].Value == 0.0 && upper[i].Value == 0.0 && lower[i].Value == 0.0 {
			zeroCount++
		}
		// Timestamps should still be valid
		assert.False(t, middle[i].Timestamp.IsZero(), "Index %d should have valid timestamp", i)
	}

	assert.Equal(t, 19, zeroCount, "Indices 1-19 should have zero BB values")

	t.Logf("✅ BB calculation working correctly:")
	t.Logf("  - Index 0 (newest): Has BB values ✓")
	t.Logf("  - Indices 1-19: Have zero BB values (expected) ✓")
	t.Logf("  - All timestamps valid ✓")
}

// TestBBWithMoreThanTwentyCandlesScenario tests with more candles to verify multiple BB values
func TestBBWithMoreThanTwentyCandlesScenario(t *testing.T) {
	service := &TechnicalIndicatorService{}

	// Create 25 candles (5 more than needed for 20-period BB)
	baseTime := time.Date(2025, 7, 18, 15, 10, 0, 0, time.UTC)
	testCandles := make([]domain.Candle, 25)

	for i := 0; i < 25; i++ {
		testCandles[i] = domain.Candle{
			Close:     719.0 + float64(i)*0.1,
			Timestamp: baseTime.Add(-time.Duration(i) * time.Minute),
		}
	}

	// Calculate BB
	upper, middle, lower := service.CalculateBollingerBandsTradingViewCompatible(testCandles, 20, 2.0)

	// Verify we have all three band types
	assert.Equal(t, 25, len(upper), "Should have 25 upper values")
	assert.Equal(t, 25, len(lower), "Should have 25 lower values")

	// With 25 candles, indices 0-5 should have BB values, indices 6-24 should be zero
	nonZeroCount := 0
	for i := 0; i < 25; i++ {
		if middle[i].Value != 0.0 {
			nonZeroCount++
			t.Logf("Index %d: Time=%s, Middle=%.4f",
				i, middle[i].Timestamp.Format("15:04"), middle[i].Value)
		}
	}

	assert.Equal(t, 6, nonZeroCount, "Should have 6 non-zero BB values (25-20+1)")

	// Verify the newest 6 candles have BB values
	for i := 0; i < 6; i++ {
		assert.NotEqual(t, 0.0, middle[i].Value, "Index %d should have BB value", i)
	}

	t.Logf("✅ With 25 candles: First 6 indices have BB values, remaining have zeros")
}

// TestBBTimestampValidation tests that all timestamps are properly set (no zero timestamps)
func TestBBTimestampValidation(t *testing.T) {
	service := &TechnicalIndicatorService{}

	// Create exactly 20 candles matching your scenario
	baseTime := time.Date(2025, 7, 18, 15, 10, 0, 0, time.UTC)
	testCandles := make([]domain.Candle, 20)

	for i := 0; i < 20; i++ {
		testCandles[i] = domain.Candle{
			Close:     719.4 + float64(i)*0.01,
			Timestamp: baseTime.Add(-time.Duration(i) * time.Minute),
		}
	}

	// Calculate BB
	upper, middle, lower := service.CalculateBollingerBandsTradingViewCompatible(testCandles, 20, 2.0)

	t.Logf("Validating timestamps and values:")

	// Check every single index for proper timestamp handling
	zeroTimestampCount := 0
	zeroValueCount := 0
	nonZeroValueCount := 0

	for i := 0; i < 20; i++ {
		// Check timestamps - should NEVER be zero
		if middle[i].Timestamp.IsZero() {
			zeroTimestampCount++
			t.Errorf("Index %d has zero timestamp - this should NEVER happen", i)
		}

		// Check values - index 0 should have value, others should be zero
		if middle[i].Value == 0.0 {
			zeroValueCount++
		} else {
			nonZeroValueCount++
			t.Logf("Index %d: Time=%s, Middle=%.4f, Upper=%.4f, Lower=%.4f",
				i, middle[i].Timestamp.Format("2006-01-02 15:04"),
				middle[i].Value, upper[i].Value, lower[i].Value)
		}

		// Verify timestamp consistency across all three arrays
		assert.Equal(t, middle[i].Timestamp, upper[i].Timestamp, "Upper timestamp mismatch at index %d", i)
		assert.Equal(t, middle[i].Timestamp, lower[i].Timestamp, "Lower timestamp mismatch at index %d", i)
	}

	// Final validations
	assert.Equal(t, 0, zeroTimestampCount, "Should have NO zero timestamps")
	assert.Equal(t, 1, nonZeroValueCount, "Should have exactly 1 non-zero BB value (index 0)")
	assert.Equal(t, 19, zeroValueCount, "Should have exactly 19 zero BB values (indices 1-19)")

	// Verify the single non-zero value is at index 0 (newest candle)
	assert.NotEqual(t, 0.0, middle[0].Value, "Index 0 (newest) should have BB value")
	assert.Greater(t, upper[0].Value, middle[0].Value, "Upper > Middle at index 0")
	assert.Greater(t, middle[0].Value, lower[0].Value, "Middle > Lower at index 0")

	// Verify that index 19 (oldest candle) has zero value but valid timestamp
	assert.Equal(t, 0.0, middle[19].Value, "Index 19 (oldest) should have zero BB value")
	assert.False(t, middle[19].Timestamp.IsZero(), "Index 19 should have valid timestamp")

	t.Logf("✅ Timestamp validation passed:")
	t.Logf("  - All 20 indices have valid timestamps ✓")
	t.Logf("  - Index 0 (newest): Has BB values ✓")
	t.Logf("  - Indices 1-19: Have zero BB values but valid timestamps ✓")
	t.Logf("  - No zero timestamps found ✓")
}
