package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/log"
)

// BatchFetchService handles batch fetching of historical data for multiple instruments
type BatchFetchService struct {
	candleService        *CandleProcessingService
	stockUniverseService *StockUniverseService
	maxConcurrent        int
	defaultFromDate      string
	defaultToDate        string
	defaultInterval      string

	// V2 Service Integration
	candleAggregationV2 CandleAggregationServiceInterface
	config              *config.Config

	// Smart Caching for Performance
	dataCache      map[string]*CacheEntry // Cache key: "instrumentKey:interval"
	cacheMutex     sync.RWMutex           // Protects cache access
	cacheEnabled   bool                   // Feature flag for caching
	cacheExpiry    time.Duration          // How long cache entries are valid
	lastCacheFlush time.Time              // When cache was last flushed
}

// CacheEntry represents cached data availability information
type CacheEntry struct {
	InstrumentKey   string    `json:"instrument_key"`
	Interval        string    `json:"interval"`
	EarliestDate    time.Time `json:"earliest_date"`
	LatestDate      time.Time `json:"latest_date"`
	LastChecked     time.Time `json:"last_checked"`
	RecordCount     int       `json:"record_count"`
	IsComplete      bool      `json:"is_complete"`       // Whether we have all expected data
	HasRecentData   bool      `json:"has_recent_data"`   // Whether data is up-to-date
	NextProcessDate time.Time `json:"next_process_date"` // Next date we need to fetch
}

// NewBatchFetchService creates a new batch fetch service with smart caching
func NewBatchFetchService(
	candleService *CandleProcessingService,
	stockUniverseService *StockUniverseService,
	maxConcurrent int,
) *BatchFetchService {
	if maxConcurrent <= 0 {
		maxConcurrent = 5 // Default to 5 concurrent requests
	}

	// Default date range is last 30 days
	now := time.Now()
	defaultToDate := now.Format("2006-01-02")
	defaultFromDate := now.AddDate(0, 0, -30).Format("2006-01-02")

	return &BatchFetchService{
		candleService:        candleService,
		stockUniverseService: stockUniverseService,
		maxConcurrent:        maxConcurrent,
		defaultFromDate:      defaultFromDate,
		defaultToDate:        defaultToDate,
		defaultInterval:      "1minute",
		// V2 services will be injected later via SetV2Services
		candleAggregationV2: nil,
		config:              nil,
		// Smart Cache initialization
		dataCache:      make(map[string]*CacheEntry),
		cacheEnabled:   true,             // Enable caching by default
		cacheExpiry:    30 * time.Minute, // Cache entries expire after 30 minutes
		lastCacheFlush: time.Now(),
	}
}

// SetV2Services injects V2 services and config for enhanced processing capabilities
func (s *BatchFetchService) SetV2Services(candleAggregationV2 CandleAggregationServiceInterface, config *config.Config) {
	s.candleAggregationV2 = candleAggregationV2
	s.config = config
	log.Info("V2 services injected into BatchFetchService - enhanced DataFrame processing enabled")
}

// 🚀 SMART CACHING METHODS FOR SUPER FAST PROCESSING

// GetOptimalIntervalDays returns the optimal interval size based on time interval type (public method for testing)
func (s *BatchFetchService) GetOptimalIntervalDays(interval string) int {
	return s.getOptimalIntervalDays(interval)
}

// getOptimalIntervalDays returns the optimal interval size based on time interval type
func (s *BatchFetchService) getOptimalIntervalDays(interval string) int {
	switch interval {
	case "1minute":
		return 4 // Fetch 4 days at a time for 1-minute data (as requested)
	case "day":
		return 100 // Fetch 100 days at a time for daily data
	case "5minute":
		return 30 // Medium interval for 5-minute data
	case "15minute":
		return 40 // Medium-large interval for 15-minute data
	case "1hour":
		return 60 // Large interval for hourly data
	default:
		return 20 // Conservative default for unknown intervals
	}
}

// getCacheKey generates a cache key for instrument and interval
func (s *BatchFetchService) getCacheKey(instrumentKey, interval string) string {
	return fmt.Sprintf("%s:%s", instrumentKey, interval)
}

// getCachedDataInfo retrieves cached data information for an instrument
func (s *BatchFetchService) getCachedDataInfo(instrumentKey, interval string) (*CacheEntry, bool) {
	if !s.cacheEnabled {
		return nil, false
	}

	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	key := s.getCacheKey(instrumentKey, interval)
	entry, exists := s.dataCache[key]

	if !exists {
		return nil, false
	}

	// Check if cache entry is still valid
	if time.Since(entry.LastChecked) > s.cacheExpiry {
		log.Debug("[CACHE] Cache entry expired for %s, will refresh", key)
		return nil, false
	}

	log.Debug("[CACHE] Cache hit for %s: latest=%s, hasRecent=%t",
		key, entry.LatestDate.Format("2006-01-02"), entry.HasRecentData)
	return entry, true
}

