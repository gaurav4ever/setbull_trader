package service

import (
	"context"
	"testing"
	"time"

	"setbull_trader/internal/core/dto/response"
	"setbull_trader/internal/domain"
)

// MockCandleRepository for testing
type MockCandleRepository struct {
	candles []domain.Candle
}

func (m *MockCandleRepository) FindByInstrumentAndTimeRange(ctx context.Context, instrumentKey, interval string, startTime, endTime time.Time) ([]domain.Candle, error) {
	return m.candles, nil
}

// Implement other required methods with minimal functionality
func (m *MockCandleRepository) Save(ctx context.Context, candle domain.Candle) error { return nil }
func (m *MockCandleRepository) FindByInstrumentKey(ctx context.Context, instrumentKey string) ([]domain.Candle, error) {
	return nil, nil
}
func (m *MockCandleRepository) CountByInstrumentAndTimeRange(ctx context.Context, instrumentKey, interval string, startTime, endTime time.Time) (int, error) {
	return len(m.candles), nil
}

// MockCandle5MinRepository for testing
type MockCandle5MinRepository struct {
	candles []domain.Candle5Min
}

func (m *MockCandle5MinRepository) Store(ctx context.Context, candle *domain.Candle5Min) error {
	m.candles = append(m.candles, *candle)
	return nil
}

func (m *MockCandle5MinRepository) FindByInstrumentAndTimeRange(ctx context.Context, instrumentKey string, startTime, endTime time.Time) ([]domain.Candle5Min, error) {
	return m.candles, nil
}

// Implement other required methods with minimal functionality
func (m *MockCandle5MinRepository) StoreBatch(ctx context.Context, candles []domain.Candle5Min) (int, error) {
	return len(candles), nil
}
func (m *MockCandle5MinRepository) FindByInstrumentKey(ctx context.Context, instrumentKey string) ([]domain.Candle5Min, error) {
	return nil, nil
}
func (m *MockCandle5MinRepository) DeleteByInstrumentAndTimeRange(ctx context.Context, instrumentKey string, startTime, endTime time.Time) error {
	return nil
}

func TestCandleAggregationServiceV2_BasicFunctionality(t *testing.T) {
	// Create mock repositories
	mockCandleRepo := &MockCandleRepository{}
	mockCandle5MinRepo := &MockCandle5MinRepository{}

	// Create mock services (minimal implementations)
	mockBatchFetch := &BatchFetchService{}
	mockTradingCalendar := &TradingCalendarService{}
	mockUtility := &UtilityService{}

	// Create test service
	service := NewCandleAggregationServiceV2(
		mockCandleRepo,
		mockCandle5MinRepo,
		mockBatchFetch,
		mockTradingCalendar,
		mockUtility,
	)

	// Test data
	now := time.Now()
	testCandles := []domain.Candle{
		{
			InstrumentKey: "RELIANCE",
			Timestamp:     now,
			Open:          100.0,
			High:          105.0,
			Low:           98.0,
			Close:         102.0,
			Volume:        1000,
			OpenInterest:  500,
		},
		{
			InstrumentKey: "RELIANCE",
			Timestamp:     now.Add(time.Minute),
			Open:          102.0,
			High:          107.0,
			Low:           100.0,
			Close:         105.0,
			Volume:        1500,
			OpenInterest:  600,
		},
		{
			InstrumentKey: "RELIANCE",
			Timestamp:     now.Add(2 * time.Minute),
			Open:          105.0,
			High:          108.0,
			Low:           103.0,
			Close:         106.0,
			Volume:        1200,
			OpenInterest:  650,
		},
		{
			InstrumentKey: "RELIANCE",
			Timestamp:     now.Add(3 * time.Minute),
			Open:          106.0,
			High:          109.0,
			Low:           104.0,
			Close:         107.0,
			Volume:        1300,
			OpenInterest:  700,
		},
		{
			InstrumentKey: "RELIANCE",
			Timestamp:     now.Add(4 * time.Minute),
			Open:          107.0,
			High:          110.0,
			Low:           105.0,
			Close:         108.0,
			Volume:        1100,
			OpenInterest:  750,
		},
	}

	mockCandleRepo.candles = testCandles

	ctx := context.Background()
	startTime := now.Add(-time.Hour)
	endTime := now.Add(time.Hour)

	// Test 1: Aggregate5MinCandlesWithIndicators
	callbackCalled := false
	err := service.Aggregate5MinCandlesWithIndicators(
		ctx,
		"RELIANCE",
		startTime,
		endTime,
		func(ctx context.Context, instrumentKey string, candle domain.AggregatedCandle) {
			callbackCalled = true
			if instrumentKey != "RELIANCE" {
				t.Errorf("Expected instrument key 'RELIANCE', got '%s'", instrumentKey)
			}
		},
	)

	if err != nil {
		t.Fatalf("Aggregate5MinCandlesWithIndicators failed: %v", err)
	}

	if !callbackCalled {
		t.Error("BB width callback was not called")
	}

	// Test 2: Get5MinCandles
	result, err := service.Get5MinCandles(ctx, "RELIANCE", startTime, endTime)
	if err != nil {
		t.Fatalf("Get5MinCandles failed: %v", err)
	}

	if len(result) == 0 {
		t.Error("Expected aggregated candles, got empty result")
	}

	// Verify aggregation logic - 5 1-minute candles should aggregate to 1 5-minute candle
	if len(result) != 1 {
		t.Errorf("Expected 1 aggregated 5-minute candle, got %d", len(result))
	}

	// Verify aggregated values
	aggregated := result[0]
	if aggregated.InstrumentKey != "RELIANCE" {
		t.Errorf("Expected instrument key 'RELIANCE', got '%s'", aggregated.InstrumentKey)
	}

	if aggregated.Open != 100.0 {
		t.Errorf("Expected open price 100.0, got %f", aggregated.Open)
	}

	if aggregated.High != 110.0 {
		t.Errorf("Expected high price 110.0, got %f", aggregated.High)
	}

	if aggregated.Low != 98.0 {
		t.Errorf("Expected low price 98.0, got %f", aggregated.Low)
	}

	if aggregated.Close != 108.0 {
		t.Errorf("Expected close price 108.0, got %f", aggregated.Close)
	}

	expectedVolume := int64(1000 + 1500 + 1200 + 1300 + 1100)
	if aggregated.Volume != expectedVolume {
		t.Errorf("Expected volume %d, got %d", expectedVolume, aggregated.Volume)
	}

	// Test 3: Analytics Metrics
	metrics := service.GetAnalyticsMetrics()
	if metrics == nil {
		t.Error("Expected analytics metrics, got nil")
	}

	t.Logf("Test completed successfully. Aggregated %d 1-minute candles into %d 5-minute candles",
		len(testCandles), len(result))
}

