package dataframe

import (
	"testing"
	"time"

	"setbull_trader/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCandleDataFrame_EmptyCandles(t *testing.T) {
	// Test with empty candles slice
	candles := []domain.Candle{}
	df := NewCandleDataFrame(candles, false)

	assert.NotNil(t, df)
	assert.True(t, df.Empty())
	assert.Equal(t, 0, df.DataFrame().Nrow())
}

func TestNewCandleDataFrame_SingleCandle(t *testing.T) {
	// Create a single test candle
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

	df := NewCandleDataFrame(candles, false)

	assert.NotNil(t, df)
	assert.False(t, df.Empty())
	assert.Equal(t, 1, df.DataFrame().Nrow())
	assert.Equal(t, 7, df.DataFrame().Ncol()) // Symbol, Timestamp, Open, High, Low, Close, Volume
}

func TestNewCandleDataFrame_MultipleCandles(t *testing.T) {
	// Create multiple test candles
	timestamp1 := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)
	timestamp2 := time.Date(2024, 1, 1, 9, 16, 0, 0, time.UTC)
	timestamp3 := time.Date(2024, 1, 1, 9, 17, 0, 0, time.UTC)

	candles := []domain.Candle{
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp1,
			Open:          100.0,
			High:          105.0,
			Low:           98.0,
			Close:         102.0,
			Volume:        1000,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp2,
			Open:          102.0,
			High:          107.0,
			Low:           101.0,
			Close:         106.0,
			Volume:        1500,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp3,
			Open:          106.0,
			High:          108.0,
			Low:           104.0,
			Close:         105.0,
			Volume:        1200,
		},
	}

	df := NewCandleDataFrame(candles, false)

	assert.NotNil(t, df)
	assert.False(t, df.Empty())
	assert.Equal(t, 3, df.DataFrame().Nrow())
	assert.Equal(t, 7, df.DataFrame().Ncol())
}

func TestCandleDataFrame_GetTimestamps(t *testing.T) {
	timestamp1 := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)
	timestamp2 := time.Date(2024, 1, 1, 9, 16, 0, 0, time.UTC)

	candles := []domain.Candle{
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp1,
			Open:          100.0,
			High:          105.0,
			Low:           98.0,
			Close:         102.0,
			Volume:        1000,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp2,
			Open:          102.0,
			High:          107.0,
			Low:           101.0,
			Close:         106.0,
			Volume:        1500,
		},
	}

	df := NewCandleDataFrame(candles, false)
	timestamps := df.GetTimestamps()

	require.Len(t, timestamps, 2)
	assert.Equal(t, timestamp1, timestamps[0])
	assert.Equal(t, timestamp2, timestamps[1])
}

func TestCandleDataFrame_GetTimestamps_Empty(t *testing.T) {
	df := NewCandleDataFrame([]domain.Candle{})
	timestamps := df.GetTimestamps()

	assert.Empty(t, timestamps)
}

func TestCandleDataFrame_GetSymbols(t *testing.T) {
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
		{
			InstrumentKey: "NSE_EQ|INE467B01029",
			Timestamp:     timestamp,
			Open:          200.0,
			High:          205.0,
			Low:           198.0,
			Close:         202.0,
			Volume:        2000,
		},
	}

	df := NewCandleDataFrame(candles, false)
	symbols := df.GetSymbols()

	require.Len(t, symbols, 2)
	assert.Equal(t, "NSE_EQ|INE002A01018", symbols[0])
	assert.Equal(t, "NSE_EQ|INE467B01029", symbols[1])
}

func TestCandleDataFrame_GetOHLCV(t *testing.T) {
	timestamp := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)
	candles := []domain.Candle{
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp,
			Open:          100.5,
			High:          105.7,
			Low:           98.2,
			Close:         102.3,
			Volume:        1000,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp,
			Open:          102.3,
			High:          107.8,
			Low:           101.1,
			Close:         106.9,
			Volume:        1500,
		},
	}

	df := NewCandleDataFrame(candles, false)

	// Test Opens
	opens := df.GetOpens()
	require.Len(t, opens, 2)
	assert.Equal(t, 100.5, opens[0])
	assert.Equal(t, 102.3, opens[1])

	// Test Highs
	highs := df.GetHighs()
	require.Len(t, highs, 2)
	assert.Equal(t, 105.7, highs[0])
	assert.Equal(t, 107.8, highs[1])

	// Test Lows
	lows := df.GetLows()
	require.Len(t, lows, 2)
	assert.Equal(t, 98.2, lows[0])
	assert.Equal(t, 101.1, lows[1])

	// Test Closes
	closes := df.GetCloses()
	require.Len(t, closes, 2)
	assert.Equal(t, 102.3, closes[0])
	assert.Equal(t, 106.9, closes[1])

	// Test Volumes
	volumes := df.GetVolumes()
	require.Len(t, volumes, 2)
	assert.Equal(t, 1000.0, volumes[0])
	assert.Equal(t, 1500.0, volumes[1])
}

func TestCandleDataFrame_GetOHLCV_Empty(t *testing.T) {
	df := NewCandleDataFrame([]domain.Candle{})

	assert.Empty(t, df.GetOpens())
	assert.Empty(t, df.GetHighs())
	assert.Empty(t, df.GetLows())
	assert.Empty(t, df.GetCloses())
	assert.Empty(t, df.GetVolumes())
}

