package concurrency

import (
	"context"
	"testing"
	"time"

	"setbull_trader/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create test candles
func createTestCandles(count int) []domain.Candle {
	candles := make([]domain.Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * time.Minute)

	for i := 0; i < count; i++ {
		candles[i] = domain.Candle{
			InstrumentKey: "TEST_STOCK",
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Open:          100.0 + float64(i)*0.5,
			High:          105.0 + float64(i)*0.5,
			Low:           95.0 + float64(i)*0.5,
			Close:         102.0 + float64(i)*0.5,
			Volume:        1000 + int64(i)*10,
		}
	}

	return candles
}

func TestIndicatorTask_EMA(t *testing.T) {
	candles := createTestCandles(50)

	task := NewIndicatorTask(
		"ema-test",
		"TEST_STOCK",
		"EMA",
		candles,
		map[string]interface{}{"period": 9},
		1,
	)

	assert.Equal(t, "ema-test", task.ID())
	assert.Equal(t, 1, task.Priority())

	ctx := context.Background()
	result, err := task.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	emaValues, ok := result.([]domain.IndicatorValue)
	require.True(t, ok)
	assert.Len(t, emaValues, len(candles))

	// Verify that EMA values are calculated (skip initial values that might be 0)
	nonZeroCount := 0
	for i, value := range emaValues {
		assert.Equal(t, candles[i].Timestamp, value.Timestamp)
		if value.Value > 0 {
			nonZeroCount++
		}
	}

	// Should have meaningful EMA values for most of the data
	assert.True(t, nonZeroCount > len(candles)/2, "Expected more than half the EMA values to be non-zero")
}

func TestIndicatorTask_RSI(t *testing.T) {
	candles := createTestCandles(30)

	task := NewIndicatorTask(
		"rsi-test",
		"TEST_STOCK",
		"RSI",
		candles,
		map[string]interface{}{"period": 14},
		1,
	)

	ctx := context.Background()
	result, err := task.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	rsiValues, ok := result.([]domain.IndicatorValue)
	require.True(t, ok)
	assert.Len(t, rsiValues, len(candles))

	// RSI should be between 0 and 100
	for _, value := range rsiValues {
		if value.Value > 0 { // Skip initial zero values
			assert.True(t, value.Value >= 0 && value.Value <= 100,
				"RSI value %f should be between 0 and 100", value.Value)
		}
	}
}

func TestIndicatorTask_SMA(t *testing.T) {
	candles := createTestCandles(20)

	task := NewIndicatorTask(
		"sma-test",
		"TEST_STOCK",
		"SMA",
		candles,
		map[string]interface{}{"period": 5},
		1,
	)

	ctx := context.Background()
	result, err := task.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	smaValues, ok := result.([]domain.IndicatorValue)
	require.True(t, ok)
	assert.Len(t, smaValues, len(candles))

	// Verify SMA calculation for the 5th candle
	expectedSMA := (candles[0].Close + candles[1].Close + candles[2].Close + candles[3].Close + candles[4].Close) / 5
	assert.InDelta(t, expectedSMA, smaValues[4].Value, 0.001)
}

func TestIndicatorTask_VWAP(t *testing.T) {
	candles := createTestCandles(15)

	task := NewIndicatorTask(
		"vwap-test",
		"TEST_STOCK",
		"VWAP",
		candles,
		map[string]interface{}{},
		1,
	)

	ctx := context.Background()
	result, err := task.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	vwapValues, ok := result.([]domain.IndicatorValue)
	require.True(t, ok)
	assert.Len(t, vwapValues, len(candles))

	// VWAP values should be reasonable
	for _, value := range vwapValues {
		assert.True(t, value.Value > 0)
	}
}

func TestIndicatorTask_BollingerBands(t *testing.T) {
	candles := createTestCandles(25)

	task := NewIndicatorTask(
		"bb-test",
		"TEST_STOCK",
		"BOLLINGER",
		candles,
		map[string]interface{}{
			"period": 20,
			"stddev": 2.0,
		},
		1,
	)

	ctx := context.Background()
	result, err := task.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Bollinger result should be some interface{}
	assert.NotNil(t, result)
}

func TestIndicatorTask_InvalidIndicatorType(t *testing.T) {
	candles := createTestCandles(10)

	task := NewIndicatorTask(
		"invalid-test",
		"TEST_STOCK",
		"INVALID_TYPE",
		candles,
		map[string]interface{}{},
		1,
	)

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported indicator type")
}

func TestIndicatorTask_MissingParameters(t *testing.T) {
	candles := createTestCandles(10)

	task := NewIndicatorTask(
		"missing-param-test",
		"TEST_STOCK",
		"EMA",
		candles,
		map[string]interface{}{}, // Missing period parameter
		1,
	)

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "missing or invalid period parameter")
}

