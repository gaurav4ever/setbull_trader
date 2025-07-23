package service

import (
	"context"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/pkg/log"
	"time"
)

// DailyDataServiceAdapter adapts the existing candle aggregation service for daily data operations
type DailyDataServiceAdapter struct {
	candleAggService     *CandleAggregationService
	stockUniverseService *StockUniverseService
}

// NewDailyDataServiceAdapter creates a new adapter for daily data operations
func NewDailyDataServiceAdapter(
	candleAggService *CandleAggregationService,
	stockUniverseService *StockUniverseService,
) DailyDataService {
	return &DailyDataServiceAdapter{
		candleAggService:     candleAggService,
		stockUniverseService: stockUniverseService,
	}
}

// InsertDailyCandles implements DailyDataService interface
func (a *DailyDataServiceAdapter) InsertDailyCandles(ctx context.Context, days int) error {
	log.Info("Starting daily candles insertion for %d days", days)

	// Get all stocks from universe
	stocks, _, err := a.stockUniverseService.GetAllStocks(ctx, false, 1, 10000)
	if err != nil {
		return fmt.Errorf("failed to get stocks from universe: %w", err)
	}

	log.Info("Processing daily candles for %d stocks", len(stocks))

	// Calculate end date (most recent trading day)
	endDate := time.Now()

	// Process each stock
	successCount := 0
	failedCount := 0

	for _, stock := range stocks {
		// Process daily candles for this stock
		result, err := a.candleAggService.ProcessStockDailyCandles(ctx, stock, endDate, days)
		if err != nil {
			log.Error("Failed to process daily candles for stock %s: %v", stock.Symbol, err)
			failedCount++
			continue
		}

		if result.Status == "success" {
			successCount++
		} else {
			failedCount++
		}
	}

	log.Info("Daily candles insertion completed. Success: %d, Failed: %d", successCount, failedCount)

	if failedCount > 0 {
		return fmt.Errorf("daily candles insertion completed with %d failures out of %d stocks", failedCount, len(stocks))
	}

	return nil
}

// FilterPipelineServiceAdapter adapts the existing stock filter pipeline for filter operations
type FilterPipelineServiceAdapter struct {
	stockFilterPipeline *StockFilterPipeline
}

// NewFilterPipelineServiceAdapter creates a new adapter for filter pipeline operations
func NewFilterPipelineServiceAdapter(stockFilterPipeline *StockFilterPipeline) FilterPipelineService {
	return &FilterPipelineServiceAdapter{
		stockFilterPipeline: stockFilterPipeline,
	}
}

// RunFilterPipeline implements FilterPipelineService interface
func (a *FilterPipelineServiceAdapter) RunFilterPipeline(ctx context.Context) error {
	log.Info("Starting filter pipeline execution")

	// Run the filter pipeline with empty instrument keys to process all stocks
	bullish, bearish, metrics, err := a.stockFilterPipeline.RunPipeline(ctx, []string{})
	if err != nil {
		return fmt.Errorf("filter pipeline execution failed: %w", err)
	}

	log.Info("Filter pipeline completed successfully. Bullish: %d, Bearish: %d, Total processed: %d",
		len(bullish), len(bearish), metrics.TotalStocks)

	return nil
}

// MinuteDataServiceAdapter adapts the existing batch fetch service for minute data operations
type MinuteDataServiceAdapter struct {
	batchFetchService *BatchFetchService
}

// NewMinuteDataServiceAdapter creates a new adapter for minute data operations
func NewMinuteDataServiceAdapter(batchFetchService *BatchFetchService) MinuteDataService {
	return &MinuteDataServiceAdapter{
		batchFetchService: batchFetchService,
	}
}

// BatchStore implements MinuteDataService interface
func (a *MinuteDataServiceAdapter) BatchStore(ctx context.Context, instrumentKeys []string, fromDate, toDate time.Time, interval string) error {
	log.Info("Starting minute data batch store for %d instruments from %s to %s",
		len(instrumentKeys), fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))

	// Create batch request
	request := &domain.BatchStoreHistoricalDataRequest{
		InstrumentKeys: instrumentKeys,
		FromDate:       fromDate.Format("2006-01-02"),
		ToDate:         toDate.Format("2006-01-02"),
		Interval:       interval,
	}

	// Process the batch request
	result, err := a.batchFetchService.ProcessBatchRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("batch store failed: %w", err)
	}

	log.Info("Minute data batch store completed. Processed: %d, Success: %d, Failed: %d",
		result.ProcessedItems, result.SuccessfulItems, result.FailedItems)

	if result.FailedItems > 0 {
		return fmt.Errorf("batch store completed with %d failures out of %d items", result.FailedItems, result.ProcessedItems)
	}

	return nil
}