// setCachedDataInfo stores data information in cache
func (s *BatchFetchService) setCachedDataInfo(instrumentKey, interval string, entry *CacheEntry) {
	if !s.cacheEnabled {
		return
	}

	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	key := s.getCacheKey(instrumentKey, interval)
	entry.LastChecked = time.Now()
	s.dataCache[key] = entry

	log.Debug("[CACHE] Cached data info for %s: earliest=%s, latest=%s, count=%d",
		key, entry.EarliestDate.Format("2006-01-02"),
		entry.LatestDate.Format("2006-01-02"), entry.RecordCount)
}

// buildCacheFromDatabase builds cache from existing database data
func (s *BatchFetchService) buildCacheFromDatabase(ctx context.Context, instrumentKeys []string, interval string) error {
	log.Info("[CACHE] Building cache from database for %d instruments with interval %s", len(instrumentKeys), interval)

	start := time.Now()
	cacheBuilt := 0
	cacheFailed := 0

	// Process in smaller batches to avoid overwhelming the database
	const batchSize = 50
	for i := 0; i < len(instrumentKeys); i += batchSize {
		end := i + batchSize
		if end > len(instrumentKeys) {
			end = len(instrumentKeys)
		}

		batch := instrumentKeys[i:end]
		log.Debug("[CACHE] Processing cache batch %d-%d of %d instruments", i+1, end, len(instrumentKeys))

		for _, instrumentKey := range batch {
			// Check if we already have cached info
			if _, exists := s.getCachedDataInfo(instrumentKey, interval); exists {
				continue // Skip if already cached
			}

			// Get data info from database
			earliest, latest, exists, err := s.candleService.candleRepo.GetCandleDateRange(ctx, instrumentKey, interval)
			if err != nil {
				log.Debug("[CACHE] Failed to get date range for %s: %v", instrumentKey, err)
				cacheFailed++
				continue
			}

			// Create cache entry
			entry := &CacheEntry{
				InstrumentKey: instrumentKey,
				Interval:      interval,
				LastChecked:   time.Now(),
			}

			if exists && !earliest.IsZero() && !latest.IsZero() {
				entry.EarliestDate = earliest
				entry.LatestDate = latest
				entry.HasRecentData = time.Since(latest) < 3*24*time.Hour // Recent if within 3 days
				entry.IsComplete = true

				// Calculate next processing date (day after latest)
				entry.NextProcessDate = latest.AddDate(0, 0, 1)

				// Estimate record count (rough calculation)
				daysDiff := int(latest.Sub(earliest).Hours() / 24)
				if interval == "day" {
					entry.RecordCount = daysDiff
				} else if interval == "1minute" {
					entry.RecordCount = daysDiff * 375 // Approx 375 minutes per trading day
				}
			} else {
				// No data exists
				entry.HasRecentData = false
				entry.IsComplete = false
				entry.NextProcessDate = time.Now().AddDate(0, 0, -30) // Start from 30 days ago
				entry.RecordCount = 0
			}

			// Store in cache
			s.setCachedDataInfo(instrumentKey, interval, entry)
			cacheBuilt++
		}

		// Small delay between batches
		time.Sleep(10 * time.Millisecond)
	}

	log.Info("[CACHE] Cache building completed in %v. Built: %d, Failed: %d",
		time.Since(start), cacheBuilt, cacheFailed)
	return nil
}

// getOptimizedDateRange returns the optimal date range for processing based on cache
func (s *BatchFetchService) getOptimizedDateRange(instrumentKey, interval string, requestedDays int) DateRange {
	// Check cache first
	if entry, exists := s.getCachedDataInfo(instrumentKey, interval); exists {
		if entry.HasRecentData && entry.IsComplete {
			// We have recent complete data, only fetch from next processing date
			return DateRange{
				FromDate: entry.NextProcessDate,
				ToDate:   time.Now(),
			}
		} else if entry.IsComplete {
			// We have data but it's not recent, fetch from latest date + 1 day
			return DateRange{
				FromDate: entry.LatestDate.AddDate(0, 0, 1),
				ToDate:   time.Now(),
			}
		}
	}

	// No cache or incomplete data, use requested range
	return DateRange{
		FromDate: time.Now().AddDate(0, 0, -requestedDays),
		ToDate:   time.Now(),
	}
}

