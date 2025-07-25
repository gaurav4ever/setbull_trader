package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"setbull_trader/internal/core/dto/response"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
)

// BBWDashboardData represents the data structure sent to the frontend
type BBWDashboardData struct {
	Symbol                   string     `json:"symbol"`
	InstrumentKey            string     `json:"instrument_key"`
	CurrentBBWidth           float64    `json:"current_bb_width"`
	HistoricalMinBBWidth     float64    `json:"historical_min_bb_width"`
	DistanceFromMinPercent   float64    `json:"distance_from_min_percent"`
	ContractingSequenceCount int        `json:"contracting_sequence_count"`
	CandlesInRangeCount      int        `json:"candles_in_range_count"` // NEW: Number of consecutive candles in optimal range
	BBWidthTrend             string     `json:"bb_width_trend"`         // "contracting", "expanding", "stable"
	AlertTriggered           bool       `json:"alert_triggered"`
	AlertTriggeredAt         *time.Time `json:"alert_triggered_at,omitempty"`
	AlertType                string     `json:"alert_type,omitempty"` // "threshold", "pattern", "squeeze"
	AlertMessage             string     `json:"alert_message,omitempty"`
	PatternStrength          string     `json:"pattern_strength,omitempty"` // "weak", "moderate", "strong"
	Timestamp                time.Time  `json:"timestamp"`
	LastUpdated              time.Time  `json:"last_updated"`
}

// BBWDashboardService provides real-time BBW data for the dashboard
type BBWDashboardService struct {
	candleAggService      *CandleAggregationService
	technicalIndicatorSvc *TechnicalIndicatorService
	stockGroupService     *StockGroupService
	universeService       *StockUniverseService
	websocketHub          *WebSocketHub
	alertService          *AlertService                   // NEW: Alert service integration
	utilityService        *UtilityService                 // NEW: Utility service for historical data
	candle5MinRepo        repository.Candle5MinRepository // NEW: Direct repository access
	mu                    sync.RWMutex
	monitoredStocks       map[string]*BBWDashboardData
	alertThreshold        float64      // 0.1% default
	contractingLookback   int          // 5 candles default
	alertHistory          []AlertEvent // NEW: Alert history tracking
	alertHistoryMutex     sync.RWMutex
}

// NewBBWDashboardService creates a new BBW dashboard service
func NewBBWDashboardService(
	candleAggService *CandleAggregationService,
	technicalIndicatorSvc *TechnicalIndicatorService,
	stockGroupService *StockGroupService,
	universeService *StockUniverseService,
	websocketHub *WebSocketHub,
	alertService *AlertService, // NEW: Alert service parameter
	utilityService *UtilityService, // NEW: Utility service parameter
	candle5MinRepo repository.Candle5MinRepository, // NEW: Candle5Min repository parameter
) *BBWDashboardService {
	return &BBWDashboardService{
		candleAggService:      candleAggService,
		technicalIndicatorSvc: technicalIndicatorSvc,
		stockGroupService:     stockGroupService,
		universeService:       universeService,
		websocketHub:          websocketHub,
		alertService:          alertService,   // NEW: Store alert service
		utilityService:        utilityService, // NEW: Store utility service
		candle5MinRepo:        candle5MinRepo, // NEW: Store candle5Min repository
		monitoredStocks:       make(map[string]*BBWDashboardData),
		alertThreshold:        0.1,                   // 0.1%
		contractingLookback:   5,                     // 5 candles
		alertHistory:          make([]AlertEvent, 0), // NEW: Initialize alert history
	}
}

