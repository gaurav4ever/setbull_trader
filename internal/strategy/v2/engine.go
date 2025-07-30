package v2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository/postgres"
	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/log"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

// StrategyEngineV2 is the core engine for V2 strategy processing
type StrategyEngineV2 struct {
	config           *config.StrategyEngineV2Config
	registry         *StrategyRegistryV2
	candleRepository *postgres.CandleRepository
	metrics          *EngineMetrics
	mu               sync.RWMutex
}

// EngineMetrics contains metrics for the strategy engine
type EngineMetrics struct {
	TotalProcessingRuns     int64         `json:"total_processing_runs"`
	TotalStocksProcessed    int64         `json:"total_stocks_processed"`
	TotalStrategiesExecuted int64         `json:"total_strategies_executed"`
	AverageProcessingTime   time.Duration `json:"average_processing_time"`
	TotalErrors             int64         `json:"total_errors"`
	LastProcessingTime      time.Time     `json:"last_processing_time"`
	MemoryUsageMB           int64         `json:"memory_usage_mb"`
}

// NewStrategyEngineV2 creates a new V2 strategy engine
func NewStrategyEngineV2(
	config *config.StrategyEngineV2Config,
	candleRepository *postgres.CandleRepository,
) *StrategyEngineV2 {
	// Create registry configuration
	registryConfig := &RegistryConfig{
		AutoDiscovery:   config.Strategies.Registry.AutoDiscovery,
		DiscoveryPath:   config.Strategies.Registry.DiscoveryPath,
		ParallelExec:    config.Strategies.Execution.ParallelStrategies,
		StrategyTimeout: config.Strategies.Execution.StrategyTimeout,
		ErrorHandling:   config.Strategies.Execution.ErrorHandling,
	}

	registry := NewStrategyRegistryV2(registryConfig)

	return &StrategyEngineV2{
		config:           config,
		registry:         registry,
		candleRepository: candleRepository,
		metrics:          &EngineMetrics{},
	}
}

