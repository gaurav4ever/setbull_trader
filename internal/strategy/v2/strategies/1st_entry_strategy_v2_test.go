package strategies

import (
	"testing"
	"time"

	v2 "setbull_trader/internal/strategy/v2"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

// TestFirstEntryStrategyV2Simple tests the 1ST_ENTRY strategy
func TestFirstEntryStrategyV2Simple(t *testing.T) {
	// Create strategy with nil params manager for basic testing
	strategy := &FirstEntryStrategyV2Simple{
		BaseStrategy:     v2.NewBaseStrategy(v2.StrategyMetadata{Name: "1ST_ENTRY_V2", Version: "1.0.0"}),
		bufferPercentage: 0.0007,
		marketOpen:       time.Date(2000, 1, 1, 9, 15, 0, 0, time.UTC),
		morningRangeEnd:  time.Date(2000, 1, 1, 9, 20, 0, 0, time.UTC),
		canGenerateLong:  true,
		canGenerateShort: true,
		mrCalculated:     false,
		paramsManager:    nil, // Will skip parameter persistence in tests
	}

	// Test strategy metadata
	if strategy.GetName() != "1ST_ENTRY_V2" {
		t.Errorf("Expected strategy name '1ST_ENTRY_V2', got '%s'", strategy.GetName())
	}

	if strategy.GetVersion() != "1.0.0" {
		t.Errorf("Expected strategy version '1.0.0', got '%s'", strategy.GetVersion())
	}

	// Test configuration
	err := strategy.Configure(map[string]interface{}{
		"enabled": true,
		"parameters": map[string]interface{}{
			"buffer_percentage": 0.001, // 0.1%
			"market_open":       "09:15",
			"morning_range_end": "09:20",
		},
	})

	if err != nil {
		t.Errorf("Failed to configure strategy: %v", err)
	}

	// Test validation
	err = strategy.ValidateConfiguration()
	if err != nil {
		t.Errorf("Strategy configuration validation failed: %v", err)
	}
}

// TestFirstEntryStrategyMRCalculation tests the morning range calculation logic
func TestFirstEntryStrategyMRCalculation(t *testing.T) {
	strategy := &FirstEntryStrategyV2Simple{
		BaseStrategy:     v2.NewBaseStrategy(v2.StrategyMetadata{Name: "1ST_ENTRY_V2", Version: "1.0.0"}),
		bufferPercentage: 0.0007,
		marketOpen:       time.Date(2000, 1, 1, 9, 15, 0, 0, time.UTC),
		morningRangeEnd:  time.Date(2000, 1, 1, 9, 20, 0, 0, time.UTC),
		paramsManager:    nil,
	}

	// Test MR calculation period detection
	candleTime1 := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC) // 9:15 AM
	candleTime2 := time.Date(2024, 1, 1, 9, 17, 0, 0, time.UTC) // 9:17 AM
	candleTime3 := time.Date(2024, 1, 1, 9, 20, 0, 0, time.UTC) // 9:20 AM
	candleTime4 := time.Date(2024, 1, 1, 9, 25, 0, 0, time.UTC) // 9:25 AM

	// Test MR calculation period
	if !strategy.isMorningRangeCalculationPeriod(candleTime1) {
		t.Error("Expected 9:15 AM to be in MR calculation period")
	}
	if !strategy.isMorningRangeCalculationPeriod(candleTime2) {
		t.Error("Expected 9:17 AM to be in MR calculation period")
	}
	if strategy.isMorningRangeCalculationPeriod(candleTime3) {
		t.Error("Expected 9:20 AM to NOT be in MR calculation period")
	}
	if strategy.isMorningRangeCalculationPeriod(candleTime4) {
		t.Error("Expected 9:25 AM to NOT be in MR calculation period")
	}

	// Test after MR calculation period
	if strategy.isAfterMorningRangeCalculation(candleTime1) {
		t.Error("Expected 9:15 AM to NOT be after MR calculation")
	}
	if strategy.isAfterMorningRangeCalculation(candleTime2) {
		t.Error("Expected 9:17 AM to NOT be after MR calculation")
	}
	if !strategy.isAfterMorningRangeCalculation(candleTime3) {
		t.Error("Expected 9:20 AM to be after MR calculation")
	}
	if !strategy.isAfterMorningRangeCalculation(candleTime4) {
		t.Error("Expected 9:25 AM to be after MR calculation")
	}
}

