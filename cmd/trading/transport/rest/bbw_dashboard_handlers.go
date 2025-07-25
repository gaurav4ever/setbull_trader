package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"setbull_trader/internal/service"
	"setbull_trader/pkg/log"
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
	log.BBWInfo("api_handler", "get_dashboard_data", "Dashboard data request received", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.UserAgent(),
	})

	data := h.bbwDashboardService.GetDashboardData()

	response := map[string]interface{}{
		"success":   true,
		"data":      data,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.BBWError("api_handler", "encode_response", "Failed to encode dashboard data response", err, map[string]interface{}{
			"remote_addr": r.RemoteAddr,
			"data_count":  len(data),
		})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.BBWInfo("api_handler", "dashboard_data_sent", "Dashboard data sent successfully", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
		"data_count":  len(data),
	})
}

// GetStockBBWData returns BBW data for a specific stock
func (h *BBWDashboardHandler) GetStockBBWData(w http.ResponseWriter, r *http.Request) {
	instrumentKey := r.URL.Query().Get("instrument_key")
	if instrumentKey == "" {
		log.BBWWarn("api_handler", "missing_instrument_key", "Missing instrument_key parameter", map[string]interface{}{
			"remote_addr":  r.RemoteAddr,
			"query_params": r.URL.RawQuery,
		})
		http.Error(w, "instrument_key parameter is required", http.StatusBadRequest)
		return
	}

	log.BBWInfo("api_handler", "get_stock_data", "Stock BBW data request received", map[string]interface{}{
		"remote_addr":    r.RemoteAddr,
		"instrument_key": instrumentKey,
	})

	data, exists := h.bbwDashboardService.GetStockBBWData(instrumentKey)
	if !exists {
		log.BBWInfo("api_handler", "stock_not_found", "Stock not found in dashboard data", map[string]interface{}{
			"remote_addr":    r.RemoteAddr,
			"instrument_key": instrumentKey,
		})
		http.Error(w, "Stock not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"data":      data,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.BBWError("api_handler", "encode_stock_response", "Failed to encode stock data response", err, map[string]interface{}{
			"remote_addr":    r.RemoteAddr,
			"instrument_key": instrumentKey,
		})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.BBWInfo("api_handler", "stock_data_sent", "Stock BBW data sent successfully", map[string]interface{}{
		"remote_addr":    r.RemoteAddr,
		"instrument_key": instrumentKey,
		"symbol":         data.Symbol,
	})
}

// GetActiveAlerts returns currently active alerts
func (h *BBWDashboardHandler) GetActiveAlerts(w http.ResponseWriter, r *http.Request) {
	log.BBWInfo("api_handler", "get_active_alerts", "Active alerts request received", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
	})

	// Get all dashboard data and filter for active alerts
	allData := h.bbwDashboardService.GetDashboardData()
	var activeAlerts []*service.BBWDashboardData

	for _, data := range allData {
		if data.AlertTriggered {
			activeAlerts = append(activeAlerts, data)
		}
	}

	response := map[string]interface{}{
		"success":   true,
		"data":      activeAlerts,
		"count":     len(activeAlerts),
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.BBWError("api_handler", "encode_alerts_response", "Failed to encode active alerts response", err, map[string]interface{}{
			"remote_addr": r.RemoteAddr,
			"alert_count": len(activeAlerts),
		})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.BBWInfo("api_handler", "active_alerts_sent", "Active alerts sent successfully", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
		"alert_count": len(activeAlerts),
	})
}

// GetAlertHistory returns alert history with optional filtering
func (h *BBWDashboardHandler) GetAlertHistory(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	alertType := r.URL.Query().Get("alert_type")
	symbol := r.URL.Query().Get("symbol")

	log.BBWInfo("api_handler", "get_alert_history", "Alert history request received", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
		"limit":       limitStr,
		"alert_type":  alertType,
		"symbol":      symbol,
	})

	limit := 50 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	history := h.bbwDashboardService.GetAlertHistory()

	// Apply filters
	var filteredHistory []service.AlertEvent
	for _, alert := range history {
		if alertType != "" && alert.AlertType != alertType {
			continue
		}
		if symbol != "" && alert.Symbol != symbol {
			continue
		}
		filteredHistory = append(filteredHistory, alert)
	}

	// Apply limit
	if len(filteredHistory) > limit {
		filteredHistory = filteredHistory[:limit]
	}

	response := map[string]interface{}{
		"success":   true,
		"data":      filteredHistory,
		"count":     len(filteredHistory),
		"total":     len(history),
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.BBWError("api_handler", "encode_history_response", "Failed to encode alert history response", err, map[string]interface{}{
			"remote_addr":    r.RemoteAddr,
			"filtered_count": len(filteredHistory),
			"total_count":    len(history),
		})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.BBWInfo("api_handler", "alert_history_sent", "Alert history sent successfully", map[string]interface{}{
		"remote_addr":    r.RemoteAddr,
		"filtered_count": len(filteredHistory),
		"total_count":    len(history),
		"limit":          limit,
	})
}