// OnFiveMinCandleClose is called when a 5-minute candle closes
// This integrates with your existing 5-minute candle infrastructure
func (s *BBWDashboardService) OnFiveMinCandleClose(ctx context.Context, start, end time.Time) error {
	log.BBWInfo("candle_processing", "start", "Processing 5-minute candle close", map[string]interface{}{
		"start_time":   start.Format("15:04"),
		"end_time":     end.Format("15:04"),
		"market_hours": s.IsMarketHours(),
	})

	// Check if we're within market hours
	if !s.IsMarketHours() {
		log.BBWDebug("candle_processing", "skip", "Outside market hours, skipping BBW processing", map[string]interface{}{
			"start_time": start.Format("15:04"),
			"end_time":   end.Format("15:04"),
		})
		return nil
	}

	// Get all stocks that need BBW monitoring
	stocks, err := s.getMonitoredStocks(ctx)
	if err != nil {
		log.BBWError("candle_processing", "get_stocks", "Failed to get monitored stocks", err, map[string]interface{}{
			"start_time": start.Format("15:04"),
			"end_time":   end.Format("15:04"),
		})
		return err
	}

	if len(stocks) == 0 {
		log.BBWDebug("candle_processing", "no_stocks", "No stocks to monitor", map[string]interface{}{
			"start_time": start.Format("15:04"),
			"end_time":   end.Format("15:04"),
		})
		return nil
	}

	log.BBWInfo("candle_processing", "process_start", "Processing BBW data for stocks", map[string]interface{}{
		"stock_count": len(stocks),
		"start_time":  start.Format("15:04"),
		"end_time":    end.Format("15:04"),
	})

	// Process each stock concurrently
	var wg sync.WaitGroup
	results := make(chan *BBWDashboardData, len(stocks))

	for _, stock := range stocks {
		wg.Add(1)
		go func(stock domain.StockUniverse) {
			defer wg.Done()
			bbwData, err := s.processStockBBW(ctx, stock, start, end)
			if err != nil {
				log.BBWError("candle_processing", "process_stock", "Failed to process BBW for stock", err, map[string]interface{}{
					"symbol":         stock.Symbol,
					"instrument_key": stock.InstrumentKey,
					"start_time":     start.Format("15:04"),
					"end_time":       end.Format("15:04"),
				})
				return
			}
			if bbwData != nil {
				results <- bbwData
			}
		}(stock)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var dashboardData []*BBWDashboardData
	for bbwData := range results {
		dashboardData = append(dashboardData, bbwData)
	}

	// Update dashboard cache and broadcast updates
	if len(dashboardData) > 0 {
		s.updateDashboardCache(dashboardData)
		s.broadcastDashboardUpdate(dashboardData)
	}

	log.BBWInfo("candle_processing", "process_complete", "Completed processing stocks", map[string]interface{}{
		"processed_count": len(dashboardData),
		"total_stocks":    len(stocks),
		"start_time":      start.Format("15:04"),
		"end_time":        end.Format("15:04"),
	})
	return nil
}

// processStockBBW processes BBW data for a single stock
func (s *BBWDashboardService) processStockBBW(ctx context.Context, stock domain.StockUniverse, start, end time.Time) (*BBWDashboardData, error) {
	log.BBWDebug("stock_processing", "start", "Processing BBW data for stock", map[string]interface{}{
		"symbol":         stock.Symbol,
		"instrument_key": stock.InstrumentKey,
		"start_time":     start.Format("15:04"),
		"end_time":       end.Format("15:04"),
	})

	// Get BBW values for the stock within the specified time range
	bbwValues, err := s.getRecentBBWValuesInRange(ctx, stock.InstrumentKey, s.contractingLookback+1, start, end)
	if err != nil {
		log.BBWError("stock_processing", "get_bbw_values", "Failed to get BBW values", err, map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"lookback":       s.contractingLookback + 1,
		})
		return nil, fmt.Errorf("failed to get BBW values: %w", err)
	}

	log.BBWDebug("stock_processing", "bbw_values_retrieved", "Retrieved BBW values for processing", map[string]interface{}{
		"symbol":         stock.Symbol,
		"instrument_key": stock.InstrumentKey,
		"data_points":    len(bbwValues),
		"required":       2,
		"values":         bbwValues,
	})

	if len(bbwValues) < 2 {
		log.BBWDebug("stock_processing", "insufficient_data", "Insufficient BBW data for stock", map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"data_points":    len(bbwValues),
			"required":       2,
		})
		return nil, nil
	}

	// Calculate current BBW
	if len(bbwValues) == 0 {
		log.BBWError("stock_processing", "no_bbw_values", "No BBW values available", nil, map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
		})
		return nil, fmt.Errorf("no BBW values available")
	}
	currentBBW := bbwValues[len(bbwValues)-1]

	// Get historical minimum BBW from CSV/database
	historicalMinBBW, err := s.getHistoricalMinBBW(ctx, stock.InstrumentKey)
	if err != nil {
		log.BBWError("stock_processing", "historical_min", "Failed to get historical minimum BBW", err, map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
		})
		// Use recent minimum as fallback
		historicalMinBBW = s.calculateRecentMinBBW(bbwValues)
		log.BBWWarn("stock_processing", "fallback", "Using recent minimum as fallback", map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"fallback_value": historicalMinBBW,
		})
	}

	// Calculate distance from minimum
	distanceFromMin := s.calculateDistanceFromMin(currentBBW, historicalMinBBW)

	// Detect contracting pattern
	contractingCount := s.detectContractingPattern(bbwValues)

	// Determine BBW trend
	trend := s.determineBBWTrend(bbwValues)

	// Calculate candles in range count
	candlesInRangeCount := s.calculateCandlesInRangeCount(ctx, stock.InstrumentKey, historicalMinBBW)

	// Persist candles in range count to database
	if err := s.updateCandlesInRangeCount(ctx, stock.InstrumentKey, candlesInRangeCount); err != nil {
		log.BBWError("stock_processing", "persistence_failed", "Failed to persist candles in range count", err, map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"count":          candlesInRangeCount,
		})
		// Continue processing even if persistence fails
	}

	// Check alert conditions and trigger alerts
	alertTriggered, alertType, alertMessage, patternStrength := s.checkAdvancedAlertConditions(
		stock.InstrumentKey, stock.Symbol, currentBBW, historicalMinBBW, contractingCount, bbwValues)

	// Create dashboard data
	dashboardData := &BBWDashboardData{
		Symbol:                   stock.Symbol,
		InstrumentKey:            stock.InstrumentKey,
		CurrentBBWidth:           currentBBW,
		HistoricalMinBBWidth:     historicalMinBBW,
		DistanceFromMinPercent:   distanceFromMin,
		ContractingSequenceCount: contractingCount,
		CandlesInRangeCount:      candlesInRangeCount,
		BBWidthTrend:             trend,
		AlertTriggered:           alertTriggered,
		AlertType:                alertType,
		AlertMessage:             alertMessage,
		PatternStrength:          patternStrength,
		Timestamp:                end, // Use the end time (candle timestamp) instead of current time
		LastUpdated:              time.Now(),
	}

	// Set alert timestamp if triggered
	if alertTriggered {
		now := time.Now()
		dashboardData.AlertTriggeredAt = &now
	}

	log.BBWDebug("stock_processing", "complete", "Completed processing stock BBW data", map[string]interface{}{
		"symbol":             stock.Symbol,
		"instrument_key":     stock.InstrumentKey,
		"current_bbw":        currentBBW,
		"historical_min_bbw": historicalMinBBW,
		"distance_percent":   distanceFromMin,
		"contracting_count":  contractingCount,
		"trend":              trend,
		"alert_triggered":    alertTriggered,
		"alert_type":         alertType,
		"pattern_strength":   patternStrength,
	})

	return dashboardData, nil
}

// getHistoricalMinBBW gets the historical minimum BBW from CSV/database
func (s *BBWDashboardService) getHistoricalMinBBW(ctx context.Context, instrumentKey string) (float64, error) {
	if s.utilityService == nil {
		return 0, fmt.Errorf("utility service not available")
	}

	// Use existing utilityService method that reads from CSV
	historicalMinBBW, err := s.utilityService.getLowestMinBBWidth(instrumentKey)
	if err != nil {
		return 0, fmt.Errorf("failed to get historical minimum BBW: %w", err)
	}

	log.BBWDebug("historical_min", "retrieved", "Retrieved historical minimum BBW", map[string]interface{}{
		"instrument_key":     instrumentKey,
		"historical_min_bbw": historicalMinBBW,
	})

	return historicalMinBBW, nil
}

// calculateRecentMinBBW calculates the minimum BBW from recent data as fallback
func (s *BBWDashboardService) calculateRecentMinBBW(bbwValues []float64) float64 {
	if len(bbwValues) == 0 {
		return 0
	}

	minBBW := bbwValues[0]
	for _, bbw := range bbwValues {
		if bbw < minBBW {
			minBBW = bbw
		}
	}

	return minBBW
}

// calculateDistanceFromMin calculates the percentage distance from historical minimum
func (s *BBWDashboardService) calculateDistanceFromMin(currentBBW, historicalMinBBW float64) float64 {
	if historicalMinBBW == 0 {
		return 0
	}
	return ((currentBBW - historicalMinBBW) / historicalMinBBW) * 100
}

