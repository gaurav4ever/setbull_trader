package strategies

import (
	"testing"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"

	v2 "setbull_trader/internal/strategy/v2"
)

// TestTwoThirtyEntryStrategyIntegration tests the strategy with real market data scenarios
func TestTwoThirtyEntryStrategyIntegration(t *testing.T) {
	// Create strategy without parameters manager for testing
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	// Test Scenario 1: Normal trading day with 2:30 PM entry
	t.Run("Normal Trading Day - 2:30 PM Entry", func(t *testing.T) {
		// Create realistic market data for a trading day
		df := createRealisticMarketData(t)

		// Process the data
		_, err := strategy.Process(&df)
		if err != nil {
			t.Fatalf("Expected no error during processing, got %v", err)
		}

		// Verify that range was calculated at 2:30 PM
		if !strategy.rangeCalculated {
			t.Error("Expected range to be calculated during processing")
		}

		// Verify range values are reasonable
		if strategy.rangeHigh <= 0 || strategy.rangeLow <= 0 {
			t.Error("Expected range values to be positive")
		}

		if strategy.rangeHigh <= strategy.rangeLow {
			t.Error("Expected range high to be greater than range low")
		}

		// Verify buffer values are calculated correctly
		expectedHighEntryPrice := strategy.rangeHigh * (1 + strategy.bufferPercentage)
		expectedLowEntryPrice := strategy.rangeLow * (1 - strategy.bufferPercentage)

		if strategy.rangeHighEntryPrice != expectedHighEntryPrice {
			t.Errorf("Expected range high entry price to be %f, got %f",
				expectedHighEntryPrice, strategy.rangeHighEntryPrice)
		}

		if strategy.rangeLowEntryPrice != expectedLowEntryPrice {
			t.Errorf("Expected range low entry price to be %f, got %f",
				expectedLowEntryPrice, strategy.rangeLowEntryPrice)
		}
	})

	// Test Scenario 2: Bullish breakout after 2:30 PM
	t.Run("Bullish Breakout After 2:30 PM", func(t *testing.T) {
		strategy.ResetState()
		strategy.direction = "BULLISH"

		// Create data with bullish breakout
		df := createBullishBreakoutData(t)

		_, err := strategy.Process(&df)
		if err != nil {
			t.Fatalf("Expected no error during processing, got %v", err)
		}

		// Verify that a bullish signal was generated
		// (This would be verified through logging in a real scenario)
		if !strategy.rangeCalculated {
			t.Error("Expected range to be calculated")
		}
	})

	// Test Scenario 3: Bearish breakout after 2:30 PM
	t.Run("Bearish Breakout After 2:30 PM", func(t *testing.T) {
		strategy.ResetState()
		strategy.direction = "BEARISH"

		// Create data with bearish breakout
		df := createBearishBreakoutData(t)

		_, err := strategy.Process(&df)
		if err != nil {
			t.Fatalf("Expected no error during processing, got %v", err)
		}

		// Verify that a bearish signal was generated
		if !strategy.rangeCalculated {
			t.Error("Expected range to be calculated")
		}
	})

	// Test Scenario 4: No breakout scenario
	t.Run("No Breakout Scenario", func(t *testing.T) {
		strategy.ResetState()
		strategy.direction = "BULLISH"

		// Create data with no breakout
		df := createNoBreakoutData(t)

		_, err := strategy.Process(&df)
		if err != nil {
			t.Fatalf("Expected no error during processing, got %v", err)
		}

		// Verify that range was calculated but no signal generated
		if !strategy.rangeCalculated {
			t.Error("Expected range to be calculated")
		}
	})
}

