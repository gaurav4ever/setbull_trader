package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"setbull_trader/internal/core/dto/response"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
)

// CandleAggregationService provides operations for aggregating candles to different timeframes
type CandleAggregationService struct {
	candleRepo           repository.CandleRepository
	candle5MinRepo       repository.Candle5MinRepository
	batchFetchService    *BatchFetchService
	tradingCalendar      *TradingCalendarService
	utilityService       *UtilityService
	candleCloseListeners []CandleCloseListener // listeners for candle close events
}

// DateRangeSegment represents a segment of time that needs to be fetched
type DateRangeSegment struct {
	StartDate time.Time
	EndDate   time.Time
	Type      string // "full_range", "historical_backfill", or "recent_update"
}

// NewCandleAggregationService creates a new candle aggregation service
func NewCandleAggregationService(
	candleRepo repository.CandleRepository,
	candle5MinRepo repository.Candle5MinRepository,
	batchFetchService *BatchFetchService,
	tradingCalendar *TradingCalendarService,
	utilityService *UtilityService,
) *CandleAggregationService {
	return &CandleAggregationService{
		candleRepo:        candleRepo,
		candle5MinRepo:    candle5MinRepo,
		batchFetchService: batchFetchService,
		tradingCalendar:   tradingCalendar,
		utilityService:    utilityService,
	}
}