// calculateCandlesInRangeCount calculates the number of consecutive candles within optimal BBW range
func (s *BBWDashboardService) calculateCandlesInRangeCount(ctx context.Context, instrumentKey string, historicalMinBBW float64) int {
	if historicalMinBBW <= 0 {
		return 0
	}

	// Calculate the optimal range (±0.1% of historical minimum)
	rangeThreshold := historicalMinBBW * 0.001 // 0.1%
	minRange := historicalMinBBW - rangeThreshold
	maxRange := historicalMinBBW + rangeThreshold

	// Get recent candles to check how many are in range
	// We'll check the last 20 candles to find consecutive ones in range
	recentCandles, err := s.getRecentCandles(ctx, instrumentKey, 20)
	if err != nil {
		log.BBWError("candles_in_range", "get_recent_candles", "Failed to get recent candles", err, map[string]interface{}{
			"instrument_key": instrumentKey,
		})
		return 0
	}

	// Count consecutive candles in range (from most recent to oldest)
	count := 0
	for i := len(recentCandles) - 1; i >= 0; i-- {
		candle := recentCandles[i]
		if candle.BBWidth >= minRange && candle.BBWidth <= maxRange {
			count++
		} else {
			// Break on first candle outside range
			break
		}
	}

	log.BBWDebug("candles_in_range", "calculated", "Calculated candles in range count", map[string]interface{}{
		"instrument_key":     instrumentKey,
		"historical_min_bbw": historicalMinBBW,
		"min_range":          minRange,
		"max_range":          maxRange,
		"candles_in_range":   count,
	})

	return count
}

// getRecentCandles gets recent 5-minute candles for a stock
func (s *BBWDashboardService) getRecentCandles(ctx context.Context, instrumentKey string, count int) ([]domain.Candle5Min, error) {
	if s.candle5MinRepo == nil {
		return nil, fmt.Errorf("candle5Min repository not available")
	}

	// Use direct repository instead of aggregation service
	candles, err := s.candle5MinRepo.GetNLatestCandles(ctx, instrumentKey, count)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent candles: %w", err)
	}

	// Validate BBW data before processing
	for _, candle := range candles {
		if candle.BBWidth <= 0 {
			log.BBWWarn("candle_validation", "invalid_bbw", "Invalid BBW value", map[string]interface{}{
				"instrument_key": instrumentKey,
				"timestamp":      candle.Timestamp,
				"bb_width":       candle.BBWidth,
			})
		}
	}

	log.BBWDebug("candles_retrieval", "success", "Retrieved recent candles", map[string]interface{}{
		"instrument_key": instrumentKey,
		"count":          len(candles),
		"requested":      count,
	})

	return candles, nil
}

// updateCandlesInRangeCount updates the candles_in_range_count in the database
func (s *BBWDashboardService) updateCandlesInRangeCount(ctx context.Context, instrumentKey string, count int) error {
	if s.candle5MinRepo == nil {
		return fmt.Errorf("candle5Min repository not available")
	}

	err := s.candle5MinRepo.UpdateCandlesInRangeCount(ctx, instrumentKey, count)
	if err != nil {
		log.BBWError("persistence", "update_failed", "Failed to update candles in range count", err, map[string]interface{}{
			"instrument_key": instrumentKey,
			"count":          count,
		})
		return err
	}

	log.BBWDebug("persistence", "update_success", "Successfully updated candles in range count", map[string]interface{}{
		"instrument_key": instrumentKey,
		"count":          count,
	})

	return nil
}

// detectContractingPattern detects consecutive contracting candles
func (s *BBWDashboardService) detectContractingPattern(bbwValues []float64) int {
	if len(bbwValues) < 2 {
		log.BBWDebug("contracting_pattern", "insufficient_data", "Insufficient data for contracting pattern detection", map[string]interface{}{
			"data_points": len(bbwValues),
			"required":    2,
		})
		return 0
	}

	count := 0
	for i := len(bbwValues) - 1; i > 0; i-- {
		if bbwValues[i] < bbwValues[i-1] {
			count++
		} else {
			break
		}
	}

	log.BBWDebug("contracting_pattern", "detection", "Contracting pattern detection completed", map[string]interface{}{
		"contracting_count": count,
		"data_points":       len(bbwValues),
	})

	return count
}

// determineBBWTrend determines the overall BBW trend
func (s *BBWDashboardService) determineBBWTrend(bbwValues []float64) string {
	if len(bbwValues) < 6 {
		log.BBWDebug("trend_determination", "insufficient_data", "Insufficient data for trend determination", map[string]interface{}{
			"data_points": len(bbwValues),
			"required":    6,
		})
		return "stable"
	}

	// Compare recent values with older values
	recent := bbwValues[len(bbwValues)-3:]

	// Safely get older values, ensuring we don't go out of bounds
	startIndex := len(bbwValues) - 6
	if startIndex < 0 {
		startIndex = 0
		log.BBWWarn("trend_determination", "bounds_adjustment", "Adjusted start index to prevent out of bounds", map[string]interface{}{
			"original_start": len(bbwValues) - 6,
			"adjusted_start": startIndex,
			"data_points":    len(bbwValues),
		})
	}
	older := bbwValues[startIndex : len(bbwValues)-3]

	// Ensure we have enough data for comparison
	if len(older) < 3 {
		log.BBWDebug("trend_determination", "insufficient_older_data", "Insufficient older data for trend comparison", map[string]interface{}{
			"older_data_points": len(older),
			"required":          3,
			"total_data_points": len(bbwValues),
		})
		return "stable"
	}

	recentAvg := (recent[0] + recent[1] + recent[2]) / 3
	olderAvg := (older[0] + older[1] + older[2]) / 3

	var changePercent float64
	if olderAvg != 0 {
		changePercent = ((recentAvg - olderAvg) / olderAvg) * 100
	} else {
		log.BBWWarn("trend_determination", "division_by_zero", "Division by zero in trend calculation", map[string]interface{}{
			"recent_avg": recentAvg,
			"older_avg":  olderAvg,
		})
		changePercent = 0
	}

	log.BBWDebug("trend_determination", "calculation", "Trend calculation completed", map[string]interface{}{
		"recent_avg":     recentAvg,
		"older_avg":      olderAvg,
		"change_percent": changePercent,
		"data_points":    len(bbwValues),
	})

	if changePercent < -5 {
		return "contracting"
	} else if changePercent > 5 {
		return "expanding"
	} else {
		return "stable"
	}
}

