package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	"setbull_trader/internal/domain"
	"setbull_trader/pkg/log"
)

// UtilityService provides utility operations for calculations and conversions
type UtilityService struct {
	fibCalculator *FibonacciCalculator
}

// NewUtilityService creates a new UtilityService
func NewUtilityService(fibCalculator *FibonacciCalculator) *UtilityService {
	return &UtilityService{
		fibCalculator: fibCalculator,
	}
}

// CalculateFibonacciLevelsWithQuantities calculates Fibonacci levels and their quantities
// for a given set of parameters without creating an execution plan
func (s *UtilityService) CalculateFibonacciLevelsWithQuantities(
	startingPrice float64,
	slPercentage float64,
	tradeSide domain.TradeSide,
	riskAmount float64,
) (int, []domain.LevelWithQuantity, error) {
	// Validate inputs
	if startingPrice <= 0 {
		return 0, nil, fmt.Errorf("starting price must be positive")
	}
	if slPercentage <= 0 || slPercentage > 5 {
		return 0, nil, fmt.Errorf("stop loss percentage must be between 0 and 5")
	}
	if riskAmount <= 0 {
		return 0, nil, fmt.Errorf("risk amount must be positive")
	}

	// Calculate Fibonacci levels
	fibLevels := s.fibCalculator.CalculateFibonacciLevels(startingPrice, slPercentage, tradeSide)

	// Calculate the stop loss points for position sizing
	var slPoints float64
	if tradeSide == domain.Buy {
		slPoints = startingPrice - fibLevels[0].Price
	} else {
		slPoints = fibLevels[0].Price - startingPrice
	}

	// Calculate total quantity based on risk
	totalQuantity := int(math.Floor(riskAmount / slPoints))
	if totalQuantity <= 0 {
		return 0, nil, fmt.Errorf("calculated quantity is too small, consider increasing risk amount or reducing stop loss distance")
	}

	// Calculate quantity per leg (distribute across 5 entry legs)
	legCount := 5
	baseQtyPerLeg := totalQuantity / legCount
	remainder := totalQuantity % legCount

	// Create level entries with quantities
	levelsWithQuantity := make([]domain.LevelWithQuantity, len(fibLevels))

	for i, level := range fibLevels {
		qty := 0
		if i > 0 { // Skip stop loss level for quantity
			qty = baseQtyPerLeg
			if i-1 < remainder {
				qty++
			}
		}

		levelsWithQuantity[i] = domain.LevelWithQuantity{
			Level:       level.Level,
			Price:       level.Price,
			Description: level.Description,
			Quantity:    qty,
		}
	}

	return totalQuantity, levelsWithQuantity, nil
}

func (s *UtilityService) getLowestMinBBWidth(instrumentKey string) (float64, error) {
	// Read from the CSV file directly
	csvPath := "python_strategies/output/bb_width_analysis.csv"

	// Read the CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open CSV file %s: %w", csvPath, err)
	}
	defer file.Close()

	// Create CSV reader
	reader := csv.NewReader(file)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Find the column indices
	instrumentKeyIndex := -1
	lowestMinBBWidthIndex := -1

	for i, col := range header {
		switch col {
		case "instrument_key":
			instrumentKeyIndex = i
		case "lowest_min_bb_width":
			lowestMinBBWidthIndex = i
		}
	}

	if instrumentKeyIndex == -1 || lowestMinBBWidthIndex == -1 {
		return 0, fmt.Errorf("required columns not found in CSV: instrument_key=%d, lowest_min_bb_width=%d",
			instrumentKeyIndex, lowestMinBBWidthIndex)
	}

	// Search for the instrument key
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to read CSV record: %w", err)
		}

		if len(record) <= instrumentKeyIndex || len(record) <= lowestMinBBWidthIndex {
			continue // Skip malformed records
		}

		if record[instrumentKeyIndex] == instrumentKey {
			// Parse the lowest_min_bb_width value
			lowestMinBBWidth, err := strconv.ParseFloat(record[lowestMinBBWidthIndex], 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse lowest_min_bb_width value '%s' for %s: %w",
					record[lowestMinBBWidthIndex], instrumentKey, err)
			}

			log.Debug("[BB Monitor] Retrieved lowest BB width for %s from CSV: %f", instrumentKey, lowestMinBBWidth)
			return lowestMinBBWidth, nil
		}
	}

	return 0, fmt.Errorf("instrument key %s not found in CSV file", instrumentKey)
}
