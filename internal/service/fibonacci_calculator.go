package service

import (
	"fmt"
	"math"

	"setbull_trader/internal/domain"
)

// FibonacciCalculator calculates Fibonacci execution levels
type FibonacciCalculator struct{}

// NewFibonacciCalculator creates a new FibonacciCalculator
func NewFibonacciCalculator() *FibonacciCalculator {
	return &FibonacciCalculator{}
}

// CalculateFibonacciLevels calculates execution price levels based on Fibonacci retracement
// Parameters:
//   - tradePrice: The initial entry price
//   - slPercentage: Stop loss percentage (positive value, e.g., 0.5 for 0.5%)
//   - side: The trade side (Buy or Sell)
//
// Returns:
//   - A slice of ExecutionLevel containing the stop loss and 5 entry levels
func (c *FibonacciCalculator) CalculateFibonacciLevels(
	tradePrice float64,
	slPercentage float64,
	side domain.TradeSide,
) []domain.ExecutionLevel {
	// Fibonacci levels
	fibLevels := []float64{0, 1, 1.25, 1.5, 1.75, 2}

	// Calculate the stop loss price
	var slPrice float64
	if side == domain.Buy {
		slPrice = tradePrice * (1 - slPercentage/100)
	} else {
		slPrice = tradePrice * (1 + slPercentage/100)
	}

	// Round the stop loss price to nearest 0.05 or 0.00
	slPrice = c.roundToNearestFiveOrZero(slPrice)

	// Calculate the range for Fibonacci calculations
	priceRange := math.Abs(tradePrice - slPrice)

	// Initialize the result slice
	result := make([]domain.ExecutionLevel, len(fibLevels))

	// Calculate execution levels
	for i, level := range fibLevels {
		var price float64

		if i == 0 {
			// Stop Loss level
			price = slPrice
			result[i] = domain.ExecutionLevel{
				Level:       level,
				Price:       price,
				Description: "Stop Loss",
			}
		} else if i == 1 {
			// First entry is the trade price
			price = tradePrice
			result[i] = domain.ExecutionLevel{
				Level:       level,
				Price:       price,
				Description: "1st Entry",
			}
		} else {
			// Calculate the price for additional entries
			if side == domain.Buy {
				// For Buy, additional entries are above the trade price
				price = tradePrice + priceRange*(level-1)
			} else {
				// For Sell, additional entries are below the trade price
				price = tradePrice - priceRange*(level-1)
			}

			// Round to nearest 0.05 or 0.00
			price = c.roundToNearestFiveOrZero(price)

			result[i] = domain.ExecutionLevel{
				Level:       level,
				Price:       price,
				Description: getOrdinal(i) + " Entry",
			}
		}
	}

	return result
}

// roundToNearestFiveOrZero rounds a price to the nearest 0.05 or 0.00
func (c *FibonacciCalculator) roundToNearestFiveOrZero(price float64) float64 {
	// Multiply by 100 to work with integers
	scaled := price * 100

	// Round to nearest integer
	rounded := math.Round(scaled)

	// Get the last digit
	lastDigit := int(rounded) % 10

	// Adjust to nearest 0 or 5
	if lastDigit < 3 {
		rounded = rounded - float64(lastDigit)
	} else if lastDigit < 8 {
		rounded = rounded - float64(lastDigit) + 5
	} else {
		rounded = rounded - float64(lastDigit) + 10
	}

	// Convert back to original scale
	return rounded / 100
}

// getOrdinal returns the ordinal suffix for a number
func getOrdinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return fmt.Sprintf("%dth", n)
	}
}