// Get5MinCandles retrieves 5-minute candles for the given instrument and time range
func (s *CandleAggregationService) Get5MinCandles(
	ctx context.Context,
	instrumentKey string,
	start, end time.Time,
) ([]domain.AggregatedCandle, error) {
	// Validate inputs
	if instrumentKey == "" {
		return nil, fmt.Errorf("instrument key is required")
	}

	if end.Before(start) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	// If end time is zero, use current time
	if end.IsZero() {
		end = time.Now()
	}

	// If start time is zero, use end time minus 7 days
	if start.IsZero() {
		start = end.AddDate(0, 0, -7)
	}

	// SOLUTION: Use trading calendar to get proper extended historical data for BB calculation
	// BB calculation needs 20 periods, so we need to fetch sufficient historical data
	bbPeriod := 20

	// Calculate how many 5-minute periods we need (20 periods for BB calculation)
	// Each trading day has ~75 5-minute periods (9:15 AM to 3:30 PM = 6.25 hours = 375 minutes = 75 periods)
	// We need at least 20 periods, so we'll fetch data from previous trading days if needed
	requiredPeriods := bbPeriod + 20 // Extra buffer for safety

	// Calculate the extended start time using trading calendar
	extendedStart, err := s.calculateExtendedStartForBB(ctx, instrumentKey, start, requiredPeriods)

	if err != nil {
		return nil, fmt.Errorf("failed to calculate extended start time: %w", err)
	}

	log.Info("Retrieving 5-minute candles for %s from %s to %s (extended from %s for BB calculation)",
		instrumentKey, extendedStart.Format(time.RFC3339), end.Format(time.RFC3339), start.Format(time.RFC3339))

	// Get the aggregated candles from the repository with extended range
	allCandles, err := s.candleRepo.GetAggregated5MinCandles(ctx, instrumentKey, extendedStart, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated 5-minute candles: %w", err)
	}

	// Calculate indicators using the full dataset
	indicatorService := NewTechnicalIndicatorService(s.candleRepo)
	candleSlice := AggregatedCandlesToCandles(allCandles)

	// Validate data ordering (Past → Latest)
	if err := ValidateDataOrdering(candleSlice); err != nil {
		log.Warn("Data ordering validation failed: %v", err)
		// Continue with calculation but log the issue
	} else {
		log.Info("Data ordering validation passed: candles are in Past → Latest order")
	}

	// Add data validation and logging
	log.Info("Calculating indicators for %d candles (need minimum %d for BB calculation)", len(candleSlice), bbPeriod)

	if len(candleSlice) < bbPeriod {
		log.Warn("Insufficient data for BB calculation: %d candles available, need %d", len(candleSlice), bbPeriod)
		// Could implement fallback calculation here
	}

	// Calculate indicators (now has sufficient historical data)
	ma9 := indicatorService.CalculateSMA(candleSlice, 9)
	bbUpper, bbMiddle, bbLower := indicatorService.CalculateBollingerBands(candleSlice, bbPeriod, 2.0)
	vwap := indicatorService.CalculateVWAP(candleSlice)
	ema5 := indicatorService.CalculateEMAV2(candleSlice, 5)
	ema9ema := indicatorService.CalculateEMAV2(candleSlice, 9)
	ema50 := indicatorService.CalculateEMAV2(candleSlice, 50)
	atr := indicatorService.CalculateATRV2(candleSlice, 14)
	rsi := indicatorService.CalculateRSIV2(candleSlice, 14)

	// Calculate three different BB Width values
	bbWidth := indicatorService.CalculateBBWidth(bbUpper, bbLower, bbMiddle)                                         // upper - lower
	bbWidthNormalized := indicatorService.CalculateBBWidthNormalized(bbUpper, bbLower, bbMiddle)                     // (upper - lower) / middle
	bbWidthNormalizedPercentage := indicatorService.CalculateBBWidthNormalizedPercentage(bbUpper, bbLower, bbMiddle) // ((upper - lower) / middle) * 100

	// Log indicator calculation results
	log.Info("Indicator calculation complete - BB Upper: %d values, BB Middle: %d values, BB Width: %d values, BB Width Normalized: %d values, BB Width Normalized Percentage: %d values",
		len(bbUpper), len(bbMiddle), len(bbWidth), len(bbWidthNormalized), len(bbWidthNormalizedPercentage))

	// Map indicator values by timestamp for fast lookup, handling NaN values
	ma9Map := make(map[time.Time]float64)
	for _, v := range ma9 {
		ma9Map[v.Timestamp] = handleNaN(v.Value)
	}
	bbUpperMap := make(map[time.Time]float64)
	bbMiddleMap := make(map[time.Time]float64)
	bbLowerMap := make(map[time.Time]float64)
	for _, v := range bbUpper {
		bbUpperMap[v.Timestamp] = handleNaN(v.Value)
	}
	for _, v := range bbMiddle {
		bbMiddleMap[v.Timestamp] = handleNaN(v.Value)
	}
	for _, v := range bbLower {
		bbLowerMap[v.Timestamp] = handleNaN(v.Value)
	}
	vwapMap := make(map[time.Time]float64)
	for _, v := range vwap {
		vwapMap[v.Timestamp] = handleNaN(v.Value)
	}
	ema5Map := make(map[time.Time]float64)
	for _, v := range ema5 {
		ema5Map[v.Timestamp] = handleNaN(v.Value)
	}
	ema9emaMap := make(map[time.Time]float64)
	for _, v := range ema9ema {
		ema9emaMap[v.Timestamp] = handleNaN(v.Value)
	}
	ema50Map := make(map[time.Time]float64)
	for _, v := range ema50 {
		ema50Map[v.Timestamp] = handleNaN(v.Value)
	}
	atrMap := make(map[time.Time]float64)
	for _, v := range atr {
		atrMap[v.Timestamp] = handleNaN(v.Value)
	}
	rsiMap := make(map[time.Time]float64)
	for _, v := range rsi {
		rsiMap[v.Timestamp] = handleNaN(v.Value)
	}
	bbWidthMap := make(map[time.Time]float64)
	for _, v := range bbWidth {
		bbWidthMap[v.Timestamp] = handleNaN(v.Value)
	}
	bbWidthNormalizedMap := make(map[time.Time]float64)
	for _, v := range bbWidthNormalized {
		bbWidthNormalizedMap[v.Timestamp] = handleNaN(v.Value)
	}
	bbWidthNormalizedPercentageMap := make(map[time.Time]float64)
	for _, v := range bbWidthNormalizedPercentage {
		bbWidthNormalizedPercentageMap[v.Timestamp] = handleNaN(v.Value)
	}

	// Filter to only return candles in the requested range
	var resultCandles []domain.AggregatedCandle
	for _, candle := range allCandles {
		if candle.Timestamp.Before(start) {
			continue // Skip candles before requested start
		}
		if candle.Timestamp.After(end) {
			break // Stop when we exceed requested end
		}
		resultCandles = append(resultCandles, candle)
	}

	// Populate indicator fields in the result candles
	for i := range resultCandles {
		ts := resultCandles[i].Timestamp
		if val, ok := ma9Map[ts]; ok {
			resultCandles[i].MA9 = val
		}
		if val, ok := bbUpperMap[ts]; ok {
			resultCandles[i].BBUpper = val
			resultCandles[i].BBUpper = math.Round(val*100) / 100
		}
		if val, ok := bbMiddleMap[ts]; ok {
			resultCandles[i].BBMiddle = val
			resultCandles[i].BBMiddle = math.Round(val*100) / 100
		}
		if val, ok := bbLowerMap[ts]; ok {
			resultCandles[i].BBLower = val
			resultCandles[i].BBLower = math.Round(val*100) / 100
		}
		if val, ok := vwapMap[ts]; ok {
			resultCandles[i].VWAP = val
			resultCandles[i].VWAP = math.Round(val*100) / 100
		}
		if val, ok := ema5Map[ts]; ok {
			resultCandles[i].EMA5 = val
			resultCandles[i].EMA5 = math.Round(val*100) / 100
		}
		if val, ok := ema9emaMap[ts]; ok {
			resultCandles[i].EMA9 = val
			resultCandles[i].EMA9 = math.Round(val*100) / 100
		}
		if val, ok := ema50Map[ts]; ok {
			resultCandles[i].EMA50 = val
			resultCandles[i].EMA50 = math.Round(val*100) / 100
		}
		if val, ok := atrMap[ts]; ok {
			resultCandles[i].ATR = val
			resultCandles[i].ATR = math.Round(val*100) / 100
		}
		if val, ok := rsiMap[ts]; ok {
			resultCandles[i].RSI = val
			resultCandles[i].RSI = math.Round(val*100) / 100
		}
		if val, ok := bbWidthMap[ts]; ok {
			resultCandles[i].BBWidth = val
			resultCandles[i].BBWidth = math.Round(val*100) / 100
		}
		if val, ok := bbWidthNormalizedMap[ts]; ok {
			resultCandles[i].BBWidthNormalized = val
			resultCandles[i].BBWidthNormalized = math.Round(val*10000) / 10000 // 4 decimal places for normalized
		}
		if val, ok := bbWidthNormalizedPercentageMap[ts]; ok {
			resultCandles[i].BBWidthNormalizedPercentage = val
			resultCandles[i].BBWidthNormalizedPercentage = math.Round(val*10000) / 10000 // 4 decimal places for percentage
		}
		lowestBBWidth, err := s.utilityService.getLowestMinBBWidth(resultCandles[i].InstrumentKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get lowest BB width: %w", err)
		}
		resultCandles[i].LowestBBWidth = lowestBBWidth
	}

	log.Info("Returning %d candles with indicators (requested range: %s to %s)",
		len(resultCandles), start.Format(time.RFC3339), end.Format(time.RFC3339))

	return resultCandles, nil
}

