package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"setbull_trader/internal/core/dto/response"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/log"
)

// BBWidthMonitorService monitors BB width for stocks in BB_RANGE groups
// and detects contracting patterns within the lowest_min_bb_width_range
type BBWidthMonitorService struct {
	stockGroupService     *StockGroupService
	technicalIndicatorSvc *TechnicalIndicatorService
	alertService          *AlertService
	universeService       *StockUniverseService
	config                *config.BBWidthMonitoringConfig
	// Add CandleAggregationService as a field if not present
	// (Assume it's available as s.candleAggService)
	candleAggService *CandleAggregationService
}

// NewBBWidthMonitorService creates a new BB width monitoring service
func NewBBWidthMonitorService(
	stockGroupService *StockGroupService,
	technicalIndicatorSvc *TechnicalIndicatorService,
	alertService *AlertService,
	universeService *StockUniverseService,
	cfg *config.BBWidthMonitoringConfig,
	candleAggService *CandleAggregationService,
) *BBWidthMonitorService {
	return &BBWidthMonitorService{
		stockGroupService:     stockGroupService,
		technicalIndicatorSvc: technicalIndicatorSvc,
		alertService:          alertService,
		universeService:       universeService,
		config:                cfg,
		candleAggService:      candleAggService,
	}
}

// MonitorBBRangeGroups monitors all BB_RANGE groups for contracting patterns
func (s *BBWidthMonitorService) MonitorBBRangeGroups(ctx context.Context, start, end time.Time) error {
	log.Info("[BB Monitor] Starting BB width monitoring for BB_RANGE groups from %s to %s",
		start.Format(time.RFC3339), end.Format(time.RFC3339))

	// Check if we're within market hours (9:15 AM - 3:30 PM IST)
	if !s.isMarketHours() {
		log.Info("[BB Monitor] Outside market hours, skipping BB width monitoring")
		return nil
	}

	// Get only BB_RANGE groups
	groups, err := s.stockGroupService.GetGroupsByEntryType(ctx, "BB_RANGE", s.universeService)
	if err != nil {
		log.Error("[BB Monitor] Failed to fetch BB_RANGE groups: %v", err)
		return fmt.Errorf("failed to fetch BB_RANGE groups: %w", err)
	}

	if len(groups) == 0 {
		log.Info("[BB Monitor] No BB_RANGE groups found")
		return nil
	}

	log.Info("[BB Monitor] Found %d BB_RANGE groups to monitor", len(groups))

	// Monitor each BB_RANGE group's stocks
	for _, group := range groups {
		err := s.monitorGroupStocks(ctx, group, start, end)
		if err != nil {
			log.Error("[BB Monitor] Failed to monitor group %s: %v", group.ID, err)
			// Continue monitoring other groups even if one fails
		}
	}

	return nil
}

// monitorGroupStocks monitors all stocks in a BB_RANGE group for contracting patterns
func (s *BBWidthMonitorService) monitorGroupStocks(ctx context.Context, group response.StockGroupResponse, start, end time.Time) error {
	for _, stock := range group.Stocks {
		if stock.InstrumentKey == "" {
			log.Warn("[BB Monitor] Skipping stock %s - no instrument key", stock.StockID)
			continue
		}

		err := s.monitorStock(ctx, stock, start, end)
		if err != nil {
			log.Error("[BB Monitor] Failed to monitor stock %s: %v", stock.InstrumentKey, err)
			// Continue monitoring other stocks even if one fails
		}
	}

	return nil
}