// invalidateCache removes expired or invalid cache entries
func (s *BatchFetchService) invalidateCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	removed := 0
	for key, entry := range s.dataCache {
		if time.Since(entry.LastChecked) > s.cacheExpiry {
			delete(s.dataCache, key)
			removed++
		}
	}

	if removed > 0 {
		log.Info("[CACHE] Invalidated %d expired cache entries", removed)
	}
}

// GetCacheStats returns cache statistics for monitoring
func (s *BatchFetchService) GetCacheStats() map[string]interface{} {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	recentCount := 0
	completeCount := 0
	totalRecords := 0

	for _, entry := range s.dataCache {
		if entry.HasRecentData {
			recentCount++
		}
		if entry.IsComplete {
			completeCount++
		}
		totalRecords += entry.RecordCount
	}

	return map[string]interface{}{
		"cache_enabled":     s.cacheEnabled,
		"total_entries":     len(s.dataCache),
		"recent_data_count": recentCount,
		"complete_count":    completeCount,
		"total_records":     totalRecords,
		"cache_expiry":      s.cacheExpiry.String(),
		"last_flush":        s.lastCacheFlush.Format("2006-01-02 15:04:05"),
	}
}

// SetV2Services injects V2 services and config for enhanced processing capabilities

// ProcessBatchRequest processes a batch request to fetch and store historical data
func (s *BatchFetchService) ProcessBatchRequest(
	ctx context.Context,
	request *domain.BatchStoreHistoricalDataRequest,
) (*domain.BatchProcessResultData, error) {
	startTime := time.Now()

	// Validate and set defaults for request parameters
	if request.Interval == "" {
		request.Interval = s.defaultInterval
	}

	if request.ToDate == "" {
		request.ToDate = s.defaultToDate
	}

	if request.FromDate == "" {
		request.FromDate = s.defaultFromDate
	}

	// Parse dates to calculate intervals
	fromDate, err := time.Parse("2006-01-02", request.FromDate)
	if err != nil {
		return nil, fmt.Errorf("invalid fromDate format: %w", err)
	}

	toDate, err := time.Parse("2006-01-02", request.ToDate)
	if err != nil {
		return nil, fmt.Errorf("invalid toDate format: %w", err)
	}

	// Determine which instrument keys to process
	var instrumentKeys []string

	if len(request.InstrumentKeys) > 0 {
		// Use provided instrument keys
		instrumentKeys = request.InstrumentKeys
		log.Info("Starting batch processing for %d specified instruments", len(instrumentKeys))
	} else {
		// Fetch all stocks from universe
		log.Info("No instrument keys provided, fetching all stocks from universe")
		stocks, _, err := s.stockUniverseService.GetAllStocks(ctx, false, 1, 10000)
		if err != nil {
			return nil, fmt.Errorf("failed to get stocks from universe: %w", err)
		}

		// Extract instrument keys from stocks
		instrumentKeys = make([]string, 0, len(stocks))
		for _, stock := range stocks {
			if stock.InstrumentKey != "" {
				instrumentKeys = append(instrumentKeys, stock.InstrumentKey)
			}
		}

		log.Info("Retrieved %d instrument keys from universe", len(instrumentKeys))
	}

	if len(instrumentKeys) == 0 {
		return &domain.BatchProcessResultData{
			ProcessedItems:  0,
			SuccessfulItems: 0,
			FailedItems:     0,
			Details:         []domain.InstrumentProcessed{},
		}, nil
	}

	// Process instruments concurrently with a semaphore to limit concurrency
	sem := make(chan struct{}, s.maxConcurrent)
	resultsChan := make(chan *domain.ProcessingResult, len(instrumentKeys))
	var wg sync.WaitGroup

	// Create a child context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Process each instrument concurrently
	for _, instrumentKey := range instrumentKeys {
		wg.Add(1)

		go func(key string) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				// Semaphore acquired
				defer func() { <-sem }() // Release semaphore
			case <-ctx.Done():
				// Context was canceled
				resultsChan <- &domain.ProcessingResult{
					InstrumentKey: key,
					Success:       false,
					Error: &domain.ProcessingError{
						InstrumentKey: key,
						ErrorType:     "context_canceled",
						Message:       "Operation was canceled",
						RawError:      ctx.Err(),
					},
				}
				return
			}

			// Process the instrument with 4-day intervals
			log.Info("Processing historical data for instrument: %s from %s to %s", key, request.FromDate, request.ToDate)
			recordCount, err := s.processInstrumentWithIntervals(
				ctx, key, request.Interval, fromDate, toDate,
			)

			if err != nil {
				log.Error("Failed to process instrument %s: %v", key, err)
				resultsChan <- &domain.ProcessingResult{
					InstrumentKey: key,
					Success:       false,
					Error: &domain.ProcessingError{
						InstrumentKey: key,
						ErrorType:     "processing_failed",
						Message:       fmt.Sprintf("Failed to process instrument: %v", err),
						RawError:      err,
					},
				}
				return
			}

			resultsChan <- &domain.ProcessingResult{
				InstrumentKey: key,
				Success:       true,
				RecordsStored: recordCount,
			}
		}(instrumentKey)

		time.Sleep(1 * time.Second)
	}

	// Wait for all goroutines to complete in a separate goroutine
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	results := make([]*domain.ProcessingResult, 0, len(instrumentKeys))
	for result := range resultsChan {
		results = append(results, result)
	}

	// Process results
	responseData := s.processResults(results)

	log.Info("Batch processing completed in %v, processed %d instruments (%d successful, %d failed)",
		time.Since(startTime), responseData.ProcessedItems,
		responseData.SuccessfulItems, responseData.FailedItems)

	return responseData, nil
}

