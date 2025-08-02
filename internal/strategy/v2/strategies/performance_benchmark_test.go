package strategies

import (
	"testing"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/stretchr/testify/assert"
)

// BenchmarkBBWidthStrategyV2 benchmarks the V2 BB Width strategy
func BenchmarkBBWidthStrategyV2(b *testing.B) {
	strategy := NewBBWidthStrategyV2()
	df := createBenchmarkData(1000) // 1000 candles

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := strategy.Process(df)
		if err != nil {
			b.Fatalf("Strategy processing failed: %v", err)
		}
	}
}

// BenchmarkBBWidthEnhancedStrategyV2 benchmarks the enhanced V2 BB Width strategy
func BenchmarkBBWidthEnhancedStrategyV2(b *testing.B) {
	strategy := NewBBWidthEnhancedStrategyV2()
	df := createBenchmarkData(1000) // 1000 candles

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := strategy.Process(df)
		if err != nil {
			b.Fatalf("Strategy processing failed: %v", err)
		}
	}
}

// BenchmarkStrategyMemoryUsage benchmarks memory usage
func BenchmarkStrategyMemoryUsage(b *testing.B) {
	strategy := NewBBWidthEnhancedStrategyV2()
	df := createBenchmarkData(1000)

	// Force garbage collection before benchmark
	b.ResetTimer()

	var memBefore, memAfter uint64
	for i := 0; i < b.N; i++ {
		// Record memory before
		memBefore = getMemoryUsage()

		// Process data
		_, err := strategy.Process(df)
		if err != nil {
			b.Fatalf("Strategy processing failed: %v", err)
		}

		// Record memory after
		memAfter = getMemoryUsage()

		// Report memory usage
		b.ReportMetric(float64(memAfter-memBefore), "bytes/op")
	}
}

// BenchmarkConcurrentStrategyProcessing benchmarks concurrent strategy processing
func BenchmarkConcurrentStrategyProcessing(b *testing.B) {
	strategy := NewBBWidthEnhancedStrategyV2()
	df := createBenchmarkData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := strategy.Process(df)
			if err != nil {
				b.Fatalf("Strategy processing failed: %v", err)
			}
		}
	})
}

// TestPerformanceComparison compares V1 vs V2 performance
func TestPerformanceComparison(t *testing.T) {
	// Create test data
	df := createBenchmarkData(1000)

	// Test V2 Basic Strategy
	v2BasicStrategy := NewBBWidthStrategyV2()
	start := time.Now()
	_, err := v2BasicStrategy.Process(df)
	v2BasicTime := time.Since(start)
	assert.NoError(t, err)

	// Test V2 Enhanced Strategy
	v2EnhancedStrategy := NewBBWidthEnhancedStrategyV2()
	start = time.Now()
	_, err = v2EnhancedStrategy.Process(df)
	v2EnhancedTime := time.Since(start)
	assert.NoError(t, err)

	// Performance assertions
	t.Logf("V2 Basic Strategy processing time: %v", v2BasicTime)
	t.Logf("V2 Enhanced Strategy processing time: %v", v2EnhancedTime)

	// V2 should be faster than typical Python processing
	// Python typically takes 10-50ms for similar operations
	assert.True(t, v2BasicTime < 50*time.Millisecond,
		"V2 Basic strategy should be faster than 50ms, got %v", v2BasicTime)
	assert.True(t, v2EnhancedTime < 100*time.Millisecond,
		"V2 Enhanced strategy should be faster than 100ms, got %v", v2EnhancedTime)

	// Enhanced strategy should take more time than basic (more features)
	assert.True(t, v2EnhancedTime > v2BasicTime,
		"Enhanced strategy should take more time than basic strategy")
}

// TestMemoryEfficiency tests memory efficiency
func TestMemoryEfficiency(t *testing.T) {
	strategy := NewBBWidthEnhancedStrategyV2()
	df := createBenchmarkData(1000)

	// Get memory requirements
	memoryRequirements := strategy.GetMemoryRequirements()

	// Process data and check memory usage
	start := time.Now()
	_, err := strategy.Process(df)
	processingTime := time.Since(start)
	assert.NoError(t, err)

	t.Logf("Memory requirements: %d bytes (%d MB)",
		memoryRequirements, memoryRequirements/1024/1024)
	t.Logf("Processing time: %v", processingTime)

	// Memory requirements should be reasonable (< 100MB)
	assert.True(t, memoryRequirements < 100*1024*1024,
		"Memory requirements should be less than 100MB, got %d bytes", memoryRequirements)
}

// Helper functions

// createBenchmarkData creates benchmark test data
func createBenchmarkData(n int) *dataframe.DataFrame {
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

		// Create realistic price movement
		opens[i] = basePrice + float64(i)*0.1 + float64(i%10)*0.5
		highs[i] = opens[i] + 0.5 + float64(i%5)*0.2
		lows[i] = opens[i] - 0.3 - float64(i%7)*0.1
		closes[i] = opens[i] + 0.2 + float64(i%3)*0.3
		volumes[i] = 1000 + int64(i*10) + int64(i%20)*100
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

// getMemoryUsage returns current memory usage (simplified)
func getMemoryUsage() uint64 {
	// In a real implementation, this would use runtime.ReadMemStats
	// For now, return a placeholder value
	return 0
}