// monitorStock monitors a single stock for BB width contracting patterns
func (s *BBWidthMonitorService) monitorStock(ctx context.Context, stock response.StockGroupStockDTO, start, end time.Time) error {
	log.Debug("[BB Monitor] Monitoring stock %s (%s) for BB width patterns", stock.Symbol, stock.InstrumentKey)

	// Use in-memory 5-min aggregation and direct BB width analysis
	if s.candleAggService == nil {
		return fmt.Errorf("candleAggService is not initialized in BBWidthMonitorService")
	}

	err := s.candleAggService.Aggregate5MinCandlesWithIndicators(
		ctx,
		stock.InstrumentKey,
		start,
		end,
		func(ctx context.Context, instrumentKey string, candle domain.AggregatedCandle) {
			// Prepare BB width history (simulate last 5 candles including this one)
			// In a real implementation, you would maintain a rolling window in memory or fetch from a cache/service
			// For now, fetch last 4 from DB and append this one
			lookback := 4
			bbWidthHistory, _ := s.getRecentBBWidthHistory(ctx, instrumentKey, lookback)
			bbWidthHistory = append(bbWidthHistory, domain.IndicatorValue{
				Timestamp: candle.Timestamp,
				Value:     candle.BBWidth,
			})
			s.ProcessBBWidth(ctx, instrumentKey, candle, bbWidthHistory, stock)
		},
	)
	if err != nil {
		log.Error("[BB Monitor] Failed to aggregate and analyze 5-min candles for %s: %v", stock.InstrumentKey, err)
		return err
	}
	return nil
}

// calculateBBWidth calculates the BB width for the given instrument and time range
func (s *BBWidthMonitorService) calculateBBWidth(ctx context.Context, instrumentKey string, start, end time.Time) (float64, error) {
	// Use existing TechnicalIndicatorService to get BB width
	// For 5-minute candles, we need to consider previous day data for proper BB calculation
	// Extend the start time to include enough historical data for BB calculation
	extendedStart := start.AddDate(0, 0, -1) // Include previous day for warm-up data

	// Get Bollinger Bands (upper, middle, lower)
	bbUpper, bbMiddle, bbLower, err := s.technicalIndicatorSvc.CalculateBollingerBandsForRange(
		ctx, instrumentKey, 20, 2.0, "5minute", extendedStart, end,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate Bollinger Bands: %w", err)
	}

	if len(bbUpper) == 0 || len(bbMiddle) == 0 || len(bbLower) == 0 {
		return 0, fmt.Errorf("no Bollinger Band values calculated")
	}

	// Calculate BB width from the bands
	bbWidthValues, err := s.technicalIndicatorSvc.CalculateBBWidthForRange(bbUpper, bbLower, bbMiddle)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate BB width: %w", err)
	}

	if len(bbWidthValues) == 0 {
		return 0, fmt.Errorf("no BB width values calculated")
	}

	// Return the latest BB width value
	latestBBWidth := bbWidthValues[len(bbWidthValues)-1].Value
	log.Debug("[BB Monitor] Calculated BB width for %s: %f", instrumentKey, latestBBWidth)

	return latestBBWidth, nil
}

// detectContractingPattern detects if there's a contracting pattern within the optimal range
func (s *BBWidthMonitorService) detectContractingPattern(ctx context.Context, stock response.StockGroupStockDTO, currentBBWidth float64) error {
	log.Debug("[BB Monitor] Detecting contracting pattern for %s (current BB width: %f)", stock.Symbol, currentBBWidth)

	// 1. Get historical BB width data for pattern analysis (last 5 candles)
	bbWidthHistory, err := s.getRecentBBWidthHistory(ctx, stock.InstrumentKey, 5)
	if err != nil {
		return fmt.Errorf("failed to get BB width history: %w", err)
	}

	// Need at least 3 candles for pattern detection
	if len(bbWidthHistory) < 3 {
		log.Debug("[BB Monitor] Insufficient BB width history for %s (need 3+, got %d)", stock.Symbol, len(bbWidthHistory))
		return nil
	}

	// 2. Check for contracting pattern (decreasing BB width)
	isContracting := s.isContractingPattern(bbWidthHistory)
	if !isContracting {
		log.Debug("[BB Monitor] No contracting pattern detected for %s", stock.Symbol)
		return nil
	}

	// 3. Get lowest_min_bb_width from stock_candle_data table
	lowestMinBBWidth, err := s.getLowestMinBBWidth(ctx, stock.InstrumentKey)
	if err != nil {
		return fmt.Errorf("failed to get lowest min BB width: %w", err)
	}

	if lowestMinBBWidth <= 0 {
		log.Debug("[BB Monitor] Invalid lowest min BB width for %s: %f", stock.Symbol, lowestMinBBWidth)
		return nil
	}

	// 4. Calculate lowest_min_bb_width_range (±0.10% of lowest_min_bb_width)
	minRange, maxRange := s.calculateBBWidthRange(lowestMinBBWidth)

	// 5. Check if current BB width is within the optimal range
	if currentBBWidth >= minRange && currentBBWidth <= maxRange {
		// 6. Pattern detected: contracting candles within optimal range
		log.Info("[BB Monitor] BB Range Alert: %s - %d consecutive contracting candles in optimal range (BB width: %f, range: %f-%f, lowest: %f)",
			stock.Symbol, len(bbWidthHistory), currentBBWidth, minRange, maxRange, lowestMinBBWidth)

		return s.triggerBBRangeAlert(ctx, stock, currentBBWidth, lowestMinBBWidth, len(bbWidthHistory))
	}

	log.Debug("[BB Monitor] Contracting pattern detected for %s but outside optimal range (BB width: %f, range: %f-%f)",
		stock.Symbol, currentBBWidth, minRange, maxRange)
	return nil
}