// TestFirstEntryStrategyBufferCalculation tests the buffer calculation logic
func TestFirstEntryStrategyBufferCalculation(t *testing.T) {
	strategy := &FirstEntryStrategyV2Simple{
		BaseStrategy:     v2.NewBaseStrategy(v2.StrategyMetadata{Name: "1ST_ENTRY_V2", Version: "1.0.0"}),
		bufferPercentage: 0.0007,
		paramsManager:    nil,
	}

	// Set morning range values
	strategy.mrHigh = 100.0
	strategy.mrLow = 98.0
	strategy.calculateBufferValues()

	// Verify buffer calculations
	expectedHighWithBuffer := 100.07 // 100.0 * (1 + 0.0007) = 100.07
	expectedLowWithBuffer := 97.93   // 98.0 * (1 - 0.0007) = 97.93

	if strategy.mrHighWithBuffer != expectedHighWithBuffer {
		t.Errorf("Expected MR high with buffer %.2f, got %.2f", expectedHighWithBuffer, strategy.mrHighWithBuffer)
	}

	if strategy.mrLowWithBuffer != expectedLowWithBuffer {
		t.Errorf("Expected MR low with buffer %.2f, got %.2f", expectedLowWithBuffer, strategy.mrLowWithBuffer)
	}
}

// TestFirstEntryStrategyEntryConditions tests the entry condition logic
func TestFirstEntryStrategyEntryConditions(t *testing.T) {
	strategy := &FirstEntryStrategyV2Simple{
		BaseStrategy:     v2.NewBaseStrategy(v2.StrategyMetadata{Name: "1ST_ENTRY_V2", Version: "1.0.0"}),
		bufferPercentage: 0.0007,
		paramsManager:    nil,
	}

	// Set morning range values and mark as calculated
	strategy.mrHigh = 100.0
	strategy.mrLow = 98.0
	strategy.mrCalculated = true
	strategy.calculateBufferValues()

	// Test long breakout condition
	params := &v2.StrategyParameters{
		CanGenerateLong:  true,
		CanGenerateShort: true,
		StrategyState:    make(map[string]interface{}),
		Metadata:         make(map[string]interface{}),
	}

	timestamp := time.Date(2024, 1, 1, 9, 25, 0, 0, time.UTC)

	// Test long breakout (high >= mr_high_with_buffer)
	signal := strategy.checkEntryConditions(100.08, 99.8, timestamp, params)
	if signal == nil {
		t.Error("Expected long breakout signal, got nil")
	} else {
		if signal.Direction != "LONG" {
			t.Errorf("Expected LONG direction, got %s", signal.Direction)
		}
		if signal.Type != "IMMEDIATE_BREAKOUT" {
			t.Errorf("Expected IMMEDIATE_BREAKOUT type, got %s", signal.Type)
		}
		// Check metadata
		if mrHigh, ok := signal.Metadata["mr_high"].(float64); !ok || mrHigh != 100.0 {
			t.Errorf("Expected mr_high 100.0 in metadata, got %v", signal.Metadata["mr_high"])
		}
		// Check signal purpose
		if purpose, ok := signal.Metadata["signal_purpose"].(string); !ok || purpose != "data_tracking" {
			t.Errorf("Expected signal_purpose 'data_tracking' in metadata, got %v", signal.Metadata["signal_purpose"])
		}
	}

	// Test short breakout condition
	strategy.ResetState()
	strategy.mrHigh = 100.0
	strategy.mrLow = 98.0
	strategy.mrCalculated = true
	strategy.calculateBufferValues()

	params = &v2.StrategyParameters{
		CanGenerateLong:  true,
		CanGenerateShort: true,
		StrategyState:    make(map[string]interface{}),
		Metadata:         make(map[string]interface{}),
	}

	// Test short breakout (low <= mr_low_with_buffer)
	signal = strategy.checkEntryConditions(98.1, 97.9, timestamp, params)
	if signal == nil {
		t.Error("Expected short breakout signal, got nil")
	} else {
		if signal.Direction != "SHORT" {
			t.Errorf("Expected SHORT direction, got %s", signal.Direction)
		}
		if signal.Type != "IMMEDIATE_BREAKOUT" {
			t.Errorf("Expected IMMEDIATE_BREAKOUT type, got %s", signal.Type)
		}
		// Check signal purpose
		if purpose, ok := signal.Metadata["signal_purpose"].(string); !ok || purpose != "data_tracking" {
			t.Errorf("Expected signal_purpose 'data_tracking' in metadata, got %v", signal.Metadata["signal_purpose"])
		}
	}
}

