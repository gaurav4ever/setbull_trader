package indicators

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/stat"
)

// Calculator provides GoNum-powered statistical calculations for technical indicators
type Calculator struct {
	// Configuration for calculations
	precision float64
}

// NewCalculator creates a new calculator instance
func NewCalculator() *Calculator {
	return &Calculator{
		precision: 1e-10, // High precision for financial calculations
	}
}

// EMA calculates Exponential Moving Average using optimized mathematical operations
func (c *Calculator) EMA(prices []float64, period int) []float64 {
	if len(prices) == 0 || period <= 0 {
		return []float64{}
	}

	if period > len(prices) {
		period = len(prices)
	}

	result := make([]float64, len(prices))
	multiplier := 2.0 / float64(period+1)

	// Initialize with Simple Moving Average for the first value
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	result[period-1] = sum / float64(period)

	// Calculate EMA for the rest
	for i := period; i < len(prices); i++ {
		result[i] = (prices[i]*multiplier + result[i-1]*(1-multiplier))
	}

	// Fill initial NaN values
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	return result
}

// SMA calculates Simple Moving Average using GoNum's optimized statistics
func (c *Calculator) SMA(prices []float64, period int) []float64 {
	if len(prices) == 0 || period <= 0 {
		return []float64{}
	}

	result := make([]float64, len(prices))

	// Fill initial values with NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	// Calculate moving average for valid windows
	for i := period - 1; i < len(prices); i++ {
		window := prices[i-period+1 : i+1]
		result[i] = stat.Mean(window, nil)
	}

	return result
}

// BollingerBands calculates Bollinger Bands using GoNum statistical functions
func (c *Calculator) BollingerBands(prices []float64, period int, stdDevMultiplier float64) (upper, middle, lower []float64) {
	if len(prices) == 0 || period <= 0 {
		return []float64{}, []float64{}, []float64{}
	}

	n := len(prices)
	upper = make([]float64, n)
	middle = make([]float64, n)
	lower = make([]float64, n)

	// Fill initial values with NaN
	for i := 0; i < period-1; i++ {
		upper[i] = math.NaN()
		middle[i] = math.NaN()
		lower[i] = math.NaN()
	}

	// Calculate Bollinger Bands for valid windows
	for i := period - 1; i < n; i++ {
		window := prices[i-period+1 : i+1]

		// Calculate mean (middle band) using GoNum
		mean := stat.Mean(window, nil)
		middle[i] = mean

		// Calculate standard deviation using GoNum
		stdDev := stat.StdDev(window, nil)

		// Calculate upper and lower bands
		upper[i] = mean + (stdDevMultiplier * stdDev)
		lower[i] = mean - (stdDevMultiplier * stdDev)
	}

	return upper, middle, lower
}

// RSI calculates Relative Strength Index using optimized gain/loss calculations
func (c *Calculator) RSI(prices []float64, period int) []float64 {
	if len(prices) == 0 || period <= 0 || len(prices) < period+1 {
		return []float64{}
	}

	n := len(prices)
	result := make([]float64, n)

	// Fill initial values with NaN
	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	// Calculate price changes
	changes := make([]float64, n-1)
	for i := 1; i < n; i++ {
		changes[i-1] = prices[i] - prices[i-1]
	}

	// Separate gains and losses
	gains := make([]float64, len(changes))
	losses := make([]float64, len(changes))

	for i, change := range changes {
		if change > 0 {
			gains[i] = change
			losses[i] = 0
		} else {
			gains[i] = 0
			losses[i] = -change
		}
	}

	// Calculate initial average gain and loss
	avgGain := stat.Mean(gains[:period], nil)
	avgLoss := stat.Mean(losses[:period], nil)

	// Calculate RSI for the first valid point
	if avgLoss == 0 {
		result[period] = 100
	} else {
		rs := avgGain / avgLoss
		result[period] = 100 - (100 / (1 + rs))
	}

	// Calculate RSI for remaining points using smoothed averages
	alpha := 1.0 / float64(period)
	for i := period + 1; i < n; i++ {
		// Smoothed average gain and loss (similar to EMA)
		avgGain = alpha*gains[i-1] + (1-alpha)*avgGain
		avgLoss = alpha*losses[i-1] + (1-alpha)*avgLoss

		if avgLoss == 0 {
			result[i] = 100
		} else {
			rs := avgGain / avgLoss
			result[i] = 100 - (100 / (1 + rs))
		}
	}

	return result
}

