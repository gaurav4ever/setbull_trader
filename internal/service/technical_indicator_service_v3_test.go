package service

import (
	"context"
	"testing"
	"time"

	"setbull_trader/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCandleRepository for testing
type MockCandleRepositoryV3 struct{}

func (m *MockCandleRepositoryV3) Store(ctx context.Context, candle *domain.Candle) error {
	return nil
}

func (m *MockCandleRepositoryV3) StoreBatch(ctx context.Context, candles []domain.Candle) (int, error) {
	return len(candles), nil
}

func (m *MockCandleRepositoryV3) FindByInstrumentKey(ctx context.Context, instrumentKey string) ([]domain.Candle, error) {
	return []domain.Candle{}, nil
}

func (m *MockCandleRepositoryV3) FindByInstrumentAndInterval(ctx context.Context, instrumentKey, interval string) ([]domain.Candle, error) {
	return []domain.Candle{}, nil
}

func (m *MockCandleRepositoryV3) FindByInstrumentAndTimeRange(ctx context.Context, instrumentKey, interval string, fromTime, toTime time.Time) ([]domain.Candle, error) {
	return []domain.Candle{}, nil
}

func (m *MockCandleRepositoryV3) DeleteByInstrumentAndTimeRange(ctx context.Context, instrumentKey, interval string, fromTime, toTime time.Time) (int, error) {
	return 0, nil
}

func (m *MockCandleRepositoryV3) CountByInstrumentAndTimeRange(ctx context.Context, instrumentKey, interval string, fromTime, toTime time.Time) (int, error) {
	return 0, nil
}

func (m *MockCandleRepositoryV3) DeleteOlderThan(ctx context.Context, olderThan time.Time) (int, error) {
	return 0, nil
}

func (m *MockCandleRepositoryV3) GetLatestCandle(ctx context.Context, instrumentKey, interval string) (*domain.Candle, error) {
	return nil, nil
}

func (m *MockCandleRepositoryV3) GetEarliestCandle(ctx context.Context, instrumentKey, interval string) (*domain.Candle, error) {
	return nil, nil
}

func (m *MockCandleRepositoryV3) GetCandleDateRange(ctx context.Context, instrumentKey, interval string) (earliest, latest time.Time, exists bool, err error) {
	return time.Time{}, time.Time{}, false, nil
}

func (m *MockCandleRepositoryV3) GetNDailyCandlesByTimeframe(ctx context.Context, instrumentKey, interval string, n int) ([]domain.Candle, error) {
	return []domain.Candle{}, nil
}

func (m *MockCandleRepositoryV3) GetAggregated5MinCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	return []domain.AggregatedCandle{}, nil
}

func (m *MockCandleRepositoryV3) GetAggregatedDailyCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	return []domain.AggregatedCandle{}, nil
}

func (m *MockCandleRepositoryV3) GetDailyCandlesByTimeframe(ctx context.Context, instrumentKey string, startTime time.Time) ([]domain.Candle, error) {
	return []domain.Candle{}, nil
}

func TestTechnicalIndicatorServiceV3_New(t *testing.T) {
	// Arrange
	mockRepo := &MockCandleRepositoryV3{}
	config := TechnicalIndicatorServiceV3Config{
		MaxWorkers:     2,
		CacheSize:      1024 * 1024, // 1MB
		MetricsEnabled: true,
	}

	// Act
	service, err := NewTechnicalIndicatorServiceV3(mockRepo, config)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, 2, service.maxWorkers)
	assert.Equal(t, 1024*1024, service.cacheSize)
	assert.True(t, service.metricsEnabled)
}

func TestTechnicalIndicatorServiceV3_CalculateIndicators(t *testing.T) {
	// Arrange
	mockRepo := &MockCandleRepositoryV3{}
	config := TechnicalIndicatorServiceV3Config{
		MaxWorkers:     2,
		CacheSize:      1024 * 1024,
		MetricsEnabled: true,
	}

	service, err := NewTechnicalIndicatorServiceV3(mockRepo, config)
	require.NoError(t, err)

	ctx := context.Background()
	instrumentKey := "TEST_STOCK"
	interval := "1m"
	start := time.Now().Add(-2 * time.Hour)
	end := time.Now()

	// Act
	indicators, err := service.CalculateIndicators(ctx, instrumentKey, interval, start, end)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, indicators)
	assert.Equal(t, 100, len(indicators.Timestamps)) // Should have 100 sample candles
}

func TestTechnicalIndicatorServiceV3_GetMetrics(t *testing.T) {
	// Arrange
	mockRepo := &MockCandleRepositoryV3{}
	config := TechnicalIndicatorServiceV3Config{
		MaxWorkers:     2,
		CacheSize:      1024 * 1024,
		MetricsEnabled: true,
	}

	service, err := NewTechnicalIndicatorServiceV3(mockRepo, config)
	require.NoError(t, err)

	// Act
	metrics := service.GetMetrics()

	// Assert
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "total_requests")
	assert.Contains(t, metrics, "cache_hits")
	assert.Contains(t, metrics, "worker_pool_size")
	assert.Equal(t, int64(0), metrics["total_requests"])
	assert.Equal(t, 2, metrics["worker_pool_size"])
}

func TestTechnicalIndicatorServiceV3_GetMetrics_Disabled(t *testing.T) {
	// Arrange
	mockRepo := &MockCandleRepositoryV3{}
	config := TechnicalIndicatorServiceV3Config{
		MaxWorkers:     2,
		CacheSize:      1024 * 1024,
		MetricsEnabled: false, // Disabled
	}

	service, err := NewTechnicalIndicatorServiceV3(mockRepo, config)
	require.NoError(t, err)

	// Act
	metrics := service.GetMetrics()

	// Assert
	assert.Nil(t, metrics)
}

func TestTechnicalIndicatorServiceV3_Shutdown(t *testing.T) {
	// Arrange
	mockRepo := &MockCandleRepositoryV3{}
	config := TechnicalIndicatorServiceV3Config{
		MaxWorkers:     2,
		CacheSize:      1024 * 1024,
		MetricsEnabled: true,
	}

	service, err := NewTechnicalIndicatorServiceV3(mockRepo, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Act
	err = service.Shutdown(ctx)

	// Assert
	assert.NoError(t, err)
}