// processInstrumentWithIntervals processes historical data for an instrument by breaking the date range into optimized intervals
func (s *BatchFetchService) processInstrumentWithIntervals(
	ctx context.Context,
	instrumentKey string,
	interval string,
	fromDate time.Time,
	toDate time.Time,
) (int, error) {
	totalRecords := 0
	currentDate := fromDate

	// Log V2 integration status
	v2Status := "V1"
	if s.isV2AggregationEnabled() {
		v2Status = "V2 DataFrame"
	}

	log.Info("[BATCH] Starting optimized batch processing for %s with interval %s from %s to %s (Aggregation: %s, Interval: %d days)",
		instrumentKey, interval, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"), v2Status, s.getOptimalIntervalDays(interval))

	// Smart interval optimization based on time interval and cache
	intervalDays := s.getOptimalIntervalDays(interval)

	// Check cache for further optimization
	if entry, exists := s.getCachedDataInfo(instrumentKey, interval); exists {
		// For recent data, use smaller chunks to be more conservative
		if entry.HasRecentData {
			if interval == "1minute" {
				intervalDays = 3 // Smaller chunks for recent 1-minute data (from 4)
			} else if interval == "day" {
				intervalDays = 75 // Smaller chunks for recent daily data (from 100)
			} else {
				intervalDays = int(float64(intervalDays) * 0.75) // 25% reduction for other intervals
			}
			log.Debug("[CACHE] Using reduced intervals (%d days) for %s with recent %s data", intervalDays, instrumentKey, interval)
		} else {
			// Use full optimal size for older/missing data
			// TODO: Not implemented yet.
			log.Debug("[CACHE] Using full optimal intervals (%d days) for %s with older/missing %s data", intervalDays, instrumentKey, interval)
		}
	}

	// Calculate total requested days and enforce interval limits
	totalRequestedDays := int(toDate.Sub(fromDate).Hours() / 24)
	maxAllowedDays := intervalDays

	log.Info("[BATCH] Total requested days: %d, Max allowed per batch: %d days for %s interval",
		totalRequestedDays, maxAllowedDays, interval)

	// Always enforce the configured interval limit - break down ANY range larger than the limit
	// This ensures consistent behavior regardless of request size
	if totalRequestedDays > maxAllowedDays {
		expectedBatches := (totalRequestedDays + maxAllowedDays - 1) / maxAllowedDays // Ceiling division
		log.Info("[BATCH] Requested range (%d days) exceeds max allowed (%d days). Will process in %d batches with 1s delays",
			totalRequestedDays, maxAllowedDays, expectedBatches)
	} else {
		log.Info("[BATCH] Requested range (%d days) fits within max allowed (%d days). Processing in single batch",
			totalRequestedDays, maxAllowedDays)
	}

	// Always use the configured interval size for consistent API behavior
	intervalDays = maxAllowedDays // Process data in optimized intervals
	for currentDate.Before(toDate) || currentDate.Equal(toDate) {
		// Calculate the end date for this interval
		intervalEndDate := currentDate.AddDate(0, 0, intervalDays)

		// If the calculated end date exceeds the requested toDate, use toDate instead
		if intervalEndDate.After(toDate) {
			intervalEndDate = toDate
		}

		// Format dates for API call
		fromDateStr := currentDate.Format("2006-01-02")
		toDateStr := intervalEndDate.Format("2006-01-02")

		log.Info("[BATCH] Processing optimized interval for %s: %s to %s (%d days)",
			instrumentKey, fromDateStr, toDateStr, intervalDays)

		// Process this interval with timeout handling
		recordCount, err := s.processIntervalWithRetry(ctx, instrumentKey, interval, fromDateStr, toDateStr)

		if err != nil {
			log.Error("[BATCH] Failed to process interval %s to %s for %s: %v", fromDateStr, toDateStr, instrumentKey, err)
			// Don't fail completely, continue with next interval
			log.Warn("[BATCH] Continuing with next interval after error for %s", instrumentKey)
		} else {
			totalRecords += recordCount
			log.Info("[BATCH] Successfully processed %d records for %s in interval %s to %s",
				recordCount, instrumentKey, fromDateStr, toDateStr)
		}

		// Trigger aggregation for 1-minute data with enhanced V2 processing
		if interval == "1minute" && recordCount > 0 {
			if err := s.processAggregationOptimized(ctx, instrumentKey, currentDate, intervalEndDate); err != nil {
				log.Error("[BATCH] Failed to aggregate 5-minute candles for %s: %v", instrumentKey, err)
				// Continue processing, don't fail the entire batch
			}
		}

		// Move to the next interval (start from the day after the current interval end)
		currentDate = intervalEndDate.AddDate(0, 0, 1)

		// Controlled delay: Always wait 1 second between API calls to ensure rate limit compliance
		// This gives our software full control over API call frequency regardless of request size
		if !currentDate.After(toDate) { // Only sleep if we have more batches to process
			log.Debug("[BATCH] Applying controlled 1-second delay between API batches for %s", instrumentKey)
			time.Sleep(1 * time.Second) // Fixed 1-second delay between all API calls
		}

		// Additional delay after errors to allow recovery
		if err != nil {
			log.Debug("[BATCH] Applying additional error recovery delay for %s", instrumentKey)
			time.Sleep(1 * time.Second) // Extra 1 second after errors
		}
	}

	log.Info("[BATCH] Completed optimized batch processing for %s: %d total records processed", instrumentKey, totalRecords)
	return totalRecords, nil
}

// hasRecentData checks if we have recent data to optimize processing intervals
func (s *BatchFetchService) hasRecentData(ctx context.Context, instrumentKey, interval string, fromDate, toDate time.Time) bool {
	// Quick check for recent data to optimize interval sizing
	candles, err := s.candleService.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, interval,
		toDate.AddDate(0, 0, -7), toDate) // Check last 7 days
	if err != nil {
		log.Debug("[BATCH] Failed to check recent data for %s: %v", instrumentKey, err)
		return false
	}
	return len(candles) > 0
}