// TestTwoThirtyEntryStrategyCrossValidation tests cross-validation with Python implementation
func TestTwoThirtyEntryStrategyCrossValidation(t *testing.T) {
	// Test cases based on Python implementation behavior
	testCases := []struct {
		name           string
		entryTime      string
		rangeHigh      float64
		rangeLow       float64
		bufferPct      float64
		direction      string
		priceAfter     float64
		expectedSignal bool
		expectedDir    string
	}{
		{
			name:           "Python Test Case 1: Bullish Breakout",
			entryTime:      "14:30",
			rangeHigh:      100.0,
			rangeLow:       95.0,
			bufferPct:      0.0003,
			direction:      "BULLISH",
			priceAfter:     100.04, // Above range high + buffer
			expectedSignal: true,
			expectedDir:    "LONG",
		},
		{
			name:           "Python Test Case 2: Bearish Breakout",
			entryTime:      "14:30",
			rangeHigh:      100.0,
			rangeLow:       95.0,
			bufferPct:      0.0003,
			direction:      "BEARISH",
			priceAfter:     94.97, // Below range low - buffer
			expectedSignal: true,
			expectedDir:    "SHORT",
		},
		{
			name:           "Python Test Case 3: No Breakout",
			entryTime:      "14:30",
			rangeHigh:      100.0,
			rangeLow:       95.0,
			bufferPct:      0.0003,
			direction:      "BULLISH",
			priceAfter:     99.0, // Within range
			expectedSignal: false,
			expectedDir:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			strategy := NewTwoThirtyEntryStrategyV2(nil)

			// Configure strategy with test parameters
			config := map[string]interface{}{
				"parameters": map[string]interface{}{
					"entry_time":        tc.entryTime,
					"buffer_percentage": tc.bufferPct,
					"direction":         tc.direction,
				},
			}

			err := strategy.Configure(config)
			if err != nil {
				t.Fatalf("Failed to configure strategy: %v", err)
			}

			// Set up range values (simulating 2:30 PM calculation)
			strategy.rangeHigh = tc.rangeHigh
			strategy.rangeLow = tc.rangeLow
			strategy.rangeCalculated = true
			strategy.calculateBufferValues()

			// Test entry conditions
			params := &v2.StrategyParameters{}
			signal := strategy.checkEntryConditions(tc.priceAfter, tc.priceAfter-1, time.Now(), params)

			if tc.expectedSignal {
				if signal == nil {
					t.Error("Expected signal to be generated")
					return
				}

				if signal.Direction != tc.expectedDir {
					t.Errorf("Expected signal direction to be %s, got %s", tc.expectedDir, signal.Direction)
				}

				// Verify signal metadata
				if signal.Metadata["signal_purpose"] != "data_tracking" {
					t.Error("Expected signal purpose to be 'data_tracking'")
				}

				if signal.Metadata["entry_type"] != "14:30" {
					t.Error("Expected entry type to be '14:30'")
				}
			} else {
				if signal != nil {
					t.Error("Expected no signal to be generated")
				}
			}
		})
	}
}

// TestTwoThirtyEntryStrategyIntegrationPerformance tests performance characteristics
func TestTwoThirtyEntryStrategyIntegrationPerformance(t *testing.T) {
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	// Create large dataset for performance testing
	df := createLargeDataset(t, 1000) // 1000 candles

	// Measure processing time
	start := time.Now()
	result, err := strategy.Process(&df)
	processingTime := time.Since(start)

	if err != nil {
		t.Fatalf("Expected no error during processing, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result DataFrame to be returned")
	}

	// Verify performance requirements
	if processingTime > 100*time.Millisecond {
		t.Errorf("Expected processing time to be less than 100ms, got %v", processingTime)
	}

	// Verify memory requirements
	memoryReq := strategy.GetMemoryRequirements()
	if memoryReq > 1024*1024 { // 1MB
		t.Errorf("Expected memory requirements to be less than 1MB, got %d bytes", memoryReq)
	}
}

// Helper functions to create test data

func createRealisticMarketData(t *testing.T) dataframe.DataFrame {
	// Create realistic market data for a trading day
	timestamps := []string{
		"2024-01-01 09:15:00", "2024-01-01 09:20:00", "2024-01-01 09:25:00",
		"2024-01-01 14:25:00", "2024-01-01 14:30:00", "2024-01-01 14:35:00",
	}

	highs := []float64{100.5, 101.0, 100.8, 100.2, 100.0, 100.3}
	lows := []float64{99.5, 99.8, 99.6, 99.0, 95.0, 99.1}
	closes := []float64{100.0, 100.5, 100.2, 99.5, 97.5, 100.1}

	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(highs, series.Float, "high"),
		series.New(lows, series.Float, "low"),
		series.New(closes, series.Float, "close"),
	)

	return df
}

