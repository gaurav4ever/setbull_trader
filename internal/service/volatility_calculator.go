package service

import (
	"math"
	"setbull_trader/internal/domain"
)

type VolatilityCalculator struct{}

func NewVolatilityCalculator() *VolatilityCalculator {
	return &VolatilityCalculator{}
}

func (vc *VolatilityCalculator) CalculateVolatility(candles []domain.Candle) float64 {
	if len(candles) < 2 {
		return 0
	}

	// Calculate daily returns
	returns := make([]float64, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		returns[i-1] = math.Log(candles[i].Close / candles[i-1].Close)
	}

	// Calculate standard deviation of returns
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns))

	// Annualize volatility
	return math.Sqrt(variance) * math.Sqrt(252) // 252 trading days in a year
}