// processIntervalWithRetry processes an interval with retry logic and timeout handling
func (s *BatchFetchService) processIntervalWithRetry(ctx context.Context, instrumentKey, interval, fromDateStr, toDateStr string) (int, error) {
	const maxRetries = 2
	const retryDelay = time.Second * 2

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Create a context with timeout for this specific interval
		intervalCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

		recordCount, err := s.candleService.ProcessHistoricalCandles(
			intervalCtx, instrumentKey, interval, fromDateStr, toDateStr,
		)
		cancel()

		if err == nil {
			return recordCount, nil
		}

		lastErr = err
		if attempt < maxRetries {
			log.Warn("[BATCH] Attempt %d failed for %s interval %s-%s: %v. Retrying in %v...",
				attempt, instrumentKey, fromDateStr, toDateStr, err, retryDelay)
			time.Sleep(retryDelay)
		}
	}

	return 0, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// processAggregationOptimized processes aggregation with V2 DataFrame optimization
func (s *BatchFetchService) processAggregationOptimized(ctx context.Context, instrumentKey string, startTime, endTime time.Time) error {
	log.Info("[BATCH] Starting optimized aggregation for %s from %s to %s",
		instrumentKey, startTime.Format("2006-01-02 15:04"), endTime.Format("2006-01-02 15:04"))

	// Get processed 1-minute candles
	processedCandles, err := s.candleService.candleRepo.FindByInstrumentAndTimeRange(
		ctx, instrumentKey, "1minute", startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to get processed candles for aggregation: %w", err)
	}

	if len(processedCandles) == 0 {
		log.Warn("[BATCH] No processed candles found for aggregation for %s", instrumentKey)
		return nil
	}

	log.Info("[BATCH] Processing 5-minute aggregation for %s with %d candles", instrumentKey, len(processedCandles))

	// Use V2 DataFrame-based aggregation if available, otherwise fallback to V1
	if err := s.processAggregation(ctx, instrumentKey, startTime, endTime); err != nil {
		return fmt.Errorf("failed to aggregate 5-minute candles: %w", err)
	}

	log.Info("[BATCH] Successfully completed optimized aggregation for %s", instrumentKey)
	return nil
}