// ClearAlertHistory clears the alert history
func (h *BBWDashboardHandler) ClearAlertHistory(w http.ResponseWriter, r *http.Request) {
	log.BBWInfo("api_handler", "clear_alert_history", "Clear alert history request received", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
	})

	h.bbwDashboardService.ClearAlertHistory()

	response := map[string]interface{}{
		"success":   true,
		"message":   "Alert history cleared successfully",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.BBWError("api_handler", "encode_clear_response", "Failed to encode clear history response", err, map[string]interface{}{
			"remote_addr": r.RemoteAddr,
		})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.BBWInfo("api_handler", "alert_history_cleared", "Alert history cleared successfully", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
	})
}

// ConfigureAlerts updates alert configuration
func (h *BBWDashboardHandler) ConfigureAlerts(w http.ResponseWriter, r *http.Request) {
	log.BBWInfo("api_handler", "configure_alerts", "Configure alerts request received", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
	})

	var config struct {
		AlertThreshold      float64 `json:"alert_threshold"`
		ContractingLookback int     `json:"contracting_lookback"`
		EnableAlerts        bool    `json:"enable_alerts"`
	}

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		log.BBWError("api_handler", "decode_config", "Failed to decode alert configuration", err, map[string]interface{}{
			"remote_addr": r.RemoteAddr,
		})
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.BBWInfo("api_handler", "config_received", "Alert configuration received", map[string]interface{}{
		"remote_addr":          r.RemoteAddr,
		"alert_threshold":      config.AlertThreshold,
		"contracting_lookback": config.ContractingLookback,
		"enable_alerts":        config.EnableAlerts,
	})

	// Update service configuration
	if config.AlertThreshold > 0 {
		h.bbwDashboardService.SetAlertThreshold(config.AlertThreshold)
	}
	if config.ContractingLookback > 0 {
		h.bbwDashboardService.SetContractingLookback(config.ContractingLookback)
	}

	response := map[string]interface{}{
		"success":   true,
		"message":   "Alert configuration updated successfully",
		"config":    config,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.BBWError("api_handler", "encode_config_response", "Failed to encode configuration response", err, map[string]interface{}{
			"remote_addr": r.RemoteAddr,
		})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.BBWInfo("api_handler", "alerts_configured", "Alert configuration updated successfully", map[string]interface{}{
		"remote_addr":          r.RemoteAddr,
		"alert_threshold":      config.AlertThreshold,
		"contracting_lookback": config.ContractingLookback,
		"enable_alerts":        config.EnableAlerts,
	})
}