// getRecentBBWidthHistory gets the recent BB width history for pattern analysis
func (s *BBWidthMonitorService) getRecentBBWidthHistory(ctx context.Context, instrumentKey string, lookbackCandles int) ([]domain.IndicatorValue, error) {
	// Get recent 5-minute candles with BB width data
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(lookbackCandles*5) * time.Minute) // 5 minutes per candle

	candles, err := s.technicalIndicatorSvc.candleRepo.FindByInstrumentAndTimeRange(
		ctx, instrumentKey, "5minute", startTime, endTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent candles: %w", err)
	}

	if len(candles) == 0 {
		return nil, fmt.Errorf("no recent candles found")
	}

	// Extract BB width values from candles
	var bbWidthHistory []domain.IndicatorValue
	for _, candle := range candles {
		if candle.BBWidth > 0 {
			bbWidthHistory = append(bbWidthHistory, domain.IndicatorValue{
				Timestamp: candle.Timestamp,
				Value:     candle.BBWidth,
			})
		}
	}

	// Sort by timestamp (oldest first)
	for i, j := 0, len(bbWidthHistory)-1; i < j; i, j = i+1, j-1 {
		bbWidthHistory[i], bbWidthHistory[j] = bbWidthHistory[j], bbWidthHistory[i]
	}

	log.Debug("[BB Monitor] Retrieved %d BB width values for %s", len(bbWidthHistory), instrumentKey)
	return bbWidthHistory, nil
}

// isContractingPattern checks if the BB width values show a contracting pattern
func (s *BBWidthMonitorService) isContractingPattern(bbWidthHistory []domain.IndicatorValue) bool {
	if len(bbWidthHistory) < 3 {
		return false
	}

	// Check for 3-5 consecutive contracting candles (decreasing BB width)
	isContracting := true
	for i := 1; i < len(bbWidthHistory); i++ {
		if bbWidthHistory[i].Value >= bbWidthHistory[i-1].Value {
			isContracting = false
			break
		}
	}

	return isContracting
}

