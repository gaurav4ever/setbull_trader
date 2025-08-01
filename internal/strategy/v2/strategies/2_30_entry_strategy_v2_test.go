package strategies

import (
	"testing"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"

	v2 "setbull_trader/internal/strategy/v2"
)

func TestNewTwoThirtyEntryStrategyV2(t *testing.T) {
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	if strategy == nil {
		t.Fatal("Expected strategy to be created")
	}

	if strategy.GetName() != "2_30_ENTRY" {
		t.Errorf("Expected strategy name to be '2_30_ENTRY', got '%s'", strategy.GetName())
	}

	if strategy.GetRequiredHistory() != 1 {
		t.Errorf("Expected required history to be 1, got %d", strategy.GetRequiredHistory())
	}

	// Check default parameters
	if strategy.direction != "BULLISH" {
		t.Errorf("Expected default direction to be 'BULLISH', got '%s'", strategy.direction)
	}

	if strategy.bufferPercentage != 0.0003 {
		t.Errorf("Expected default buffer percentage to be 0.0003, got %f", strategy.bufferPercentage)
	}

	if strategy.minPriceMovement != 0.001 {
		t.Errorf("Expected default min price movement to be 0.001, got %f", strategy.minPriceMovement)
	}
}

func TestTwoThirtyEntryStrategyTimeDetection(t *testing.T) {
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	// Test entry time detection (2:30 PM)
	entryTime := time.Date(2024, 1, 1, 14, 30, 0, 0, time.UTC)
	if !strategy.isEntryTime(entryTime) {
		t.Error("Expected 2:30 PM to be detected as entry time")
	}

	// Test non-entry time
	nonEntryTime := time.Date(2024, 1, 1, 14, 29, 0, 0, time.UTC)
	if strategy.isEntryTime(nonEntryTime) {
		t.Error("Expected 2:29 PM to not be detected as entry time")
	}

	// Test after entry time detection
	afterEntryTime := time.Date(2024, 1, 1, 14, 31, 0, 0, time.UTC)
	if !strategy.isAfterEntryTime(afterEntryTime) {
		t.Error("Expected 2:31 PM to be detected as after entry time")
	}

	// Test before entry time
	beforeEntryTime := time.Date(2024, 1, 1, 14, 29, 0, 0, time.UTC)
	if strategy.isAfterEntryTime(beforeEntryTime) {
		t.Error("Expected 2:29 PM to not be detected as after entry time")
	}
}

func TestTwoThirtyEntryStrategyRangeCalculation(t *testing.T) {
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	// Test range calculation
	high := 100.0
	low := 95.0
	timestamp := time.Date(2024, 1, 1, 14, 30, 0, 0, time.UTC)
	params := &v2.StrategyParameters{}

	strategy.calculateEntryRange(high, low, timestamp, params)

	if !strategy.rangeCalculated {
		t.Error("Expected range to be calculated")
	}

	if strategy.rangeHigh != high {
		t.Errorf("Expected range high to be %f, got %f", high, strategy.rangeHigh)
	}

	if strategy.rangeLow != low {
		t.Errorf("Expected range low to be %f, got %f", low, strategy.rangeLow)
	}

	// Test buffer calculation
	expectedHighEntryPrice := high * (1 + strategy.bufferPercentage)
	expectedLowEntryPrice := low * (1 - strategy.bufferPercentage)

	if strategy.rangeHighEntryPrice != expectedHighEntryPrice {
		t.Errorf("Expected range high entry price to be %f, got %f", expectedHighEntryPrice, strategy.rangeHighEntryPrice)
	}

	if strategy.rangeLowEntryPrice != expectedLowEntryPrice {
		t.Errorf("Expected range low entry price to be %f, got %f", expectedLowEntryPrice, strategy.rangeLowEntryPrice)
	}
}