// TestFirstEntryStrategyProcessing tests the strategy processing logic
func TestFirstEntryStrategyProcessing(t *testing.T) {
	strategy := &FirstEntryStrategyV2Simple{
		BaseStrategy:     v2.NewBaseStrategy(v2.StrategyMetadata{Name: "1ST_ENTRY_V2", Version: "1.0.0"}),
		bufferPercentage: 0.0007,
		paramsManager:    nil,
	}

	// Create test DataFrame with MR calculation and breakout candles
	timestamps := []string{
		"2024-01-01 09:15:00", // MR calculation start
		"2024-01-01 09:17:00", // MR calculation continue
		"2024-01-01 09:20:00", // MR calculation end, start checking breakouts
		"2024-01-01 09:25:00", // Long breakout
		"2024-01-01 09:30:00", // Short breakout
	}

	highs := []float64{100.5, 100.8, 100.2, 100.08, 97.9} // 100.08 should trigger long breakout
	lows := []float64{99.5, 99.8, 99.9, 99.8, 97.8}       // 97.8 should trigger short breakout

	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(highs, series.Float, "high"),
		series.New(lows, series.Float, "low"),
		series.New([]float64{100.0, 100.0, 100.0, 100.0, 97.85}, series.Float, "close"),
		series.New([]int{1000, 1000, 1000, 1000, 1000}, series.Int, "volume"),
	)

	// Process DataFrame
	result, err := strategy.Process(&df)
	if err != nil {
		t.Errorf("Strategy processing failed: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result DataFrame")
	}

	// Verify that MR was calculated correctly
	// MR should be: High=100.8, Low=99.5 (from 9:15-9:20 AM candles)
	if strategy.mrHigh != 100.8 {
		t.Errorf("Expected MR high 100.8, got %.2f", strategy.mrHigh)
	}
	if strategy.mrLow != 99.5 {
		t.Errorf("Expected MR low 99.5, got %.2f", strategy.mrLow)
	}
	if !strategy.mrCalculated {
		t.Error("Expected MR to be calculated")
	}
}

// TestFirstEntryStrategyPerformance tests performance characteristics
func TestFirstEntryStrategyPerformance(t *testing.T) {
	strategy := &FirstEntryStrategyV2Simple{
		BaseStrategy:     v2.NewBaseStrategy(v2.StrategyMetadata{Name: "1ST_ENTRY_V2", Version: "1.0.0"}),
		bufferPercentage: 0.0007,
		paramsManager:    nil,
	}

	// Test processing time
	processingTime := strategy.GetEstimatedProcessingTime()
	if processingTime > 100*time.Millisecond {
		t.Errorf("Expected processing time < 100ms, got %v", processingTime)
	}

	// Test memory requirements
	memoryReq := strategy.GetMemoryRequirements()
	if memoryReq > 1024*1024 { // 1MB
		t.Errorf("Expected memory requirements < 1MB, got %d bytes", memoryReq)
	}

	// Test required history
	requiredHistory := strategy.GetRequiredHistory()
	if requiredHistory != 5 {
		t.Errorf("Expected required history 5, got %d", requiredHistory)
	}
}

// TestFirstEntryStrategyConfiguration tests configuration validation
func TestFirstEntryStrategyConfiguration(t *testing.T) {
	strategy := &FirstEntryStrategyV2Simple{
		BaseStrategy:     v2.NewBaseStrategy(v2.StrategyMetadata{Name: "1ST_ENTRY_V2", Version: "1.0.0"}),
		bufferPercentage: 0.0007,
		paramsManager:    nil,
	}

	// Test valid configuration
	err := strategy.Configure(map[string]interface{}{
		"enabled": true,
		"parameters": map[string]interface{}{
			"buffer_percentage": 0.001,
			"market_open":       "09:15",
			"morning_range_end": "09:20",
		},
	})
	if err != nil {
		t.Errorf("Valid configuration failed: %v", err)
	}

	// Test invalid buffer percentage
	err = strategy.Configure(map[string]interface{}{
		"enabled": true,
		"parameters": map[string]interface{}{
			"buffer_percentage": -0.001, // Invalid negative value
			"market_open":       "09:15",
			"morning_range_end": "09:20",
		},
	})
	if err == nil {
		t.Error("Expected error for invalid buffer percentage")
	}

	// Test too high buffer percentage
	err = strategy.Configure(map[string]interface{}{
		"enabled": true,
		"parameters": map[string]interface{}{
			"buffer_percentage": 0.2, // Too high
			"market_open":       "09:15",
			"morning_range_end": "09:20",
		},
	})
	if err == nil {
		t.Error("Expected error for too high buffer percentage")
	}
}
