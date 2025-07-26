package analytics

import (
	"context"
	"testing"
	"time"

	"setbull_trader/internal/analytics/dataframe"
	"setbull_trader/internal/domain"
)

func TestAnalyticsProcessor_BasicFunctionality(t *testing.T) {
	// Create test processor
	config := DefaultAnalyticsConfig()
	processor := NewProcessor(config)

	// Create test candles
	now := time.Now()
	candles := []domain.Candle{
		{
			InstrumentKey: "RELIANCE",
			Timestamp:     now,
			Open:          100.0,
			High:          105.0,
			Low:           98.0,
			Close:         102.0,
			Volume:        1000,
		},
		{
			InstrumentKey: "RELIANCE",
			Timestamp:     now.Add(time.Minute),
			Open:          102.0,
			High:          107.0,
			Low:           100.0,
			Close:         105.0,
			Volume:        1500,
		},
	}

	// Test ProcessCandles
	result, err := processor.ProcessCandles(context.Background(), candles)
	if err != nil {
		t.Fatalf("ProcessCandles failed: %v", err)
	}

	if result == nil {
		t.Fatal("ProcessCandles returned nil result")
	}

	if result.DataFrame.Nrow() != 2 {
		t.Errorf("Expected 2 rows in DataFrame, got %d", result.DataFrame.Nrow())
	}

	// Test empty candles
	emptyResult, err := processor.ProcessCandles(context.Background(), []domain.Candle{})
	if err != nil {
		t.Fatalf("ProcessCandles with empty candles failed: %v", err)
	}

	if emptyResult.DataFrame.Nrow() != 0 {
		t.Errorf("Expected 0 rows for empty candles, got %d", emptyResult.DataFrame.Nrow())
	}

	// Test AggregateTimeframes
	candleData := &CandleData{Candles: candles}
	aggregated, err := processor.AggregateTimeframes(context.Background(), candleData, "5m")
	if err != nil {
		t.Fatalf("AggregateTimeframes failed: %v", err)
	}

	if aggregated == nil {
		t.Fatal("AggregateTimeframes returned nil")
	}

	// Test metrics
	metrics := processor.GetMetrics()
	if metrics.TotalProcessed == 0 {
		t.Error("Expected non-zero TotalProcessed metric")
	}

	t.Logf("Test completed successfully. Processed %d operations", metrics.TotalProcessed)
}

func TestCandleDataFrame_BasicOperations(t *testing.T) {
	// Create test candles
	candles := []domain.Candle{
		{
			InstrumentKey: "RELIANCE",
			Timestamp:     time.Now(),
			Open:          100.0,
			High:          105.0,
			Low:           98.0,
			Close:         102.0,
			Volume:        1000,
		},
	}

	// Create DataFrame
	df := dataframe.NewCandleDataFrame(candles, false)
	if df.Empty() {
		t.Fatal("DataFrame should not be empty")
	}

	// Test conversion back to candles
	convertedCandles := df.ToCandles()
	if len(convertedCandles) != 1 {
		t.Errorf("Expected 1 candle, got %d", len(convertedCandles))
	}

	if convertedCandles[0].InstrumentKey != "RELIANCE" {
		t.Errorf("Expected RELIANCE, got %s", convertedCandles[0].InstrumentKey)
	}

	// Test aggregated candles conversion
	aggregatedCandles := df.ToAggregatedCandles()
	if len(aggregatedCandles) != 1 {
		t.Errorf("Expected 1 aggregated candle, got %d", len(aggregatedCandles))
	}

	if aggregatedCandles[0].InstrumentKey != "RELIANCE" {
		t.Errorf("Expected RELIANCE, got %s", aggregatedCandles[0].InstrumentKey)
	}

	t.Log("CandleDataFrame operations test completed successfully")
}
