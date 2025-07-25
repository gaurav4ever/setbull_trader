package service

import (
	"context"
	"testing"
	"time"

	"setbull_trader/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// BenchmarkResults stores performance comparison results
type BenchmarkResults struct {
	Implementation   string
	DatasetSize      int
	ProcessingTime   time.Duration
	MemoryAllocated  int64
	CacheHitRate     float64
	ErrorCount       int
	OperationsPerSec float64
}

// CompleteMockCandleRepository implements the full CandleRepository interface for testing
type CompleteMockCandleRepository struct {
	mock.Mock
	testData map[string][]domain.Candle
}

// Implement all required CandleRepository methods
func (m *CompleteMockCandleRepository) Store(ctx context.Context, candle *domain.Candle) error {
	args := m.Called(ctx, candle)
	return args.Error(0)
}

func (m *CompleteMockCandleRepository) StoreBatch(ctx context.Context, candles []domain.Candle) (int, error) {
	args := m.Called(ctx, candles)
	return args.Int(0), args.Error(1)
}

func (m *CompleteMockCandleRepository) FindByInstrumentKey(ctx context.Context, instrumentKey string) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey)
	return args.Get(0).([]domain.Candle), args.Error(1)
}

func (m *CompleteMockCandleRepository) FindByInstrumentAndInterval(ctx context.Context, instrumentKey, interval string) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, interval)
	return args.Get(0).([]domain.Candle), args.Error(1)
}

func (m *CompleteMockCandleRepository) FindByInstrumentAndTimeRange(
	ctx context.Context,
	instrumentKey string,
	interval string,
	fromTime, toTime time.Time,
) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, interval, fromTime, toTime)

	// Return pre-configured test data if available
	if data, exists := m.testData[instrumentKey]; exists {
		return data, args.Error(1)
	}

	return args.Get(0).([]domain.Candle), args.Error(1)
}

func (m *CompleteMockCandleRepository) DeleteByInstrumentAndTimeRange(
	ctx context.Context,
	instrumentKey string,
	interval string,
	fromTime, toTime time.Time,
) (int, error) {
	args := m.Called(ctx, instrumentKey, interval, fromTime, toTime)
	return args.Int(0), args.Error(1)
}

func (m *CompleteMockCandleRepository) CountByInstrumentAndTimeRange(
	ctx context.Context,
	instrumentKey string,
	interval string,
	fromTime, toTime time.Time,
) (int, error) {
	args := m.Called(ctx, instrumentKey, interval, fromTime, toTime)
	return args.Int(0), args.Error(1)
}

func (m *CompleteMockCandleRepository) DeleteOlderThan(ctx context.Context, olderThan time.Time) (int, error) {
	args := m.Called(ctx, olderThan)
	return args.Int(0), args.Error(1)
}

func (m *CompleteMockCandleRepository) GetLatestCandle(ctx context.Context, instrumentKey, interval string) (*domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, interval)
	return args.Get(0).(*domain.Candle), args.Error(1)
}

func (m *CompleteMockCandleRepository) GetEarliestCandle(ctx context.Context, instrumentKey string, interval string) (*domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, interval)
	return args.Get(0).(*domain.Candle), args.Error(1)
}

func (m *CompleteMockCandleRepository) GetCandleDateRange(ctx context.Context, instrumentKey string, interval string) (earliest, latest time.Time, exists bool, err error) {
	args := m.Called(ctx, instrumentKey, interval)
	return args.Get(0).(time.Time), args.Get(1).(time.Time), args.Bool(2), args.Error(3)
}

func (m *CompleteMockCandleRepository) GetNDailyCandlesByTimeframe(ctx context.Context, instrumentKey string, interval string, n int) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, interval, n)
	return args.Get(0).([]domain.Candle), args.Error(1)
}

func (m *CompleteMockCandleRepository) GetAggregated5MinCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	args := m.Called(ctx, instrumentKey, start, end)
	return args.Get(0).([]domain.AggregatedCandle), args.Error(1)
}

func (m *CompleteMockCandleRepository) GetAggregatedDailyCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	args := m.Called(ctx, instrumentKey, start, end)
	return args.Get(0).([]domain.AggregatedCandle), args.Error(1)
}

func (m *CompleteMockCandleRepository) GetDailyCandlesByTimeframe(ctx context.Context, instrumentKey string, startTime time.Time) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, startTime)
	return args.Get(0).([]domain.Candle), args.Error(1)
}

func (m *CompleteMockCandleRepository) GetStocksWithExistingDailyCandles(ctx context.Context, startDate, endDate time.Time) ([]string, error) {
	args := m.Called(ctx, startDate, endDate)
	return args.Get(0).([]string), args.Error(1)
}

func (m *CompleteMockCandleRepository) StoreAggregatedCandles(ctx context.Context, candles []domain.CandleData) error {
	args := m.Called(ctx, candles)
	return args.Error(0)
}