// ATR calculates Average True Range using GoNum's statistical functions
func (c *Calculator) ATR(high, low, close []float64, period int) []float64 {
	if len(high) == 0 || len(low) == 0 || len(close) == 0 || period <= 0 {
		return []float64{}
	}

	n := len(high)
	if len(low) != n || len(close) != n {
		return []float64{}
	}

	if n < period+1 {
		return make([]float64, n) // Return zeros if not enough data
	}

	result := make([]float64, n)
	trueRanges := make([]float64, n-1)

	// Calculate True Range for each period
	for i := 1; i < n; i++ {
		tr1 := high[i] - low[i]
		tr2 := math.Abs(high[i] - close[i-1])
		tr3 := math.Abs(low[i] - close[i-1])

		trueRanges[i-1] = math.Max(tr1, math.Max(tr2, tr3))
	}

	// Fill initial values with NaN
	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	// Calculate ATR using smoothed moving average (similar to EMA)
	if period <= len(trueRanges) {
		// Initial ATR as simple average
		result[period] = stat.Mean(trueRanges[:period], nil)

		// Calculate subsequent ATR values using smoothed average
		alpha := 1.0 / float64(period)
		for i := period + 1; i < n; i++ {
			result[i] = alpha*trueRanges[i-1] + (1-alpha)*result[i-1]
		}
	}

	return result
}

// VWAP calculates Volume Weighted Average Price
func (c *Calculator) VWAP(prices, volumes []float64) []float64 {
	if len(prices) == 0 || len(volumes) == 0 || len(prices) != len(volumes) {
		return []float64{}
	}

	n := len(prices)
	result := make([]float64, n)

	cumulativeTypicalPriceVolume := 0.0
	cumulativeVolume := 0.0

	for i := 0; i < n; i++ {
		typicalPrice := prices[i] // For close prices, typical price is just the close
		volumeWeightedPrice := typicalPrice * volumes[i]

		cumulativeTypicalPriceVolume += volumeWeightedPrice
		cumulativeVolume += volumes[i]

		if cumulativeVolume > 0 {
			result[i] = cumulativeTypicalPriceVolume / cumulativeVolume
		} else {
			result[i] = prices[i]
		}
	}

	return result
}

// BBWidth calculates Bollinger Band Width (normalized)
func (c *Calculator) BBWidth(upper, middle, lower []float64) []float64 {
	if len(upper) == 0 || len(middle) == 0 || len(lower) == 0 {
		return []float64{}
	}

	n := len(upper)
	if len(middle) != n || len(lower) != n {
		return []float64{}
	}

	result := make([]float64, n)

	for i := 0; i < n; i++ {
		if math.IsNaN(upper[i]) || math.IsNaN(middle[i]) || math.IsNaN(lower[i]) || middle[i] == 0 {
			result[i] = math.NaN()
		} else {
			result[i] = (upper[i] - lower[i]) / middle[i]
		}
	}

	return result
}

// Utilities

// IsValidNumber checks if a float64 value is valid (not NaN or Inf)
func (c *Calculator) IsValidNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// HandleNaN replaces NaN values with a default value
func (c *Calculator) HandleNaN(values []float64, defaultValue float64) []float64 {
	result := make([]float64, len(values))
	for i, v := range values {
		if math.IsNaN(v) {
			result[i] = defaultValue
		} else {
			result[i] = v
		}
	}
	return result
}

// ValidateInputs performs common validation for indicator calculations
func (c *Calculator) ValidateInputs(prices []float64, period int) error {
	if len(prices) == 0 {
		return fmt.Errorf("prices slice is empty")
	}
	if period <= 0 {
		return fmt.Errorf("period must be positive, got %d", period)
	}
	if period > len(prices) {
		return fmt.Errorf("period (%d) cannot be greater than data length (%d)", period, len(prices))
	}
	return nil
}