func TestTwoThirtyEntryStrategyDirectionBias(t *testing.T) {
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	// Set up range values
	strategy.rangeHigh = 100.0
	strategy.rangeLow = 95.0
	strategy.rangeCalculated = true
	strategy.calculateBufferValues()

	// Test bullish entry
	strategy.direction = "BULLISH"
	strategy.canGenerateLong = true

	// Price above range high entry price should generate long signal
	high := 101.0
	low := 99.0
	timestamp := time.Date(2024, 1, 1, 14, 31, 0, 0, time.UTC)
	params := &v2.StrategyParameters{}

	signal := strategy.checkEntryConditions(high, low, timestamp, params)

	if signal == nil {
		t.Error("Expected bullish signal to be generated")
	}

	if signal.Direction != "LONG" {
		t.Errorf("Expected signal direction to be 'LONG', got '%s'", signal.Direction)
	}

	if signal.Price != strategy.rangeHighEntryPrice {
		t.Errorf("Expected signal price to be %f, got %f", strategy.rangeHighEntryPrice, signal.Price)
	}

	// Check signal metadata
	if signal.Metadata["signal_purpose"] != "data_tracking" {
		t.Error("Expected signal purpose to be 'data_tracking'")
	}

	// Test bearish entry
	strategy.ResetState()
	strategy.rangeHigh = 100.0
	strategy.rangeLow = 95.0
	strategy.rangeCalculated = true
	strategy.calculateBufferValues()
	strategy.direction = "BEARISH"
	strategy.canGenerateShort = true

	// Price below range low entry price should generate short signal
	high = 96.0
	low = 94.0

	signal = strategy.checkEntryConditions(high, low, timestamp, params)

	if signal == nil {
		t.Error("Expected bearish signal to be generated")
	}

	if signal.Direction != "SHORT" {
		t.Errorf("Expected signal direction to be 'SHORT', got '%s'", signal.Direction)
	}

	if signal.Price != strategy.rangeLowEntryPrice {
		t.Errorf("Expected signal price to be %f, got %f", strategy.rangeLowEntryPrice, signal.Price)
	}
}

func TestTwoThirtyEntryStrategyBufferCalculation(t *testing.T) {
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	// Set up range values
	strategy.rangeHigh = 100.0
	strategy.rangeLow = 95.0

	strategy.calculateBufferValues()

	// Test buffer calculation (0.03% buffer)
	expectedHighEntryPrice := 100.0 * (1 + 0.0003) // 100.03
	expectedLowEntryPrice := 95.0 * (1 - 0.0003)   // 94.9715

	if strategy.rangeHighEntryPrice != expectedHighEntryPrice {
		t.Errorf("Expected range high entry price to be %f, got %f", expectedHighEntryPrice, strategy.rangeHighEntryPrice)
	}

	if strategy.rangeLowEntryPrice != expectedLowEntryPrice {
		t.Errorf("Expected range low entry price to be %f, got %f", expectedLowEntryPrice, strategy.rangeLowEntryPrice)
	}
}

func TestTwoThirtyEntryStrategyProcessing(t *testing.T) {
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	// Create test DataFrame
	df := dataframe.New(
		series.New([]string{"2024-01-01 14:29:00", "2024-01-01 14:30:00", "2024-01-01 14:31:00"}, series.String, "timestamp"),
		series.New([]float64{99.0, 100.0, 101.0}, series.Float, "high"),
		series.New([]float64{94.0, 95.0, 96.0}, series.Float, "low"),
		series.New([]float64{96.5, 97.5, 98.5}, series.Float, "close"),
	)

	// Process DataFrame
	result, err := strategy.Process(&df)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result DataFrame to be returned")
	}

	// Check that range was calculated at 2:30 PM
	if !strategy.rangeCalculated {
		t.Error("Expected range to be calculated during processing")
	}

	// Check that range values are set
	if strategy.rangeHigh != 100.0 {
		t.Errorf("Expected range high to be 100.0, got %f", strategy.rangeHigh)
	}

	if strategy.rangeLow != 95.0 {
		t.Errorf("Expected range low to be 95.0, got %f", strategy.rangeLow)
	}
}

