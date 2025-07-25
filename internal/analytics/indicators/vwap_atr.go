package indicators

import (
	"math"

	"setbull_trader/internal/domain"
)

// VWAPCalculator provides VWAP calculation utilities
type VWAPCalculator struct {
	calculator *Calculator
}

// NewVWAPCalculator creates a new VWAP calculator
func NewVWAPCalculator() *VWAPCalculator {
	return &VWAPCalculator{
		calculator: NewCalculator(),
	}
}

// CalculateVWAP calculates Volume Weighted Average Price from candles
func (v *VWAPCalculator) CalculateVWAP(candles []domain.Candle) []domain.IndicatorValue {
	if len(candles) == 0 {
		return []domain.IndicatorValue{}
	}

	// Extract typical prices (using close price) and volumes
	prices := make([]float64, len(candles))
	volumes := make([]float64, len(candles))

	for i, candle := range candles {
		// Typical price = (High + Low + Close) / 3, but using Close for simplicity
		prices[i] = candle.Close
		volumes[i] = float64(candle.Volume)
	}

	// Calculate VWAP using calculator
	vwapValues := v.calculator.VWAP(prices, volumes)

	// Convert to domain model
	result := make([]domain.IndicatorValue, len(candles))
	for i, candle := range candles {
		result[i] = domain.IndicatorValue{
			Timestamp: candle.Timestamp,
			Value:     vwapValues[i],
		}
	}

	return result
}

// CalculateVWAPTypical calculates VWAP using typical price (HLC/3)
func (v *VWAPCalculator) CalculateVWAPTypical(candles []domain.Candle) []domain.IndicatorValue {
	if len(candles) == 0 {
		return []domain.IndicatorValue{}
	}

	// Calculate typical prices and volumes
	typicalPrices := make([]float64, len(candles))
	volumes := make([]float64, len(candles))

	for i, candle := range candles {
		typicalPrices[i] = (candle.High + candle.Low + candle.Close) / 3.0
		volumes[i] = float64(candle.Volume)
	}

	// Calculate VWAP using calculator
	vwapValues := v.calculator.VWAP(typicalPrices, volumes)

	// Convert to domain model
	result := make([]domain.IndicatorValue, len(candles))
	for i, candle := range candles {
		result[i] = domain.IndicatorValue{
			Timestamp: candle.Timestamp,
			Value:     vwapValues[i],
		}
	}

	return result
}

// GetVWAPSignals returns VWAP-based trading signals
func (v *VWAPCalculator) GetVWAPSignals(candles []domain.Candle) []domain.IndicatorValue {
	vwapValues := v.CalculateVWAP(candles)

	result := make([]domain.IndicatorValue, len(candles))
	for i, candle := range candles {
		var signal float64

		if math.IsNaN(vwapValues[i].Value) {
			signal = math.NaN()
		} else if candle.Close > vwapValues[i].Value {
			signal = 1.0 // Above VWAP - bullish
		} else if candle.Close < vwapValues[i].Value {
			signal = -1.0 // Below VWAP - bearish
		} else {
			signal = 0.0 // At VWAP - neutral
		}

		result[i] = domain.IndicatorValue{
			Timestamp: candle.Timestamp,
			Value:     signal,
		}
	}

	return result
}

// ATRCalculator provides ATR calculation utilities
type ATRCalculator struct {
	calculator *Calculator
}

// NewATRCalculator creates a new ATR calculator
func NewATRCalculator() *ATRCalculator {
	return &ATRCalculator{
		calculator: NewCalculator(),
	}
}

