package concurrency

import (
	"context"
	"fmt"
	"time"

	"setbull_trader/internal/analytics/indicators"
	"setbull_trader/internal/domain"
)

// IndicatorTask represents a task for calculating technical indicators
type IndicatorTask struct {
	id            string
	instrumentKey string
	candles       []domain.Candle
	indicatorType string
	parameters    map[string]interface{}
	priority      int
}

// NewIndicatorTask creates a new indicator calculation task
func NewIndicatorTask(
	id, instrumentKey, indicatorType string,
	candles []domain.Candle,
	parameters map[string]interface{},
	priority int,
) *IndicatorTask {
	return &IndicatorTask{
		id:            id,
		instrumentKey: instrumentKey,
		candles:       candles,
		indicatorType: indicatorType,
		parameters:    parameters,
		priority:      priority,
	}
}

// ID returns the task ID
func (it *IndicatorTask) ID() string {
	return it.id
}

// Priority returns the task priority
func (it *IndicatorTask) Priority() int {
	return it.priority
}

// Execute executes the indicator calculation task
func (it *IndicatorTask) Execute(ctx context.Context) (interface{}, error) {
	if len(it.candles) == 0 {
		return nil, fmt.Errorf("no candles provided for indicator calculation")
	}

	switch it.indicatorType {
	case "EMA":
		return it.calculateEMA()
	case "RSI":
		return it.calculateRSI()
	case "BOLLINGER":
		return it.calculateBollingerBands()
	case "VWAP":
		return it.calculateVWAP()
	case "ATR":
		return it.calculateATR()
	case "SMA":
		return it.calculateSMA()
	default:
		return nil, fmt.Errorf("unsupported indicator type: %s", it.indicatorType)
	}
}

// calculateEMA calculates Exponential Moving Average
func (it *IndicatorTask) calculateEMA() ([]domain.IndicatorValue, error) {
	period, ok := it.parameters["period"].(int)
	if !ok {
		return nil, fmt.Errorf("missing or invalid period parameter for EMA")
	}

	calculator := indicators.NewEMACalculator()
	return calculator.CalculateEMA(it.candles, period), nil
}

// calculateRSI calculates Relative Strength Index
func (it *IndicatorTask) calculateRSI() ([]domain.IndicatorValue, error) {
	period, ok := it.parameters["period"].(int)
	if !ok {
		return nil, fmt.Errorf("missing or invalid period parameter for RSI")
	}

	calculator := indicators.NewRSICalculator()
	return calculator.CalculateRSI(it.candles, period), nil
}

// calculateBollingerBands calculates Bollinger Bands
func (it *IndicatorTask) calculateBollingerBands() (interface{}, error) {
	period, ok := it.parameters["period"].(int)
	if !ok {
		return nil, fmt.Errorf("missing or invalid period parameter for Bollinger Bands")
	}

	stdDev, ok := it.parameters["stddev"].(float64)
	if !ok {
		stdDev = 2.0 // Default standard deviation
	}

	calculator := indicators.NewBollingerCalculator()
	return calculator.CalculateBollingerBands(it.candles, period, stdDev), nil
}

// calculateVWAP calculates Volume Weighted Average Price
func (it *IndicatorTask) calculateVWAP() ([]domain.IndicatorValue, error) {
	calculator := indicators.NewVWAPCalculator()
	return calculator.CalculateVWAP(it.candles), nil
}

// calculateATR calculates Average True Range
func (it *IndicatorTask) calculateATR() ([]domain.IndicatorValue, error) {
	period, ok := it.parameters["period"].(int)
	if !ok {
		return nil, fmt.Errorf("missing or invalid period parameter for ATR")
	}

	calculator := indicators.NewATRCalculator()
	return calculator.CalculateATR(it.candles, period), nil
}

