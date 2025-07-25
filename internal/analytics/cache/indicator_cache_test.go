package cache

import (
	"fmt"
	"testing"
	"time"

	"setbull_trader/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndicatorCache_BasicOperations(t *testing.T) {
	cache := NewIndicatorCache(10) // 10MB cache

	// Create test key and indicators
	key := CacheKey{
		Symbol:      "TESTSTOCK",
		Timeframe:   "5m",
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
		Indicators:  []string{"EMA9", "RSI14"},
		DataVersion: "v1.0",
	}

	indicators := &domain.TechnicalIndicators{
		InstrumentKey: "TESTSTOCK",
		Interval:      "5m",
		StartTime:     key.StartTime,
		EndTime:       key.EndTime,
		EMA9: []domain.IndicatorValue{
			{Timestamp: time.Now(), Value: 100.0},
			{Timestamp: time.Now().Add(5 * time.Minute), Value: 101.0},
		},
		RSI14: []domain.IndicatorValue{
			{Timestamp: time.Now(), Value: 65.0},
			{Timestamp: time.Now().Add(5 * time.Minute), Value: 67.0},
		},
	}

	// Test cache miss
	result, found := cache.GetIndicators(key)
	assert.False(t, found)
	assert.Nil(t, result)

	// Test cache set
	err := cache.SetIndicators(key, indicators, 100*time.Millisecond, 5*time.Minute)
	assert.NoError(t, err)

	// Test cache hit
	result, found = cache.GetIndicators(key)
	assert.True(t, found)
	assert.NotNil(t, result)
	assert.Equal(t, "TESTSTOCK", result.InstrumentKey)
	assert.Equal(t, "5m", result.Interval)
	assert.Len(t, result.EMA9, 2)
	assert.Len(t, result.RSI14, 2)
}

func TestIndicatorCache_GetOrCalculate(t *testing.T) {
	cache := NewIndicatorCache(10)

	key := CacheKey{
		Symbol:      "TESTSTOCK",
		Timeframe:   "1m",
		StartTime:   time.Now().Add(-30 * time.Minute),
		EndTime:     time.Now(),
		Indicators:  []string{"EMA9"},
		DataVersion: "v1.0",
	}

	calculatorCalled := false
	calculator := func() (*domain.TechnicalIndicators, error) {
		calculatorCalled = true
		return &domain.TechnicalIndicators{
			InstrumentKey: "TESTSTOCK",
			Interval:      "1m",
			EMA9: []domain.IndicatorValue{
				{Timestamp: time.Now(), Value: 50.0},
			},
		}, nil
	}

	// First call should invoke calculator
	result, err := cache.GetOrCalculate(key, calculator, 5*time.Minute)
	assert.NoError(t, err)
	assert.True(t, calculatorCalled)
	assert.NotNil(t, result)
	assert.Equal(t, "TESTSTOCK", result.InstrumentKey)

	// Reset flag
	calculatorCalled = false

	// Second call should use cache
	result2, err := cache.GetOrCalculate(key, calculator, 5*time.Minute)
	assert.NoError(t, err)
	assert.False(t, calculatorCalled) // Should not be called again
	assert.NotNil(t, result2)
	assert.Equal(t, result.InstrumentKey, result2.InstrumentKey)
}

