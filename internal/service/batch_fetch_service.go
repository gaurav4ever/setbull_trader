package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"setbull_trader/internal/domain"
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
}

// NewBatchFetchService creates a new batch fetch service
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
	}
}

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

			// Process the instrument
			log.Info("Processing historical data for instrument: %s", key)
			recordCount, err := s.candleService.ProcessHistoricalCandles(
				ctx, key, request.Interval, request.FromDate, request.ToDate,
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
