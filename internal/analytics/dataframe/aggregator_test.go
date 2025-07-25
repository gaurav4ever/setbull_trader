package dataframe

import (
	"testing"
	"time"

	"setbull_trader/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAggregator_DefaultConfig(t *testing.T) {
	aggregator := NewAggregator(nil)

	assert.NotNil(t, aggregator)
	assert.NotNil(t, aggregator.config)
	assert.Equal(t, "5m", aggregator.config.DefaultTimeframe)
	assert.Equal(t, 10000, aggregator.config.MaxCandles)
	assert.Equal(t, 30*time.Second, aggregator.config.Timeout)
}

func TestNewAggregator_CustomConfig(t *testing.T) {
	config := &AggregatorConfig{
		DefaultTimeframe: "1m",
		MaxCandles:       5000,
		Timeout:          60 * time.Second,
	}

	aggregator := NewAggregator(config)

	assert.NotNil(t, aggregator)
	assert.Equal(t, config, aggregator.config)
	assert.Equal(t, "1m", aggregator.config.DefaultTimeframe)
	assert.Equal(t, 5000, aggregator.config.MaxCandles)
	assert.Equal(t, 60*time.Second, aggregator.config.Timeout)
}

func TestAggregate5MinCandles_EmptyInput(t *testing.T) {
	aggregator := NewAggregator(nil)
	result, err := aggregator.Aggregate5MinCandles([]domain.Candle{})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestAggregate5MinCandles_SingleCandle(t *testing.T) {
	aggregator := NewAggregator(nil)
	timestamp := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	candles := []domain.Candle{
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp,
			Open:          100.0,
			High:          105.0,
			Low:           98.0,
			Close:         102.0,
			Volume:        1000,
		},
	}

	result, err := aggregator.Aggregate5MinCandles(candles)

	require.NoError(t, err)
	require.Len(t, result, 1)

	aggregated := result[0]
	assert.Equal(t, "NSE_EQ|INE002A01018", aggregated.InstrumentKey)
	assert.Equal(t, timestamp, aggregated.Timestamp)
	assert.Equal(t, 100.0, aggregated.Open)
	assert.Equal(t, 105.0, aggregated.High)
	assert.Equal(t, 98.0, aggregated.Low)
	assert.Equal(t, 102.0, aggregated.Close)
	assert.Equal(t, int64(1000), aggregated.Volume)
	assert.Equal(t, "5m", aggregated.TimeInterval)
}

func TestAggregate5MinCandles_MultipleCandles_SameInterval(t *testing.T) {
	aggregator := NewAggregator(nil)
	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	// 3 candles within the same 5-minute interval (9:15-9:20)
	candles := []domain.Candle{
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime,
			Open:          100.0,
			High:          105.0,
			Low:           98.0,
			Close:         102.0,
			Volume:        1000,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime.Add(1 * time.Minute),
			Open:          102.0,
			High:          107.0,
			Low:           101.0,
			Close:         106.0,
			Volume:        1500,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime.Add(2 * time.Minute),
			Open:          106.0,
			High:          108.0,
			Low:           104.0,
			Close:         105.0,
			Volume:        1200,
		},
	}

	result, err := aggregator.Aggregate5MinCandles(candles)

	require.NoError(t, err)
	require.Len(t, result, 1)

	aggregated := result[0]
	assert.Equal(t, "NSE_EQ|INE002A01018", aggregated.InstrumentKey)
	assert.Equal(t, baseTime, aggregated.Timestamp) // Should be aligned to 9:15
	assert.Equal(t, 100.0, aggregated.Open)         // First candle's open
	assert.Equal(t, 108.0, aggregated.High)         // Highest high
	assert.Equal(t, 98.0, aggregated.Low)           // Lowest low
	assert.Equal(t, 105.0, aggregated.Close)        // Last candle's close
	assert.Equal(t, int64(3700), aggregated.Volume) // Sum of volumes (1000+1500+1200)
	assert.Equal(t, "5m", aggregated.TimeInterval)
}

