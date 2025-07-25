package indicators

import (
	"math"

	"setbull_trader/internal/domain"
)

// BollingerCalculator provides Bollinger Bands calculation utilities with GoNum optimization
type BollingerCalculator struct {
	calculator *Calculator
}

// NewBollingerCalculator creates a new Bollinger Bands calculator
func NewBollingerCalculator() *BollingerCalculator {
	return &BollingerCalculator{
		calculator: NewCalculator(),
	}
}

// BollingerBandsResult holds the complete Bollinger Bands calculation result
type BollingerBandsResult struct {
	Upper  []domain.IndicatorValue
	Middle []domain.IndicatorValue
	Lower  []domain.IndicatorValue
	Width  []domain.IndicatorValue
}

// CalculateBollingerBands calculates complete Bollinger Bands with width
func (b *BollingerCalculator) CalculateBollingerBands(candles []domain.Candle, period int, stdDevMultiplier float64) *BollingerBandsResult {
	if len(candles) == 0 {
		return &BollingerBandsResult{
			Upper:  []domain.IndicatorValue{},
			Middle: []domain.IndicatorValue{},
			Lower:  []domain.IndicatorValue{},
			Width:  []domain.IndicatorValue{},
		}
	}

	// Extract close prices
	prices := make([]float64, len(candles))
	for i, candle := range candles {
		prices[i] = candle.Close
	}

	// Calculate Bollinger Bands using GoNum-optimized calculator
	upperValues, middleValues, lowerValues := b.calculator.BollingerBands(prices, period, stdDevMultiplier)

	// Calculate BB Width
	widthValues := b.calculator.BBWidth(upperValues, middleValues, lowerValues)

	// Convert to domain models
	result := &BollingerBandsResult{
		Upper:  make([]domain.IndicatorValue, len(candles)),
		Middle: make([]domain.IndicatorValue, len(candles)),
		Lower:  make([]domain.IndicatorValue, len(candles)),
		Width:  make([]domain.IndicatorValue, len(candles)),
	}

	for i, candle := range candles {
		timestamp := candle.Timestamp

		result.Upper[i] = domain.IndicatorValue{
			Timestamp: timestamp,
			Value:     upperValues[i],
		}
		result.Middle[i] = domain.IndicatorValue{
			Timestamp: timestamp,
			Value:     middleValues[i],
		}
		result.Lower[i] = domain.IndicatorValue{
			Timestamp: timestamp,
			Value:     lowerValues[i],
		}
		result.Width[i] = domain.IndicatorValue{
			Timestamp: timestamp,
			Value:     widthValues[i],
		}
	}

	return result
}

// CalculateBollingerBandsCompatible calculates BB compatible with TradingView
func (b *BollingerCalculator) CalculateBollingerBandsCompatible(candles []domain.Candle, period int, multiplier float64) (upper, middle, lower []domain.IndicatorValue) {
	result := b.CalculateBollingerBands(candles, period, multiplier)
	return result.Upper, result.Middle, result.Lower
}

// CalculateBBWidth calculates Bollinger Band Width from existing bands
func (b *BollingerCalculator) CalculateBBWidth(upper, middle, lower []domain.IndicatorValue) []domain.IndicatorValue {
	if len(upper) == 0 || len(middle) == 0 || len(lower) == 0 {
		return []domain.IndicatorValue{}
	}

	minLen := len(upper)
	if len(middle) < minLen {
		minLen = len(middle)
	}
	if len(lower) < minLen {
		minLen = len(lower)
	}

	result := make([]domain.IndicatorValue, minLen)

	for i := 0; i < minLen; i++ {
		timestamp := upper[i].Timestamp

		if math.IsNaN(upper[i].Value) || math.IsNaN(middle[i].Value) || math.IsNaN(lower[i].Value) || middle[i].Value == 0 {
			result[i] = domain.IndicatorValue{
				Timestamp: timestamp,
				Value:     math.NaN(),
			}
		} else {
			width := (upper[i].Value - lower[i].Value) / middle[i].Value
			result[i] = domain.IndicatorValue{
				Timestamp: timestamp,
				Value:     width,
			}
		}
	}

	return result
}

// CalculateBBWidthNormalized calculates normalized BB width
func (b *BollingerCalculator) CalculateBBWidthNormalized(upper, middle, lower []domain.IndicatorValue) []domain.IndicatorValue {
	widthValues := b.CalculateBBWidth(upper, middle, lower)

	if len(widthValues) == 0 {
		return []domain.IndicatorValue{}
	}

	// Find min and max for normalization (excluding NaN values)
	var minWidth, maxWidth float64
	firstValidFound := false

	for _, w := range widthValues {
		if !math.IsNaN(w.Value) {
			if !firstValidFound {
				minWidth = w.Value
				maxWidth = w.Value
				firstValidFound = true
			} else {
				if w.Value < minWidth {
					minWidth = w.Value
				}
				if w.Value > maxWidth {
					maxWidth = w.Value
				}
			}
		}
	}

	// Normalize values
	result := make([]domain.IndicatorValue, len(widthValues))
	widthRange := maxWidth - minWidth

	for i, w := range widthValues {
		if math.IsNaN(w.Value) || widthRange == 0 {
			result[i] = domain.IndicatorValue{
				Timestamp: w.Timestamp,
				Value:     math.NaN(),
			}
		} else {
			normalized := (w.Value - minWidth) / widthRange
			result[i] = domain.IndicatorValue{
				Timestamp: w.Timestamp,
				Value:     normalized,
			}
		}
	}

	return result
}

