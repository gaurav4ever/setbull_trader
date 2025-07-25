package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"setbull_trader/internal/service"
)

// BBWDashboardHandler handles BBW dashboard API requests
type BBWDashboardHandler struct {
	bbwDashboardService *service.BBWDashboardService
}

// NewBBWDashboardHandler creates a new BBW dashboard handler
func NewBBWDashboardHandler(bbwDashboardService *service.BBWDashboardService) *BBWDashboardHandler {
	return &BBWDashboardHandler{
		bbwDashboardService: bbwDashboardService,
	}
}

// GetDashboardData returns all BBW dashboard data
func (h *BBWDashboardHandler) GetDashboardData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get dashboard data
	dashboardData := h.bbwDashboardService.GetDashboardData()

	// Create response
	response := map[string]interface{}{
		"status":    "success",
		"data":      dashboardData,
		"count":     len(dashboardData),
		"timestamp": time.Now(),
	}

	respondSuccess(w, response)
}

// GetStockBBWData returns BBW data for a specific stock
func (h *BBWDashboardHandler) GetStockBBWData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract instrument key from query parameters
	instrumentKey := r.URL.Query().Get("instrument_key")
	if instrumentKey == "" {
		respondWithError(w, http.StatusBadRequest, "instrument_key parameter is required")
		return
	}

	// Get stock BBW data
	bbwData, exists := h.bbwDashboardService.GetStockBBWData(instrumentKey)
	if !exists {
		respondWithError(w, http.StatusNotFound, "BBW data not found for instrument key: "+instrumentKey)
		return
	}

	// Create response
	response := map[string]interface{}{
		"status":    "success",
		"data":      bbwData,
		"timestamp": time.Now(),
	}

	respondSuccess(w, response)
}

// UpdateAlertThreshold updates the alert threshold for BBW monitoring
func (h *BBWDashboardHandler) UpdateAlertThreshold(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var request struct {
		Threshold float64 `json:"threshold"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format: "+err.Error())
		return
	}

	// Validate threshold
	if request.Threshold < 0 || request.Threshold > 100 {
		respondWithError(w, http.StatusBadRequest, "Threshold must be between 0 and 100")
		return
	}

	// Update threshold
	h.bbwDashboardService.SetAlertThreshold(request.Threshold)

	// Create response
	response := map[string]interface{}{
		"status":    "success",
		"message":   "Alert threshold updated successfully",
		"threshold": request.Threshold,
		"timestamp": time.Now(),
	}

	respondSuccess(w, response)
}

// UpdateContractingLookback updates the contracting lookback period
func (h *BBWDashboardHandler) UpdateContractingLookback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var request struct {
		Lookback int `json:"lookback"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format: "+err.Error())
		return
	}

	// Validate lookback
	if request.Lookback < 1 || request.Lookback > 20 {
		respondWithError(w, http.StatusBadRequest, "Lookback must be between 1 and 20")
		return
	}

	// Update lookback
	h.bbwDashboardService.SetContractingLookback(request.Lookback)

	// Create response
	response := map[string]interface{}{
		"status":    "success",
		"message":   "Contracting lookback updated successfully",
		"lookback":  request.Lookback,
		"timestamp": time.Now(),
	}

	respondSuccess(w, response)
}

// GetDashboardStats returns dashboard statistics
func (h *BBWDashboardHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get dashboard data
	dashboardData := h.bbwDashboardService.GetDashboardData()

	// Calculate statistics
	var (
		totalStocks        = len(dashboardData)
		alertedStocks      = 0
		contractingStocks  = 0
		expandingStocks    = 0
		stableStocks       = 0
		avgDistanceFromMin = 0.0
	)

	for _, data := range dashboardData {
		if data.AlertTriggered {
			alertedStocks++
		}

		switch data.BBWidthTrend {
		case "contracting":
			contractingStocks++
		case "expanding":
			expandingStocks++
		case "stable":
			stableStocks++
		}

		avgDistanceFromMin += data.DistanceFromMinPercent
	}

	if totalStocks > 0 {
		avgDistanceFromMin = avgDistanceFromMin / float64(totalStocks)
	}

	// Create response
	stats := map[string]interface{}{
		"total_stocks":          totalStocks,
		"alerted_stocks":        alertedStocks,
		"contracting_stocks":    contractingStocks,
		"expanding_stocks":      expandingStocks,
		"stable_stocks":         stableStocks,
		"avg_distance_from_min": avgDistanceFromMin,
		"market_hours":          h.bbwDashboardService.IsMarketHours(),
	}

	response := map[string]interface{}{
		"status":    "success",
		"data":      stats,
		"timestamp": time.Now(),
	}

	respondSuccess(w, response)
}

// GetStockBBWHistory returns historical BBW data for a stock
func (h *BBWDashboardHandler) GetStockBBWHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract parameters
	instrumentKey := r.URL.Query().Get("instrument_key")
	if instrumentKey == "" {
		respondWithError(w, http.StatusBadRequest, "instrument_key parameter is required")
		return
	}

	// Parse timeframe (default: 1 day)
	timeframeStr := r.URL.Query().Get("timeframe")
	if timeframeStr == "" {
		timeframeStr = "1d"
	}

	// Calculate time range
	end := time.Now()
	var start time.Time

	switch timeframeStr {
	case "1h":
		start = end.Add(-1 * time.Hour)
	case "4h":
		start = end.Add(-4 * time.Hour)
	case "1d":
		start = end.Add(-24 * time.Hour)
	case "1w":
		start = end.Add(-7 * 24 * time.Hour)
	default:
		respondWithError(w, http.StatusBadRequest, "Invalid timeframe. Use: 1h, 4h, 1d, 1w")
		return
	}

	// Get historical BBW data
	// Note: This would need to be implemented in the BBWDashboardService
	// For now, return a placeholder response
	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"instrument_key": instrumentKey,
			"timeframe":      timeframeStr,
			"start_time":     start,
			"end_time":       end,
			"history":        []interface{}{}, // Placeholder for historical data
		},
		"timestamp": time.Now(),
	}

	respondSuccess(w, response)
}