// checkAdvancedAlertConditions checks for various alert conditions and triggers alerts
func (s *BBWDashboardService) checkAdvancedAlertConditions(
	instrumentKey, symbol string,
	currentBBW, historicalMinBBW float64,
	contractingCount int,
	bbwValues []float64) (bool, string, string, string) {

	log.BBWDebug("alert_check", "start", "Checking alert conditions", map[string]interface{}{
		"symbol":            symbol,
		"instrument_key":    instrumentKey,
		"current_bbw":       currentBBW,
		"historical_min":    historicalMinBBW,
		"contracting_count": contractingCount,
		"data_points":       len(bbwValues),
	})

	// Calculate pattern strength
	patternStrength := s.calculatePatternStrength(bbwValues, contractingCount)

	// Check threshold alert (within 0.1% of historical minimum)
	thresholdRange := historicalMinBBW * (s.alertThreshold / 100.0)
	minRange := historicalMinBBW - thresholdRange
	maxRange := historicalMinBBW + thresholdRange

	if currentBBW >= minRange && currentBBW <= maxRange {
		// Check if this is a new alert (not already triggered)
		s.mu.RLock()
		existingData, exists := s.monitoredStocks[instrumentKey]
		s.mu.RUnlock()

		if !exists || !existingData.AlertTriggered {
			// Trigger threshold alert
			alertType := "threshold"
			alertMessage := fmt.Sprintf("BB Width entered optimal range (%.4f)", currentBBW)

			log.PatternDetectionInfo(symbol, "threshold_alert", "Threshold alert triggered", map[string]interface{}{
				"current_bbw":      currentBBW,
				"historical_min":   historicalMinBBW,
				"threshold_range":  s.alertThreshold,
				"min_range":        minRange,
				"max_range":        maxRange,
				"pattern_strength": patternStrength,
			})

			s.triggerAlert(symbol, currentBBW, historicalMinBBW, contractingCount, alertType, alertMessage, patternStrength)
			return true, alertType, alertMessage, patternStrength
		}
	}

	// Check for strong contracting pattern (5+ consecutive candles)
	if contractingCount >= 5 && patternStrength == "strong" {
		alertType := "pattern"
		alertMessage := fmt.Sprintf("Strong contracting pattern detected (%d candles)", contractingCount)

		log.PatternDetectionInfo(symbol, "pattern_alert", "Strong contracting pattern alert triggered", map[string]interface{}{
			"contracting_count": contractingCount,
			"pattern_strength":  patternStrength,
			"current_bbw":       currentBBW,
			"historical_min":    historicalMinBBW,
		})

		s.triggerAlert(symbol, currentBBW, historicalMinBBW, contractingCount, alertType, alertMessage, patternStrength)
		return true, alertType, alertMessage, patternStrength
	}

	// Check for squeeze condition (very low BB width)
	squeezeThreshold := historicalMinBBW * 0.05 // 5% of historical minimum
	if currentBBW <= squeezeThreshold {
		alertType := "squeeze"
		alertMessage := fmt.Sprintf("BB Width squeeze detected (%.4f)", currentBBW)

		log.PatternDetectionInfo(symbol, "squeeze_alert", "Squeeze alert triggered", map[string]interface{}{
			"current_bbw":       currentBBW,
			"historical_min":    historicalMinBBW,
			"squeeze_threshold": squeezeThreshold,
			"pattern_strength":  patternStrength,
		})

		s.triggerAlert(symbol, currentBBW, historicalMinBBW, contractingCount, alertType, alertMessage, patternStrength)
		return true, alertType, alertMessage, patternStrength
	}

	log.BBWDebug("alert_check", "no_alert", "No alert conditions met", map[string]interface{}{
		"symbol":           symbol,
		"pattern_strength": patternStrength,
	})

	return false, "", "", patternStrength
}

// calculatePatternStrength determines the strength of the pattern
func (s *BBWDashboardService) calculatePatternStrength(bbwValues []float64, contractingCount int) string {
	if len(bbwValues) < 3 {
		log.BBWDebug("pattern_strength", "insufficient_data", "Insufficient data for pattern strength calculation", map[string]interface{}{
			"data_points": len(bbwValues),
			"required":    3,
		})
		return "weak"
	}

	// Calculate rate of change
	recentValues := bbwValues[len(bbwValues)-3:]
	// recentValues[0] is the most recent, recentValues[2] is the oldest in the 3-element slice
	var rateOfChange float64
	if recentValues[2] != 0 {
		rateOfChange = (recentValues[2] - recentValues[0]) / recentValues[2] * 100
	} else {
		log.BBWWarn("pattern_strength", "division_by_zero", "Division by zero in rate of change calculation", map[string]interface{}{
			"recent_values": recentValues,
		})
		rateOfChange = 0
	}

	log.BBWDebug("pattern_strength", "calculation", "Pattern strength calculation completed", map[string]interface{}{
		"contracting_count": contractingCount,
		"rate_of_change":    rateOfChange,
		"recent_values":     recentValues,
		"data_points":       len(bbwValues),
	})

	// Determine strength based on contracting count and rate of change
	if contractingCount >= 5 && rateOfChange > 10 {
		return "strong"
	} else if contractingCount >= 3 && rateOfChange > 5 {
		return "moderate"
	} else {
		return "weak"
	}
}

// triggerAlert triggers an audio alert and logs the alert
func (s *BBWDashboardService) triggerAlert(symbol string, currentBBW, historicalMinBBW float64,
	contractingCount int, alertType, alertMessage, patternStrength string) {

	log.AlertInfo(alertType, symbol, "Alert triggered", map[string]interface{}{
		"current_bbw":        currentBBW,
		"historical_min_bbw": historicalMinBBW,
		"pattern_length":     contractingCount,
		"pattern_strength":   patternStrength,
		"alert_message":      alertMessage,
	})

	if s.alertService == nil {
		log.BBWWarn("alert_system", "service_unavailable", "Alert service not available", map[string]interface{}{
			"symbol":     symbol,
			"alert_type": alertType,
		})
		return
	}

	// Create alert event
	alert := AlertEvent{
		Symbol:           symbol,
		BBWidth:          currentBBW,
		LowestMinBBWidth: historicalMinBBW,
		PatternLength:    contractingCount,
		AlertType:        alertType,
		Timestamp:        time.Now(),
		GroupID:          "BBW_DASHBOARD",
		Message:          alertMessage,
	}

	// Play audio alert
	if err := s.alertService.PlayAlert(alert); err != nil {
		log.AlertError(alertType, symbol, "Failed to play alert", err, map[string]interface{}{
			"current_bbw":        currentBBW,
			"historical_min_bbw": historicalMinBBW,
			"pattern_length":     contractingCount,
		})
	}

	// Add to alert history
	s.addToAlertHistory(alert)

	log.AlertInfo(alertType, symbol, "Alert processing completed", map[string]interface{}{
		"pattern_strength": patternStrength,
		"alert_message":    alertMessage,
	})
}

