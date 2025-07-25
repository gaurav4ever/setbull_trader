package cache

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDataFramePool_GetPut(t *testing.T) {
	pool := NewDataFramePool()

	// Get a DataFrame from the pool
	df1 := pool.Get()
	assert.NotNil(t, df1)

	// Get another DataFrame
	df2 := pool.Get()
	assert.NotNil(t, df2)

	// Put one back
	pool.Put(df1)

	// Get again - should potentially reuse the put-back DataFrame
	df3 := pool.Get()
	assert.NotNil(t, df3)

	// Put all back
	pool.Put(df2)
	pool.Put(df3)
}

func TestDataFramePool_MultipleOperations(t *testing.T) {
	pool := NewDataFramePool()

	dataframes := make([]*CandleDataFrame, 100)

	// Get many DataFrames
	for i := 0; i < 100; i++ {
		dataframes[i] = pool.Get()
		assert.NotNil(t, dataframes[i])
	}

	// Put them all back
	for i := 0; i < 100; i++ {
		pool.Put(dataframes[i])
	}

	// Get them again
	for i := 0; i < 100; i++ {
		df := pool.Get()
		assert.NotNil(t, df)
		pool.Put(df)
	}
}

func TestCandleDataFrame_BasicOperations(t *testing.T) {
	pool := NewDataFramePool()
	df := pool.Get()

	// Test initial state
	assert.NotNil(t, df.DataFrame)
	assert.False(t, df.InUse)
	assert.Nil(t, df.ReleaseFunc)

	// Test acquire
	df.Acquire()
	assert.True(t, df.InUse)

	// Test release
	released := false
	df.ReleaseFunc = func() {
		released = true
	}

	df.Release()
	assert.False(t, df.InUse)
	assert.True(t, released)
}

func TestCandleDataFrame_Reset(t *testing.T) {
	pool := NewDataFramePool()
	df := pool.Get()

	// Simulate usage
	df.Acquire()
	df.ReleaseFunc = func() {}

	// Reset
	df.Reset()

	// Should be in clean state
	assert.False(t, df.InUse)
	assert.Nil(t, df.ReleaseFunc)
	assert.NotNil(t, df.DataFrame) // DataFrame should still exist
}

func TestProcessingPool_GetDataFramePool(t *testing.T) {
	processingPool := NewProcessingPool()

	// Get DataFrame pool
	dfPool := processingPool.GetDataFramePool()
	assert.NotNil(t, dfPool)

	// Should return the same instance
	dfPool2 := processingPool.GetDataFramePool()
	assert.Equal(t, dfPool, dfPool2)
}

func TestProcessingPool_PooledCandleDataFrame(t *testing.T) {
	processingPool := NewProcessingPool()

	// Get pooled candle DataFrame
	candleDF := processingPool.GetPooledCandleDataFrame()
	assert.NotNil(t, candleDF)
	assert.NotNil(t, candleDF.DataFrame)
	assert.True(t, candleDF.InUse)
	assert.NotNil(t, candleDF.ReleaseFunc)

	// Release it
	candleDF.Release()
	assert.False(t, candleDF.InUse)

	// Get another one
	candleDF2 := processingPool.GetPooledCandleDataFrame()
	assert.NotNil(t, candleDF2)
	assert.True(t, candleDF2.InUse)

	// Clean up
	candleDF2.Release()
}

func TestProcessingPool_Stats(t *testing.T) {
	processingPool := NewProcessingPool()

	// Get stats
	stats := processingPool.Stats()
	assert.NotNil(t, stats)

	// Should contain expected keys
	assert.Contains(t, stats, "pool_type")
	assert.Contains(t, stats, "dataframe_pool_active")

	// Values should be reasonable
	assert.Equal(t, "memory_pool", stats["pool_type"])
	assert.IsType(t, 0, stats["dataframe_pool_active"])
}

func TestProcessingPool_ConcurrentAccess(t *testing.T) {
	processingPool := NewProcessingPool()

	const numGoroutines = 50
	const operationsPerGoroutine = 20

	done := make(chan bool, numGoroutines)

	// Start multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Get pooled DataFrame
				candleDF := processingPool.GetPooledCandleDataFrame()
				assert.NotNil(t, candleDF)
				assert.True(t, candleDF.InUse)

				// Simulate some work
				time.Sleep(time.Microsecond)

				// Release it
				candleDF.Release()
				assert.False(t, candleDF.InUse)
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out")
		}
	}
}

// Benchmark tests
func BenchmarkDataFramePool_GetPut(b *testing.B) {
	pool := NewDataFramePool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		df := pool.Get()
		pool.Put(df)
	}
}

func BenchmarkDataFramePool_GetPutParallel(b *testing.B) {
	pool := NewDataFramePool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			df := pool.Get()
			pool.Put(df)
		}
	})
}

func BenchmarkProcessingPool_GetPooledDataFrame(b *testing.B) {
	processingPool := NewProcessingPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		candleDF := processingPool.GetPooledCandleDataFrame()
		candleDF.Release()
	}
}

func BenchmarkProcessingPool_GetPooledDataFrameParallel(b *testing.B) {
	processingPool := NewProcessingPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			candleDF := processingPool.GetPooledCandleDataFrame()
			candleDF.Release()
		}
	})
}

// Memory usage tests
func TestDataFramePool_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	pool := NewDataFramePool()

	// Force garbage collection to get baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Create and release many DataFrames
	for i := 0; i < 1000; i++ {
		df := pool.Get()
		// Simulate some usage
		df.Acquire()
		df.Release()
		pool.Put(df)
	}

	// Force garbage collection
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Calculate memory increase safely (handle potential uint64 overflow)
	var memIncrease uint64
	if m2.Alloc >= m1.Alloc {
		memIncrease = m2.Alloc - m1.Alloc
	} else {
		// Handle wrap-around case
		memIncrease = 0
	}

	t.Logf("Memory increase: %d bytes", memIncrease)
	t.Logf("Initial alloc: %d, Final alloc: %d", m1.Alloc, m2.Alloc)

	// This is more of an informational test - actual limits depend on system
	// Allow for reasonable memory usage (up to 50MB for this test)
	assert.True(t, memIncrease < 50*1024*1024, "Memory usage seems excessive: %d bytes", memIncrease)
}

func TestCandleDataFrame_LifecycleManagement(t *testing.T) {
	pool := NewDataFramePool()

	// Test lifecycle
	df := pool.Get()

	// Initial state
	assert.NotNil(t, df)
	assert.False(t, df.InUse)
	assert.Nil(t, df.ReleaseFunc)

	// Acquire
	df.Acquire()
	assert.True(t, df.InUse)

	// Set release function
	releaseCallCount := 0
	df.ReleaseFunc = func() {
		releaseCallCount++
	}

	// Release
	df.Release()
	assert.False(t, df.InUse)
	assert.Equal(t, 1, releaseCallCount)

	// Multiple releases should not cause issues
	df.Release()
	assert.Equal(t, 1, releaseCallCount) // Should still be 1

	// Reset and reuse
	df.Reset()
	assert.False(t, df.InUse)
	assert.Nil(t, df.ReleaseFunc)

	// Return to pool
	pool.Put(df)
}
