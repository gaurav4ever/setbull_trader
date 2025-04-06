package service

import (
	"math"
	"setbull_trader/internal/domain"
)

type TrendAnalyzer struct{}

func NewTrendAnalyzer() *TrendAnalyzer {
	return &TrendAnalyzer{}
}

func (ta *TrendAnalyzer) CalculateTrendStrength(candles []domain.Candle) float64 {
	if len(candles) < 2 {
		return 0
	}

	// Calculate linear regression
	n := float64(len(candles))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, candle := range candles {
		x := float64(i)
		y := candle.Close

		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// Calculate slope
	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// Calculate R-squared
	meanY := sumY / n
	totalSS := 0.0
	residualSS := 0.0

	for i, candle := range candles {
		yPred := slope*float64(i) + (sumY-slope*sumX)/n
		totalSS += math.Pow(candle.Close-meanY, 2)
		residualSS += math.Pow(candle.Close-yPred, 2)
	}

	rSquared := 1 - (residualSS / totalSS)

	// Combine slope direction and R-squared for trend strength
	slopeDirection := math.Copysign(1, slope)
	trendStrength := slopeDirection * math.Sqrt(math.Abs(rSquared))

	return math.Max(-1, math.Min(1, trendStrength))
}