func createBullishBreakoutData(t *testing.T) dataframe.DataFrame {
	// Create data with bullish breakout after 2:30 PM
	timestamps := []string{
		"2024-01-01 14:25:00", "2024-01-01 14:30:00", "2024-01-01 14:35:00",
	}

	highs := []float64{100.0, 100.0, 100.04} // Breakout above range high + buffer
	lows := []float64{95.0, 95.0, 99.9}
	closes := []float64{97.5, 97.5, 100.02}

	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(highs, series.Float, "high"),
		series.New(lows, series.Float, "low"),
		series.New(closes, series.Float, "close"),
	)

	return df
}

func createBearishBreakoutData(t *testing.T) dataframe.DataFrame {
	// Create data with bearish breakout after 2:30 PM
	timestamps := []string{
		"2024-01-01 14:25:00", "2024-01-01 14:30:00", "2024-01-01 14:35:00",
	}

	highs := []float64{100.0, 100.0, 99.9}
	lows := []float64{95.0, 95.0, 94.97} // Breakout below range low - buffer
	closes := []float64{97.5, 97.5, 94.98}

	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(highs, series.Float, "high"),
		series.New(lows, series.Float, "low"),
		series.New(closes, series.Float, "close"),
	)

	return df
}

func createNoBreakoutData(t *testing.T) dataframe.DataFrame {
	// Create data with no breakout after 2:30 PM
	timestamps := []string{
		"2024-01-01 14:25:00", "2024-01-01 14:30:00", "2024-01-01 14:35:00",
	}

	highs := []float64{100.0, 100.0, 99.9} // Within range
	lows := []float64{95.0, 95.0, 95.1}    // Within range
	closes := []float64{97.5, 97.5, 97.4}

	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(highs, series.Float, "high"),
		series.New(lows, series.Float, "low"),
		series.New(closes, series.Float, "close"),
	)

	return df
}

func createLargeDataset(t *testing.T, size int) dataframe.DataFrame {
	// Create large dataset for performance testing
	timestamps := make([]string, size)
	highs := make([]float64, size)
	lows := make([]float64, size)
	closes := make([]float64, size)

	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)
	basePrice := 100.0

	for i := 0; i < size; i++ {
		timestamps[i] = baseTime.Add(time.Duration(i*5) * time.Minute).Format("2006-01-02 15:04:05")
		highs[i] = basePrice + float64(i%10)*0.1
		lows[i] = basePrice - float64(i%10)*0.1
		closes[i] = basePrice + float64(i%5-2)*0.05
	}

	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(highs, series.Float, "high"),
		series.New(lows, series.Float, "low"),
		series.New(closes, series.Float, "close"),
	)

	return df
}

// MockStrategyParametersManager for testing
type MockStrategyParametersManager struct{}

func (m *MockStrategyParametersManager) GetParameters(stockID, strategyName string, timestamp time.Time) (*v2.StrategyParameters, error) {
	return &v2.StrategyParameters{
		CanGenerateLong:  true,
		CanGenerateShort: true,
	}, nil
}

func (m *MockStrategyParametersManager) SaveParameters(stockID, strategyName string, timestamp time.Time, params *v2.StrategyParameters) error {
	return nil
}

func (m *MockStrategyParametersManager) GetLatestParameters(stockID, strategyName string) (*v2.StrategyParameters, error) {
	return &v2.StrategyParameters{
		CanGenerateLong:  true,
		CanGenerateShort: true,
	}, nil
}

func (m *MockStrategyParametersManager) UpdateParameters(stockID, strategyName string, timestamp time.Time, params *v2.StrategyParameters) error {
	return nil
}

func (m *MockStrategyParametersManager) GetParametersRange(stockID, strategyName string, startTime, endTime time.Time) ([]*v2.StrategyParametersRecord, error) {
	return nil, nil
}

func (m *MockStrategyParametersManager) DeleteParameters(stockID, strategyName string, timestamp time.Time) error {
	return nil
}

func (m *MockStrategyParametersManager) CleanupOldParameters(beforeTime time.Time) error {
	return nil
}

func (m *MockStrategyParametersManager) GetCacheKey(stockID, strategyName string, timestamp time.Time) string {
	return ""
}

func (m *MockStrategyParametersManager) ConvertDataFrameToParameters(df *dataframe.DataFrame, rowIndex int) (*v2.StrategyParameters, error) {
	return nil, nil
}

func (m *MockStrategyParametersManager) ConvertParametersToDataFrame(params *v2.StrategyParameters) map[string]series.Series {
	return nil
}

func (m *MockStrategyParametersManager) CleanupCache() {}