func TestCandleDataFrame_ToCandles(t *testing.T) {
	timestamp1 := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)
	timestamp2 := time.Date(2024, 1, 1, 9, 16, 0, 0, time.UTC)

	originalCandles := []domain.Candle{
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp1,
			Open:          100.5,
			High:          105.7,
			Low:           98.2,
			Close:         102.3,
			Volume:        1000,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp2,
			Open:          102.3,
			High:          107.8,
			Low:           101.1,
			Close:         106.9,
			Volume:        1500,
		},
	}

	df := NewCandleDataFrame(originalCandles)
	convertedCandles := df.ToCandles()

	require.Len(t, convertedCandles, 2)

	// Test first candle
	assert.Equal(t, originalCandles[0].InstrumentKey, convertedCandles[0].InstrumentKey)
	assert.Equal(t, originalCandles[0].Timestamp, convertedCandles[0].Timestamp)
	assert.Equal(t, originalCandles[0].Open, convertedCandles[0].Open)
	assert.Equal(t, originalCandles[0].High, convertedCandles[0].High)
	assert.Equal(t, originalCandles[0].Low, convertedCandles[0].Low)
	assert.Equal(t, originalCandles[0].Close, convertedCandles[0].Close)
	assert.Equal(t, originalCandles[0].Volume, convertedCandles[0].Volume)

	// Test second candle
	assert.Equal(t, originalCandles[1].InstrumentKey, convertedCandles[1].InstrumentKey)
	assert.Equal(t, originalCandles[1].Timestamp, convertedCandles[1].Timestamp)
	assert.Equal(t, originalCandles[1].Open, convertedCandles[1].Open)
	assert.Equal(t, originalCandles[1].High, convertedCandles[1].High)
	assert.Equal(t, originalCandles[1].Low, convertedCandles[1].Low)
	assert.Equal(t, originalCandles[1].Close, convertedCandles[1].Close)
	assert.Equal(t, originalCandles[1].Volume, convertedCandles[1].Volume)
}

func TestCandleDataFrame_ToCandles_Empty(t *testing.T) {
	df := NewCandleDataFrame([]domain.Candle{})
	candles := df.ToCandles()

	assert.Empty(t, candles)
}

func TestCandleDataFrame_ToAggregatedCandles(t *testing.T) {
	timestamp1 := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)
	timestamp2 := time.Date(2024, 1, 1, 9, 16, 0, 0, time.UTC)

	originalCandles := []domain.Candle{
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp1,
			Open:          100.5,
			High:          105.7,
			Low:           98.2,
			Close:         102.3,
			Volume:        1000,
		},
		{
			InstrumentKey: "NSE_EQ|INE002A01018",
			Timestamp:     timestamp2,
			Open:          102.3,
			High:          107.8,
			Low:           101.1,
			Close:         106.9,
			Volume:        1500,
		},
	}

	df := NewCandleDataFrame(originalCandles)
	aggregatedCandles := df.ToAggregatedCandles()

	require.Len(t, aggregatedCandles, 2)

	// Test first aggregated candle
	assert.Equal(t, "NSE_EQ|INE002A01018", aggregatedCandles[0].InstrumentKey)
	assert.Equal(t, timestamp1, aggregatedCandles[0].Timestamp)
	assert.Equal(t, 100.5, aggregatedCandles[0].Open)
	assert.Equal(t, 105.7, aggregatedCandles[0].High)
	assert.Equal(t, 98.2, aggregatedCandles[0].Low)
	assert.Equal(t, 102.3, aggregatedCandles[0].Close)
	assert.Equal(t, int64(1000), aggregatedCandles[0].Volume)
	assert.Equal(t, int64(0), aggregatedCandles[0].OpenInterest)
	assert.Equal(t, "", aggregatedCandles[0].TimeInterval)

	// Test second aggregated candle
	assert.Equal(t, "NSE_EQ|INE002A01018", aggregatedCandles[1].InstrumentKey)
	assert.Equal(t, timestamp2, aggregatedCandles[1].Timestamp)
	assert.Equal(t, 102.3, aggregatedCandles[1].Open)
	assert.Equal(t, 107.8, aggregatedCandles[1].High)
	assert.Equal(t, 101.1, aggregatedCandles[1].Low)
	assert.Equal(t, 106.9, aggregatedCandles[1].Close)
	assert.Equal(t, int64(1500), aggregatedCandles[1].Volume)
	assert.Equal(t, int64(0), aggregatedCandles[1].OpenInterest)
	assert.Equal(t, "", aggregatedCandles[1].TimeInterval)
}

func TestCandleDataFrame_ToAggregatedCandles_Empty(t *testing.T) {
	df := NewCandleDataFrame([]domain.Candle{})
	aggregatedCandles := df.ToAggregatedCandles()

	assert.Empty(t, aggregatedCandles)
}

func TestCandleDataFrame_DataFrame(t *testing.T) {
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

	df := NewCandleDataFrame(candles, false)
	gotaDF := df.DataFrame()

	assert.NotNil(t, gotaDF)
	assert.Equal(t, 1, gotaDF.Nrow())
	assert.Equal(t, 7, gotaDF.Ncol())

	// Verify column names
	expectedColumns := []string{"Symbol", "Timestamp", "Open", "High", "Low", "Close", "Volume"}
	actualColumns := gotaDF.Names()
	assert.Equal(t, expectedColumns, actualColumns)
}

// Benchmark tests for performance validation
func BenchmarkNewCandleDataFrame_1000Candles(b *testing.B) {
	// Create 1000 test candles
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
		NewCandleDataFrame(candles, false)
	}
}

func BenchmarkCandleDataFrame_ToCandles_1000Rows(b *testing.B) {
	// Create 1000 test candles
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

	df := NewCandleDataFrame(candles, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		df.ToCandles()
	}
}

func BenchmarkCandleDataFrame_GetCloses_1000Rows(b *testing.B) {
	// Create 1000 test candles
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

	df := NewCandleDataFrame(candles, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		df.GetCloses()
	}
}