// CalculateATR calculates Average True Range from candles
func (a *ATRCalculator) CalculateATR(candles []domain.Candle, period int) []domain.IndicatorValue {
	if len(candles) == 0 {
		return []domain.IndicatorValue{}
	}

	// Extract OHLC data
	high := make([]float64, len(candles))
	low := make([]float64, len(candles))
	close := make([]float64, len(candles))

	for i, candle := range candles {
		high[i] = candle.High
		low[i] = candle.Low
		close[i] = candle.Close
	}

	// Calculate ATR using calculator
	atrValues := a.calculator.ATR(high, low, close, period)

	// Convert to domain model
	result := make([]domain.IndicatorValue, len(candles))
	for i, candle := range candles {
		result[i] = domain.IndicatorValue{
			Timestamp: candle.Timestamp,
			Value:     atrValues[i],
		}
	}

	return result
}

// CalculateATRFromData calculates ATR directly from OHLC slices
func (a *ATRCalculator) CalculateATRFromData(high, low, close []float64, period int) []float64 {
	return a.calculator.ATR(high, low, close, period)
}

// GetValidATRValues returns only the non-NaN ATR values
func (a *ATRCalculator) GetValidATRValues(candles []domain.Candle, period int) []domain.IndicatorValue {
	result := a.CalculateATR(candles, period)

	// Filter out NaN values
	validResults := make([]domain.IndicatorValue, 0, len(result))
	for _, indicator := range result {
		if !math.IsNaN(indicator.Value) {
			validResults = append(validResults, indicator)
		}
	}

	return validResults
}

// CalculateATRPercentage calculates ATR as percentage of current price
func (a *ATRCalculator) CalculateATRPercentage(candles []domain.Candle, period int) []domain.IndicatorValue {
	atrValues := a.CalculateATR(candles, period)

	result := make([]domain.IndicatorValue, len(atrValues))
	for i, atr := range atrValues {
		var percentage float64

		if math.IsNaN(atr.Value) || candles[i].Close == 0 {
			percentage = math.NaN()
		} else {
			percentage = (atr.Value / candles[i].Close) * 100
		}

		result[i] = domain.IndicatorValue{
			Timestamp: atr.Timestamp,
			Value:     percentage,
		}
	}

	return result
}

// GetATRVolatilityLevels returns volatility level classification based on ATR
func (a *ATRCalculator) GetATRVolatilityLevels(candles []domain.Candle, period int, atrPeriod int) []domain.IndicatorValue {
	atrValues := a.CalculateATR(candles, period)

	if len(atrValues) < atrPeriod {
		return []domain.IndicatorValue{}
	}

	result := make([]domain.IndicatorValue, len(atrValues))

	// Calculate average ATR for normalization
	for i := atrPeriod - 1; i < len(atrValues); i++ {
		// Get recent ATR values for comparison
		recentATRs := make([]float64, 0, atrPeriod)
		for j := i - atrPeriod + 1; j <= i; j++ {
			if !math.IsNaN(atrValues[j].Value) {
				recentATRs = append(recentATRs, atrValues[j].Value)
			}
		}

		if len(recentATRs) == 0 {
			result[i] = domain.IndicatorValue{
				Timestamp: atrValues[i].Timestamp,
				Value:     math.NaN(),
			}
			continue
		}

		// Calculate mean and std of recent ATRs
		var sum float64
		for _, val := range recentATRs {
			sum += val
		}
		mean := sum / float64(len(recentATRs))

		var level float64
		currentATR := atrValues[i].Value

		if math.IsNaN(currentATR) {
			level = math.NaN()
		} else if currentATR > mean*1.5 {
			level = 3.0 // High volatility
		} else if currentATR > mean*1.2 {
			level = 2.0 // Medium-high volatility
		} else if currentATR > mean*0.8 {
			level = 1.0 // Normal volatility
		} else {
			level = 0.0 // Low volatility
		}

		result[i] = domain.IndicatorValue{
			Timestamp: atrValues[i].Timestamp,
			Value:     level,
		}
	}

	// Fill initial values with NaN
	for i := 0; i < atrPeriod-1; i++ {
		result[i] = domain.IndicatorValue{
			Timestamp: atrValues[i].Timestamp,
			Value:     math.NaN(),
		}
	}

	return result
}