// calculateSMA calculates Simple Moving Average
func (it *IndicatorTask) calculateSMA() ([]domain.IndicatorValue, error) {
	period, ok := it.parameters["period"].(int)
	if !ok {
		return nil, fmt.Errorf("missing or invalid period parameter for SMA")
	}

	if len(it.candles) < period {
		return nil, fmt.Errorf("insufficient data: need at least %d candles, got %d", period, len(it.candles))
	}

	// Extract close prices
	prices := make([]float64, len(it.candles))
	for i, candle := range it.candles {
		prices[i] = candle.Close
	}

	// Calculate SMA manually
	result := make([]domain.IndicatorValue, len(it.candles))

	// Fill initial values with 0 or NaN before we have enough data
	for i := 0; i < period-1; i++ {
		result[i] = domain.IndicatorValue{
			Timestamp: it.candles[i].Timestamp,
			Value:     0.0, // or math.NaN()
		}
	}

	// Calculate SMA for remaining values
	for i := period - 1; i < len(it.candles); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += prices[j]
		}
		sma := sum / float64(period)

		result[i] = domain.IndicatorValue{
			Timestamp: it.candles[i].Timestamp,
			Value:     sma,
		}
	}

	return result, nil
}

// BatchIndicatorTask represents a task for calculating multiple indicators in parallel
type BatchIndicatorTask struct {
	id            string
	instrumentKey string
	candles       []domain.Candle
	indicators    []IndicatorRequest
	priority      int
}

// IndicatorRequest represents a single indicator calculation request
type IndicatorRequest struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// BatchIndicatorResult represents the result of a batch indicator calculation
type BatchIndicatorResult struct {
	InstrumentKey string                   `json:"instrument_key"`
	Results       map[string]interface{}   `json:"results"`
	Errors        map[string]error         `json:"errors"`
	Timing        map[string]time.Duration `json:"timing"`
}

// NewBatchIndicatorTask creates a new batch indicator calculation task
func NewBatchIndicatorTask(
	id, instrumentKey string,
	candles []domain.Candle,
	indicators []IndicatorRequest,
	priority int,
) *BatchIndicatorTask {
	return &BatchIndicatorTask{
		id:            id,
		instrumentKey: instrumentKey,
		candles:       candles,
		indicators:    indicators,
		priority:      priority,
	}
}

// ID returns the task ID
func (bit *BatchIndicatorTask) ID() string {
	return bit.id
}

// Priority returns the task priority
func (bit *BatchIndicatorTask) Priority() int {
	return bit.priority
}

// Execute executes the batch indicator calculation task
func (bit *BatchIndicatorTask) Execute(ctx context.Context) (interface{}, error) {
	result := &BatchIndicatorResult{
		InstrumentKey: bit.instrumentKey,
		Results:       make(map[string]interface{}),
		Errors:        make(map[string]error),
		Timing:        make(map[string]time.Duration),
	}

	// Calculate each indicator
	for i, indicatorReq := range bit.indicators {
		indicatorID := fmt.Sprintf("%s_%d", indicatorReq.Type, i)

		startTime := time.Now()

		// Create individual task for this indicator
		task := NewIndicatorTask(
			indicatorID,
			bit.instrumentKey,
			indicatorReq.Type,
			bit.candles,
			indicatorReq.Parameters,
			bit.priority,
		)

		// Execute the calculation
		data, err := task.Execute(ctx)

		duration := time.Since(startTime)
		result.Timing[indicatorID] = duration

		if err != nil {
			result.Errors[indicatorID] = err
		} else {
			result.Results[indicatorID] = data
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
			// Continue
		}
	}

	return result, nil
}

// PriorityTask represents a task with priority ordering
type PriorityTask struct {
	Task         Task
	TaskPriority int
	TaskID       string
}

// NewPriorityTask creates a new priority task wrapper
func NewPriorityTask(task Task, priority int) *PriorityTask {
	return &PriorityTask{
		Task:         task,
		TaskPriority: priority,
		TaskID:       task.ID(),
	}
}

// Execute executes the wrapped task
func (pt *PriorityTask) Execute(ctx context.Context) (interface{}, error) {
	return pt.Task.Execute(ctx)
}

// ID returns the task ID
func (pt *PriorityTask) ID() string {
	return pt.TaskID
}

// Priority returns the task priority
func (pt *PriorityTask) Priority() int {
	return pt.TaskPriority
}
