package service

import (
	"context"
	"testing"
	"time"

	"setbull_trader/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCandleRepository for testing
type MockCandleRepository struct {
	mock.Mock
}

func (m *MockCandleRepository) FindByInstrumentAndTimeRange(
	ctx context.Context,
	instrumentKey, interval string,
	start, end time.Time,
) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, interval, start, end)
	return args.Get(0).([]domain.Candle), args.Error(1)
}

// Test cache integration in TechnicalIndicatorServiceV2
func TestTechnicalIndicatorServiceV2_CacheIntegration(t *testing.T) {
	mockRepo := new(MockCandleRepository)
	service := NewTechnicalIndicatorServiceV2(mockRepo)

	ctx := context.Background()
	instrumentKey := "NSE_EQ|INE002A01018"
	interval := "5minute"
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()

	// Mock candle data
	mockCandles := []domain.Candle{
		{
			InstrumentKey: instrumentKey,
			Timestamp:     start,
			Open:          100.0,
			High:          105.0,
			Low:           99.0,
			Close:         103.0,
			Volume:        1000,
		},
		{
			InstrumentKey: instrumentKey,
			Timestamp:     start.Add(5 * time.Minute),
			Open:          103.0,
			High:          107.0,
			Low:           102.0,
			Close:         106.0,
			Volume:        1200,
		},
	}

	// Set up mock expectations - should only be called once due to caching
	mockRepo.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, start, end).
		Return(mockCandles, nil).Once()

	// First call - should hit the repository and cache the result
	result1, err := service.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	assert.NoError(t, err)
	assert.NotNil(t, result1)
	assert.Len(t, result1.Timestamps, 2)

	// Second call - should use cached result, repository should not be called again
	result2, err := service.CalculateAllIndicators(ctx, instrumentKey, interval, start, end)
	assert.NoError(t, err)
	assert.NotNil(t, result2)
	assert.Len(t, result2.Timestamps, 2)

	// Verify results are consistent
	assert.Equal(t, len(result1.EMA9), len(result2.EMA9))
	assert.Equal(t, len(result1.RSI), len(result2.RSI))

	// Verify cache metrics
	metrics := service.GetServiceMetrics()
	assert.Contains(t, metrics, "cache_hits")
	assert.Contains(t, metrics, "cache_misses")
	assert.Contains(t, metrics, "cache_hit_rate")

	// Should have at least one cache hit from the second call
	cacheHits := metrics["cache_hits"].(int64)
	assert.True(t, cacheHits >= 1, "Expected at least 1 cache hit, got %d", cacheHits)

	// Verify all mock expectations were met
	mockRepo.AssertExpectations(t)
}

func TestTechnicalIndicatorServiceV2_CacheClearing(t *testing.T) {
	mockRepo := new(MockCandleRepository)
	service := NewTechnicalIndicatorServiceV2(mockRepo)

	// Test cache clearing
	service.ClearCache()

	// Check initial metrics after clearing
	metrics := service.GetServiceMetrics()
	assert.Equal(t, int64(0), metrics["cache_hits"])
	assert.Equal(t, int64(0), metrics["cache_misses"])
}

func TestTechnicalIndicatorServiceV2_EMA_CacheIntegration(t *testing.T) {
	mockRepo := new(MockCandleRepository)
	service := NewTechnicalIndicatorServiceV2(mockRepo)

	ctx := context.Background()
	instrumentKey := "NSE_EQ|INE002A01018"
	interval := "5minute"
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	period := 9

	// Mock candle data - enough for EMA calculation
	mockCandles := make([]domain.Candle, 20)
	for i := 0; i < 20; i++ {
		mockCandles[i] = domain.Candle{
			InstrumentKey: instrumentKey,
			Timestamp:     start.Add(time.Duration(i*5) * time.Minute),
			Open:          100.0 + float64(i),
			High:          105.0 + float64(i),
			Low:           99.0 + float64(i),
			Close:         103.0 + float64(i),
			Volume:        1000,
		}
	}

	// Set up mock expectations - should only be called once due to caching
	mockRepo.On("FindByInstrumentAndTimeRange", ctx, instrumentKey, interval, start, end).
		Return(mockCandles, nil).Once()

	// First call - should hit the repository and cache the result
	result1, err := service.CalculateEMA(ctx, instrumentKey, period, interval, start, end)
	assert.NoError(t, err)
	assert.NotNil(t, result1)
	assert.Greater(t, len(result1), 0)

	// Second call - should use cached result
	result2, err := service.CalculateEMA(ctx, instrumentKey, period, interval, start, end)
	assert.NoError(t, err)
	assert.NotNil(t, result2)
	assert.Equal(t, len(result1), len(result2))

	// Verify results are identical (cached)
	for i := range result1 {
		assert.Equal(t, result1[i].Value, result2[i].Value)
		assert.Equal(t, result1[i].Timestamp, result2[i].Timestamp)
	}

	// Verify all mock expectations were met
	mockRepo.AssertExpectations(t)
}