func TestTwoThirtyEntryStrategyConfiguration(t *testing.T) {
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	// Test configuration
	config := map[string]interface{}{
		"parameters": map[string]interface{}{
			"entry_time":         "15:00",
			"buffer_percentage":  0.0005,
			"direction":          "BEARISH",
			"min_price_movement": 0.002,
		},
	}

	err := strategy.Configure(config)
	if err != nil {
		t.Fatalf("Expected no error during configuration, got %v", err)
	}

	// Check that parameters were updated
	expectedEntryTime := time.Date(2000, 1, 1, 15, 0, 0, 0, time.UTC)
	if !strategy.entryTime.Equal(expectedEntryTime) {
		t.Errorf("Expected entry time to be %v, got %v", expectedEntryTime, strategy.entryTime)
	}

	if strategy.bufferPercentage != 0.0005 {
		t.Errorf("Expected buffer percentage to be 0.0005, got %f", strategy.bufferPercentage)
	}

	if strategy.direction != "BEARISH" {
		t.Errorf("Expected direction to be 'BEARISH', got '%s'", strategy.direction)
	}

	if strategy.minPriceMovement != 0.002 {
		t.Errorf("Expected min price movement to be 0.002, got %f", strategy.minPriceMovement)
	}
}

func TestTwoThirtyEntryStrategyValidation(t *testing.T) {
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	// Test valid configuration
	err := strategy.ValidateConfiguration()
	if err != nil {
		t.Errorf("Expected no validation error, got %v", err)
	}

	// Test invalid buffer percentage
	strategy.bufferPercentage = 0.2 // Too high
	err = strategy.ValidateConfiguration()
	if err == nil {
		t.Error("Expected validation error for high buffer percentage")
	}

	// Test invalid direction
	strategy.bufferPercentage = 0.0003 // Reset to valid
	strategy.direction = "INVALID"
	err = strategy.ValidateConfiguration()
	if err == nil {
		t.Error("Expected validation error for invalid direction")
	}

	// Test invalid min price movement
	strategy.direction = "BULLISH"  // Reset to valid
	strategy.minPriceMovement = 0.2 // Too high
	err = strategy.ValidateConfiguration()
	if err == nil {
		t.Error("Expected validation error for high min price movement")
	}
}

func TestTwoThirtyEntryStrategyResetState(t *testing.T) {
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	// Set some state
	strategy.canGenerateLong = false
	strategy.canGenerateShort = false
	strategy.rangeCalculated = true
	strategy.rangeHigh = 100.0
	strategy.rangeLow = 95.0
	strategy.rangeHighEntryPrice = 100.03
	strategy.rangeLowEntryPrice = 94.97

	// Reset state
	strategy.ResetState()

	// Check that state was reset
	if !strategy.canGenerateLong {
		t.Error("Expected canGenerateLong to be reset to true")
	}

	if !strategy.canGenerateShort {
		t.Error("Expected canGenerateShort to be reset to true")
	}

	if strategy.rangeCalculated {
		t.Error("Expected rangeCalculated to be reset to false")
	}

	if strategy.rangeHigh != 0 {
		t.Error("Expected rangeHigh to be reset to 0")
	}

	if strategy.rangeLow != 0 {
		t.Error("Expected rangeLow to be reset to 0")
	}

	if strategy.rangeHighEntryPrice != 0 {
		t.Error("Expected rangeHighEntryPrice to be reset to 0")
	}

	if strategy.rangeLowEntryPrice != 0 {
		t.Error("Expected rangeLowEntryPrice to be reset to 0")
	}
}

func TestTwoThirtyEntryStrategyPerformance(t *testing.T) {
	strategy := NewTwoThirtyEntryStrategyV2(nil)

	// Test processing time
	processingTime := strategy.GetEstimatedProcessingTime()
	if processingTime > 100*time.Millisecond {
		t.Errorf("Expected processing time to be less than 100ms, got %v", processingTime)
	}

	// Test memory requirements
	memoryReq := strategy.GetMemoryRequirements()
	if memoryReq > 1024*1024 { // 1MB
		t.Errorf("Expected memory requirements to be less than 1MB, got %d bytes", memoryReq)
	}
}