// getLowestMinBBWidth gets the lowest_min_bb_width from the CSV analysis file
func (s *BBWidthMonitorService) getLowestMinBBWidth(ctx context.Context, instrumentKey string) (float64, error) {
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

// calculateBBWidthRange calculates the optimal range (±0.10% of lowest_min_bb_width)
func (s *BBWidthMonitorService) calculateBBWidthRange(lowestMinBBWidth float64) (minRange, maxRange float64) {
	rangeThresholdPercent := 0.10 // Default ±0.10%
	if s.config != nil {
		rangeThresholdPercent = s.config.PatternDetection.RangeThresholdPercent
	}

	rangeThreshold := lowestMinBBWidth * (rangeThresholdPercent / 100.0)
	minRange = lowestMinBBWidth - rangeThreshold
	maxRange = lowestMinBBWidth + rangeThreshold

	log.Debug("[BB Monitor] Calculated BB width range: %f ± %f = [%f, %f]",
		lowestMinBBWidth, rangeThreshold, minRange, maxRange)

	return minRange, maxRange
}

// triggerBBRangeAlert triggers an alert for BB range pattern detection
func (s *BBWidthMonitorService) triggerBBRangeAlert(ctx context.Context, stock response.StockGroupStockDTO, currentBBWidth, lowestMinBBWidth float64, patternLength int) error {
	// Check if alert service is available
	if s.alertService == nil {
		log.Warn("[BB Monitor] Alert service not available for %s", stock.Symbol)
		return nil
	}

	// Create alert event
	alert := AlertEvent{
		Symbol:           stock.Symbol,
		BBWidth:          currentBBWidth,
		LowestMinBBWidth: lowestMinBBWidth,
		PatternLength:    patternLength,
		AlertType:        "bb_range_contracting",
		Timestamp:        time.Now(),
		GroupID:          "", // Will be set by the calling function if needed
		Message:          fmt.Sprintf("BB Range Alert: %s - %d consecutive contracting candles in optimal range", stock.Symbol, patternLength),
	}

	// Trigger the alert
	err := s.alertService.PlayAlert(alert)
	if err != nil {
		log.Error("[BB Monitor] Failed to trigger alert for %s: %v", stock.Symbol, err)
		return fmt.Errorf("failed to trigger alert: %w", err)
	}

	log.Info("[BB Monitor] Successfully triggered BB range alert for %s", stock.Symbol)
	return nil
}

// isMarketHours checks if current time is within market hours (9:15 AM - 3:30 PM IST)
func (s *BBWidthMonitorService) isMarketHours() bool {
	now := time.Now()

	// Convert to IST (UTC+5:30)
	ist := now.UTC().Add(5*time.Hour + 30*time.Minute)

	// Market hours: 9:15 AM to 3:30 PM IST
	marketStart := time.Date(ist.Year(), ist.Month(), ist.Day(), 9, 15, 0, 0, ist.Location())
	marketEnd := time.Date(ist.Year(), ist.Month(), ist.Day(), 15, 30, 0, 0, ist.Location())

	return ist.After(marketStart) && ist.Before(marketEnd)
}

// ProcessBBWidth processes BB width directly from an aggregated 5-min candle
func (s *BBWidthMonitorService) ProcessBBWidth(
	ctx context.Context,
	instrumentKey string,
	candle domain.AggregatedCandle,
	bbWidthHistory []domain.IndicatorValue, // pass recent BB width history (including this candle)
	stockMeta response.StockGroupStockDTO, // pass stock meta for alerting
) error {
	bbWidth := candle.BBWidth
	if bbWidth <= 0 {
		log.Debug("[BB Monitor] Invalid BB width for %s: %f", instrumentKey, bbWidth)
		return nil
	}
	// Pattern detection: use provided bbWidthHistory (should be sorted oldest to newest)
	if len(bbWidthHistory) < 3 {
		log.Debug("[BB Monitor] Insufficient BB width history for %s (need 3+, got %d)", instrumentKey, len(bbWidthHistory))
		return nil
	}
	isContracting := s.isContractingPattern(bbWidthHistory)
	if !isContracting {
		log.Debug("[BB Monitor] No contracting pattern detected for %s", instrumentKey)
		return nil
	}
	// Get lowest_min_bb_width from CSV (unchanged)
	lowestMinBBWidth, err := s.getLowestMinBBWidth(ctx, instrumentKey)
	if err != nil {
		return fmt.Errorf("failed to get lowest min BB width: %w", err)
	}
	if lowestMinBBWidth <= 0 {
		log.Debug("[BB Monitor] Invalid lowest min BB width for %s: %f", instrumentKey, lowestMinBBWidth)
		return nil
	}
	minRange, maxRange := s.calculateBBWidthRange(lowestMinBBWidth)
	if bbWidth >= minRange && bbWidth <= maxRange {
		log.Info("[BB Monitor] BB Range Alert: %s - %d consecutive contracting candles in optimal range (BB width: %f, range: %f-%f, lowest: %f)",
			instrumentKey, len(bbWidthHistory), bbWidth, minRange, maxRange, lowestMinBBWidth)
		return s.triggerBBRangeAlert(ctx, stockMeta, bbWidth, lowestMinBBWidth, len(bbWidthHistory))
	}
	log.Debug("[BB Monitor] Contracting pattern detected for %s but outside optimal range (BB width: %f, range: %f-%f)",
		instrumentKey, bbWidth, minRange, maxRange)
	return nil
}
