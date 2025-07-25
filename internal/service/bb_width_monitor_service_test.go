package service

import (
	"context"
	"testing"
	"time"

	"setbull_trader/internal/core/dto/response"
	"setbull_trader/internal/domain"
)

func TestNewBBWidthMonitorService(t *testing.T) {
	// Test service creation
	service := NewBBWidthMonitorService(nil, nil, nil, nil, nil, nil)
	if service == nil {
		t.Fatal("Expected BBWidthMonitorService to be created")
	}
}

func TestBBWidthMonitorService_MonitorBBRangeGroups_NoGroups(t *testing.T) {
	// Test the service creation and basic functionality without calling methods that require database
	service := NewBBWidthMonitorService(
		nil, // stock group service
		nil, // technical indicator service
		nil, // alert service
		nil, // universe service
		nil, // config
		nil, // candle aggregation service
	)

	if service == nil {
		t.Fatal("Expected BBWidthMonitorService to be created")
	}

	// Test that the service can be created without panicking
	t.Log("BBWidthMonitorService created successfully")
}

func TestBBWidthMonitorService_MonitorGroupStocks_EmptyGroup(t *testing.T) {
	service := NewBBWidthMonitorService(nil, nil, nil, nil, nil, nil)

	ctx := context.Background()
	start := time.Now()
	end := start.Add(5 * time.Minute)

	// Test with empty group
	emptyGroup := response.StockGroupResponse{
		ID:        "test-group",
		EntryType: "BB_RANGE",
		Status:    "PENDING",
		Stocks:    []response.StockGroupStockDTO{},
	}

	err := service.monitorGroupStocks(ctx, emptyGroup, start, end)
	if err != nil {
		t.Errorf("Expected no error for empty group, got: %v", err)
	}
}

func TestBBWidthMonitorService_IsContractingPattern(t *testing.T) {
	service := NewBBWidthMonitorService(nil, nil, nil, nil, nil, nil)

	tests := []struct {
		name     string
		values   []domain.IndicatorValue
		expected bool
	}{
		{
			name:     "Empty values",
			values:   []domain.IndicatorValue{},
			expected: false,
		},
		{
			name: "Less than 3 values",
			values: []domain.IndicatorValue{
				{Value: 0.1},
				{Value: 0.09},
			},
			expected: false,
		},
		{
			name: "Contracting pattern",
			values: []domain.IndicatorValue{
				{Value: 0.1},
				{Value: 0.09},
				{Value: 0.08},
				{Value: 0.07},
			},
			expected: true,
		},
		{
			name: "Non-contracting pattern",
			values: []domain.IndicatorValue{
				{Value: 0.1},
				{Value: 0.09},
				{Value: 0.1}, // Increasing
				{Value: 0.08},
			},
			expected: false,
		},
		{
			name: "Sideways pattern",
			values: []domain.IndicatorValue{
				{Value: 0.1},
				{Value: 0.1},
				{Value: 0.1},
				{Value: 0.1},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isContractingPattern(tt.values)
			if result != tt.expected {
				t.Errorf("isContractingPattern() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBBWidthMonitorService_CalculateBBWidthRange(t *testing.T) {
	service := NewBBWidthMonitorService(nil, nil, nil, nil, nil, nil)

	tests := []struct {
		name             string
		lowestMinBBWidth float64
		expectedMinRange float64
		expectedMaxRange float64
	}{
		{
			name:             "Normal case",
			lowestMinBBWidth: 0.05,
			expectedMinRange: 0.04995, // 0.05 - 0.00005
			expectedMaxRange: 0.05005, // 0.05 + 0.00005
		},
		{
			name:             "Zero case",
			lowestMinBBWidth: 0.0,
			expectedMinRange: 0.0,
			expectedMaxRange: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minRange, maxRange := service.calculateBBWidthRange(tt.lowestMinBBWidth)

			// Use approximate comparison for floating point
			if abs(minRange-tt.expectedMinRange) > 0.000001 {
				t.Errorf("calculateBBWidthRange() minRange = %v, want %v", minRange, tt.expectedMinRange)
			}
			if abs(maxRange-tt.expectedMaxRange) > 0.000001 {
				t.Errorf("calculateBBWidthRange() maxRange = %v, want %v", maxRange, tt.expectedMaxRange)
			}
		})
	}
}

func TestBBWidthMonitorService_Integration_ContractingPatternDetection(t *testing.T) {
	// This test verifies the integration of contracting pattern detection
	// without requiring actual database connections

	service := NewBBWidthMonitorService(nil, nil, nil, nil, nil, nil)

	// Test contracting pattern detection
	contractingValues := []domain.IndicatorValue{
		{Value: 0.1, Timestamp: time.Now().Add(-15 * time.Minute)},
		{Value: 0.09, Timestamp: time.Now().Add(-10 * time.Minute)},
		{Value: 0.08, Timestamp: time.Now().Add(-5 * time.Minute)},
		{Value: 0.07, Timestamp: time.Now()},
	}

	isContracting := service.isContractingPattern(contractingValues)
	if !isContracting {
		t.Error("Expected contracting pattern to be detected")
	}

	// Test range calculation
	lowestMinBBWidth := 0.05
	minRange, maxRange := service.calculateBBWidthRange(lowestMinBBWidth)

	expectedMinRange := 0.04995 // 0.05 - 0.00005
	expectedMaxRange := 0.05005 // 0.05 + 0.00005

	if abs(minRange-expectedMinRange) > 0.000001 {
		t.Errorf("Expected minRange %f, got %f", expectedMinRange, minRange)
	}
	if abs(maxRange-expectedMaxRange) > 0.000001 {
		t.Errorf("Expected maxRange %f, got %f", expectedMaxRange, maxRange)
	}

	// Test that current BB width (0.07) is outside the optimal range (0.04995-0.05005)
	currentBBWidth := 0.07
	if currentBBWidth >= minRange && currentBBWidth <= maxRange {
		t.Error("Current BB width should be outside optimal range")
	}

	// Test with BB width within optimal range
	currentBBWidthInRange := 0.05
	if !(currentBBWidthInRange >= minRange && currentBBWidthInRange <= maxRange) {
		t.Error("BB width 0.05 should be within optimal range")
	}
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