// processAggregation handles 5-minute aggregation using V2 services when available, with V1 fallback
func (s *BatchFetchService) processAggregation(ctx context.Context, instrumentKey string, startTime, endTime time.Time) error {
	// Check if V2 candle aggregation service is available and enabled
	if s.isV2AggregationEnabled() {
		log.Info("[BATCH] Using V2 DataFrame-based aggregation for %s", instrumentKey)

		// Use V2 service with DataFrame processing and automatic indicator calculation
		if candleAggV2, ok := s.candleAggregationV2.(*CandleAggregationServiceV2); ok {
			err := candleAggV2.Aggregate5MinCandlesWithIndicators(ctx, instrumentKey, startTime, endTime, nil)
			if err != nil {
				log.Error("[BATCH] V2 aggregation failed for %s, falling back to V1: %v", instrumentKey, err)
				// Fallback to V1 aggregation
				return s.candleService.AggregateAndStore5MinCandlesForRange(ctx, instrumentKey, startTime, endTime)
			}
			log.Info("[BATCH] V2 DataFrame aggregation with indicators completed successfully for %s", instrumentKey)
			return nil
		} else {
			// Interface doesn't support V2 specific methods, use interface method
			log.Info("[BATCH] Using V2 interface aggregation for %s", instrumentKey)
			err := s.candleAggregationV2.Aggregate5MinCandlesWithIndicators(ctx, instrumentKey, startTime, endTime, nil)
			if err != nil {
				log.Error("[BATCH] V2 interface aggregation failed for %s, falling back to V1: %v", instrumentKey, err)
				return s.candleService.AggregateAndStore5MinCandlesForRange(ctx, instrumentKey, startTime, endTime)
			}
			return nil
		}
	}

	// Fallback to V1 aggregation (original implementation)
	log.Info("[BATCH] Using V1 aggregation for %s (V2 not available)", instrumentKey)
	return s.candleService.AggregateAndStore5MinCandlesForRange(ctx, instrumentKey, startTime, endTime)
}

// isV2AggregationEnabled checks if V2 candle aggregation should be used
func (s *BatchFetchService) isV2AggregationEnabled() bool {
	return s.candleAggregationV2 != nil &&
		s.config != nil &&
		s.config.Features.CandleAggregationV2
}

// GetV2Status returns the current V2 integration status for monitoring
func (s *BatchFetchService) GetV2Status() map[string]interface{} {
	status := map[string]interface{}{
		"v2_services_available":  s.candleAggregationV2 != nil,
		"config_available":       s.config != nil,
		"v2_aggregation_enabled": s.isV2AggregationEnabled(),
		"enhanced_processing":    s.isV2AggregationEnabled(),
	}

	if s.config != nil {
		status["feature_flags"] = map[string]bool{
			"candle_aggregation_v2":     s.config.Features.CandleAggregationV2,
			"technical_indicators_v2":   s.config.Features.TechnicalIndicatorsV2,
			"use_dataframe_aggregation": s.config.Features.UseDataFrameAggregation,
		}
	}

	return status
}

// processResults converts processing results to a response structure
func (s *BatchFetchService) processResults(results []*domain.ProcessingResult) *domain.BatchProcessResultData {
	data := &domain.BatchProcessResultData{
		ProcessedItems: len(results),
		Details:        make([]domain.InstrumentProcessed, 0, len(results)),
	}

	for _, result := range results {
		detail := domain.InstrumentProcessed{
			InstrumentKey: result.InstrumentKey,
			RecordsStored: result.RecordsStored,
		}

		if result.Success {
			detail.Status = "success"
			detail.Message = fmt.Sprintf("Successfully processed %d records", result.RecordsStored)
			data.SuccessfulItems++
		} else {
			detail.Status = "failed"
			if result.Error != nil {
				detail.Message = result.Error.Message
			} else {
				detail.Message = "Processing failed with no specific error"
			}
			data.FailedItems++
		}

		data.Details = append(data.Details, detail)
	}

	return data
}