// addToAlertHistory adds an alert to the history
func (s *BBWDashboardService) addToAlertHistory(alert AlertEvent) {
	s.alertHistoryMutex.Lock()
	defer s.alertHistoryMutex.Unlock()

	// Keep only last 100 alerts
	if len(s.alertHistory) >= 100 {
		removedCount := len(s.alertHistory) - 99
		s.alertHistory = s.alertHistory[1:]
		log.BBWDebug("alert_history", "cleanup", "Removed old alerts from history", map[string]interface{}{
			"removed_count": removedCount,
			"remaining":     len(s.alertHistory),
		})
	}

	s.alertHistory = append(s.alertHistory, alert)

	log.BBWDebug("alert_history", "add", "Added alert to history", map[string]interface{}{
		"symbol":      alert.Symbol,
		"alert_type":  alert.AlertType,
		"total_count": len(s.alertHistory),
	})
}

// GetAlertHistory returns the alert history
func (s *BBWDashboardService) GetAlertHistory() []AlertEvent {
	s.alertHistoryMutex.RLock()
	defer s.alertHistoryMutex.RUnlock()

	// Return a copy to avoid race conditions
	history := make([]AlertEvent, len(s.alertHistory))
	copy(history, s.alertHistory)
	return history
}

// ClearAlertHistory clears the alert history
func (s *BBWDashboardService) ClearAlertHistory() {
	s.alertHistoryMutex.Lock()
	defer s.alertHistoryMutex.Unlock()
	s.alertHistory = make([]AlertEvent, 0)
	log.Info("[BBW Dashboard] Alert history cleared")
}

// getRecentBBWValues gets recent BBW values for a stock
func (s *BBWDashboardService) getRecentBBWValues(ctx context.Context, instrumentKey string, count int) ([]float64, error) {
	return s.getRecentBBWValuesInRange(ctx, instrumentKey, count, time.Time{}, time.Time{})
}

// getRecentBBWValuesInRange gets BBW values for a stock within a specific time range or recent candles if no range specified
func (s *BBWDashboardService) getRecentBBWValuesInRange(ctx context.Context, instrumentKey string, count int, startTime, endTime time.Time) ([]float64, error) {
	if s.candle5MinRepo == nil {
		return nil, fmt.Errorf("candle5Min repository not available")
	}

	var candles []domain.Candle5Min
	var err error

	// If time range is specified, get candles within that range
	if !startTime.IsZero() && !endTime.IsZero() {
		// Calculate the time range to get enough candles for the lookback period
		// We need to go back further to get enough candles for the contracting pattern analysis
		lookbackStartTime := startTime.Add(-time.Duration(count*5) * time.Minute) // 5 minutes per candle

		candles, err = s.candle5MinRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, lookbackStartTime, endTime)
		if err != nil {
			return nil, fmt.Errorf("failed to get candles in time range for BBW values: %w", err)
		}

		log.BBWDebug("bbw_values", "time_range", "Getting BBW values from time range", map[string]interface{}{
			"instrument_key": instrumentKey,
			"start_time":     startTime.Format("15:04"),
			"end_time":       endTime.Format("15:04"),
			"lookback_start": lookbackStartTime.Format("15:04"),
			"requested":      count,
		})
	} else {
		// Get recent candles from the database (default behavior)
		candles, err = s.candle5MinRepo.GetNLatestCandles(ctx, instrumentKey, count)
		if err != nil {
			return nil, fmt.Errorf("failed to get recent candles for BBW values: %w", err)
		}

		log.BBWDebug("bbw_values", "recent", "Getting recent BBW values", map[string]interface{}{
			"instrument_key": instrumentKey,
			"requested":      count,
		})
	}

	if len(candles) == 0 {
		log.BBWDebug("bbw_values", "no_data", "No candles found for BBW values", map[string]interface{}{
			"instrument_key": instrumentKey,
			"requested":      count,
		})
		return []float64{}, nil
	}

	// Extract BBW values from candles
	var bbwValues []float64
	for _, candle := range candles {
		if candle.BBWidth > 0 {
			bbwValues = append(bbwValues, candle.BBWidth)
		} else {
			log.BBWWarn("bbw_values", "invalid_bbw", "Invalid BBW value in candle", map[string]interface{}{
				"instrument_key": instrumentKey,
				"timestamp":      candle.Timestamp,
				"bb_width":       candle.BBWidth,
			})
		}
	}

	// Ensure we have valid data and log the data points for debugging
	if len(bbwValues) > 0 {
		log.BBWDebug("bbw_values", "data_validation", "BBW values validation", map[string]interface{}{
			"instrument_key": instrumentKey,
			"total_candles":  len(candles),
			"valid_bbw":      len(bbwValues),
			"first_value":    bbwValues[0],
			"last_value":     bbwValues[len(bbwValues)-1],
			"all_values":     bbwValues,
		})
	} else {
		log.BBWWarn("bbw_values", "no_valid_data", "No valid BBW values found", map[string]interface{}{
			"instrument_key": instrumentKey,
			"total_candles":  len(candles),
		})
	}

	// Verify we have enough valid BBW values
	if len(bbwValues) < 2 {
		log.BBWDebug("bbw_values", "insufficient_valid", "Insufficient valid BBW values", map[string]interface{}{
			"instrument_key": instrumentKey,
			"valid_count":    len(bbwValues),
			"required":       2,
		})
		return bbwValues, nil
	}

	log.BBWDebug("bbw_values", "retrieved", "Retrieved BBW values", map[string]interface{}{
		"instrument_key": instrumentKey,
		"requested":      count,
		"retrieved":      len(bbwValues),
		"valid_count":    len(bbwValues),
		"values":         bbwValues,
	})

	return bbwValues, nil
}