// UpdateAlertThreshold updates the alert threshold for BBW monitoring
func (h *BBWDashboardHandler) UpdateAlertThreshold(w http.ResponseWriter, r *http.Request) {
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
	log.BBWInfo("api_handler", "get_dashboard_stats", "Dashboard stats request received", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
	})

	data := h.bbwDashboardService.GetDashboardData()

	// Calculate statistics
	var minBBW, maxBBW, totalBBW float64
	var alertedCount, contractingCount, expandingCount, stableCount int
	var recentAlerts []service.AlertEvent

	if len(data) > 0 {
		minBBW = data[0].CurrentBBWidth
		maxBBW = data[0].CurrentBBWidth

		for _, item := range data {
			// BBW range
			if item.CurrentBBWidth < minBBW {
				minBBW = item.CurrentBBWidth
			}
			if item.CurrentBBWidth > maxBBW {
				maxBBW = item.CurrentBBWidth
			}
			totalBBW += item.CurrentBBWidth

			// Count by trend
			switch item.BBWidthTrend {
			case "contracting":
				contractingCount++
			case "expanding":
				expandingCount++
			case "stable":
				stableCount++
			}

			// Count alerts
			if item.AlertTriggered {
				alertedCount++
			}
		}
	}

	// Get recent alerts
	alertHistory := h.bbwDashboardService.GetAlertHistory()
	if len(alertHistory) > 5 {
		recentAlerts = alertHistory[len(alertHistory)-5:]
	} else {
		recentAlerts = alertHistory
	}

	avgBBW := 0.0
	if len(data) > 0 {
		avgBBW = totalBBW / float64(len(data))
	}

	stats := map[string]interface{}{
		"total_stocks":      len(data),
		"alerted_stocks":    alertedCount,
		"contracting_count": contractingCount,
		"expanding_count":   expandingCount,
		"stable_count":      stableCount,
		"min_bb_width":      minBBW,
		"max_bb_width":      maxBBW,
		"avg_bb_width":      avgBBW,
		"recent_alerts":     recentAlerts,
		"total_alerts":      len(alertHistory),
	}

	response := map[string]interface{}{
		"success":   true,
		"data":      stats,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.BBWError("api_handler", "encode_stats_response", "Failed to encode dashboard stats response", err, map[string]interface{}{
			"remote_addr":  r.RemoteAddr,
			"total_stocks": len(data),
		})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.BBWInfo("api_handler", "dashboard_stats_sent", "Dashboard stats sent successfully", map[string]interface{}{
		"remote_addr":       r.RemoteAddr,
		"total_stocks":      len(data),
		"alerted_count":     alertedCount,
		"contracting_count": contractingCount,
		"expanding_count":   expandingCount,
		"stable_count":      stableCount,
	})
}

// GetLatestAvailableDayData returns the most recent available BBW data for all monitored stocks
// regardless of market hours - useful for dashboard access outside trading hours
func (h *BBWDashboardHandler) GetLatestAvailableDayData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get the latest available day data
	dashboardData, err := h.bbwDashboardService.GetLatestAvailableDayData(ctx)
	if err != nil {
		log.Error("[BBW Handler] Failed to get latest available day data: %v", err)
		http.Error(w, "Failed to get latest available day data", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var response []map[string]interface{}
	for _, data := range dashboardData {
		response = append(response, map[string]interface{}{
			"symbol":                     data.Symbol,
			"instrument_key":             data.InstrumentKey,
			"current_bb_width":           data.CurrentBBWidth,
			"historical_min_bb_width":    data.HistoricalMinBBWidth,
			"distance_from_min_percent":  data.DistanceFromMinPercent,
			"contracting_sequence_count": data.ContractingSequenceCount,
			"bb_width_trend":             data.BBWidthTrend,
			"alert_triggered":            data.AlertTriggered,
			"alert_triggered_at":         data.AlertTriggeredAt,
			"alert_type":                 data.AlertType,
			"alert_message":              data.AlertMessage,
			"pattern_strength":           data.PatternStrength,
			"timestamp":                  data.Timestamp,
			"last_updated":               data.LastUpdated,
		})
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    response,
		"count":   len(response),
		"message": "Latest available day data retrieved successfully",
	})
}

// GetStockBBWHistory returns historical BBW data for a specific stock
func (h *BBWDashboardHandler) GetStockBBWHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get query parameters
	instrumentKey := r.URL.Query().Get("instrument_key")
	if instrumentKey == "" {
		http.Error(w, "instrument_key parameter is required", http.StatusBadRequest)
		return
	}

	// Parse days parameter (default to 7 days)
	daysStr := r.URL.Query().Get("days")
	days := 7 // default
	if daysStr != "" {
		if parsedDays, err := strconv.Atoi(daysStr); err == nil && parsedDays > 0 {
			days = parsedDays
		}
	}

	// Get historical data
	historicalData, err := h.bbwDashboardService.GetStockBBWHistory(ctx, instrumentKey, days)
	if err != nil {
		log.Error("[BBW Handler] Failed to get stock BBW history for %s: %v", instrumentKey, err)
		http.Error(w, "Failed to get stock BBW history", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var response []map[string]interface{}
	for _, data := range historicalData {
		response = append(response, map[string]interface{}{
			"symbol":                     data.Symbol,
			"instrument_key":             data.InstrumentKey,
			"current_bb_width":           data.CurrentBBWidth,
			"historical_min_bb_width":    data.HistoricalMinBBWidth,
			"distance_from_min_percent":  data.DistanceFromMinPercent,
			"contracting_sequence_count": data.ContractingSequenceCount,
			"bb_width_trend":             data.BBWidthTrend,
			"alert_triggered":            data.AlertTriggered,
			"alert_triggered_at":         data.AlertTriggeredAt,
			"alert_type":                 data.AlertType,
			"alert_message":              data.AlertMessage,
			"pattern_strength":           data.PatternStrength,
			"timestamp":                  data.Timestamp,
			"last_updated":               data.LastUpdated,
		})
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"data":           response,
		"count":          len(response),
		"instrument_key": instrumentKey,
		"days":           days,
		"message":        "Stock BBW history retrieved successfully",
	})
}

