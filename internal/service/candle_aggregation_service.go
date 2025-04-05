package service

import (
	"context"
	"fmt"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
)

// CandleAggregationService provides operations for aggregating candles to different timeframes
type CandleAggregationService struct {
	candleRepo        repository.CandleRepository
	batchFetchService *BatchFetchService
	tradingCalendar   *TradingCalendarService
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
	batchFetchService *BatchFetchService,
	tradingCalendar *TradingCalendarService) *CandleAggregationService {
	return &CandleAggregationService{
		candleRepo:        candleRepo,
		batchFetchService: batchFetchService,
		tradingCalendar:   tradingCalendar,
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

	log.Info("Retrieving 5-minute candles for %s from %s to %s",
		instrumentKey, start.Format(time.RFC3339), end.Format(time.RFC3339))

	// Get the aggregated candles from the repository
	candles, err := s.candleRepo.GetAggregated5MinCandles(ctx, instrumentKey, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated 5-minute candles: %w", err)
	}

	return candles, nil
}

// GetDailyCandles retrieves daily candles for the given instrument and time range
func (s *CandleAggregationService) GetDailyCandles(
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

	// If start time is zero, use end time minus 30 days
	if start.IsZero() {
		start = end.AddDate(0, 0, -30)
	}

	log.Info("Retrieving daily candles for %s from %s to %s",
		instrumentKey, start.Format(time.RFC3339), end.Format(time.RFC3339))

	// Get the aggregated candles from the repository
	candles, err := s.candleRepo.GetAggregatedDailyCandles(ctx, instrumentKey, start, end)
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

		case "day":
			candles, err = s.GetDailyCandles(ctx, instrumentKey, start, end)
			if err != nil {
				return nil, fmt.Errorf("failed to get daily candles: %w", err)
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
	case "day":
		candles, err = s.GetDailyCandles(ctx, instrumentKey, start, end)
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

// // processDailyCandlesParallel processes stocks in parallel
// func (s *Server) processDailyCandlesParallel(
// 	ctx context.Context,
// 	stocks []domain.StockUniverse,
// 	endDate time.Time,
// 	maxDays int,
// ) *DailyCandles {
// 	result := &DailyCandles{
// 		TotalStocks:  len(stocks),
// 		StockResults: make([]StockProcessResult, 0, len(stocks)),
// 		StartTime:    time.Now(),
// 	}

// 	// Use a mutex to protect concurrent access to the result
// 	var resultMutex sync.Mutex

// 	// Use a wait group to wait for all goroutines to finish
// 	var wg sync.WaitGroup

// 	// Use a semaphore to limit concurrency
// 	maxConcurrency := 5 // Adjust based on your system capabilities and API rate limits
// 	semaphore := make(chan struct{}, maxConcurrency)

// 	// Process each stock in parallel
// 	for _, stock := range stocks {
// 		wg.Add(1)

// 		go func(stock domain.StockUniverse) {
// 			defer wg.Done()

// 			// Acquire semaphore slot
// 			semaphore <- struct{}{}
// 			defer func() { <-semaphore }()

// 			stockStartTime := time.Now()

// 			// Skip stocks without instrument key
// 			if stock.InstrumentKey == "" {
// 				log.Warn("Stock %s has no instrument key, skipping", stock.Symbol)

// 				stockResult := StockProcessResult{
// 					Symbol:        stock.Symbol,
// 					InstrumentKey: "",
// 					Status:        "failed",
// 					Error:         "no instrument key",
// 					Duration:      time.Since(stockStartTime).String(),
// 				}

// 				// Update result with mutex protection
// 				resultMutex.Lock()
// 				result.StockResults = append(result.StockResults, stockResult)
// 				result.ProcessedStocks++
// 				result.FailedStocks++
// 				resultMutex.Unlock()

// 				return
// 			}

// 			// Process the stock using the server's candleAggService
// 			processResult, _ := s.candleAggService.ProcessStockDailyCandles(ctx, stock, endDate, maxDays)

// 			// Convert service result to handler result
// 			stockResult := StockProcessResult{
// 				Symbol:           processResult.Symbol,
// 				InstrumentKey:    processResult.InstrumentKey,
// 				Status:           processResult.Status,
// 				Message:          processResult.Message,
// 				Error:            processResult.Error,
// 				CandlesProcessed: processResult.CandlesProcessed,
// 				Segments:         processResult.Segments,
// 				SegmentDetails:   processResult.SegmentDetails,
// 				Duration:         time.Since(stockStartTime).String(),
// 			}

// 			// Update result with mutex protection
// 			resultMutex.Lock()
// 			result.StockResults = append(result.StockResults, stockResult)
// 			result.ProcessedStocks++

// 			// Update counters based on status
// 			switch processResult.Status {
// 			case "success":
// 				result.SuccessfulStocks++
// 			case "skipped":
// 				result.SkippedStocks++
// 			case "failed":
// 				result.FailedStocks++
// 			}
// 			resultMutex.Unlock()

// 		}(stock)
// 	}

// 	// Wait for all goroutines to finish
// 	wg.Wait()

// 	return result
// }

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