// getMonitoredStocks gets all stocks that need BBW monitoring
func (s *BBWDashboardService) getMonitoredStocks(ctx context.Context) ([]domain.StockUniverse, error) {
	// Get stocks from BB_RANGE groups
	groups, err := s.stockGroupService.GetGroupsByEntryType(ctx, "BB_RANGE", s.universeService)
	if err != nil {
		return nil, fmt.Errorf("failed to get BB_RANGE groups: %w", err)
	}

	var stocks []domain.StockUniverse
	stockMap := make(map[string]bool) // Avoid duplicates

	for _, group := range groups {
		for _, stock := range group.Stocks {
			if !stockMap[stock.InstrumentKey] {
				stockMap[stock.InstrumentKey] = true
				stocks = append(stocks, domain.StockUniverse{
					Symbol:        stock.Symbol,
					InstrumentKey: stock.InstrumentKey,
				})
			}
		}
	}

	return stocks, nil
}

// updateDashboardCache updates the in-memory cache with latest BBW data
func (s *BBWDashboardService) updateDashboardCache(dashboardData []*BBWDashboardData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, data := range dashboardData {
		s.monitoredStocks[data.InstrumentKey] = data
	}
}

// broadcastDashboardUpdate sends real-time updates to frontend via WebSocket
func (s *BBWDashboardService) broadcastDashboardUpdate(dashboardData []*BBWDashboardData) {
	if s.websocketHub == nil {
		log.BBWWarn("websocket", "hub_unavailable", "WebSocket hub not available", map[string]interface{}{
			"data_count": len(dashboardData),
		})
		return
	}

	// Create update message
	update := map[string]interface{}{
		"type":      "bbw_dashboard_update",
		"data":      dashboardData,
		"timestamp": time.Now(),
	}

	// Convert to JSON
	jsonData, err := json.Marshal(update)
	if err != nil {
		log.WebSocketError("broadcast", "Failed to marshal dashboard update", err, map[string]interface{}{
			"data_count": len(dashboardData),
		})
		return
	}

	// Broadcast to all connected clients
	s.websocketHub.Broadcast(jsonData)

	log.WebSocketInfo("broadcast", "Dashboard update broadcasted", map[string]interface{}{
		"data_count":   len(dashboardData),
		"message_size": len(jsonData),
	})
}

// GetDashboardData returns current dashboard data for all monitored stocks
func (s *BBWDashboardService) GetDashboardData() []*BBWDashboardData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var data []*BBWDashboardData
	for _, stockData := range s.monitoredStocks {
		data = append(data, stockData)
	}

	return data
}

// GetStockBBWData returns BBW data for a specific stock
func (s *BBWDashboardService) GetStockBBWData(instrumentKey string) (*BBWDashboardData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.monitoredStocks[instrumentKey]
	return data, exists
}

// IsMarketHours checks if current time is within market hours (9:15 AM - 3:30 PM IST)
func (s *BBWDashboardService) IsMarketHours() bool {
	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Kolkata")
	now = now.In(loc)

	// Check if it's a weekday
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}

	// Check market hours (9:15 AM - 3:30 PM IST)
	marketOpen := time.Date(now.Year(), now.Month(), now.Day(), 9, 15, 0, 0, loc)
	marketClose := time.Date(now.Year(), now.Month(), now.Day(), 15, 30, 0, 0, loc)

	return now.After(marketOpen) && now.Before(marketClose)
}

// SetAlertThreshold sets the alert threshold percentage
func (s *BBWDashboardService) SetAlertThreshold(threshold float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertThreshold = threshold
}

// SetContractingLookback sets the number of candles to look back for contracting patterns
func (s *BBWDashboardService) SetContractingLookback(lookback int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contractingLookback = lookback
}

// GetLatestAvailableDayData retrieves the most recent available BBW data for all monitored stocks
// regardless of market hours - useful for dashboard access outside trading hours
func (s *BBWDashboardService) GetLatestAvailableDayData(ctx context.Context) ([]*BBWDashboardData, error) {
	log.Info("[BBW Dashboard] Getting latest available day data for all monitored stocks")

	// Get all BB_RANGE groups
	groups, err := s.stockGroupService.GetGroupsByEntryType(ctx, "BB_RANGE", s.universeService)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch BB_RANGE groups: %w", err)
	}

	if len(groups) == 0 {
		log.Info("[BBW Dashboard] No BB_RANGE groups found")
		return []*BBWDashboardData{}, nil
	}

	var allStockData []*BBWDashboardData

	// Process each group's stocks
	for _, group := range groups {
		for _, stock := range group.Stocks {
			if stock.InstrumentKey == "" {
				continue
			}

			// Get the latest available 5-minute candle data
			stockData, err := s.getLatestStockBBWData(ctx, stock)
			if err != nil {
				log.Error("[BBW Dashboard] Failed to get latest data for %s: %v", stock.Symbol, err)
				continue
			}

			allStockData = append(allStockData, stockData)
		}
	}

	log.Info("[BBW Dashboard] Retrieved latest data for %d stocks", len(allStockData))
	return allStockData, nil
}

// getLatestStockBBWData gets the most recent BBW data for a single stock
func (s *BBWDashboardService) getLatestStockBBWData(ctx context.Context, stock response.StockGroupStockDTO) (*BBWDashboardData, error) {
	// Get the latest 5-minute candle (last 24 hours to ensure we get today's data)
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -1) // Last 24 hours

	candles, err := s.candleAggService.Get5MinCandles(ctx, stock.InstrumentKey, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get 5-minute candles: %w", err)
	}

	if len(candles) == 0 {
		return nil, fmt.Errorf("no 5-minute candles found")
	}

	// Get the latest candle
	latestCandle := candles[len(candles)-1]

	// Convert to domain.StockUniverse for processing
	stockUniverse := domain.StockUniverse{
		InstrumentKey: stock.InstrumentKey,
		Symbol:        stock.Symbol,
	}

	// Process the stock BBW data using existing method
	stockData, err := s.processStockBBW(ctx, stockUniverse, latestCandle.Timestamp, latestCandle.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to process BBW data: %w", err)
	}

	return stockData, nil
}

