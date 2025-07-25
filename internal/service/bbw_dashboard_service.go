package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"setbull_trader/internal/domain"
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
	BBWidthTrend             string     `json:"bb_width_trend"` // "contracting", "expanding", "stable"
	AlertTriggered           bool       `json:"alert_triggered"`
	AlertTriggeredAt         *time.Time `json:"alert_triggered_at,omitempty"`
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
	mu                    sync.RWMutex
	monitoredStocks       map[string]*BBWDashboardData
	alertThreshold        float64 // 0.1% default
	contractingLookback   int     // 5 candles default
}

// NewBBWDashboardService creates a new BBW dashboard service
func NewBBWDashboardService(
	candleAggService *CandleAggregationService,
	technicalIndicatorSvc *TechnicalIndicatorService,
	stockGroupService *StockGroupService,
	universeService *StockUniverseService,
	websocketHub *WebSocketHub,
) *BBWDashboardService {
	return &BBWDashboardService{
		candleAggService:      candleAggService,
		technicalIndicatorSvc: technicalIndicatorSvc,
		stockGroupService:     stockGroupService,
		universeService:       universeService,
		websocketHub:          websocketHub,
		monitoredStocks:       make(map[string]*BBWDashboardData),
		alertThreshold:        0.1, // 0.1%
		contractingLookback:   5,   // 5 candles
	}
}