func TestIndicatorTask_EmptyCandles(t *testing.T) {
	task := NewIndicatorTask(
		"empty-candles-test",
		"TEST_STOCK",
		"EMA",
		[]domain.Candle{}, // Empty candles
		map[string]interface{}{"period": 9},
		1,
	)

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no candles provided")
}

func TestBatchIndicatorTask(t *testing.T) {
	candles := createTestCandles(30)

	indicators := []IndicatorRequest{
		{Type: "EMA", Parameters: map[string]interface{}{"period": 9}},
		{Type: "RSI", Parameters: map[string]interface{}{"period": 14}},
		{Type: "SMA", Parameters: map[string]interface{}{"period": 5}},
	}

	task := NewBatchIndicatorTask(
		"batch-test",
		"TEST_STOCK",
		candles,
		indicators,
		1,
	)

	assert.Equal(t, "batch-test", task.ID())
	assert.Equal(t, 1, task.Priority())

	ctx := context.Background()
	result, err := task.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	batchResult, ok := result.(*BatchIndicatorResult)
	require.True(t, ok)

	assert.Equal(t, "TEST_STOCK", batchResult.InstrumentKey)
	assert.Len(t, batchResult.Results, 3) // Should have 3 results
	assert.Len(t, batchResult.Timing, 3)  // Should have timing for 3 indicators

	// Verify each indicator was calculated
	for indicatorID, data := range batchResult.Results {
		assert.NotNil(t, data)

		if values, ok := data.([]domain.IndicatorValue); ok {
			assert.Len(t, values, len(candles))
		}

		// Check timing
		timing, exists := batchResult.Timing[indicatorID]
		assert.True(t, exists)
		assert.True(t, timing > 0)
	}
}

func TestBatchIndicatorTask_WithErrors(t *testing.T) {
	candles := createTestCandles(30)

	indicators := []IndicatorRequest{
		{Type: "EMA", Parameters: map[string]interface{}{"period": 9}},
		{Type: "INVALID", Parameters: map[string]interface{}{}}, // This will error
		{Type: "RSI", Parameters: map[string]interface{}{"period": 14}},
	}

	task := NewBatchIndicatorTask(
		"batch-error-test",
		"TEST_STOCK",
		candles,
		indicators,
		1,
	)

	ctx := context.Background()
	result, err := task.Execute(ctx)

	require.NoError(t, err) // The task itself should not error
	require.NotNil(t, result)

	batchResult, ok := result.(*BatchIndicatorResult)
	require.True(t, ok)

	// Should have 2 successful results and 1 error
	assert.Len(t, batchResult.Results, 2)
	assert.Len(t, batchResult.Errors, 1)
	assert.Len(t, batchResult.Timing, 3) // Timing for all attempts
}

func TestPriorityTask(t *testing.T) {
	mockTask := NewMockTask("mock-1", 1, 10*time.Millisecond, false)
	priorityTask := NewPriorityTask(mockTask, 5)

	assert.Equal(t, "mock-1", priorityTask.ID())
	assert.Equal(t, 5, priorityTask.Priority())

	ctx := context.Background()
	result, err := priorityTask.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the underlying task was executed
	assert.Equal(t, int32(1), mockTask.GetExecuteCount())
}

// Benchmark tests for indicator tasks
func BenchmarkIndicatorTask_EMA(b *testing.B) {
	candles := createTestCandles(100)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := NewIndicatorTask(
			"ema-bench",
			"TEST_STOCK",
			"EMA",
			candles,
			map[string]interface{}{"period": 9},
			1,
		)

		_, err := task.Execute(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBatchIndicatorTask(b *testing.B) {
	candles := createTestCandles(100)
	indicators := []IndicatorRequest{
		{Type: "EMA", Parameters: map[string]interface{}{"period": 9}},
		{Type: "RSI", Parameters: map[string]interface{}{"period": 14}},
		{Type: "SMA", Parameters: map[string]interface{}{"period": 20}},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := NewBatchIndicatorTask(
			"batch-bench",
			"TEST_STOCK",
			candles,
			indicators,
			1,
		)

		_, err := task.Execute(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