func TestAggregate5MinCandles_MultipleCandles_DifferentIntervals(t *testing.T) {
	aggregator := NewAggregator(nil)
	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	// Candles spanning two 5-minute intervals (9:15-9:20 and 9:20-9:25)
	candles := []domain.Candle{
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime,
			Open:          100.0,
			High:          105.0,
			Low:           98.0,
			Close:         102.0,
			Volume:        1000,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime.Add(3 * time.Minute),
			Open:          102.0,
			High:          107.0,
			Low:           101.0,
			Close:         106.0,
			Volume:        1500,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime.Add(5 * time.Minute), // Next interval: 9:20
			Open:          106.0,
			High:          108.0,
			Low:           104.0,
			Close:         105.0,
			Volume:        1200,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime.Add(7 * time.Minute),
			Open:          105.0,
			High:          110.0,
			Low:           103.0,
			Close:         109.0,
			Volume:        1800,
		},
	}

	result, err := aggregator.Aggregate5MinCandles(candles)

	require.NoError(t, err)
	require.Len(t, result, 2)

	// First interval (9:15-9:20)
	first := result[0]
	assert.Equal(t, baseTime, first.Timestamp)
	assert.Equal(t, 100.0, first.Open)
	assert.Equal(t, 107.0, first.High)
	assert.Equal(t, 98.0, first.Low)
	assert.Equal(t, 106.0, first.Close)
	assert.Equal(t, int64(2500), first.Volume) // 1000 + 1500

	// Second interval (9:20-9:25)
	second := result[1]
	assert.Equal(t, baseTime.Add(5*time.Minute), second.Timestamp)
	assert.Equal(t, 106.0, second.Open)
	assert.Equal(t, 110.0, second.High)
	assert.Equal(t, 103.0, second.Low)
	assert.Equal(t, 109.0, second.Close)
	assert.Equal(t, int64(3000), second.Volume) // 1200 + 1800
}