// calculateExtendedStartForBB calculates the proper extended start time for BB calculation
// considering trading hours and market boundaries
func (s *CandleAggregationService) calculateExtendedStartForBB(
	ctx context.Context,
	instrumentKey string,
	requestedStart time.Time,
	requiredPeriods int,
) (time.Time, error) {
	// Indian market hours: 9:15 AM to 3:30 PM (IST)
	// Each trading day has 75 5-minute periods (375 minutes / 5 = 75)
	periodsPerDay := 75

	// Calculate how many trading days we need to go back
	tradingDaysNeeded := (requiredPeriods + periodsPerDay - 1) / periodsPerDay // Ceiling division

	log.Info("Calculating extended start for BB calculation: need %d periods, %d trading days back from %s",
		requiredPeriods, tradingDaysNeeded, requestedStart.Format(time.RFC3339))

	// Start from the requested start time and go back by trading days
	extendedStart := requestedStart

	// Go back by the required number of trading days
	for i := 0; i < tradingDaysNeeded; i++ {
		extendedStart = s.tradingCalendar.PreviousTradingDay(extendedStart)
	}

	extendedStart = extendedStart.In(time.FixedZone("Asia/Kolkata", 5*60*60))

	// // Set the time to market open (9:15 AM IST)
	// year, month, day := extendedStart.Date()
	// extendedStart = time.Date(year, month, day, 9, 15, 0, 0, time.FixedZone("Asia/Kolkata", 5*60*60))

	// log.Info("Extended start calculated: %s (went back %d trading days)",
	// 	extendedStart.Format(time.RFC3339), tradingDaysNeeded)

	return extendedStart, nil
}

