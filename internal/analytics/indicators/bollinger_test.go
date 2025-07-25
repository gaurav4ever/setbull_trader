package indicators

import (
	"math"
	"testing"
	"time"

	"setbull_trader/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBollingerCalculator(t *testing.T) {
	calc := NewBollingerCalculator()

	assert.NotNil(t, calc)
	assert.NotNil(t, calc.calculator)
}

func TestBollingerCalculator_CalculateBollingerBands_EmptyCandles(t *testing.T) {
	calc := NewBollingerCalculator()
	result := calc.CalculateBollingerBands([]domain.Candle{}, 20, 2.0)

	assert.NotNil(t, result)
	assert.Empty(t, result.Upper)
	assert.Empty(t, result.Middle)
	assert.Empty(t, result.Lower)
	assert.Empty(t, result.Width)
}

func TestBollingerCalculator_CalculateBollingerBands_ValidCalculation(t *testing.T) {
	calc := NewBollingerCalculator()
	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	// Create test candles with ascending close prices
	candles := make([]domain.Candle, 25)
	for i := 0; i < 25; i++ {
		candles[i] = domain.Candle{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Open:          100.0 + float64(i)*0.5,
			High:          105.0 + float64(i)*0.5,
			Low:           98.0 + float64(i)*0.5,
			Close:         102.0 + float64(i)*0.5,
			Volume:        1000,
		}
	}

	period := 20
	multiplier := 2.0
	result := calc.CalculateBollingerBands(candles, period, multiplier)

	require.NotNil(t, result)
	require.Len(t, result.Upper, len(candles))
	require.Len(t, result.Middle, len(candles))
	require.Len(t, result.Lower, len(candles))
	require.Len(t, result.Width, len(candles))

	// First period-1 values should be NaN
	for i := 0; i < period-1; i++ {
		assert.True(t, math.IsNaN(result.Upper[i].Value), "Expected NaN at index %d", i)
		assert.True(t, math.IsNaN(result.Middle[i].Value), "Expected NaN at index %d", i)
		assert.True(t, math.IsNaN(result.Lower[i].Value), "Expected NaN at index %d", i)
		assert.True(t, math.IsNaN(result.Width[i].Value), "Expected NaN at index %d", i)
	}

	// Valid values should maintain band relationships
	for i := period - 1; i < len(candles); i++ {
		if !math.IsNaN(result.Upper[i].Value) && !math.IsNaN(result.Lower[i].Value) && !math.IsNaN(result.Middle[i].Value) {
			assert.True(t, result.Upper[i].Value > result.Middle[i].Value, "Upper should be above middle at index %d", i)
			assert.True(t, result.Lower[i].Value < result.Middle[i].Value, "Lower should be below middle at index %d", i)
			assert.True(t, result.Width[i].Value > 0, "Width should be positive at index %d", i)
		}

		// Verify timestamps
		assert.Equal(t, candles[i].Timestamp, result.Upper[i].Timestamp)
		assert.Equal(t, candles[i].Timestamp, result.Middle[i].Timestamp)
		assert.Equal(t, candles[i].Timestamp, result.Lower[i].Timestamp)
		assert.Equal(t, candles[i].Timestamp, result.Width[i].Timestamp)
	}
}

func TestBollingerCalculator_CalculateBollingerBandsCompatible(t *testing.T) {
	calc := NewBollingerCalculator()
	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	candles := make([]domain.Candle, 25)
	for i := 0; i < 25; i++ {
		candles[i] = domain.Candle{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Close:         102.0 + float64(i)*0.5,
		}
	}

	upper, middle, lower := calc.CalculateBollingerBandsCompatible(candles, 20, 2.0)

	require.Len(t, upper, len(candles))
	require.Len(t, middle, len(candles))
	require.Len(t, lower, len(candles))

	// Should be equivalent to full calculation
	fullResult := calc.CalculateBollingerBands(candles, 20, 2.0)
	for i := 0; i < len(candles); i++ {
		if math.IsNaN(fullResult.Upper[i].Value) && math.IsNaN(upper[i].Value) {
			// Both are NaN, which is expected for early periods
			continue
		}
		assert.Equal(t, fullResult.Upper[i].Value, upper[i].Value)

		if math.IsNaN(fullResult.Middle[i].Value) && math.IsNaN(middle[i].Value) {
			// Both are NaN, which is expected for early periods
			continue
		}
		assert.Equal(t, fullResult.Middle[i].Value, middle[i].Value)

		if math.IsNaN(fullResult.Lower[i].Value) && math.IsNaN(lower[i].Value) {
			// Both are NaN, which is expected for early periods
			continue
		}
		assert.Equal(t, fullResult.Lower[i].Value, lower[i].Value)
	}
}