// ProcessDailyCandlesOptimized processes daily candles for multiple stocks with smart caching for super fast performance
func (s *BatchFetchService) ProcessDailyCandlesOptimized(ctx context.Context, stocks []string) error {
	log.Info("[BATCH] Starting SUPER FAST optimized daily candles processing for %d stocks with smart caching", len(stocks))

	startTime := time.Now()

	// Step 1: Build cache from database (super fast database scan)
	log.Info("[CACHE] Building smart cache for existing data...")
	if err := s.buildCacheFromDatabase(ctx, stocks, "day"); err != nil {
		log.Warn("[CACHE] Failed to build cache, proceeding without cache: %v", err)
	}

	// Step 2: Analyze cache to determine what needs processing
	needsProcessing := make([]string, 0)
	alreadyComplete := 0
	needsUpdate := 0

	for _, instrumentKey := range stocks {
		if entry, exists := s.getCachedDataInfo(instrumentKey, "day"); exists {
			if entry.HasRecentData && entry.IsComplete {
				alreadyComplete++
				log.Debug("[CACHE] Skipping %s - already has recent complete data (latest: %s)",
					instrumentKey, entry.LatestDate.Format("2006-01-02"))
				continue
			} else if entry.IsComplete {
				needsUpdate++
				log.Debug("[CACHE] %s needs update from %s to today",
					instrumentKey, entry.NextProcessDate.Format("2006-01-02"))
			}
		}
		needsProcessing = append(needsProcessing, instrumentKey)
	}

	log.Info("[CACHE] Analysis complete: %d already complete, %d need updates, %d need full processing",
		alreadyComplete, needsUpdate, len(needsProcessing)-needsUpdate)

	// Step 3: Process only what's needed (super fast targeted processing)
	if len(needsProcessing) == 0 {
		log.Info("[BATCH] 🚀 SUPER FAST: All stocks already have recent data! Completed in %v",
			time.Since(startTime))
		return nil
	}

	log.Info("[BATCH] Processing %d stocks that need updates (skipped %d with recent data)",
		len(needsProcessing), alreadyComplete)

	var wg sync.WaitGroup
	const maxConcurrency = 4 // Slightly higher since we're doing less work
	semaphore := make(chan struct{}, maxConcurrency)

	successCount := 0
	errorCount := 0
	var mu sync.Mutex

	for _, stock := range needsProcessing {
		wg.Add(1)
		go func(instrumentKey string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Get optimized date range using cache
			dateRange := s.getOptimizedDateRange(instrumentKey, "day", 30)

			// Skip if no processing needed (edge case)
			if dateRange.FromDate.After(dateRange.ToDate) || dateRange.FromDate.Equal(dateRange.ToDate) {
				log.Debug("[CACHE] Skipping %s - no new data needed", instrumentKey)
				mu.Lock()
				successCount++
				mu.Unlock()
				return
			}

			log.Info("[CACHE] 🎯 Targeted processing for %s: %s to %s (%d days)",
				instrumentKey,
				dateRange.FromDate.Format("2006-01-02"),
				dateRange.ToDate.Format("2006-01-02"),
				int(dateRange.ToDate.Sub(dateRange.FromDate).Hours()/24))

			totalRecords, err := s.processInstrumentWithIntervals(
				ctx, instrumentKey, "day",
				dateRange.FromDate, dateRange.ToDate,
			)

			mu.Lock()
			if err != nil {
				errorCount++
				log.Error("[BATCH] Failed to process daily candles for %s: %v", instrumentKey, err)
			} else {
				successCount++
				log.Info("[BATCH] ✅ Successfully processed %d daily candle records for %s", totalRecords, instrumentKey)

				// Update cache with new data
				if entry, exists := s.getCachedDataInfo(instrumentKey, "day"); exists {
					entry.LatestDate = dateRange.ToDate
					entry.HasRecentData = true
					entry.IsComplete = true
					entry.NextProcessDate = dateRange.ToDate.AddDate(0, 0, 1)
					entry.RecordCount += totalRecords
					s.setCachedDataInfo(instrumentKey, "day", entry)
				}
			}
			mu.Unlock()
		}(stock)
	}

	wg.Wait()

	// Step 4: Performance summary
	totalDuration := time.Since(startTime)
	log.Info("[BATCH] 🚀 SUPER FAST processing completed in %v", totalDuration)
	log.Info("[BATCH] Results: %d skipped (cached), %d processed (%d success, %d errors)",
		alreadyComplete, len(needsProcessing), successCount, errorCount)

	// Print cache stats
	cacheStats := s.GetCacheStats()
	log.Info("[CACHE] Cache stats: %d total entries, %d with recent data, efficiency: %.1f%%",
		cacheStats["total_entries"], cacheStats["recent_data_count"],
		float64(alreadyComplete)/float64(len(stocks))*100)

	if errorCount > 0 {
		return fmt.Errorf("encountered %d errors during batch processing (success: %d, skipped: %d)",
			errorCount, successCount, alreadyComplete)
	}

	return nil
}