// CalculateBBWidthPercentage calculates BB width as percentage
func (b *BollingerCalculator) CalculateBBWidthPercentage(upper, middle, lower []domain.IndicatorValue) []domain.IndicatorValue {
	widthValues := b.CalculateBBWidth(upper, middle, lower)

	result := make([]domain.IndicatorValue, len(widthValues))
	for i, w := range widthValues {
		if math.IsNaN(w.Value) {
			result[i] = domain.IndicatorValue{
				Timestamp: w.Timestamp,
				Value:     math.NaN(),
			}
		} else {
			percentage := w.Value * 100 // Convert to percentage
			result[i] = domain.IndicatorValue{
				Timestamp: w.Timestamp,
				Value:     percentage,
			}
		}
	}

	return result
}

// GetBBSqueeze detects Bollinger Band squeeze conditions
func (b *BollingerCalculator) GetBBSqueeze(upper, middle, lower []domain.IndicatorValue, squeezeThreshold float64) []domain.IndicatorValue {
	widthValues := b.CalculateBBWidth(upper, middle, lower)

	result := make([]domain.IndicatorValue, len(widthValues))
	for i, w := range widthValues {
		var squeezeValue float64
		if math.IsNaN(w.Value) {
			squeezeValue = math.NaN()
		} else if w.Value <= squeezeThreshold {
			squeezeValue = 1.0 // Squeeze detected
		} else {
			squeezeValue = 0.0 // No squeeze
		}

		result[i] = domain.IndicatorValue{
			Timestamp: w.Timestamp,
			Value:     squeezeValue,
		}
	}

	return result
}

// GetBBPosition calculates price position within Bollinger Bands (0 = lower band, 1 = upper band)
func (b *BollingerCalculator) GetBBPosition(candles []domain.Candle, upper, middle, lower []domain.IndicatorValue) []domain.IndicatorValue {
	if len(candles) == 0 || len(upper) == 0 || len(lower) == 0 {
		return []domain.IndicatorValue{}
	}

	minLen := len(candles)
	if len(upper) < minLen {
		minLen = len(upper)
	}
	if len(lower) < minLen {
		minLen = len(lower)
	}

	result := make([]domain.IndicatorValue, minLen)

	for i := 0; i < minLen; i++ {
		price := candles[i].Close
		timestamp := candles[i].Timestamp

		if math.IsNaN(upper[i].Value) || math.IsNaN(lower[i].Value) {
			result[i] = domain.IndicatorValue{
				Timestamp: timestamp,
				Value:     math.NaN(),
			}
		} else {
			bandRange := upper[i].Value - lower[i].Value
			if bandRange == 0 {
				result[i] = domain.IndicatorValue{
					Timestamp: timestamp,
					Value:     0.5, // Middle position if no range
				}
			} else {
				position := (price - lower[i].Value) / bandRange
				result[i] = domain.IndicatorValue{
					Timestamp: timestamp,
					Value:     position,
				}
			}
		}
	}

	return result
}

// GetValidBBValues returns only non-NaN Bollinger Band values
func (b *BollingerCalculator) GetValidBBValues(result *BollingerBandsResult) *BollingerBandsResult {
	if result == nil {
		return &BollingerBandsResult{}
	}

	validResult := &BollingerBandsResult{
		Upper:  make([]domain.IndicatorValue, 0),
		Middle: make([]domain.IndicatorValue, 0),
		Lower:  make([]domain.IndicatorValue, 0),
		Width:  make([]domain.IndicatorValue, 0),
	}

	minLen := len(result.Upper)
	if len(result.Middle) < minLen {
		minLen = len(result.Middle)
	}
	if len(result.Lower) < minLen {
		minLen = len(result.Lower)
	}
	if len(result.Width) < minLen {
		minLen = len(result.Width)
	}

	for i := 0; i < minLen; i++ {
		if !math.IsNaN(result.Upper[i].Value) &&
			!math.IsNaN(result.Middle[i].Value) &&
			!math.IsNaN(result.Lower[i].Value) {
			validResult.Upper = append(validResult.Upper, result.Upper[i])
			validResult.Middle = append(validResult.Middle, result.Middle[i])
			validResult.Lower = append(validResult.Lower, result.Lower[i])
			validResult.Width = append(validResult.Width, result.Width[i])
		}
	}

	return validResult
}