func TestCandleAggregationServiceV2_NotifyOnNew5MinCandles(t *testing.T) {
	// Create mock repositories
	mockCandleRepo := &MockCandleRepository{}
	mockCandle5MinRepo := &MockCandle5MinRepository{}

	// Create mock services
	mockBatchFetch := &BatchFetchService{}
	mockTradingCalendar := &TradingCalendarService{}
	mockUtility := &UtilityService{}

	// Create test service
	service := NewCandleAggregationServiceV2(
		mockCandleRepo,
		mockCandle5MinRepo,
		mockBatchFetch,
		mockTradingCalendar,
		mockUtility,
	)

	// Test data
	now := time.Now()
	testCandles := []domain.Candle{
		{
			InstrumentKey: "TATASTEEL",
			Timestamp:     now,
			Open:          200.0,
			High:          205.0,
			Low:           198.0,
			Close:         202.0,
			Volume:        2000,
		},
	}

	mockCandleRepo.candles = testCandles

	// Register a test listener
	listenerCalled := false
	service.RegisterCandleCloseListener(func(candles []domain.AggregatedCandle, stock response.StockGroupStockDTO) {
		listenerCalled = true
		if len(candles) == 0 {
			t.Error("Expected candles in listener callback, got empty slice")
		}
		if stock.InstrumentKey != "TATASTEEL" {
			t.Errorf("Expected stock instrument key 'TATASTEEL', got '%s'", stock.InstrumentKey)
		}
	})

	// Test notification
	ctx := context.Background()
	stock := response.StockGroupStockDTO{
		InstrumentKey: "TATASTEEL",
	}

	err := service.NotifyOnNew5MinCandles(ctx, stock, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("NotifyOnNew5MinCandles failed: %v", err)
	}

	if !listenerCalled {
		t.Error("Candle close listener was not called")
	}

	t.Log("Notification test completed successfully")
}

func TestCandleAggregationServiceV2_PerformanceComparison(t *testing.T) {
	// This test demonstrates the performance improvement potential
	// In a real scenario, you would compare with the old service

	mockCandleRepo := &MockCandleRepository{}
	mockCandle5MinRepo := &MockCandle5MinRepository{}

	service := NewCandleAggregationServiceV2(
		mockCandleRepo,
		mockCandle5MinRepo,
		&BatchFetchService{},
		&TradingCalendarService{},
		&UtilityService{},
	)

	// Generate larger dataset for performance testing
	now := time.Now()
	largeDataset := make([]domain.Candle, 1000) // 1000 1-minute candles
	for i := 0; i < 1000; i++ {
		largeDataset[i] = domain.Candle{
			InstrumentKey: "PERFORMANCE_TEST",
			Timestamp:     now.Add(time.Duration(i) * time.Minute),
			Open:          100.0 + float64(i%10),
			High:          105.0 + float64(i%10),
			Low:           95.0 + float64(i%10),
			Close:         102.0 + float64(i%10),
			Volume:        1000 + int64(i%100),
		}
	}

	mockCandleRepo.candles = largeDataset

	ctx := context.Background()
	startTime := time.Now()

	// Test processing large dataset
	result, err := service.Get5MinCandles(ctx, "PERFORMANCE_TEST", now, now.Add(1000*time.Minute))
	if err != nil {
		t.Fatalf("Performance test failed: %v", err)
	}

	processingTime := time.Since(startTime)

	// Should aggregate 1000 1-minute candles to 200 5-minute candles
	expectedCandles := 200
	if len(result) != expectedCandles {
		t.Errorf("Expected %d aggregated candles, got %d", expectedCandles, len(result))
	}

	// Get analytics metrics
	metrics := service.GetAnalyticsMetrics()

	t.Logf("Performance test completed:")
	t.Logf("- Processed %d 1-minute candles", len(largeDataset))
	t.Logf("- Generated %d 5-minute candles", len(result))
	t.Logf("- Processing time: %v", processingTime)
	t.Logf("- Analytics metrics: %+v", metrics)

	// Performance assertion - should be fast
	if processingTime > time.Second {
		t.Errorf("Processing took too long: %v (expected < 1 second)", processingTime)
	}
}