// DateRange represents a date range for processing
type DateRange struct {
	FromDate time.Time
	ToDate   time.Time
}

// getBulkDateRanges gets optimal date ranges for all stocks in a single operation
func (s *BatchFetchService) getBulkDateRanges(ctx context.Context, stocks []string, interval string) (map[string]DateRange, error) {
	log.Info("[BATCH] Fetching bulk date ranges for %d stocks", len(stocks))

	dateRanges := make(map[string]DateRange)
	defaultFromDate := time.Now().AddDate(0, 0, -30) // Default 30 days
	defaultToDate := time.Now()

	// Create a context with timeout for bulk operations
	bulkCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Process in smaller batches to avoid overwhelming the database
	const batchSize = 20 // Smaller batch size for better performance
	for i := 0; i < len(stocks); i += batchSize {
		end := i + batchSize
		if end > len(stocks) {
			end = len(stocks)
		}

		batch := stocks[i:end]
		log.Info("[BATCH] Processing date range batch %d-%d of %d stocks", i+1, end, len(stocks))

		for _, stock := range batch {
			// Try to get date range with timeout, fallback to default if it fails
			dateRange, err := s.getDateRangeWithTimeout(bulkCtx, stock, interval)
			if err != nil {
				log.Warn("[BATCH] Failed to get date range for %s: %v, using default", stock, err)
				dateRange = DateRange{
					FromDate: defaultFromDate,
					ToDate:   defaultToDate,
				}
			}
			dateRanges[stock] = dateRange
		}

		// Small delay between batches
		time.Sleep(200 * time.Millisecond)
	}

	log.Info("[BATCH] Successfully fetched %d date ranges", len(dateRanges))
	return dateRanges, nil
}

// getDateRangeWithTimeout gets date range for a single stock with timeout
func (s *BatchFetchService) getDateRangeWithTimeout(ctx context.Context, instrumentKey, interval string) (DateRange, error) {
	// Create a short timeout context for this specific query
	queryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	earliest, latest, exists, err := s.candleService.candleRepo.GetCandleDateRange(queryCtx, instrumentKey, interval)
	if err != nil {
		return DateRange{}, fmt.Errorf("failed to get date range: %w", err)
	}

	if !exists {
		// No data found, use default range
		return DateRange{
			FromDate: time.Now().AddDate(0, 0, -30),
			ToDate:   time.Now(),
		}, nil
	}

	// Calculate optimal date range
	fromDate := latest.AddDate(0, 0, -30) // Last 30 days from max date
	if fromDate.Before(earliest) {
		fromDate = earliest
	}

	return DateRange{
		FromDate: fromDate,
		ToDate:   time.Now(),
	}, nil
}

// processWithDefaults processes stocks with default date ranges when bulk optimization fails
func (s *BatchFetchService) processWithDefaults(ctx context.Context, stocks []string) error {
	log.Warn("[BATCH] Using default date ranges for all stocks due to bulk optimization failure")

	defaultFromDate := time.Now().AddDate(0, 0, -30)
	defaultToDate := time.Now()

	var wg sync.WaitGroup
	const maxConcurrency = 2 // Very conservative limit
	semaphore := make(chan struct{}, maxConcurrency)

	successCount := 0
	errorCount := 0
	var mu sync.Mutex

	for _, stock := range stocks {
		wg.Add(1)
		go func(instrumentKey string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			log.Info("[BATCH] Processing daily candles for %s with default range %s to %s",
				instrumentKey,
				defaultFromDate.Format("2006-01-02"),
				defaultToDate.Format("2006-01-02"))

			totalRecords, err := s.processInstrumentWithIntervals(
				ctx, instrumentKey, "day",
				defaultFromDate, defaultToDate,
			)

			mu.Lock()
			if err != nil {
				errorCount++
				log.Error("[BATCH] Failed to process daily candles for %s: %v", instrumentKey, err)
			} else {
				successCount++
				log.Info("[BATCH] Successfully processed %d daily candle records for %s", totalRecords, instrumentKey)
			}
			mu.Unlock()
		}(stock)
	}

	wg.Wait()

	log.Info("[BATCH] Completed default daily candles processing. Success: %d, Errors: %d", successCount, errorCount)

	if errorCount > 0 {
		return fmt.Errorf("encountered %d errors during default processing (success: %d)", errorCount, successCount)
	}

	return nil
}