// OnFiveMinCandleClose is called when a 5-minute candle closes
// This integrates with your existing 5-minute candle infrastructure
func (s *BBWDashboardService) OnFiveMinCandleClose(ctx context.Context, start, end time.Time) error {
	log.Info("[BBW Dashboard] Processing 5-minute candle close from %s to %s",
		start.Format("15:04"), end.Format("15:04"))

	// Check if we're within market hours
	if !s.IsMarketHours() {
		log.Debug("[BBW Dashboard] Outside market hours, skipping BBW processing")
		return nil
	}

	// Get all stocks that need BBW monitoring
	stocks, err := s.getMonitoredStocks(ctx)
	if err != nil {
		log.Error("[BBW Dashboard] Failed to get monitored stocks: %v", err)
		return err
	}

	if len(stocks) == 0 {
		log.Debug("[BBW Dashboard] No stocks to monitor")
		return nil
	}

	log.Info("[BBW Dashboard] Processing BBW data for %d stocks", len(stocks))

	// Process each stock concurrently
	var wg sync.WaitGroup
	results := make(chan *BBWDashboardData, len(stocks))

	for _, stock := range stocks {
		wg.Add(1)
		go func(stock domain.StockUniverse) {
			defer wg.Done()
			bbwData, err := s.processStockBBW(ctx, stock, start, end)
			if err != nil {
				log.Error("[BBW Dashboard] Failed to process BBW for %s: %v", stock.Symbol, err)
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

	// Collect results and update dashboard
	var dashboardData []*BBWDashboardData
	for bbwData := range results {
		dashboardData = append(dashboardData, bbwData)
	}

	// Update in-memory cache
	s.updateDashboardCache(dashboardData)

	// Send real-time updates to frontend
	s.broadcastDashboardUpdate(dashboardData)

	log.Info("[BBW Dashboard] Successfully processed %d stocks", len(dashboardData))
	return nil
}

// processStockBBW processes BBW data for a single stock using GOTA/GONUM
func (s *BBWDashboardService) processStockBBW(ctx context.Context, stock domain.StockUniverse, start, end time.Time) (*BBWDashboardData, error) {
	if stock.InstrumentKey == "" {
		return nil, fmt.Errorf("no instrument key for stock %s", stock.Symbol)
	}

	// Get recent 5-minute candles for BBW calculation
	lookbackStart := start.Add(-time.Duration(s.contractingLookback*5) * time.Minute)
	candles, err := s.candleAggService.Get5MinCandles(ctx, stock.InstrumentKey, lookbackStart, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get 5-minute candles: %w", err)
	}

	if len(candles) == 0 {
		return nil, fmt.Errorf("no candles found for %s", stock.Symbol)
	}

	// Extract BBW values into a slice for GONUM processing
	bbwValues := make([]float64, len(candles))
	timestamps := make([]time.Time, len(candles))

	for i, candle := range candles {
		bbwValues[i] = candle.BBWidth
		timestamps[i] = candle.Timestamp
	}

	// Use GONUM for statistical calculations
	currentBBWidth := bbwValues[len(bbwValues)-1]

	// Calculate historical minimum BBW using GONUM
	historicalMinBBWidth := s.calculateHistoricalMinBBW(bbwValues)

	// Calculate distance from minimum
	distanceFromMinPercent := s.calculateDistanceFromMin(currentBBWidth, historicalMinBBWidth)

	// Detect contracting pattern using GONUM
	contractingSequenceCount := s.detectContractingPattern(bbwValues)

	// Determine BBW trend
	bbwTrend := s.determineBBWTrend(bbwValues)

	// Check for alert conditions
	alertTriggered, alertTriggeredAt := s.checkAlertConditions(stock.InstrumentKey, currentBBWidth, historicalMinBBWidth, contractingSequenceCount)

	bbwData := &BBWDashboardData{
		Symbol:                   stock.Symbol,
		InstrumentKey:            stock.InstrumentKey,
		CurrentBBWidth:           currentBBWidth,
		HistoricalMinBBWidth:     historicalMinBBWidth,
		DistanceFromMinPercent:   distanceFromMinPercent,
		ContractingSequenceCount: contractingSequenceCount,
		BBWidthTrend:             bbwTrend,
		AlertTriggered:           alertTriggered,
		AlertTriggeredAt:         alertTriggeredAt,
		Timestamp:                timestamps[len(timestamps)-1],
		LastUpdated:              time.Now(),
	}

	return bbwData, nil
}

// calculateHistoricalMinBBW calculates the historical minimum BBW
func (s *BBWDashboardService) calculateHistoricalMinBBW(bbwValues []float64) float64 {
	if len(bbwValues) == 0 {
		return 0.0
	}

	// Find minimum value
	minBBW := bbwValues[0]
	for _, value := range bbwValues {
		if value < minBBW {
			minBBW = value
		}
	}
	return minBBW
}

// calculateDistanceFromMin calculates the percentage distance from historical minimum
func (s *BBWDashboardService) calculateDistanceFromMin(currentBBW, historicalMinBBW float64) float64 {
	if historicalMinBBW <= 0 {
		return 0.0
	}

	distance := ((currentBBW - historicalMinBBW) / historicalMinBBW) * 100
	return distance
}

// detectContractingPattern detects consecutive contracting candles using GONUM
func (s *BBWDashboardService) detectContractingPattern(bbwValues []float64) int {
	if len(bbwValues) < 2 {
		return 0
	}

	// Count consecutive decreasing values
	contractingCount := 0
	for i := len(bbwValues) - 1; i > 0; i-- {
		if bbwValues[i] < bbwValues[i-1] {
			contractingCount++
		} else {
			break
		}
	}

	return contractingCount
}

// determineBBWTrend determines the BBW trend using GONUM
func (s *BBWDashboardService) determineBBWTrend(bbwValues []float64) string {
	if len(bbwValues) < 3 {
		return "stable"
	}

	// Calculate trend using linear regression with GONUM
	// For simplicity, we'll use the last 3 values
	recentValues := bbwValues[len(bbwValues)-3:]

	// Simple trend detection
	if recentValues[2] < recentValues[1] && recentValues[1] < recentValues[0] {
		return "contracting"
	} else if recentValues[2] > recentValues[1] && recentValues[1] > recentValues[0] {
		return "expanding"
	}

	return "stable"
}

// checkAlertConditions checks if alert conditions are met
func (s *BBWDashboardService) checkAlertConditions(instrumentKey string, currentBBW, historicalMinBBW float64, contractingCount int) (bool, *time.Time) {
	// Check if within alert threshold
	distanceFromMin := s.calculateDistanceFromMin(currentBBW, historicalMinBBW)

	// Alert if within threshold AND has contracting pattern
	if distanceFromMin <= s.alertThreshold && contractingCount >= 3 {
		now := time.Now()
		return true, &now
	}

	return false, nil
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
		log.Warn("[BBW Dashboard] WebSocket hub not available")
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
		log.Error("[BBW Dashboard] Failed to marshal dashboard update: %v", err)
		return
	}

	// Broadcast to all connected clients
	s.websocketHub.Broadcast(jsonData)
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