// SetTestData configures test data for specific instruments
func (m *CompleteMockCandleRepository) SetTestData(instrumentKey string, candles []domain.Candle) {
	if m.testData == nil {
		m.testData = make(map[string][]domain.Candle)
	}
	m.testData[instrumentKey] = candles
}

// generateBenchmarkCandles creates realistic test candle data
func generateBenchmarkCandles(count int, instrumentKey string) []domain.Candle {
	candles := make([]domain.Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * time.Minute)
	basePrice := 100.0

	for i := 0; i < count; i++ {
		// Generate realistic OHLCV data with some volatility
		variation := float64(i%10-5) * 0.1
		open := basePrice + variation
		high := open + 0.5
		low := open - 0.3
		close := open + variation*0.5
		volume := int64(1000 + i*10)

		candles[i] = domain.Candle{
			InstrumentKey: instrumentKey,
			TimeInterval:  "1minute",
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Open:          open,
			High:          high,
			Low:           low,
			Close:         close,
			Volume:        volume,
		}
	}

	return candles
}

// TestPerformanceComparison_V1_vs_V2_SmallDataset compares V1 and V2 performance on small dataset
func TestPerformanceComparison_V1_vs_V2_SmallDataset(t *testing.T) {
	ctx := context.Background()
	instrumentKey := "NSE_EQ|INE002A01018"
	interval := "1minute"
	start := time.Now().Add(-2 * time.Hour)
	end := time.Now()

	// Generate test data (100 candles)
	testCandles := generateBenchmarkCandles(100, instrumentKey)

	// Test V1 (Original) Service
	mockRepoV1 := &CompleteMockCandleRepository{}
	mockRepoV1.SetTestData(instrumentKey, testCandles)
	mockRepoV1.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	serviceV1 := NewTechnicalIndicatorService(mockRepoV1)

	startTime := time.Now()
	resultV1, errV1 := serviceV1.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	v1Duration := time.Since(startTime)

	// Test V2 (Optimized) Service
	mockRepoV2 := &CompleteMockCandleRepository{}
	mockRepoV2.SetTestData(instrumentKey, testCandles)
	mockRepoV2.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	serviceV2 := NewTechnicalIndicatorServiceV2(mockRepoV2)

	startTime = time.Now()
	resultV2, errV2 := serviceV2.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	v2Duration := time.Since(startTime)

	// Validate results
	assert.NoError(t, errV1, "V1 service should not error")
	assert.NoError(t, errV2, "V2 service should not error")
	assert.NotNil(t, resultV1, "V1 should return results")
	assert.NotNil(t, resultV2, "V2 should return results")

	// Calculate performance improvement
	speedImprovement := float64(v1Duration-v2Duration) / float64(v1Duration) * 100

	t.Logf("Performance Comparison (100 candles):")
	t.Logf("  V1 Original: %v", v1Duration)
	t.Logf("  V2 Optimized: %v", v2Duration)
	t.Logf("  Speed Improvement: %.2f%%", speedImprovement)

	// V2 should be faster (though improvement might be minimal for small datasets)
	assert.True(t, v2Duration <= v1Duration*2, "V2 should not be significantly slower than V1")

	// Verify mock expectations
	mockRepoV1.AssertExpectations(t)
	mockRepoV2.AssertExpectations(t)
}

// TestPerformanceComparison_V1_vs_V2_LargeDataset compares performance on larger dataset
func TestPerformanceComparison_V1_vs_V2_LargeDataset(t *testing.T) {
	ctx := context.Background()
	instrumentKey := "NSE_EQ|INE002A01018"
	interval := "1minute"
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	// Generate larger test data (1000 candles)
	testCandles := generateBenchmarkCandles(1000, instrumentKey)

	// Test V1 (Original) Service
	mockRepoV1 := &CompleteMockCandleRepository{}
	mockRepoV1.SetTestData(instrumentKey, testCandles)
	mockRepoV1.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	serviceV1 := NewTechnicalIndicatorService(mockRepoV1)

	startTime := time.Now()
	resultV1, errV1 := serviceV1.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	v1Duration := time.Since(startTime)

	// Test V2 (Optimized) Service
	mockRepoV2 := &CompleteMockCandleRepository{}
	mockRepoV2.SetTestData(instrumentKey, testCandles)
	mockRepoV2.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	serviceV2 := NewTechnicalIndicatorServiceV2(mockRepoV2)

	startTime = time.Now()
	resultV2, errV2 := serviceV2.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	v2Duration := time.Since(startTime)

	// Validate results
	assert.NoError(t, errV1, "V1 service should not error")
	assert.NoError(t, errV2, "V2 service should not error")
	assert.NotNil(t, resultV1, "V1 should return results")
	assert.NotNil(t, resultV2, "V2 should return results")

	// Calculate performance improvement
	speedImprovement := float64(v1Duration-v2Duration) / float64(v1Duration) * 100

	t.Logf("Performance Comparison (1000 candles):")
	t.Logf("  V1 Original: %v", v1Duration)
	t.Logf("  V2 Optimized: %v", v2Duration)
	t.Logf("  Speed Improvement: %.2f%%", speedImprovement)

	// For larger datasets, V2 should show measurable improvement
	assert.True(t, speedImprovement >= 0, "V2 should be at least as fast as V1 for large datasets")

	// V2 should provide cache metrics
	metrics := serviceV2.GetServiceMetrics()
	assert.Contains(t, metrics, "service_type")
	assert.Equal(t, "GoNum-optimized", metrics["service_type"])

	// Verify mock expectations
	mockRepoV1.AssertExpectations(t)
	mockRepoV2.AssertExpectations(t)
}