// GetDailyCandles retrieves daily candles for the given instrument and time range
func (s *CandleAggregationService) GetDailyCandles(
	ctx context.Context,
	instrumentKey string,
	start, end time.Time,
) ([]domain.Candle, error) {
	// Validate inputs
	if instrumentKey == "" {
		return nil, fmt.Errorf("instrument key is required")
	}

	if end.Before(start) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	// If end time is zero, use current time
	if end.IsZero() {
		end = time.Now()
	}

	// If start time is zero, use end time minus 30 days
	if start.IsZero() {
		start = end.AddDate(0, 0, -30)
	}

	log.Info("Retrieving daily candles for %s from %s to %s",
		instrumentKey, start.Format(time.RFC3339), end.Format(time.RFC3339))

	// Get the aggregated candles from the repository
	candles, err := s.candleRepo.GetDailyCandlesByTimeframe(ctx, instrumentKey, start)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated daily candles: %w", err)
	}

	return candles, nil
}

// GetMultiTimeframeCandles retrieves candles for multiple timeframes for an instrument
func (s *CandleAggregationService) GetMultiTimeframeCandles(
	ctx context.Context,
	instrumentKey string,
	timeframes []string,
	start, end time.Time,
) (map[string][]domain.AggregatedCandle, error) {
	result := make(map[string][]domain.AggregatedCandle)

	for _, timeframe := range timeframes {
		var candles []domain.AggregatedCandle
		var err error

		switch timeframe {
		case "1minute":
			// For 1-minute candles, fetch directly from the database
			minuteCandles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, "1minute", start, end)
			if err != nil {
				return nil, fmt.Errorf("failed to get 1-minute candles: %w", err)
			}

			// Convert to AggregatedCandle format for consistency
			candles = make([]domain.AggregatedCandle, len(minuteCandles))
			for i, c := range minuteCandles {
				candles[i] = domain.AggregatedCandle{
					InstrumentKey: c.InstrumentKey,
					Timestamp:     c.Timestamp,
					Open:          c.Open,
					High:          c.High,
					Low:           c.Low,
					Close:         c.Close,
					Volume:        c.Volume,
					OpenInterest:  c.OpenInterest,
					TimeInterval:  c.TimeInterval,
				}
			}

		case "5minute":
			candles, err = s.Get5MinCandles(ctx, instrumentKey, start, end)
			if err != nil {
				return nil, fmt.Errorf("failed to get 5-minute candles: %w", err)
			}

		default:
			log.Warn("Unsupported timeframe: %s", timeframe)
			continue
		}

		result[timeframe] = candles
	}

	return result, nil
}

// CacheAggregatedCandles caches aggregated candles for future use
// This is useful for timeframes that are queried frequently
func (s *CandleAggregationService) CacheAggregatedCandles(
	ctx context.Context,
	instrumentKey string,
	timeframe string,
	start, end time.Time,
) error {
	var candles []domain.AggregatedCandle
	var err error

	// Get the aggregated candles based on timeframe
	switch timeframe {
	case "5minute":
		candles, err = s.Get5MinCandles(ctx, instrumentKey, start, end)
	default:
		return fmt.Errorf("unsupported timeframe for caching: %s", timeframe)
	}

	if err != nil {
		return fmt.Errorf("failed to get aggregated candles for caching: %w", err)
	}

	// Convert to CandleData for storage
	candleData := make([]domain.CandleData, len(candles))
	for i, c := range candles {
		candleData[i] = domain.CandleData{
			InstrumentKey: c.InstrumentKey,
			Timestamp:     c.Timestamp,
			Open:          c.Open,
			High:          c.High,
			Low:           c.Low,
			Close:         c.Close,
			Volume:        c.Volume,
			OpenInterest:  c.OpenInterest,
			Interval:      c.TimeInterval,
		}
	}

	// Store the converted data
	err = s.candleRepo.StoreAggregatedCandles(ctx, candleData)
	if err != nil {
		return fmt.Errorf("failed to store aggregated candles: %w", err)
	}

	log.Info("Cached %d %s candles for %s from %s to %s",
		len(candles), timeframe, instrumentKey,
		start.Format(time.RFC3339), end.Format(time.RFC3339))

	return nil
}

// GetStocksWithExistingDailyCandles returns a list of instrument keys that already have daily candle data
// for the specified date range
func (s *CandleAggregationService) GetStocksWithExistingDailyCandles(
	ctx context.Context,
	startDate, endDate time.Time,
) ([]string, error) {
	return s.candleRepo.GetStocksWithExistingDailyCandles(ctx, startDate, endDate)
}