func TestIndicatorCache_TTLExpiration(t *testing.T) {
	cache := NewIndicatorCache(10)

	key := CacheKey{
		Symbol:      "TESTSTOCK",
		Timeframe:   "5m",
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
		Indicators:  []string{"EMA9"},
		DataVersion: "v1.0",
	}

	indicators := &domain.TechnicalIndicators{
		InstrumentKey: "TESTSTOCK",
		Interval:      "5m",
		EMA9: []domain.IndicatorValue{
			{Timestamp: time.Now(), Value: 100.0},
		},
	}

	// Set with very short TTL
	err := cache.SetIndicators(key, indicators, 50*time.Millisecond, 100*time.Millisecond)
	assert.NoError(t, err)

	// Should be available immediately
	result, found := cache.GetIndicators(key)
	assert.True(t, found)
	assert.NotNil(t, result)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired and not found
	result, found = cache.GetIndicators(key)
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestIndicatorCache_Metrics(t *testing.T) {
	cache := NewIndicatorCache(10)

	key := CacheKey{
		Symbol:      "TESTSTOCK",
		Timeframe:   "5m",
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
		Indicators:  []string{"EMA9"},
		DataVersion: "v1.0",
	}

	indicators := &domain.TechnicalIndicators{
		InstrumentKey: "TESTSTOCK",
		Interval:      "5m",
	}

	// Initial metrics
	metrics := cache.GetMetrics()
	assert.Equal(t, int64(0), metrics.CacheHits)
	assert.Equal(t, int64(0), metrics.CacheMisses)
	assert.Equal(t, int64(0), metrics.TotalRequests)

	// Cache miss
	_, found := cache.GetIndicators(key)
	assert.False(t, found)

	metrics = cache.GetMetrics()
	assert.Equal(t, int64(0), metrics.CacheHits)
	assert.Equal(t, int64(1), metrics.CacheMisses)
	assert.Equal(t, int64(1), metrics.TotalRequests)

	// Cache set and hit
	err := cache.SetIndicators(key, indicators, 100*time.Millisecond, 5*time.Minute)
	assert.NoError(t, err)

	_, found = cache.GetIndicators(key)
	assert.True(t, found)

	metrics = cache.GetMetrics()
	assert.Equal(t, int64(1), metrics.CacheHits)
	assert.Equal(t, int64(1), metrics.CacheMisses)
	assert.Equal(t, int64(2), metrics.TotalRequests)
}

func TestIndicatorCache_ClearAndReset(t *testing.T) {
	cache := NewIndicatorCache(10)

	key := CacheKey{
		Symbol:      "TESTSTOCK",
		Timeframe:   "5m",
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
		Indicators:  []string{"EMA9"},
		DataVersion: "v1.0",
	}

	indicators := &domain.TechnicalIndicators{
		InstrumentKey: "TESTSTOCK",
		Interval:      "5m",
	}

	// Set indicators
	err := cache.SetIndicators(key, indicators, 100*time.Millisecond, 5*time.Minute)
	assert.NoError(t, err)

	// Verify it's cached
	_, found := cache.GetIndicators(key)
	assert.True(t, found)

	// Clear cache
	cache.Clear()

	// Should not be found after clear
	_, found = cache.GetIndicators(key)
	assert.False(t, found)

	// Reset metrics
	cache.ResetMetrics()
	metrics := cache.GetMetrics()
	assert.Equal(t, int64(0), metrics.CacheHits)
	assert.Equal(t, int64(0), metrics.CacheMisses)
	assert.Equal(t, int64(0), metrics.TotalRequests)
}

func TestIndicatorCache_KeyGeneration(t *testing.T) {
	cache := NewIndicatorCache(10)

	key1 := CacheKey{
		Symbol:      "STOCK1",
		Timeframe:   "5m",
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
		Indicators:  []string{"EMA9"},
		DataVersion: "v1.0",
	}

	key2 := CacheKey{
		Symbol:      "STOCK2",
		Timeframe:   "5m",
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
		Indicators:  []string{"EMA9"},
		DataVersion: "v1.0",
	}

	// Generate keys
	keyStr1 := cache.generateKey(key1)
	keyStr2 := cache.generateKey(key2)

	// Keys should be different
	assert.NotEqual(t, keyStr1, keyStr2)

	// Same key should generate same string
	keyStr1Again := cache.generateKey(key1)
	assert.Equal(t, keyStr1, keyStr1Again)
}

func TestDataFramePool_BasicOperations(t *testing.T) {
	pool := NewDataFramePool()

	// Get DataFrame from pool
	df1 := pool.Get()
	assert.NotNil(t, df1)

	// Get another DataFrame
	df2 := pool.Get()
	assert.NotNil(t, df2)

	// Put back to pool
	pool.Put(df1)
	pool.Put(df2)

	// Should be able to get again
	df3 := pool.Get()
	assert.NotNil(t, df3)
}

func TestProcessingPool_Management(t *testing.T) {
	processingPool := NewProcessingPool()

	// Get DataFrame pool
	dfPool := processingPool.GetDataFramePool()
	assert.NotNil(t, dfPool)

	// Get pooled candle DataFrame
	candleDF := processingPool.GetPooledCandleDataFrame()
	assert.NotNil(t, candleDF)

	// Release it
	candleDF.Release()

	// Get stats
	stats := processingPool.Stats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "dataframe_pool_active")
	assert.Contains(t, stats, "pool_type")
}

// Benchmark tests
func BenchmarkIndicatorCache_GetIndicators(b *testing.B) {
	cache := NewIndicatorCache(100)

	key := CacheKey{
		Symbol:      "TESTSTOCK",
		Timeframe:   "5m",
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
		Indicators:  []string{"EMA9", "RSI14"},
		DataVersion: "v1.0",
	}

	indicators := &domain.TechnicalIndicators{
		InstrumentKey: "TESTSTOCK",
		Interval:      "5m",
		EMA9: []domain.IndicatorValue{
			{Timestamp: time.Now(), Value: 100.0},
		},
		RSI14: []domain.IndicatorValue{
			{Timestamp: time.Now(), Value: 65.0},
		},
	}

	// Pre-populate cache
	err := cache.SetIndicators(key, indicators, 100*time.Millisecond, 10*time.Minute)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, found := cache.GetIndicators(key)
		if !found {
			b.Fatal("Expected cache hit")
		}
	}
}

func BenchmarkIndicatorCache_SetIndicators(b *testing.B) {
	cache := NewIndicatorCache(100)

	indicators := &domain.TechnicalIndicators{
		InstrumentKey: "TESTSTOCK",
		Interval:      "5m",
		EMA9: []domain.IndicatorValue{
			{Timestamp: time.Now(), Value: 100.0},
		},
		RSI14: []domain.IndicatorValue{
			{Timestamp: time.Now(), Value: 65.0},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := CacheKey{
			Symbol:      fmt.Sprintf("STOCK%d", i),
			Timeframe:   "5m",
			StartTime:   time.Now().Add(-1 * time.Hour),
			EndTime:     time.Now(),
			Indicators:  []string{"EMA9", "RSI14"},
			DataVersion: "v1.0",
		}

		err := cache.SetIndicators(key, indicators, 100*time.Millisecond, 10*time.Minute)
		if err != nil {
			b.Fatal(err)
		}
	}
}
