package indicators

import (
	"math"
	"time"

	"setbull_trader/internal/domain"
)

// EMACalculator provides EMA calculation utilities with GoNum optimization
type EMACalculator struct {
	calculator *Calculator
}

// NewEMACalculator creates a new EMA calculator
func NewEMACalculator() *EMACalculator {
	return &EMACalculator{
		calculator: NewCalculator(),
	}
}

// CalculateEMA calculates EMA from domain.Candle slice and returns domain.IndicatorValue slice
func (e *EMACalculator) CalculateEMA(candles []domain.Candle, period int) []domain.IndicatorValue {
	if len(candles) == 0 {
		return []domain.IndicatorValue{}
	}

	// Extract close prices
	prices := make([]float64, len(candles))
	for i, candle := range candles {
		prices[i] = candle.Close
	}

	// Calculate EMA using GoNum-optimized calculator
	emaValues := e.calculator.EMA(prices, period)

	// Convert back to domain model
	result := make([]domain.IndicatorValue, len(candles))
	for i, candle := range candles {
		result[i] = domain.IndicatorValue{
			Timestamp: candle.Timestamp,
			Value:     emaValues[i],
		}
	}

	return result
}

// CalculateMultipleEMAs calculates multiple EMA periods efficiently
func (e *EMACalculator) CalculateMultipleEMAs(candles []domain.Candle, periods []int) map[int][]domain.IndicatorValue {
	result := make(map[int][]domain.IndicatorValue)

	if len(candles) == 0 || len(periods) == 0 {
		return result
	}

	// Extract prices once
	prices := make([]float64, len(candles))
	timestamps := make([]time.Time, len(candles))
	for i, candle := range candles {
		prices[i] = candle.Close
		timestamps[i] = candle.Timestamp
	}

	// Calculate each EMA period
	for _, period := range periods {
		emaValues := e.calculator.EMA(prices, period)

		// Convert to domain model
		indicators := make([]domain.IndicatorValue, len(candles))
		for i := 0; i < len(candles); i++ {
			indicators[i] = domain.IndicatorValue{
				Timestamp: timestamps[i],
				Value:     emaValues[i],
			}
		}

		result[period] = indicators
	}

	return result
}

// CalculateEMAFromPrices calculates EMA directly from price slice
func (e *EMACalculator) CalculateEMAFromPrices(prices []float64, period int) []float64 {
	return e.calculator.EMA(prices, period)
}

// CalculateEMAWithTimestamps calculates EMA and returns with timestamp mapping
func (e *EMACalculator) CalculateEMAWithTimestamps(candles []domain.Candle, period int) ([]time.Time, []float64) {
	if len(candles) == 0 {
		return []time.Time{}, []float64{}
	}

	prices := make([]float64, len(candles))
	timestamps := make([]time.Time, len(candles))

	for i, candle := range candles {
		prices[i] = candle.Close
		timestamps[i] = candle.Timestamp
	}

	emaValues := e.calculator.EMA(prices, period)
	return timestamps, emaValues
}

// GetValidEMAValues returns only the non-NaN EMA values with their timestamps
func (e *EMACalculator) GetValidEMAValues(candles []domain.Candle, period int) []domain.IndicatorValue {
	result := e.CalculateEMA(candles, period)

	// Filter out NaN values
	validResults := make([]domain.IndicatorValue, 0, len(result))
	for _, indicator := range result {
		if !math.IsNaN(indicator.Value) {
			validResults = append(validResults, indicator)
		}
	}

	return validResults
}

// CalculateEMASlope calculates the slope/trend of EMA values
func (e *EMACalculator) CalculateEMASlope(candles []domain.Candle, period int, lookback int) []domain.IndicatorValue {
	emaIndicators := e.CalculateEMA(candles, period)

	if len(emaIndicators) < lookback+1 {
		return []domain.IndicatorValue{}
	}

	result := make([]domain.IndicatorValue, len(emaIndicators))

	// Fill initial values with NaN
	for i := 0; i < lookback; i++ {
		result[i] = domain.IndicatorValue{
			Timestamp: emaIndicators[i].Timestamp,
			Value:     math.NaN(),
		}
	}

	// Calculate slope for valid points
	for i := lookback; i < len(emaIndicators); i++ {
		if !math.IsNaN(emaIndicators[i].Value) && !math.IsNaN(emaIndicators[i-lookback].Value) {
			slope := (emaIndicators[i].Value - emaIndicators[i-lookback].Value) / float64(lookback)
			result[i] = domain.IndicatorValue{
				Timestamp: emaIndicators[i].Timestamp,
				Value:     slope,
			}
		} else {
			result[i] = domain.IndicatorValue{
				Timestamp: emaIndicators[i].Timestamp,
				Value:     math.NaN(),
			}
		}
	}

	return result
}