func TestBollingerCalculator_CalculateBBWidth_EmptyInputs(t *testing.T) {
	calc := NewBollingerCalculator()
	result := calc.CalculateBBWidth([]domain.IndicatorValue{}, []domain.IndicatorValue{}, []domain.IndicatorValue{})

	assert.Empty(t, result)
}

func TestBollingerCalculator_CalculateBBWidth_ValidCalculation(t *testing.T) {
	calc := NewBollingerCalculator()
	timestamp := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	upper := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 110.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 112.0},
		{Timestamp: timestamp.Add(2 * time.Minute), Value: 114.0},
	}
	middle := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 105.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 107.0},
		{Timestamp: timestamp.Add(2 * time.Minute), Value: 109.0},
	}
	lower := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 100.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 102.0},
		{Timestamp: timestamp.Add(2 * time.Minute), Value: 104.0},
	}

	result := calc.CalculateBBWidth(upper, middle, lower)

	require.Len(t, result, 3)

	// Calculate expected widths
	expectedWidth1 := (110.0 - 100.0) / 105.0
	expectedWidth2 := (112.0 - 102.0) / 107.0
	expectedWidth3 := (114.0 - 104.0) / 109.0

	assert.InDelta(t, expectedWidth1, result[0].Value, 1e-10)
	assert.InDelta(t, expectedWidth2, result[1].Value, 1e-10)
	assert.InDelta(t, expectedWidth3, result[2].Value, 1e-10)

	// Verify timestamps
	assert.Equal(t, timestamp, result[0].Timestamp)
	assert.Equal(t, timestamp.Add(time.Minute), result[1].Timestamp)
	assert.Equal(t, timestamp.Add(2*time.Minute), result[2].Timestamp)
}

func TestBollingerCalculator_CalculateBBWidth_WithNaN(t *testing.T) {
	calc := NewBollingerCalculator()
	timestamp := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	upper := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: math.NaN()},
		{Timestamp: timestamp.Add(time.Minute), Value: 112.0},
	}
	middle := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: math.NaN()},
		{Timestamp: timestamp.Add(time.Minute), Value: 107.0},
	}
	lower := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: math.NaN()},
		{Timestamp: timestamp.Add(time.Minute), Value: 102.0},
	}

	result := calc.CalculateBBWidth(upper, middle, lower)

	require.Len(t, result, 2)
	assert.True(t, math.IsNaN(result[0].Value))
	assert.False(t, math.IsNaN(result[1].Value))
}

func TestBollingerCalculator_CalculateBBWidthNormalized(t *testing.T) {
	calc := NewBollingerCalculator()
	timestamp := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	upper := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 110.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 115.0},
		{Timestamp: timestamp.Add(2 * time.Minute), Value: 120.0},
	}
	middle := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 100.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 100.0},
		{Timestamp: timestamp.Add(2 * time.Minute), Value: 100.0},
	}
	lower := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 90.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 85.0},
		{Timestamp: timestamp.Add(2 * time.Minute), Value: 80.0},
	}

	result := calc.CalculateBBWidthNormalized(upper, middle, lower)

	require.Len(t, result, 3)

	// Calculate expected widths
	width1 := (110.0 - 90.0) / 100.0 // 0.2
	width2 := (115.0 - 85.0) / 100.0 // 0.3
	width3 := (120.0 - 80.0) / 100.0 // 0.4

	// Normalize: min = 0.2, max = 0.4, range = 0.2
	expectedNorm1 := (width1 - 0.2) / 0.2 // 0.0
	expectedNorm2 := (width2 - 0.2) / 0.2 // 0.5
	expectedNorm3 := (width3 - 0.2) / 0.2 // 1.0

	assert.InDelta(t, expectedNorm1, result[0].Value, 1e-10)
	assert.InDelta(t, expectedNorm2, result[1].Value, 1e-10)
	assert.InDelta(t, expectedNorm3, result[2].Value, 1e-10)
}

func TestBollingerCalculator_CalculateBBWidthPercentage(t *testing.T) {
	calc := NewBollingerCalculator()
	timestamp := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	upper := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 110.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 112.0},
	}
	middle := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 100.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 100.0},
	}
	lower := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 90.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 88.0},
	}

	result := calc.CalculateBBWidthPercentage(upper, middle, lower)

	require.Len(t, result, 2)

	// Calculate expected percentages
	width1 := (110.0 - 90.0) / 100.0 * 100 // 20%
	width2 := (112.0 - 88.0) / 100.0 * 100 // 24%

	assert.InDelta(t, width1, result[0].Value, 1e-10)
	assert.InDelta(t, width2, result[1].Value, 1e-10)
}