// GetMarketStatus returns current market status and last available data timestamp
func (h *BBWDashboardHandler) GetMarketStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if market is currently open
	isMarketOpen := h.bbwDashboardService.IsMarketHours()

	// Get current time in IST
	now := time.Now()
	ist := now.UTC().Add(5*time.Hour + 30*time.Minute)

	// Get the latest available data timestamp
	latestData, err := h.bbwDashboardService.GetLatestAvailableDayData(ctx)
	var lastDataTimestamp *time.Time
	if err == nil && len(latestData) > 0 {
		// Find the most recent timestamp
		for _, data := range latestData {
			if lastDataTimestamp == nil || data.Timestamp.After(*lastDataTimestamp) {
				lastDataTimestamp = &data.Timestamp
			}
		}
	}

	// Return market status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"market_open":         isMarketOpen,
			"current_time":        ist.Format("2006-01-02 15:04:05"),
			"current_time_ist":    ist.Format("15:04:05"),
			"market_hours":        "09:15 - 15:30 IST",
			"last_data_timestamp": lastDataTimestamp,
			"last_data_age_minutes": func() int {
				if lastDataTimestamp == nil {
					return -1
				}
				return int(time.Since(*lastDataTimestamp).Minutes())
			}(),
		},
		"message": "Market status retrieved successfully",
	})
}

// GetStatistics returns market-wide BBW statistics
func (h *BBWDashboardHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	log.BBWInfo("api_handler", "get_statistics", "Market statistics request received", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
	})

	data := h.bbwDashboardService.GetDashboardData()

	// Calculate market-wide statistics
	var totalBBW, minBBW, maxBBW float64
	var alertDistribution map[string]int
	var trendDistribution map[string]int

	if len(data) > 0 {
		minBBW = data[0].CurrentBBWidth
		maxBBW = data[0].CurrentBBWidth
		alertDistribution = make(map[string]int)
		trendDistribution = make(map[string]int)

		for _, item := range data {
			// BBW range
			if item.CurrentBBWidth < minBBW {
				minBBW = item.CurrentBBWidth
			}
			if item.CurrentBBWidth > maxBBW {
				maxBBW = item.CurrentBBWidth
			}
			totalBBW += item.CurrentBBWidth

			// Alert distribution
			if item.AlertTriggered {
				alertDistribution[item.AlertType]++
			}

			// Trend distribution
			trendDistribution[item.BBWidthTrend]++
		}
	}

	avgBBW := 0.0
	if len(data) > 0 {
		avgBBW = totalBBW / float64(len(data))
	}

	stats := map[string]interface{}{
		"total_stocks":          len(data),
		"avg_bb_width":          avgBBW,
		"min_bb_width":          minBBW,
		"max_bb_width":          maxBBW,
		"alert_distribution":    alertDistribution,
		"trend_distribution":    trendDistribution,
		"market_volatility":     "medium", // Placeholder
		"squeeze_opportunities": len(alertDistribution),
	}

	response := map[string]interface{}{
		"success":   true,
		"data":      stats,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.BBWError("api_handler", "encode_market_stats_response", "Failed to encode market statistics response", err, map[string]interface{}{
			"remote_addr":  r.RemoteAddr,
			"total_stocks": len(data),
		})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.BBWInfo("api_handler", "market_stats_sent", "Market statistics sent successfully", map[string]interface{}{
		"remote_addr":           r.RemoteAddr,
		"total_stocks":          len(data),
		"avg_bb_width":          avgBBW,
		"squeeze_opportunities": len(alertDistribution),
	})
}