// TestCacheEffectiveness_V2 tests cache effectiveness in V2 service
func TestCacheEffectiveness_V2(t *testing.T) {
	ctx := context.Background()
	instrumentKey := "NSE_EQ|INE002A01018"
	interval := "1minute"
	start := time.Now().Add(-2 * time.Hour)
	end := time.Now()

	// Generate test data
	testCandles := generateBenchmarkCandles(500, instrumentKey)

	// Setup mock repository
	mockRepo := &CompleteMockCandleRepository{}
	mockRepo.SetTestData(instrumentKey, testCandles)
	// Repository should only be called once due to caching
	mockRepo.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil).Once()

	serviceV2 := NewTechnicalIndicatorServiceV2(mockRepo)

	// First call - should hit repository
	startTime := time.Now()
	result1, err1 := serviceV2.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	firstCallDuration := time.Since(startTime)

	// Second call - should hit cache
	startTime = time.Now()
	result2, err2 := serviceV2.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	secondCallDuration := time.Since(startTime)

	// Validate results
	assert.NoError(t, err1, "First call should not error")
	assert.NoError(t, err2, "Second call should not error")
	assert.NotNil(t, result1, "First call should return results")
	assert.NotNil(t, result2, "Second call should return results")

	// Calculate cache effectiveness
	cacheSpeedup := float64(firstCallDuration) / float64(secondCallDuration)

	t.Logf("Cache Effectiveness Test:")
	t.Logf("  First call (repository): %v", firstCallDuration)
	t.Logf("  Second call (cache): %v", secondCallDuration)
	t.Logf("  Cache speedup: %.2fx", cacheSpeedup)

	// Cache should provide significant speedup
	assert.True(t, cacheSpeedup >= 2.0, "Cache should provide at least 2x speedup")

	// Verify cache metrics
	metrics := serviceV2.GetServiceMetrics()
	cacheHits := metrics["cache_hits"].(int64)
	assert.True(t, cacheHits >= 1, "Should have at least 1 cache hit")

	// Verify mock expectations (should only be called once)
	mockRepo.AssertExpectations(t)
}

// Benchmark tests for continuous performance monitoring
func BenchmarkTechnicalIndicatorService_V1_Small(b *testing.B) {
	benchmarkService(b, "V1", 100)
}

func BenchmarkTechnicalIndicatorService_V2_Small(b *testing.B) {
	benchmarkService(b, "V2", 100)
}

func BenchmarkTechnicalIndicatorService_V1_Medium(b *testing.B) {
	benchmarkService(b, "V1", 1000)
}

func BenchmarkTechnicalIndicatorService_V2_Medium(b *testing.B) {
	benchmarkService(b, "V2", 1000)
}

// benchmarkService runs benchmark for specific service version and dataset size
func benchmarkService(b *testing.B, version string, datasetSize int) {
	ctx := context.Background()
	instrumentKey := "NSE_EQ|INE002A01018"
	interval := "1minute"
	start := time.Now().Add(-time.Duration(datasetSize) * time.Minute)
	end := time.Now()

	// Generate test data
	testCandles := generateBenchmarkCandles(datasetSize, instrumentKey)

	// Setup mock repository
	mockRepo := &CompleteMockCandleRepository{}
	mockRepo.SetTestData(instrumentKey, testCandles)
	mockRepo.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, mock.Anything, mock.Anything).
		Return(testCandles, nil)

	b.ResetTimer()

	switch version {
	case "V1":
		service := NewTechnicalIndicatorService(mockRepo)
		for i := 0; i < b.N; i++ {
			_, err := service.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
			if err != nil {
				b.Fatalf("Error in V1 service: %v", err)
			}
		}
	case "V2":
		service := NewTechnicalIndicatorServiceV2(mockRepo)
		for i := 0; i < b.N; i++ {
			_, err := service.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
			if err != nil {
				b.Fatalf("Error in V2 service: %v", err)
			}
		}
	default:
		b.Fatalf("Unknown service version: %s", version)
	}
}