func TestAlignToInterval_1Minute(t *testing.T) {
	testCases := []struct {
		input    time.Time
		expected time.Time
	}{
		{
			input:    time.Date(2024, 1, 1, 9, 15, 30, 500000000, time.UTC),
			expected: time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC),
		},
		{
			input:    time.Date(2024, 1, 1, 9, 16, 45, 123456789, time.UTC),
			expected: time.Date(2024, 1, 1, 9, 16, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		result := alignToInterval(tc.input, time.Minute)
		assert.Equal(t, tc.expected, result)
	}
}

func TestAlignToInterval_5Minutes(t *testing.T) {
	testCases := []struct {
		input    time.Time
		expected time.Time
	}{
		{
			input:    time.Date(2024, 1, 1, 9, 17, 30, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC),
		},
		{
			input:    time.Date(2024, 1, 1, 9, 23, 45, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 9, 20, 0, 0, time.UTC),
		},
		{
			input:    time.Date(2024, 1, 1, 9, 20, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 9, 20, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		result := alignToInterval(tc.input, 5*time.Minute)
		assert.Equal(t, tc.expected, result)
	}
}

func TestAlignToInterval_15Minutes(t *testing.T) {
	testCases := []struct {
		input    time.Time
		expected time.Time
	}{
		{
			input:    time.Date(2024, 1, 1, 9, 17, 30, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC),
		},
		{
			input:    time.Date(2024, 1, 1, 9, 32, 45, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		result := alignToInterval(tc.input, 15*time.Minute)
		assert.Equal(t, tc.expected, result)
	}
}

func TestAlignToInterval_1Hour(t *testing.T) {
	testCases := []struct {
		input    time.Time
		expected time.Time
	}{
		{
			input:    time.Date(2024, 1, 1, 9, 45, 30, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		},
		{
			input:    time.Date(2024, 1, 1, 14, 15, 45, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		result := alignToInterval(tc.input, time.Hour)
		assert.Equal(t, tc.expected, result)
	}
}

func TestFormatInterval(t *testing.T) {
	testCases := []struct {
		interval time.Duration
		expected string
	}{
		{time.Minute, "1m"},
		{5 * time.Minute, "5m"},
		{15 * time.Minute, "15m"},
		{time.Hour, "1h"},
		{2 * time.Hour, "2h0m0s"},
	}

	for _, tc := range testCases {
		result := formatInterval(tc.interval)
		assert.Equal(t, tc.expected, result)
	}
}

func TestValidateTimeframe(t *testing.T) {
	aggregator := NewAggregator(nil)

	validTimeframes := []string{"1m", "3m", "5m", "15m", "30m", "1h", "4h", "1d"}
	for _, tf := range validTimeframes {
		err := aggregator.ValidateTimeframe(tf)
		assert.NoError(t, err, "Expected %s to be valid", tf)
	}

	invalidTimeframes := []string{"2m", "10m", "2h", "6h", "1w", ""}
	for _, tf := range invalidTimeframes {
		err := aggregator.ValidateTimeframe(tf)
		assert.Error(t, err, "Expected %s to be invalid", tf)
	}
}

func TestParseTimeframe(t *testing.T) {
	aggregator := NewAggregator(nil)

	testCases := []struct {
		timeframe string
		expected  time.Duration
		shouldErr bool
	}{
		{"1m", time.Minute, false},
		{"3m", 3 * time.Minute, false},
		{"5m", 5 * time.Minute, false},
		{"15m", 15 * time.Minute, false},
		{"30m", 30 * time.Minute, false},
		{"1h", time.Hour, false},
		{"4h", 4 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"2m", 0, true},
		{"invalid", 0, true},
	}

	for _, tc := range testCases {
		result, err := aggregator.ParseTimeframe(tc.timeframe)
		if tc.shouldErr {
			assert.Error(t, err, "Expected error for timeframe %s", tc.timeframe)
		} else {
			assert.NoError(t, err, "Expected no error for timeframe %s", tc.timeframe)
			assert.Equal(t, tc.expected, result)
		}
	}
}

func TestAggregateWithIndicators(t *testing.T) {
	aggregator := NewAggregator(nil)
	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	candles := []domain.Candle{
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime,
			Open:          100.0,
			High:          105.0,
			Low:           98.0,
			Close:         102.0,
			Volume:        1000,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime.Add(5 * time.Minute),
			Open:          102.0,
			High:          107.0,
			Low:           101.0,
			Close:         106.0,
			Volume:        1500,
		},
	}

	result, err := aggregator.AggregateWithIndicators(candles, 5*time.Minute)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Empty())
	assert.Equal(t, 2, result.DataFrame().Nrow())
}

func TestAggregateGroup(t *testing.T) {
	aggregator := NewAggregator(nil)
	intervalStart := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	candles := []domain.Candle{
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     intervalStart,
			Open:          100.0,
			High:          105.0,
			Low:           98.0,
			Close:         102.0,
			Volume:        1000,
			OpenInterest:  50,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     intervalStart.Add(1 * time.Minute),
			Open:          102.0,
			High:          107.0,
			Low:           96.0, // Lowest low
			Close:         106.0,
			Volume:        1500,
			OpenInterest:  75,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     intervalStart.Add(2 * time.Minute),
			Open:          106.0,
			High:          110.0, // Highest high
			Low:           104.0,
			Close:         108.0, // Last close
			Volume:        1200,
			OpenInterest:  60,
		},
	}

	result := aggregator.aggregateGroup(candles, intervalStart, 5*time.Minute)

	assert.Equal(t, "NSE_EQ|INE002A01018", result.InstrumentKey)
	assert.Equal(t, intervalStart, result.Timestamp)
	assert.Equal(t, 100.0, result.Open)              // First open
	assert.Equal(t, 110.0, result.High)              // Highest high
	assert.Equal(t, 96.0, result.Low)                // Lowest low
	assert.Equal(t, 108.0, result.Close)             // Last close
	assert.Equal(t, int64(3700), result.Volume)      // Sum of volumes
	assert.Equal(t, int64(185), result.OpenInterest) // Sum of open interest
	assert.Equal(t, "5m", result.TimeInterval)
}

func TestAggregateGroup_EmptyCandles(t *testing.T) {
	aggregator := NewAggregator(nil)
	intervalStart := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	result := aggregator.aggregateGroup([]domain.Candle{}, intervalStart, 5*time.Minute)

	assert.Equal(t, domain.Candle{}, result)
}

// Benchmark tests for performance validation
func BenchmarkAggregate5MinCandles_1000Candles(b *testing.B) {
	aggregator := NewAggregator(nil)
	candles := make([]domain.Candle, 1000)
	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	for i := 0; i < 1000; i++ {
		candles[i] = domain.Candle{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Open:          100.0 + float64(i)*0.1,
			High:          105.0 + float64(i)*0.1,
			Low:           98.0 + float64(i)*0.1,
			Close:         102.0 + float64(i)*0.1,
			Volume:        1000 + int64(i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := aggregator.Aggregate5MinCandles(candles)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAlignToInterval_5Minutes(b *testing.B) {
	timestamp := time.Date(2024, 1, 1, 9, 17, 30, 123456789, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alignToInterval(timestamp, 5*time.Minute)
	}
}

func BenchmarkAggregateGroup_10Candles(b *testing.B) {
	aggregator := NewAggregator(nil)
	intervalStart := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)

	candles := make([]domain.Candle, 10)
	for i := 0; i < 10; i++ {
		candles[i] = domain.Candle{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     intervalStart.Add(time.Duration(i) * time.Minute),
			Open:          100.0 + float64(i),
			High:          105.0 + float64(i),
			Low:           98.0 + float64(i),
			Close:         102.0 + float64(i),
			Volume:        1000 + int64(i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		aggregator.aggregateGroup(candles, intervalStart, 5*time.Minute)
	}
}