// GetOptimalDateRangeForStock determines the optimal date range(s) to fetch for a stock
// based on the existing data and requested date range
func (s *CandleAggregationService) GetOptimalDateRangeForStock(
	ctx context.Context,
	instrumentKey string,
	interval string,
	endDate time.Time,
	maxDays int,
) ([]DateRangeSegment, bool, error) {
	// If no end date specified, use current date
	if endDate.IsZero() {
		endDate = time.Now()
	}

	// If endDate is not a trading day, adjust to the previous trading day
	if !s.tradingCalendar.IsTradingDay(endDate) {
		endDate = s.tradingCalendar.PreviousTradingDay(endDate)
		log.Info("Adjusted end date to previous trading day: %s", endDate.Format("2006-01-02"))
	}

	// Calculate the requested start date based on trading days
	requestedStartDate := s.tradingCalendar.SubtractTradingDays(endDate, maxDays)

	log.Info("Determining optimal date range for %s with interval %s from %s to %s",
		instrumentKey,
		interval,
		requestedStartDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))

	// Get the date range of existing candles
	earliestExisting, latestExisting, exists, err := s.candleRepo.GetCandleDateRange(ctx, instrumentKey, interval)
	if err != nil {
		log.Error("Failed to get candle date range for %s: %v", instrumentKey, err)
		return nil, false, fmt.Errorf("failed to get candle date range: %w", err)
	}

	// If no data exists, fetch the full range
	if !exists {
		log.Info("No existing data for %s - fetching full range from %s to %s",
			instrumentKey,
			requestedStartDate.Format("2006-01-02"),
			endDate.Format("2006-01-02"))

		return []DateRangeSegment{
			{
				StartDate: requestedStartDate,
				EndDate:   endDate,
				Type:      "full_range",
			},
		}, false, nil
	}

	// Log the existing data range
	log.Info("Existing data for %s spans from %s to %s",
		instrumentKey,
		earliestExisting.Format("2006-01-02"),
		latestExisting.Format("2006-01-02"))

	// Initialize segments slice to hold the date ranges we need to fetch
	var segments []DateRangeSegment

	// Check if we need to backfill historical data
	if requestedStartDate.Before(earliestExisting) {
		// Calculate the day before the earliest existing data
		dayBeforeEarliest := s.tradingCalendar.PreviousTradingDay(earliestExisting)

		// Add a segment for the historical backfill
		segments = append(segments, DateRangeSegment{
			StartDate: requestedStartDate,
			EndDate:   dayBeforeEarliest,
			Type:      "historical_backfill",
		})

		log.Info("Need to backfill historical data for %s from %s to %s",
			instrumentKey,
			requestedStartDate.Format("2006-01-02"),
			dayBeforeEarliest.Format("2006-01-02"))
	}

	// Check if we need to fetch recent data
	if latestExisting.Before(endDate) {
		// Calculate the day after the latest existing data
		dayAfterLatest := s.tradingCalendar.NextTradingDay(latestExisting)

		// Add a segment for the recent data
		segments = append(segments, DateRangeSegment{
			StartDate: dayAfterLatest,
			EndDate:   endDate,
			Type:      "recent_update",
		})

		log.Info("Need to fetch recent data for %s from %s to %s",
			instrumentKey,
			dayAfterLatest.Format("2006-01-02"),
			endDate.Format("2006-01-02"))
	}

	// If no segments were added, the data is already up to date
	if len(segments) == 0 {
		log.Info("Data for %s is already up to date for the requested range",
			instrumentKey)
		return nil, true, nil
	}

	return segments, false, nil
}