func TestBollingerCalculator_GetBBSqueeze(t *testing.T) {
	calc := NewBollingerCalculator()
	timestamp := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	upper := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 105.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 110.0},
		{Timestamp: timestamp.Add(2 * time.Minute), Value: 103.0},
	}
	middle := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 100.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 100.0},
		{Timestamp: timestamp.Add(2 * time.Minute), Value: 100.0},
	}
	lower := []domain.IndicatorValue{
		{Timestamp: timestamp, Value: 95.0},
		{Timestamp: timestamp.Add(time.Minute), Value: 90.0},
		{Timestamp: timestamp.Add(2 * time.Minute), Value: 97.0},
	}

	squeezeThreshold := 0.08

	result := calc.GetBBSqueeze(upper, middle, lower, squeezeThreshold)

	require.Len(t, result, 3)

	// Expected results:
	// width1 = (105.0 - 95.0) / 100.0 = 0.1 > 0.08 → no squeeze
	// width2 = (110.0 - 90.0) / 100.0 = 0.2 > 0.08 → no squeeze
	// width3 = (103.0 - 97.0) / 100.0 = 0.06 <= 0.08 → squeeze

	assert.Equal(t, 0.0, result[0].Value) // No squeeze
	assert.Equal(t, 0.0, result[1].Value) // No squeeze
	assert.Equal(t, 1.0, result[2].Value) // Squeeze detected
}

func TestBollingerCalculator_GetBBPosition(t *testing.T) {
	calc := NewBollingerCalculator()
	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	candles := []domain.Candle{
		{
			Timestamp: baseTime,
			Close:     95.0, // Below lower band
		},
		{
			Timestamp: baseTime.Add(time.Minute),
			Close:     100.0, // At middle
		},
		{
			Timestamp: baseTime.Add(2 * time.Minute),
			Close:     105.0, // Above upper band
		},
		{
			Timestamp: baseTime.Add(3 * time.Minute),
			Close:     102.5, // Between middle and upper
		},
	}

	upper := []domain.IndicatorValue{
		{Timestamp: baseTime, Value: 110.0},
		{Timestamp: baseTime.Add(time.Minute), Value: 110.0},
		{Timestamp: baseTime.Add(2 * time.Minute), Value: 110.0},
		{Timestamp: baseTime.Add(3 * time.Minute), Value: 110.0},
	}
	middle := []domain.IndicatorValue{
		{Timestamp: baseTime, Value: 100.0},
		{Timestamp: baseTime.Add(time.Minute), Value: 100.0},
		{Timestamp: baseTime.Add(2 * time.Minute), Value: 100.0},
		{Timestamp: baseTime.Add(3 * time.Minute), Value: 100.0},
	}
	lower := []domain.IndicatorValue{
		{Timestamp: baseTime, Value: 90.0},
		{Timestamp: baseTime.Add(time.Minute), Value: 90.0},
		{Timestamp: baseTime.Add(2 * time.Minute), Value: 90.0},
		{Timestamp: baseTime.Add(3 * time.Minute), Value: 90.0},
	}

	result := calc.GetBBPosition(candles, upper, middle, lower)

	require.Len(t, result, 4)

	// Calculate expected positions
	// Position = (price - lower) / (upper - lower)
	// Band range = 110 - 90 = 20

	expectedPos1 := (95.0 - 90.0) / 20.0  // 0.25
	expectedPos2 := (100.0 - 90.0) / 20.0 // 0.5
	expectedPos3 := (105.0 - 90.0) / 20.0 // 0.75
	expectedPos4 := (102.5 - 90.0) / 20.0 // 0.625

	assert.InDelta(t, expectedPos1, result[0].Value, 1e-10)
	assert.InDelta(t, expectedPos2, result[1].Value, 1e-10)
	assert.InDelta(t, expectedPos3, result[2].Value, 1e-10)
	assert.InDelta(t, expectedPos4, result[3].Value, 1e-10)
}

func TestBollingerCalculator_GetBBPosition_EmptyInputs(t *testing.T) {
	calc := NewBollingerCalculator()
	result := calc.GetBBPosition([]domain.Candle{}, []domain.IndicatorValue{}, []domain.IndicatorValue{}, []domain.IndicatorValue{})

	assert.Empty(t, result)
}

func TestBollingerCalculator_GetBBPosition_WithNaN(t *testing.T) {
	calc := NewBollingerCalculator()
	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	candles := []domain.Candle{
		{Timestamp: baseTime, Close: 100.0},
		{Timestamp: baseTime.Add(time.Minute), Close: 102.0},
	}

	upper := []domain.IndicatorValue{
		{Timestamp: baseTime, Value: math.NaN()},
		{Timestamp: baseTime.Add(time.Minute), Value: 110.0},
	}
	lower := []domain.IndicatorValue{
		{Timestamp: baseTime, Value: math.NaN()},
		{Timestamp: baseTime.Add(time.Minute), Value: 90.0},
	}

	result := calc.GetBBPosition(candles, upper, []domain.IndicatorValue{}, lower)

	require.Len(t, result, 2)
	assert.True(t, math.IsNaN(result[0].Value))
	assert.False(t, math.IsNaN(result[1].Value))
}