// ProcessStockGroups processes all stock groups with registered strategies
func (e *StrategyEngineV2) ProcessStockGroups(
	ctx context.Context,
	stockGroups []domain.StockGroup,
	currentTime time.Time,
) (map[string]map[string]*StrategyResult, error) {
	startTime := time.Now()
	e.metrics.LastProcessingTime = startTime
	e.metrics.TotalProcessingRuns++

	log.Info("Starting V2 strategy processing for %d stock groups", len(stockGroups))

	// Fetch historical data for all stock groups
	stockDataFrames, err := e.fetchHistoricalData(ctx, stockGroups, currentTime)
	if err != nil {
		e.metrics.TotalErrors++
		return nil, fmt.Errorf("failed to fetch historical data: %w", err)
	}

	// Process each stock group
	results := make(map[string]map[string]*StrategyResult)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var processingErrors []error

	// Use worker pool for parallel processing
	semaphore := make(chan struct{}, e.config.Processing.ConcurrentWorkers)

	for _, group := range stockGroups {
		wg.Add(1)
		go func(group domain.StockGroup) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			groupResults, err := e.processStockGroup(ctx, group, stockDataFrames, currentTime)
			if err != nil {
				mu.Lock()
				processingErrors = append(processingErrors, fmt.Errorf("group %s: %w", group.ID, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			results[group.ID] = groupResults
			mu.Unlock()
		}(group)
	}

	wg.Wait()

	// Handle processing errors
	if len(processingErrors) > 0 {
		log.Error("Encountered %d processing errors", len(processingErrors))
		for _, err := range processingErrors {
			log.Error("Processing error: %v", err)
		}
		// Continue processing even with errors (graceful degradation)
	}

	// Update metrics
	processingTime := time.Since(startTime)
	e.metrics.AverageProcessingTime = (e.metrics.AverageProcessingTime + processingTime) / 2
	e.metrics.TotalStocksProcessed += int64(len(stockGroups))

	log.Info("Completed V2 strategy processing in %v", processingTime)
	return results, nil
}

// fetchHistoricalData fetches historical candle data for all stock groups
func (e *StrategyEngineV2) fetchHistoricalData(
	ctx context.Context,
	stockGroups []domain.StockGroup,
	currentTime time.Time,
) (map[string]*dataframe.DataFrame, error) {
	log.Info("Fetching historical data for %d stock groups", len(stockGroups))

	// Get maximum required history from all strategies
	maxHistory := e.getMaxRequiredHistory()
	if maxHistory == 0 {
		maxHistory = e.config.Processing.MaxHistoricalCandles
	}

	// Calculate time range
	endTime := currentTime
	startTime := endTime.Add(-time.Duration(maxHistory) * 5 * time.Minute) // 5-minute candles

	stockDataFrames := make(map[string]*dataframe.DataFrame)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var fetchErrors []error

	// Use worker pool for parallel fetching
	semaphore := make(chan struct{}, e.config.Processing.ConcurrentWorkers)

	for _, group := range stockGroups {
		wg.Add(1)
		go func(group domain.StockGroup) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			// Fetch candles for each stock in the group
			for _, stock := range group.Stocks {
				candles, err := e.candleRepository.FindByInstrumentAndTimeRange(
					ctx,
					stock.StockID,
					"5minute",
					startTime,
					endTime,
				)
				if err != nil {
					mu.Lock()
					fetchErrors = append(fetchErrors, fmt.Errorf("stock %s: %w", stock.StockID, err))
					mu.Unlock()
					continue
				}

				// Convert to DataFrame
				df, err := e.convertCandlesToDataFrame(candles)
				if err != nil {
					mu.Lock()
					fetchErrors = append(fetchErrors, fmt.Errorf("convert %s: %w", stock.StockID, err))
					mu.Unlock()
					continue
				}

				mu.Lock()
				stockDataFrames[stock.StockID] = df
				mu.Unlock()
			}
		}(group)
	}

	wg.Wait()

	// Handle fetch errors
	if len(fetchErrors) > 0 {
		log.Warn("Encountered %d fetch errors", len(fetchErrors))
		for _, err := range fetchErrors {
			log.Warn("Fetch error: %v", err)
		}
	}

	log.Info("Fetched historical data for %d stocks", len(stockDataFrames))
	return stockDataFrames, nil
}

// processStockGroup processes a single stock group with all registered strategies
func (e *StrategyEngineV2) processStockGroup(
	ctx context.Context,
	group domain.StockGroup,
	stockDataFrames map[string]*dataframe.DataFrame,
	currentTime time.Time,
) (map[string]*StrategyResult, error) {
	log.Info("Processing stock group %s with %d stocks", group.ID, len(group.Stocks))

	results := make(map[string]*StrategyResult)

	for _, stock := range group.Stocks {
		df, exists := stockDataFrames[stock.StockID]
		if !exists {
			log.Warn("No data found for stock %s", stock.StockID)
			continue
		}

		// Process with all registered strategies
		strategyResults, err := e.registry.ProcessAll(ctx, df)
		if err != nil {
			log.Error("Strategy processing failed for stock %s: %v", stock.StockID, err)
			continue
		}

		// Aggregate results for this stock
		for strategyName, result := range strategyResults {
			results[fmt.Sprintf("%s_%s", stock.StockID, strategyName)] = result
		}

		e.metrics.TotalStrategiesExecuted += int64(len(strategyResults))
	}

	return results, nil
}

// convertCandlesToDataFrame converts domain.Candle slice to Gota DataFrame
func (e *StrategyEngineV2) convertCandlesToDataFrame(candles []domain.Candle) (*dataframe.DataFrame, error) {
	if len(candles) == 0 {
		return nil, fmt.Errorf("no candles provided")
	}

	// Prepare data arrays
	timestamps := make([]string, len(candles))
	opens := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	volumes := make([]int64, len(candles))

	// Convert candles to arrays
	for i, candle := range candles {
		timestamps[i] = candle.Timestamp.Format("2006-01-02 15:04:05")
		opens[i] = candle.Open
		highs[i] = candle.High
		lows[i] = candle.Low
		closes[i] = candle.Close
		volumes[i] = candle.Volume
	}

	// Create DataFrame
	df := dataframe.New(
		series.New(timestamps, series.String, "timestamp"),
		series.New(opens, series.Float, "open"),
		series.New(highs, series.Float, "high"),
		series.New(lows, series.Float, "low"),
		series.New(closes, series.Float, "close"),
		series.New(volumes, series.Int, "volume"),
	)

	// Add technical indicators if available
	if len(candles) > 0 && candles[0].BBWidth != 0 {
		bbWidths := make([]float64, len(candles))
		bbUppers := make([]float64, len(candles))
		bbMiddles := make([]float64, len(candles))
		bbLowers := make([]float64, len(candles))

		for i, candle := range candles {
			bbWidths[i] = candle.BBWidth
			bbUppers[i] = candle.BBUpper
			bbMiddles[i] = candle.BBMiddle
			bbLowers[i] = candle.BBLower
		}

		df = df.Mutate(series.New(bbWidths, series.Float, "bb_width"))
		df = df.Mutate(series.New(bbUppers, series.Float, "bb_upper"))
		df = df.Mutate(series.New(bbMiddles, series.Float, "bb_middle"))
		df = df.Mutate(series.New(bbLowers, series.Float, "bb_lower"))
	}

	return &df, nil
}

// getMaxRequiredHistory returns the maximum historical candles required by any strategy
func (e *StrategyEngineV2) getMaxRequiredHistory() int {
	strategies := e.registry.ListActiveStrategies()
	maxHistory := 0

	for _, strategy := range strategies {
		required := strategy.GetRequiredHistory()
		if required > maxHistory {
			maxHistory = required
		}
	}

	return maxHistory
}

// RegisterStrategy registers a strategy with the engine
func (e *StrategyEngineV2) RegisterStrategy(strategy StrategyV2) error {
	return e.registry.Register(strategy)
}

// GetMetrics returns the engine metrics
func (e *StrategyEngineV2) GetMetrics() *EngineMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *e.metrics
	return &metrics
}

// ResetMetrics resets the engine metrics
func (e *StrategyEngineV2) ResetMetrics() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.metrics = &EngineMetrics{}
}

// GetRegistry returns the strategy registry
func (e *StrategyEngineV2) GetRegistry() *StrategyRegistryV2 {
	return e.registry
}