// ProcessStockDailyCandles processes daily candles for a single stock with backfill support
func (s *CandleAggregationService) ProcessStockDailyCandles(
	ctx context.Context,
	stock domain.StockUniverse,
	endDate time.Time,
	maxDays int,
) (ProcessResult, error) {
	result := ProcessResult{
		Symbol:        stock.Symbol,
		InstrumentKey: stock.InstrumentKey,
	}

	if stock.InstrumentKey == "" {
		result.Status = "failed"
		result.Error = "instrument key is required"
		return result, fmt.Errorf("instrument key is required")
	}

	// Determine the optimal date range segments
	segments, isUpToDate, err := s.GetOptimalDateRangeForStock(ctx, stock.InstrumentKey, "day", endDate, maxDays)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to determine date ranges: %v", err)
		return result, fmt.Errorf("failed to determine date ranges: %w", err)
	}

	// If data is already up to date, return early
	if isUpToDate {
		result.Status = "skipped"
		result.Message = "data is already up to date"
		return result, nil
	}

	totalProcessed := 0

	// Process each segment using the existing batchFetchService
	for _, segment := range segments {
		// Format dates for API call
		fromDateStr := segment.StartDate.Format("2006-01-02")
		toDateStr := segment.EndDate.Format("2006-01-02")

		log.Info("Processing %s data for %s (%s) from %s to %s",
			segment.Type,
			stock.Symbol,
			stock.InstrumentKey,
			fromDateStr,
			toDateStr)

		// Use the batch fetch service to process this segment
		batchRequest := &domain.BatchStoreHistoricalDataRequest{
			InstrumentKeys: []string{stock.InstrumentKey},
			Interval:       "day",
			FromDate:       fromDateStr,
			ToDate:         toDateStr,
		}

		batchResult, err := s.batchFetchService.ProcessBatchRequest(ctx, batchRequest)
		if err != nil {
			result.Status = "failed"
			result.Error = fmt.Sprintf("failed to process %s segment: %v", segment.Type, err)
			return result, fmt.Errorf("failed to process %s segment: %w", segment.Type, err)
		}

		log.Info("Successfully processed %d candles for %s segment of %s",
			batchResult.SuccessfulItems,
			segment.Type,
			stock.Symbol)

		totalProcessed += batchResult.SuccessfulItems
	}

	// Update the result
	result.Status = "success"
	result.CandlesProcessed = totalProcessed
	result.Segments = len(segments)

	// Add segment details
	for _, segment := range segments {
		result.SegmentDetails = append(result.SegmentDetails, SegmentDetail{
			Type:      segment.Type,
			StartDate: segment.StartDate.Format("2006-01-02"),
			EndDate:   segment.EndDate.Format("2006-01-02"),
		})
	}

	return result, nil
}

// ProcessResult represents the result of processing a stock
type ProcessResult struct {
	Symbol           string          `json:"symbol"`
	InstrumentKey    string          `json:"instrument_key"`
	Status           string          `json:"status"` // "success", "skipped", or "failed"
	Message          string          `json:"message,omitempty"`
	Error            string          `json:"error,omitempty"`
	CandlesProcessed int             `json:"candles_processed"`
	Segments         int             `json:"segments"`
	SegmentDetails   []SegmentDetail `json:"segment_details,omitempty"`
}

