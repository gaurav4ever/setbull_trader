package rest

import (
	"net/http"
	"strconv"

	"setbull_trader/internal/domain"
)

// CalculateFibonacciLevels calculates Fibonacci levels without creating an execution plan
func (s *Server) CalculateFibonacciLevels(w http.ResponseWriter, r *http.Request) {
	// ctx := r.Context()
	query := r.URL.Query()

	// Get and validate query parameters
	startingPriceStr := query.Get("startingPrice")
	if startingPriceStr == "" {
		respondWithError(w, http.StatusBadRequest, "Starting price is required")
		return
	}
	startingPrice, err := strconv.ParseFloat(startingPriceStr, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid starting price: "+err.Error())
		return
	}

	slPercentageStr := query.Get("slPercentage")
	if slPercentageStr == "" {
		respondWithError(w, http.StatusBadRequest, "Stop loss percentage is required")
		return
	}
	slPercentage, err := strconv.ParseFloat(slPercentageStr, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid stop loss percentage: "+err.Error())
		return
	}

	tradeSideStr := query.Get("tradeSide")
	if tradeSideStr == "" {
		respondWithError(w, http.StatusBadRequest, "Trade side is required")
		return
	}
	tradeSide := domain.TradeSide(tradeSideStr)
	if tradeSide != domain.Buy && tradeSide != domain.Sell {
		respondWithError(w, http.StatusBadRequest, "Trade side must be either BUY or SELL")
		return
	}

	// Optional risk amount parameter with default value
	riskAmountStr := query.Get("riskAmount")
	riskAmount := 30.0 // Default value
	if riskAmountStr != "" {
		riskAmount, err = strconv.ParseFloat(riskAmountStr, 64)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid risk amount: "+err.Error())
			return
		}
	}

	// Use the utility service to calculate levels and quantities
	totalQuantity, levelsWithQuantity, err := s.utilityService.CalculateFibonacciLevelsWithQuantities(
		startingPrice,
		slPercentage,
		tradeSide,
		riskAmount,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to calculate Fibonacci levels: "+err.Error())
		return
	}

	// Create response
	response := domain.FibonacciLevelsResponse{
		TotalQuantity: totalQuantity,
		Levels:        levelsWithQuantity,
	}

	respondSuccess(w, response)
}
