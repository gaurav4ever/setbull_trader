package strategies

import (
	"testing"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

func TestBBWidthStrategyV2(t *testing.T) {
	// Create strategy
	strategy := NewBBWidthStrategyV2()

	// Create sample data
	closePrices := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110}
	timestamps := make([]string, len(closePrices))
	for i := range timestamps {
		timestamps[i] = time.Now().Add(time.Duration(i) * time.Minute).Format("2006-01-02 15:04:05")
	}

	// Create DataFrame
	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(closePrices, series.Float, "close"),
	)

	// Process data
	result, err := strategy.Process(&df)
	if err != nil {
		t.Fatalf("Strategy processing failed: %v", err)
	}

	// Verify result
	if result.Nrow() != len(closePrices) {
		t.Fatalf("Expected %d rows, got %d", len(closePrices), result.Nrow())
	}

	// Check if new columns were added
	columns := result.Names()
	expectedColumns := []string{"timestamp", "close", "bb_upper_v2", "bb_middle_v2", "bb_lower_v2", "bb_width_v2", "lowest_bb_width_v2", "distance_from_lowest_v2"}

	for _, expectedCol := range expectedColumns {
		found := false
		for _, col := range columns {
			if col == expectedCol {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected column '%s' not found in result", expectedCol)
		}
	}

	// Verify BB width values are calculated (should have NaN for first few values)
	bbWidthCol := result.Col("bb_width_v2")
	if bbWidthCol.Err != nil {
		t.Fatalf("Failed to get BB width column: %v", bbWidthCol.Err)
	}

	bbWidthValues := bbWidthCol.Float()
	if len(bbWidthValues) != len(closePrices) {
		t.Fatalf("Expected %d BB width values, got %d", len(closePrices), len(bbWidthValues))
	}

	// Check that first few values are NaN (not enough data for calculation)
	// With period = 20 and only 11 data points, all values should be NaN
	for i := 0; i < len(bbWidthValues); i++ {
		if !isNaN(bbWidthValues[i]) {
			t.Errorf("Expected NaN at index %d, got %f", i, bbWidthValues[i])
		}
	}
}

func TestStrategyConfiguration(t *testing.T) {
	// Create strategy
	strategy := NewBBWidthStrategyV2()

	// Test configuration
	config := map[string]interface{}{
		"enabled":     true,
		"timeout":     5 * time.Second,
		"max_retries": 3,
		"parameters": map[string]interface{}{
			"period":   15,
			"std_dev":  1.5,
			"lookback": 3,
		},
	}

	err := strategy.Configure(config)
	if err != nil {
		t.Fatalf("Configuration failed: %v", err)
	}

	// Verify configuration
	if !strategy.IsEnabled() {
		t.Error("Strategy should be enabled")
	}

	// Verify parameters were set (we can't directly access them, but we can test behavior)
	requiredHistory := strategy.GetRequiredHistory()
	expectedHistory := 15 + 3 // period + lookback
	if requiredHistory != expectedHistory {
		t.Errorf("Expected required history %d, got %d", expectedHistory, requiredHistory)
	}
}

// Helper function to check if a value is NaN
func isNaN(f float64) bool {
	return f != f
}