// SegmentDetail represents details about a processed segment
type SegmentDetail struct {
	Type      string `json:"type"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// CandleCloseListener is a callback function type for candle close events
// It receives a slice of newly closed aggregated candles (e.g., 5-min)
type CandleCloseListener func(candles []domain.AggregatedCandle, stock response.StockGroupStockDTO)

// RegisterCandleCloseListener registers a listener for candle close events
func (s *CandleAggregationService) RegisterCandleCloseListener(listener CandleCloseListener) {
	s.candleCloseListeners = append(s.candleCloseListeners, listener)
}

// FireCandleCloseEvent notifies all registered listeners of new closed candles
func (s *CandleAggregationService) FireCandleCloseEvent(candles []domain.AggregatedCandle, stock response.StockGroupStockDTO) {
	for _, listener := range s.candleCloseListeners {
		go listener(candles, stock) // fire in goroutine to avoid blocking
	}
}

// Example: Call this after fetching/aggregating new 5-min candles
func (s *CandleAggregationService) NotifyOnNew5MinCandles(ctx context.Context, stock response.StockGroupStockDTO, start, end time.Time) error {
	log.Info("Notifying listeners of new 5-min candles for %s from %s to %s", stock.InstrumentKey, start.Format(time.RFC3339), end.Format(time.RFC3339))
	candles, err := s.Get5MinCandles(ctx, stock.InstrumentKey, start, end)
	if err != nil {
		return err
	}
	if len(candles) > 0 {
		s.FireCandleCloseEvent(candles, stock)
	}
	return nil
}

// Example stub: How a scheduler or other service would register a listener
func ExampleRegisterCandleCloseListener(s *CandleAggregationService) {
	s.RegisterCandleCloseListener(func(candles []domain.AggregatedCandle, stock response.StockGroupStockDTO) {
		for _, candle := range candles {
			log.Info("[Listener] 5-min candle closed: %+v", candle)
			// Here you would trigger group execution logic, etc.
		}
	})
}

// handleNaN converts NaN and Infinity values to 0.0 to prevent JSON marshaling issues
func handleNaN(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0.0
	}
	return value
}

// Aggregate 1-minute candles to 5-minute candles in-memory, calculate indicators, and trigger BB width monitoring
func (s *CandleAggregationService) Aggregate5MinCandlesWithIndicators(
	ctx context.Context,
	instrumentKey string,
	startTime, endTime time.Time,
	bbWidthCallback func(ctx context.Context, instrumentKey string, candle domain.AggregatedCandle),
) error {
	// 1. Fetch all 1-minute candles for the time range
	oneMinCandles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, "1minute", startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to fetch 1-min candles: %w", err)
	}
	if len(oneMinCandles) == 0 {
		return fmt.Errorf("no 1-min candles found for aggregation")
	}
	// 2. Aggregate to 5-minute candles
	fiveMinCandles := aggregateTo5Min(oneMinCandles)
	if len(fiveMinCandles) == 0 {
		return fmt.Errorf("aggregation to 5-min candles produced no results")
	}
	// Convert []AggregatedCandle to []Candle for indicator calculation
	candleSlice := AggregatedCandlesToCandles(fiveMinCandles)
	// 3. Calculate indicators on 5-minute data
	indicatorService := NewTechnicalIndicatorService(s.candleRepo)
	ma9 := indicatorService.CalculateSMA(candleSlice, 9)
	bbUpper, bbMiddle, bbLower := indicatorService.CalculateBollingerBands(candleSlice, 20, 2.0)
	bbWidth := indicatorService.CalculateBBWidth(bbUpper, bbLower, bbMiddle)
	vwap := indicatorService.CalculateVWAP(candleSlice)
	ema5 := indicatorService.CalculateEMAV2(candleSlice, 5)
	ema9 := indicatorService.CalculateEMAV2(candleSlice, 9)
	ema50 := indicatorService.CalculateEMAV2(candleSlice, 50)
	atr := indicatorService.CalculateATRV2(candleSlice, 14)
	rsi := indicatorService.CalculateRSIV2(candleSlice, 14)
	// Map indicators by timestamp
	ma9Map := make(map[time.Time]float64)
	for _, v := range ma9 {
		ma9Map[v.Timestamp] = v.Value
	}
	bbUpperMap := make(map[time.Time]float64)
	bbMiddleMap := make(map[time.Time]float64)
	bbLowerMap := make(map[time.Time]float64)
	for i := range bbUpper {
		bbUpperMap[bbUpper[i].Timestamp] = bbUpper[i].Value
		bbMiddleMap[bbMiddle[i].Timestamp] = bbMiddle[i].Value
		bbLowerMap[bbLower[i].Timestamp] = bbLower[i].Value
	}
	bbWidthMap := make(map[time.Time]float64)
	for _, v := range bbWidth {
		bbWidthMap[v.Timestamp] = v.Value
	}
	vwapMap := make(map[time.Time]float64)
	for _, v := range vwap {
		vwapMap[v.Timestamp] = v.Value
	}
	ema5Map := make(map[time.Time]float64)
	for _, v := range ema5 {
		ema5Map[v.Timestamp] = v.Value
	}
	ema9Map := make(map[time.Time]float64)
	for _, v := range ema9 {
		ema9Map[v.Timestamp] = v.Value
	}
	ema50Map := make(map[time.Time]float64)
	for _, v := range ema50 {
		ema50Map[v.Timestamp] = v.Value
	}
	atrMap := make(map[time.Time]float64)
	for _, v := range atr {
		atrMap[v.Timestamp] = v.Value
	}
	rsiMap := make(map[time.Time]float64)
	for _, v := range rsi {
		rsiMap[v.Timestamp] = v.Value
	}
	// 4. Enrich 5-min candles with indicators and trigger callback
	for i := range fiveMinCandles {
		ts := fiveMinCandles[i].Timestamp
		if v, ok := ma9Map[ts]; ok {
			fiveMinCandles[i].MA9 = v
		}
		if v, ok := bbUpperMap[ts]; ok {
			fiveMinCandles[i].BBUpper = v
		}
		if v, ok := bbMiddleMap[ts]; ok {
			fiveMinCandles[i].BBMiddle = v
		}
		if v, ok := bbLowerMap[ts]; ok {
			fiveMinCandles[i].BBLower = v
		}
		if v, ok := bbWidthMap[ts]; ok {
			fiveMinCandles[i].BBWidth = v
		}
		if v, ok := vwapMap[ts]; ok {
			fiveMinCandles[i].VWAP = v
		}
		if v, ok := ema5Map[ts]; ok {
			fiveMinCandles[i].EMA5 = v
		}
		if v, ok := ema9Map[ts]; ok {
			fiveMinCandles[i].EMA9 = v
		}
		if v, ok := ema50Map[ts]; ok {
			fiveMinCandles[i].EMA50 = v
		}
		if v, ok := atrMap[ts]; ok {
			fiveMinCandles[i].ATR = v
		}
		if v, ok := rsiMap[ts]; ok {
			fiveMinCandles[i].RSI = v
		}
		// Trigger BB width monitoring callback
		if bbWidthCallback != nil {
			bbWidthCallback(ctx, instrumentKey, fiveMinCandles[i])
		}
	}
	return nil
}

// Helper: Aggregate 1-min candles to 5-min candles (OHLCV)
func aggregateTo5Min(oneMinCandles []domain.Candle) []domain.AggregatedCandle {
	if len(oneMinCandles) == 0 {
		return nil
	}
	var result []domain.AggregatedCandle
	var bucket []domain.Candle
	var currentBucketStart time.Time
	for _, c := range oneMinCandles {
		bucketStart := c.Timestamp.Truncate(5 * time.Minute)
		if currentBucketStart.IsZero() || !bucketStart.Equal(currentBucketStart) {
			if len(bucket) > 0 {
				result = append(result, aggregateBucketTo5Min(bucket))
			}
			bucket = bucket[:0]
			currentBucketStart = bucketStart
		}
		bucket = append(bucket, c)
	}
	if len(bucket) > 0 {
		result = append(result, aggregateBucketTo5Min(bucket))
	}
	return result
}

// Helper: Aggregate a bucket of 1-min candles to a single 5-min candle
func aggregateBucketTo5Min(bucket []domain.Candle) domain.AggregatedCandle {
	if len(bucket) == 0 {
		return domain.AggregatedCandle{}
	}
	open := bucket[0].Open
	high := bucket[0].High
	low := bucket[0].Low
	close := bucket[len(bucket)-1].Close
	volume := int64(0)
	openInterest := int64(0)
	for _, c := range bucket {
		if c.High > high {
			high = c.High
		}
		if c.Low < low {
			low = c.Low
		}
		volume += c.Volume
		openInterest = c.OpenInterest // Use last open interest
	}
	return domain.AggregatedCandle{
		InstrumentKey: bucket[0].InstrumentKey,
		Timestamp:     bucket[0].Timestamp.Truncate(5 * time.Minute),
		Open:          open,
		High:          high,
		Low:           low,
		Close:         close,
		Volume:        volume,
		OpenInterest:  openInterest,
		TimeInterval:  "5minute",
	}
}

// Store5MinCandles stores 5-minute candles with indicators in the 5-minute repository
func (s *CandleAggregationService) Store5MinCandles(ctx context.Context, candles []domain.AggregatedCandle) error {
	if len(candles) == 0 {
		return nil
	}

	// Convert AggregatedCandle to Candle5Min for storage
	candlesForStorage := make([]domain.Candle5Min, len(candles))
	for i, aggCandle := range candles {
		candlesForStorage[i] = domain.Candle5Min{
			InstrumentKey: aggCandle.InstrumentKey,
			Timestamp:     aggCandle.Timestamp,
			Open:          aggCandle.Open,
			High:          aggCandle.High,
			Low:           aggCandle.Low,
			Close:         aggCandle.Close,
			Volume:        aggCandle.Volume,
			OpenInterest:  aggCandle.OpenInterest,
			TimeInterval:  "5minute",
			// Copy indicator values
			BBUpper:                     aggCandle.BBUpper,
			BBMiddle:                    aggCandle.BBMiddle,
			BBLower:                     aggCandle.BBLower,
			BBWidth:                     aggCandle.BBWidth,
			BBWidthNormalized:           aggCandle.BBWidthNormalized,
			BBWidthNormalizedPercentage: aggCandle.BBWidthNormalizedPercentage,
			EMA5:                        aggCandle.EMA5,
			EMA9:                        aggCandle.EMA9,
			EMA20:                       aggCandle.EMA20,
			EMA50:                       aggCandle.EMA50,
			ATR:                         aggCandle.ATR,
			RSI:                         aggCandle.RSI,
			VWAP:                        aggCandle.VWAP,
			MA9:                         aggCandle.MA9,
			LowestBBWidth:               aggCandle.LowestBBWidth,
		}
	}

	// Store 5-minute candles
	_, err := s.candle5MinRepo.StoreBatch(ctx, candlesForStorage)
	if err != nil {
		return fmt.Errorf("failed to store 5-minute candles: %w", err)
	}

	log.Info("Stored %d 5-minute candles with indicators", len(candlesForStorage))
	return nil
}