// GetStockBBWHistory retrieves historical BBW data for a specific stock
func (s *BBWDashboardService) GetStockBBWHistory(ctx context.Context, instrumentKey string, days int) ([]*BBWDashboardData, error) {
	if days <= 0 {
		days = 7 // Default to 7 days
	}

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	// Use the candle5MinRepo directly to get historical 5-minute candles with BBW data
	candles, err := s.candle5MinRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical candles: %w", err)
	}

	var historicalData []*BBWDashboardData

	// Get stock metadata
	stock, err := s.getStockMetadata(ctx, instrumentKey)
	if err != nil {
		log.Warn("[BBW Dashboard] Failed to get stock metadata for %s: %v", instrumentKey, err)
		// Continue with basic data
		stock = response.StockGroupStockDTO{
			InstrumentKey: instrumentKey,
			Symbol:        instrumentKey,
		}
	}

	// Convert to domain.StockUniverse for processing
	stockUniverse := domain.StockUniverse{
		InstrumentKey: stock.InstrumentKey,
		Symbol:        stock.Symbol,
	}

	// Process each candle with historical context
	log.Info("[BBW Dashboard] Processing %d historical candles for %s", len(candles), stock.Symbol)
	for i, candle := range candles {
		if i < 5 || i > len(candles)-5 { // Log first 5 and last 5 candles
			log.Info("[BBW Dashboard] Processing candle %d/%d: %s", i+1, len(candles), candle.Timestamp.Format("2006-01-02 15:04:05"))
		}
		stockData, err := s.processHistoricalStockBBWFromCandle5Min(ctx, stockUniverse, candle)
		if err != nil {
			log.Error("[BBW Dashboard] Failed to process historical data for %s: %v", stock.Symbol, err)
			continue
		}
		// Only append non-nil data
		if stockData != nil {
			historicalData = append(historicalData, stockData)
		} else {
			log.BBWDebug("historical_processing", "skip_nil", "Skipping nil data for candle", map[string]interface{}{
				"symbol":         stock.Symbol,
				"instrument_key": stock.InstrumentKey,
				"candle_index":   i,
				"timestamp":      candle.Timestamp.Format("2006-01-02 15:04:05"),
			})
		}
	}

	return historicalData, nil
}

// processHistoricalStockBBWFromCandle5Min processes BBW data for a specific historical Candle5Min
func (s *BBWDashboardService) processHistoricalStockBBWFromCandle5Min(ctx context.Context, stock domain.StockUniverse, candle domain.Candle5Min) (*BBWDashboardData, error) {
	log.BBWDebug("historical_processing", "start", "Processing historical BBW data for stock from Candle5Min", map[string]interface{}{
		"symbol":         stock.Symbol,
		"instrument_key": stock.InstrumentKey,
		"candle_time":    candle.Timestamp.Format("15:04"),
		"bb_width":       candle.BBWidth,
	})

	// For historical data, we need to get BBW values leading up to this specific candle
	// Calculate the time range to get enough candles for the lookback period
	lookbackStartTime := candle.Timestamp.Add(-time.Duration(s.contractingLookback*5) * time.Minute) // 5 minutes per candle

	// Get BBW values for the stock within the historical time range
	bbwValues, err := s.getRecentBBWValuesInRange(ctx, stock.InstrumentKey, s.contractingLookback+1, lookbackStartTime, candle.Timestamp)
	if err != nil {
		log.BBWError("historical_processing", "get_bbw_values", "Failed to get BBW values", err, map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"lookback":       s.contractingLookback + 1,
			"candle_time":    candle.Timestamp.Format("15:04"),
		})
		return nil, fmt.Errorf("failed to get BBW values: %w", err)
	}

	if len(bbwValues) < 2 {
		log.BBWDebug("historical_processing", "insufficient_data", "Insufficient BBW data for stock", map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"data_points":    len(bbwValues),
			"required":       2,
			"candle_time":    candle.Timestamp.Format("15:04"),
		})
		return nil, nil
	}

	// Use the BBW value from the actual candle in the database
	currentBBW := candle.BBWidth

	// Get historical minimum BBW from CSV/database
	historicalMinBBW, err := s.getHistoricalMinBBW(ctx, stock.InstrumentKey)
	if err != nil {
		log.BBWError("historical_processing", "historical_min", "Failed to get historical minimum BBW", err, map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
		})
		// Use recent minimum as fallback
		historicalMinBBW = s.calculateRecentMinBBW(bbwValues)
		log.BBWWarn("historical_processing", "fallback", "Using recent minimum as fallback", map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"fallback_value": historicalMinBBW,
		})
	}

	// Calculate distance from minimum
	distanceFromMin := s.calculateDistanceFromMin(currentBBW, historicalMinBBW)

	// Detect contracting pattern
	contractingCount := s.detectContractingPattern(bbwValues)

	// Determine BBW trend
	trend := s.determineBBWTrend(bbwValues)

	// Calculate candles in range count (for historical data, we don't persist this)
	candlesInRangeCount := s.calculateCandlesInRangeCount(ctx, stock.InstrumentKey, historicalMinBBW)

	// Check alert conditions (for historical data, we don't trigger alerts)
	alertTriggered, alertType, alertMessage, patternStrength := s.checkAdvancedAlertConditions(
		stock.InstrumentKey, stock.Symbol, currentBBW, historicalMinBBW, contractingCount, bbwValues)

	// Create dashboard data with the actual candle timestamp
	dashboardData := &BBWDashboardData{
		Symbol:                   stock.Symbol,
		InstrumentKey:            stock.InstrumentKey,
		CurrentBBWidth:           currentBBW,
		HistoricalMinBBWidth:     historicalMinBBW,
		DistanceFromMinPercent:   distanceFromMin,
		ContractingSequenceCount: contractingCount,
		CandlesInRangeCount:      candlesInRangeCount,
		BBWidthTrend:             trend,
		AlertTriggered:           alertTriggered,
		AlertType:                alertType,
		AlertMessage:             alertMessage,
		PatternStrength:          patternStrength,
		Timestamp:                candle.Timestamp, // Use the actual candle timestamp
		LastUpdated:              time.Now(),
	}

	log.BBWDebug("historical_processing", "complete", "Completed processing historical stock BBW data from Candle5Min", map[string]interface{}{
		"symbol":             stock.Symbol,
		"instrument_key":     stock.InstrumentKey,
		"current_bbw":        currentBBW,
		"historical_min_bbw": historicalMinBBW,
		"distance_percent":   distanceFromMin,
		"contracting_count":  contractingCount,
		"trend":              trend,
		"candle_time":        candle.Timestamp.Format("15:04"),
	})

	return dashboardData, nil
}

