package service

import (
	"context"
	"testing"
	"time"

	"setbull_trader/internal/domain"
)

// SimpleMockCandleRepository for basic testing
type SimpleMockCandleRepository struct{}

func (m *SimpleMockCandleRepository) GetCandles(ctx context.Context, instrumentKey string, startTime, endTime time.Time) ([]domain.Candle, error) {
	// Generate simple test data
	var candles []domain.Candle
	current := startTime

	for current.Before(endTime) && len(candles) < 120 { // Limit to 120 candles (2 hours of minute data)
		basePrice := 100.0 + float64(len(candles)%10)
		candle := domain.Candle{
			InstrumentKey: instrumentKey,
			Timestamp:     current,
			Open:          basePrice,
			High:          basePrice + 2.0,
			Low:           basePrice - 1.0,
			Close:         basePrice + 1.0,
			Volume:        1000,
		}
		candles = append(candles, candle)
		current = current.Add(time.Minute)
	}

	return candles, nil
}

func (m *SimpleMockCandleRepository) GetCandlesByTimeframe(ctx context.Context, instrumentKey string, timeframe string, startTime, endTime time.Time) ([]domain.Candle, error) {
	return m.GetCandles(ctx, instrumentKey, startTime, endTime)
}

func (m *SimpleMockCandleRepository) SaveCandle(ctx context.Context, candle domain.Candle) error {
	return nil
}

func (m *SimpleMockCandleRepository) SaveCandles(ctx context.Context, candles []domain.Candle) error {
	return nil
}

func (m *SimpleMockCandleRepository) GetLatestCandle(ctx context.Context, instrumentKey string) (*domain.Candle, error) {
	now := time.Now()
	candle := &domain.Candle{
		InstrumentKey: instrumentKey,
		Timestamp:     now,
		Open:          100.0,
		High:          105.0,
		Low:           95.0,
		Close:         102.0,
		Volume:        1000,
	}
	return candle, nil
}

func (m *SimpleMockCandleRepository) CountByInstrumentAndTimeRange(ctx context.Context, instrumentKey string, interval string, startTime, endTime time.Time) (int, error) {
	return 120, nil // Mock count for CandleRepository
}

func (m *SimpleMockCandleRepository) DeleteByInstrumentAndTimeRange(ctx context.Context, instrumentKey string, interval string, startTime, endTime time.Time) (int, error) {
	return 0, nil // Mock delete
}

func (m *SimpleMockCandleRepository) DeleteOlderThan(ctx context.Context, olderThan time.Time) (int, error) {
	return 0, nil
}

// SimpleMock5MinRepository for 5-min repository interface
type SimpleMock5MinRepository struct{}

func (m *SimpleMock5MinRepository) SaveCandle5Min(ctx context.Context, candle domain.Candle5Min) error {
	return nil
}

func (m *SimpleMock5MinRepository) SaveCandles5Min(ctx context.Context, candles []domain.Candle5Min) error {
	return nil
}

func (m *SimpleMock5MinRepository) GetCandles5Min(ctx context.Context, instrumentKey string, startTime, endTime time.Time) ([]domain.Candle5Min, error) {
	return []domain.Candle5Min{}, nil
}

func (m *SimpleMock5MinRepository) GetLatestCandle5Min(ctx context.Context, instrumentKey string) (*domain.Candle5Min, error) {
	return nil, nil
}

func (m *SimpleMock5MinRepository) CountByInstrumentAndTimeRange(ctx context.Context, instrumentKey string, startTime, endTime time.Time) (int, error) {
	return 24, nil // Mock count for 5MinRepository (no interval parameter)
}

func (m *SimpleMock5MinRepository) DeleteOlderThan(ctx context.Context, olderThan time.Time) (int, error) {
	return 0, nil
}

// Benchmark DataFrame-based aggregation
func BenchmarkDataFrameAggregation(b *testing.B) {
	instrumentKey := "NSE_EQ|INE002A01018"
	endTime := time.Now()
	startTime := endTime.Add(-2 * time.Hour)

	mockRepo := &SimpleMockCandleRepository{}
	mock5MinRepo := &SimpleMock5MinRepository{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service := NewCandleAggregationServiceV2(mockRepo, mock5MinRepo, nil, nil, nil)

		var candleCount int
		err := service.Aggregate5MinCandlesWithIndicators(
			context.Background(),
			instrumentKey,
			startTime,
			endTime,
			func(ctx context.Context, instrumentKey string, candle domain.AggregatedCandle) {
				candleCount++
			},
		)

		if err != nil {
			b.Errorf("Aggregation failed: %v", err)
		}

		if candleCount == 0 {
			b.Error("No candles produced")
		}
	}
}

// Test basic functionality
func TestDataFrameAggregationBasic(t *testing.T) {
	instrumentKey := "NSE_EQ|INE002A01018"
	endTime := time.Now()
	startTime := endTime.Add(-2 * time.Hour)

	mockRepo := &SimpleMockCandleRepository{}
	mock5MinRepo := &SimpleMock5MinRepository{}
	service := NewCandleAggregationServiceV2(mockRepo, mock5MinRepo, nil, nil, nil)

	var candleCount int
	err := service.Aggregate5MinCandlesWithIndicators(
		context.Background(),
		instrumentKey,
		startTime,
		endTime,
		func(ctx context.Context, instrumentKey string, candle domain.AggregatedCandle) {
			candleCount++
		},
	)

	if err != nil {
		t.Errorf("Aggregation failed: %v", err)
	}

	if candleCount == 0 {
		t.Error("No candles produced")
	}

	t.Logf("DataFrame aggregation test passed. Produced %d candles", candleCount)
}