func TestBollingerCalculator_GetValidBBValues(t *testing.T) {
	calc := NewBollingerCalculator()

	// Nil input
	result := calc.GetValidBBValues(nil)
	assert.NotNil(t, result)
	assert.Empty(t, result.Upper)

	timestamp := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	// Input with mixed valid and NaN values
	input := &BollingerBandsResult{
		Upper: []domain.IndicatorValue{
			{Timestamp: timestamp, Value: math.NaN()},
			{Timestamp: timestamp.Add(time.Minute), Value: 110.0},
			{Timestamp: timestamp.Add(2 * time.Minute), Value: 112.0},
		},
		Middle: []domain.IndicatorValue{
			{Timestamp: timestamp, Value: math.NaN()},
			{Timestamp: timestamp.Add(time.Minute), Value: 105.0},
			{Timestamp: timestamp.Add(2 * time.Minute), Value: 107.0},
		},
		Lower: []domain.IndicatorValue{
			{Timestamp: timestamp, Value: math.NaN()},
			{Timestamp: timestamp.Add(time.Minute), Value: 100.0},
			{Timestamp: timestamp.Add(2 * time.Minute), Value: 102.0},
		},
		Width: []domain.IndicatorValue{
			{Timestamp: timestamp, Value: math.NaN()},
			{Timestamp: timestamp.Add(time.Minute), Value: 0.1},
			{Timestamp: timestamp.Add(2 * time.Minute), Value: 0.093},
		},
	}

	result = calc.GetValidBBValues(input)

	require.NotNil(t, result)
	require.Len(t, result.Upper, 2)
	require.Len(t, result.Middle, 2)
	require.Len(t, result.Lower, 2)
	require.Len(t, result.Width, 2)

	// Should only contain the valid values (indices 1 and 2)
	assert.Equal(t, 110.0, result.Upper[0].Value)
	assert.Equal(t, 112.0, result.Upper[1].Value)
	assert.Equal(t, 105.0, result.Middle[0].Value)
	assert.Equal(t, 107.0, result.Middle[1].Value)
	assert.Equal(t, 100.0, result.Lower[0].Value)
	assert.Equal(t, 102.0, result.Lower[1].Value)
}

// Benchmark tests for performance validation
func BenchmarkBollingerCalculator_CalculateBollingerBands_1000Candles(b *testing.B) {
	calc := NewBollingerCalculator()
	candles := make([]domain.Candle, 1000)
	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	for i := 0; i < 1000; i++ {
		candles[i] = domain.Candle{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Close:         100.0 + float64(i)*0.1,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.CalculateBollingerBands(candles, 20, 2.0)
	}
}

func BenchmarkBollingerCalculator_CalculateBBWidth_1000Values(b *testing.B) {
	calc := NewBollingerCalculator()
	timestamp := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	upper := make([]domain.IndicatorValue, 1000)
	middle := make([]domain.IndicatorValue, 1000)
	lower := make([]domain.IndicatorValue, 1000)

	for i := 0; i < 1000; i++ {
		ts := timestamp.Add(time.Duration(i) * time.Minute)
		upper[i] = domain.IndicatorValue{Timestamp: ts, Value: 110.0 + float64(i)*0.1}
		middle[i] = domain.IndicatorValue{Timestamp: ts, Value: 100.0 + float64(i)*0.1}
		lower[i] = domain.IndicatorValue{Timestamp: ts, Value: 90.0 + float64(i)*0.1}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.CalculateBBWidth(upper, middle, lower)
	}
}

func BenchmarkBollingerCalculator_GetBBPosition_1000Values(b *testing.B) {
	calc := NewBollingerCalculator()
	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	candles := make([]domain.Candle, 1000)
	upper := make([]domain.IndicatorValue, 1000)
	lower := make([]domain.IndicatorValue, 1000)

	for i := 0; i < 1000; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Minute)
		candles[i] = domain.Candle{Timestamp: ts, Close: 100.0 + float64(i)*0.1}
		upper[i] = domain.IndicatorValue{Timestamp: ts, Value: 110.0 + float64(i)*0.1}
		lower[i] = domain.IndicatorValue{Timestamp: ts, Value: 90.0 + float64(i)*0.1}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.GetBBPosition(candles, upper, []domain.IndicatorValue{}, lower)
	}
}
