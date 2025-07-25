package indicators

import (
	"math"

	"setbull_trader/internal/domain"
)

// RSICalculator provides RSI calculation utilities with GoNum optimization
type RSICalculator struct {
	calculator *Calculator
}

// NewRSICalculator creates a new RSI calculator
func NewRSICalculator() *RSICalculator {
	return &RSICalculator{
		calculator: NewCalculator(),
	}
}

// CalculateRSI calculates RSI from domain.Candle slice and returns domain.IndicatorValue slice
func (r *RSICalculator) CalculateRSI(candles []domain.Candle, period int) []domain.IndicatorValue {
	if len(candles) == 0 {
		return []domain.IndicatorValue{}
	}

	// Extract close prices
	prices := make([]float64, len(candles))
	for i, candle := range candles {
		prices[i] = candle.Close
	}

	// Calculate RSI using GoNum-optimized calculator
	rsiValues := r.calculator.RSI(prices, period)

	// Convert back to domain model
	result := make([]domain.IndicatorValue, len(candles))
	for i, candle := range candles {
		result[i] = domain.IndicatorValue{
			Timestamp: candle.Timestamp,
			Value:     rsiValues[i],
		}
	}

	return result
}

// CalculateRSIFromPrices calculates RSI directly from price slice
func (r *RSICalculator) CalculateRSIFromPrices(prices []float64, period int) []float64 {
	return r.calculator.RSI(prices, period)
}

// GetValidRSIValues returns only the non-NaN RSI values
func (r *RSICalculator) GetValidRSIValues(candles []domain.Candle, period int) []domain.IndicatorValue {
	result := r.CalculateRSI(candles, period)

	// Filter out NaN values
	validResults := make([]domain.IndicatorValue, 0, len(result))
	for _, indicator := range result {
		if !math.IsNaN(indicator.Value) {
			validResults = append(validResults, indicator)
		}
	}

	return validResults
}

// GetRSISignals returns RSI-based trading signals
func (r *RSICalculator) GetRSISignals(candles []domain.Candle, period int, overboughtLevel, oversoldLevel float64) []domain.IndicatorValue {
	rsiValues := r.CalculateRSI(candles, period)

	result := make([]domain.IndicatorValue, len(rsiValues))
	for i, rsi := range rsiValues {
		var signal float64

		if math.IsNaN(rsi.Value) {
			signal = math.NaN()
		} else if rsi.Value >= overboughtLevel {
			signal = -1.0 // Sell signal
		} else if rsi.Value <= oversoldLevel {
			signal = 1.0 // Buy signal
		} else {
			signal = 0.0 // Neutral
		}

		result[i] = domain.IndicatorValue{
			Timestamp: rsi.Timestamp,
			Value:     signal,
		}
	}

	return result
}

// GetRSIDivergence detects RSI divergence patterns
func (r *RSICalculator) GetRSIDivergence(candles []domain.Candle, period, lookback int) []domain.IndicatorValue {
	if len(candles) < lookback+1 {
		return []domain.IndicatorValue{}
	}

	rsiValues := r.CalculateRSI(candles, period)
	result := make([]domain.IndicatorValue, len(candles))

	// Fill initial values with NaN
	for i := 0; i < lookback; i++ {
		result[i] = domain.IndicatorValue{
			Timestamp: candles[i].Timestamp,
			Value:     math.NaN(),
		}
	}

	// Detect divergence
	for i := lookback; i < len(candles); i++ {
		currentPrice := candles[i].Close
		currentRSI := rsiValues[i].Value
		pastPrice := candles[i-lookback].Close
		pastRSI := rsiValues[i-lookback].Value

		var divergence float64

		if math.IsNaN(currentRSI) || math.IsNaN(pastRSI) {
			divergence = math.NaN()
		} else {
			priceChange := currentPrice - pastPrice
			rsiChange := currentRSI - pastRSI

			// Bullish divergence: price down, RSI up
			if priceChange < 0 && rsiChange > 0 {
				divergence = 1.0
			} else if priceChange > 0 && rsiChange < 0 {
				// Bearish divergence: price up, RSI down
				divergence = -1.0
			} else {
				divergence = 0.0
			}
		}

		result[i] = domain.IndicatorValue{
			Timestamp: candles[i].Timestamp,
			Value:     divergence,
		}
	}

	return result
}

// CalculateRSISlope calculates the slope/trend of RSI values
func (r *RSICalculator) CalculateRSISlope(candles []domain.Candle, period int, lookback int) []domain.IndicatorValue {
	rsiValues := r.CalculateRSI(candles, period)

	if len(rsiValues) < lookback+1 {
		return []domain.IndicatorValue{}
	}

	result := make([]domain.IndicatorValue, len(rsiValues))

	// Fill initial values with NaN
	for i := 0; i < lookback; i++ {
		result[i] = domain.IndicatorValue{
			Timestamp: rsiValues[i].Timestamp,
			Value:     math.NaN(),
		}
	}

	// Calculate slope for valid points
	for i := lookback; i < len(rsiValues); i++ {
		if !math.IsNaN(rsiValues[i].Value) && !math.IsNaN(rsiValues[i-lookback].Value) {
			slope := (rsiValues[i].Value - rsiValues[i-lookback].Value) / float64(lookback)
			result[i] = domain.IndicatorValue{
				Timestamp: rsiValues[i].Timestamp,
				Value:     slope,
			}
		} else {
			result[i] = domain.IndicatorValue{
				Timestamp: rsiValues[i].Timestamp,
				Value:     math.NaN(),
			}
		}
	}

	return result
}

// GetRSILevels returns RSI levels classification
func (r *RSICalculator) GetRSILevels(candles []domain.Candle, period int) []domain.IndicatorValue {
	rsiValues := r.CalculateRSI(candles, period)

	result := make([]domain.IndicatorValue, len(rsiValues))
	for i, rsi := range rsiValues {
		var level float64

		if math.IsNaN(rsi.Value) {
			level = math.NaN()
		} else if rsi.Value >= 70 {
			level = 3.0 // Overbought
		} else if rsi.Value >= 50 {
			level = 2.0 // Bullish
		} else if rsi.Value >= 30 {
			level = 1.0 // Bearish
		} else {
			level = 0.0 // Oversold
		}

		result[i] = domain.IndicatorValue{
			Timestamp: rsi.Timestamp,
			Value:     level,
		}
	}

	return result
}