// processHistoricalStockBBW processes BBW data for a specific historical candle
func (s *BBWDashboardService) processHistoricalStockBBW(ctx context.Context, stock domain.StockUniverse, candle domain.AggregatedCandle) (*BBWDashboardData, error) {
	log.BBWDebug("historical_processing", "start", "Processing historical BBW data for stock", map[string]interface{}{
		"symbol":         stock.Symbol,
		"instrument_key": stock.InstrumentKey,
		"candle_time":    candle.Timestamp.Format("15:04"),
	})

	// For historical data, we need to get BBW values leading up to this specific candle
	// Calculate the time range to get enough candles for the lookback period
	lookbackStartTime := candle.Timestamp.Add(-time.Duration(s.contractingLookback*5) * time.Minute) // 5 minutes per candle

	// Get BBW values for the stock within the historical time range
	bbwValues, err := s.getRecentBBWValuesInRange(ctx, stock.InstrumentKey, s.contractingLookback+1, lookbackStartTime, candle.Timestamp)
	if err != nil {
		log.BBWError("historical_processing", "get_bbw_values", "Failed to get BBW values", err, map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"lookback":       s.contractingLookback + 1,
			"candle_time":    candle.Timestamp.Format("15:04"),
		})
		return nil, fmt.Errorf("failed to get BBW values: %w", err)
	}

	if len(bbwValues) < 2 {
		log.BBWDebug("historical_processing", "insufficient_data", "Insufficient BBW data for stock", map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"data_points":    len(bbwValues),
			"required":       2,
			"candle_time":    candle.Timestamp.Format("15:04"),
		})
		return nil, nil
	}

	// Get the actual BBW value from the database for this specific timestamp
	// We need to get the 5-minute candle from the database to get the correct BBW value
	actualCandle, err := s.candle5MinRepo.FindByInstrumentAndTimeRange(ctx, stock.InstrumentKey, candle.Timestamp, candle.Timestamp)
	if err != nil {
		log.BBWError("historical_processing", "get_actual_candle", "Failed to get actual candle from database", err, map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"candle_time":    candle.Timestamp.Format("15:04"),
		})
		return nil, fmt.Errorf("failed to get actual candle: %w", err)
	}

	// Use the BBW value from the actual candle in the database
	var currentBBW float64
	if len(actualCandle) > 0 {
		currentBBW = actualCandle[0].BBWidth
		log.BBWDebug("historical_processing", "found_actual_candle", "Found actual candle in database", map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"candle_time":    candle.Timestamp.Format("15:04"),
			"bbw_value":      currentBBW,
			"candles_found":  len(actualCandle),
		})
	} else {
		// Fallback to the aggregated candle BBW if no actual candle found
		currentBBW = candle.BBWidth
		log.BBWWarn("historical_processing", "fallback_bbw", "Using aggregated candle BBW as fallback", map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"candle_time":    candle.Timestamp.Format("15:04"),
			"bbw_value":      currentBBW,
		})
	}

	// Get historical minimum BBW from CSV/database
	historicalMinBBW, err := s.getHistoricalMinBBW(ctx, stock.InstrumentKey)
	if err != nil {
		log.BBWError("historical_processing", "historical_min", "Failed to get historical minimum BBW", err, map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
		})
		// Use recent minimum as fallback
		historicalMinBBW = s.calculateRecentMinBBW(bbwValues)
		log.BBWWarn("historical_processing", "fallback", "Using recent minimum as fallback", map[string]interface{}{
			"symbol":         stock.Symbol,
			"instrument_key": stock.InstrumentKey,
			"fallback_value": historicalMinBBW,
		})
	}

	// Calculate distance from minimum
	distanceFromMin := s.calculateDistanceFromMin(currentBBW, historicalMinBBW)

	// Detect contracting pattern
	contractingCount := s.detectContractingPattern(bbwValues)

	// Determine BBW trend
	trend := s.determineBBWTrend(bbwValues)

	// Calculate candles in range count (for historical data, we don't persist this)
	candlesInRangeCount := s.calculateCandlesInRangeCount(ctx, stock.InstrumentKey, historicalMinBBW)

	// Check alert conditions (for historical data, we don't trigger alerts)
	alertTriggered, alertType, alertMessage, patternStrength := s.checkAdvancedAlertConditions(
		stock.InstrumentKey, stock.Symbol, currentBBW, historicalMinBBW, contractingCount, bbwValues)

	// Create dashboard data with the actual candle timestamp
	dashboardData := &BBWDashboardData{
		Symbol:                   stock.Symbol,
		InstrumentKey:            stock.InstrumentKey,
		CurrentBBWidth:           currentBBW,
		HistoricalMinBBWidth:     historicalMinBBW,
		DistanceFromMinPercent:   distanceFromMin,
		ContractingSequenceCount: contractingCount,
		CandlesInRangeCount:      candlesInRangeCount,
		BBWidthTrend:             trend,
		AlertTriggered:           alertTriggered,
		AlertType:                alertType,
		AlertMessage:             alertMessage,
		PatternStrength:          patternStrength,
		Timestamp:                candle.Timestamp, // Use the actual candle timestamp
		LastUpdated:              time.Now(),
	}

	log.BBWDebug("historical_processing", "complete", "Completed processing historical stock BBW data", map[string]interface{}{
		"symbol":             stock.Symbol,
		"instrument_key":     stock.InstrumentKey,
		"current_bbw":        currentBBW,
		"historical_min_bbw": historicalMinBBW,
		"distance_percent":   distanceFromMin,
		"contracting_count":  contractingCount,
		"trend":              trend,
		"candle_time":        candle.Timestamp.Format("15:04"),
	})

	return dashboardData, nil
}

// getStockMetadata retrieves stock metadata from groups
func (s *BBWDashboardService) getStockMetadata(ctx context.Context, instrumentKey string) (response.StockGroupStockDTO, error) {
	groups, err := s.stockGroupService.GetGroupsByEntryType(ctx, "BB_RANGE", s.universeService)
	if err != nil {
		return response.StockGroupStockDTO{}, err
	}

	for _, group := range groups {
		for _, stock := range group.Stocks {
			if stock.InstrumentKey == instrumentKey {
				return stock, nil
			}
		}
	}

	return response.StockGroupStockDTO{}, fmt.Errorf("stock not found in BB_RANGE groups")
}
