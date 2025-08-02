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

// ProductionReadyEngineV2 is a production-ready strategy engine with enhanced scalability
type ProductionReadyEngineV2 struct {
	config            *config.StrategyEngineV2Config
	registry          *StrategyRegistryV2
	candleRepository  *postgres.CandleRepository
	parallelProcessor *ParallelProcessorV2
	stateManager      *AdvancedStateManagerV2
	monitoring        *ProductionMonitoringV2
	healthChecker     *HealthCheckerV2
	metrics           *ProductionEngineMetrics
	paramsManager     *StrategyParametersManager
	progressManager   *ProgressTrackerManager
	debugManager      *DebugManager
	logger            DebugLogger
	mu                sync.RWMutex
	shutdownChan      chan struct{}
}

// ProductionEngineMetrics contains comprehensive metrics for the production engine
type ProductionEngineMetrics struct {
	// Processing metrics
	TotalProcessingRuns     int64         `json:"total_processing_runs"`
	TotalStocksProcessed    int64         `json:"total_stocks_processed"`
	TotalStrategiesExecuted int64         `json:"total_strategies_executed"`
	AverageProcessingTime   time.Duration `json:"average_processing_time"`

	// Performance metrics
	MemoryUsageMB    int64   `json:"memory_usage_mb"`
	CPUUsagePercent  float64 `json:"cpu_usage_percent"`
	ActiveGoroutines int     `json:"active_goroutines"`

	// Error metrics
	TotalErrors   int64     `json:"total_errors"`
	ErrorRate     float64   `json:"error_rate"`
	LastErrorTime time.Time `json:"last_error_time"`

	// State management metrics
	TotalStatesManaged    int64   `json:"total_states_managed"`
	StateCacheHitRate     float64 `json:"state_cache_hit_rate"`
	StateValidationErrors int64   `json:"state_validation_errors"`

	// Parallel processing metrics
	ActiveWorkers     int     `json:"active_workers"`
	QueueLength       int     `json:"queue_length"`
	WorkerUtilization float64 `json:"worker_utilization"`

	// System health
	IsHealthy       bool          `json:"is_healthy"`
	LastHealthCheck time.Time     `json:"last_health_check"`
	Uptime          time.Duration `json:"uptime"`

	// Timestamps
	LastProcessingTime time.Time `json:"last_processing_time"`
	StartTime          time.Time `json:"start_time"`
}

// NewProductionReadyEngineV2 creates a new production-ready strategy engine
func NewProductionReadyEngineV2(
	config *config.StrategyEngineV2Config,
	candleRepository *postgres.CandleRepository,
) *ProductionReadyEngineV2 {
	engine := &ProductionReadyEngineV2{
		config:           config,
		candleRepository: candleRepository,
		shutdownChan:     make(chan struct{}),
		metrics: &ProductionEngineMetrics{
			StartTime: time.Now(),
		},
	}

	// Initialize logging and debug components
	engine.initializeLoggingAndDebug()

	// Initialize components
	engine.initializeComponents()

	// Start background processes
	engine.startBackgroundProcesses()

	log.Info("Production-ready V2 strategy engine initialized")
	return engine
}

// initializeLoggingAndDebug initializes logging and debug components
func (engine *ProductionReadyEngineV2) initializeLoggingAndDebug() {
	// Initialize global logger if not already initialized
	if GetGlobalLogger() == nil {
		err := InitializeGlobalLogger("./logs", "v2_production_engine.log")
		if err != nil {
			log.Error("Failed to initialize global logger: %v", err)
		}
	}

	engine.logger = GetGlobalLogger()

	// Initialize debug manager
	engine.debugManager = NewDebugManager(engine.logger)

	// Initialize progress tracker manager
	engine.progressManager = NewProgressTrackerManager(engine.logger, engine.debugManager)

	// Enable debug features by default for now (can be made configurable later)
	engine.debugManager.EnableDebug()
}

// initializeComponents initializes all engine components
func (engine *ProductionReadyEngineV2) initializeComponents() {
	// Initialize registry
	registryConfig := &RegistryConfig{
		AutoDiscovery:   engine.config.Strategies.Registry.AutoDiscovery,
		DiscoveryPath:   engine.config.Strategies.Registry.DiscoveryPath,
		ParallelExec:    engine.config.Strategies.Execution.ParallelStrategies,
		StrategyTimeout: engine.config.Strategies.Execution.StrategyTimeout,
		ErrorHandling:   engine.config.Strategies.Execution.ErrorHandling,
	}
	engine.registry = NewStrategyRegistryV2(registryConfig)

	// Initialize parallel processor
	parallelConfig := DefaultParallelProcessorConfig()
	engine.parallelProcessor = NewParallelProcessorV2(parallelConfig, engine.progressManager, engine.debugManager, engine.logger)

	// Initialize state manager
	stateConfig := DefaultAdvancedStateConfig()
	engine.stateManager = NewAdvancedStateManagerV2(stateConfig)

	// Initialize monitoring
	engine.monitoring = NewProductionMonitoringV2(engine.config)

	// Initialize health checker
	engine.healthChecker = NewHealthCheckerV2(engine.config)
}

// startBackgroundProcesses starts all background processes
func (engine *ProductionReadyEngineV2) startBackgroundProcesses() {
	// Start health monitoring
	go engine.healthMonitoringWorker()

	// Start metrics collection
	go engine.metricsCollectionWorker()

	// Start performance monitoring
	go engine.performanceMonitoringWorker()

	// Start auto-scaling
	go engine.autoScalingWorker()
}

// ProcessStockGroups processes stock groups with production-ready features
func (engine *ProductionReadyEngineV2) ProcessStockGroups(
	ctx context.Context,
	stockGroups []domain.StockGroup,
	currentTime time.Time,
) (map[string]map[string]*StrategyResult, error) {
	startTime := time.Now()
	engine.metrics.LastProcessingTime = startTime
	engine.metrics.TotalProcessingRuns++

	// Create progress tracker for engine-level processing
	totalJobs := len(stockGroups)
	engine.progressManager.CreateTracker("engine_processing", "ProductionReadyEngineV2", totalJobs, LevelDetailed, map[string]interface{}{
		"current_time":       currentTime.Format(time.RFC3339),
		"stock_groups_count": len(stockGroups),
		"start_time":         startTime.Format(time.RFC3339),
	})

	// Start progress tracking
	engine.progressManager.StartTracker("engine_processing", map[string]interface{}{
		"processing_start": startTime.Format(time.RFC3339),
	})

	// Take initial state snapshot
	engine.debugManager.TakeStateSnapshot("engine", map[string]interface{}{
		"stock_groups_count": len(stockGroups),
		"current_time":       currentTime.Format(time.RFC3339),
		"start_time":         startTime.Format(time.RFC3339),
	})

	log.Info("Starting production processing for %d stock groups", len(stockGroups))

	// Health check before processing
	if !engine.healthChecker.IsHealthy() {
		engine.progressManager.UpdateProgress("engine_processing", 0, 1, StatusFailed, map[string]interface{}{
			"health_check_failed": true,
			"error":               "engine is not healthy",
		})
		return nil, fmt.Errorf("engine is not healthy, skipping processing")
	}

	// Update progress - Health check passed
	engine.progressManager.UpdateProgress("engine_processing", 1, 0, StatusRunning, map[string]interface{}{
		"step": "health_check_passed",
	})

	// Fetch historical data
	stockDataFrames, err := engine.fetchHistoricalData(ctx, stockGroups, currentTime)
	if err != nil {
		engine.recordError(err)
		engine.progressManager.UpdateProgress("engine_processing", 1, 1, StatusFailed, map[string]interface{}{
			"step":  "fetch_historical_data",
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to fetch historical data: %w", err)
	}

	// Update progress - Historical data fetched
	engine.progressManager.UpdateProgress("engine_processing", 2, 0, StatusRunning, map[string]interface{}{
		"step":             "historical_data_fetched",
		"dataframes_count": len(stockDataFrames),
	})

	// Get active strategies
	strategies := engine.registry.ListActiveStrategies()
	if len(strategies) == 0 {
		log.Warn("No active strategies found")
		engine.progressManager.CompleteTracker("engine_processing", map[string]interface{}{
			"step":             "no_strategies_found",
			"strategies_count": 0,
		})
		return make(map[string]map[string]*StrategyResult), nil
	}

	// Update progress - Strategies loaded
	engine.progressManager.UpdateProgress("engine_processing", 3, 0, StatusRunning, map[string]interface{}{
		"step":             "strategies_loaded",
		"strategies_count": len(strategies),
	})

	// Process using parallel processor
	results, err := engine.parallelProcessor.ProcessStockGroups(
		ctx, stockGroups, strategies, stockDataFrames, currentTime,
	)
	if err != nil {
		engine.recordError(err)
		engine.progressManager.UpdateProgress("engine_processing", 3, 1, StatusFailed, map[string]interface{}{
			"step":  "parallel_processing",
			"error": err.Error(),
		})
		return nil, fmt.Errorf("parallel processing failed: %w", err)
	}

	// Update progress - Parallel processing completed
	engine.progressManager.UpdateProgress("engine_processing", 4, 0, StatusRunning, map[string]interface{}{
		"step":          "parallel_processing_completed",
		"results_count": len(results),
	})

	// Update metrics
	processingTime := time.Since(startTime)
	engine.metrics.AverageProcessingTime = (engine.metrics.AverageProcessingTime + processingTime) / 2
	engine.metrics.TotalStocksProcessed += int64(len(stockGroups))
	engine.metrics.TotalStrategiesExecuted += int64(len(strategies) * len(stockGroups))

	// Track performance
	engine.debugManager.TrackPerformance("ProductionReadyEngineV2", "ProcessStockGroups", processingTime)

	// Update state management
	engine.updateStateManagement(stockGroups, strategies, results)

	// Update progress - State management completed
	engine.progressManager.UpdateProgress("engine_processing", 5, 0, StatusRunning, map[string]interface{}{
		"step": "state_management_completed",
	})

	// Complete progress tracking
	engine.progressManager.CompleteTracker("engine_processing", map[string]interface{}{
		"processing_time":           processingTime.String(),
		"total_stocks_processed":    len(stockGroups),
		"total_strategies_executed": len(strategies),
		"results_count":             len(results),
		"completion_time":           time.Now().Format(time.RFC3339),
	})

	log.Info("Production processing completed in %v for %d stock groups",
		processingTime, len(stockGroups))

	return results, nil
}

// fetchHistoricalData fetches historical data for all stock groups
func (engine *ProductionReadyEngineV2) fetchHistoricalData(
	ctx context.Context,
	stockGroups []domain.StockGroup,
	currentTime time.Time,
) (map[string]*dataframe.DataFrame, error) {
	stockDataFrames := make(map[string]*dataframe.DataFrame)

	// Get maximum required history across all strategies
	maxHistory := engine.getMaxRequiredHistory()

	// Calculate start time for historical data
	startTime := currentTime.Add(-time.Duration(maxHistory) * 5 * time.Minute)

	log.Info("Fetching historical data from %s to %s (max history: %d candles)",
		startTime.Format("2006-01-02 15:04:05"),
		currentTime.Format("2006-01-02 15:04:05"),
		maxHistory)

	for _, group := range stockGroups {
		// Get the first stock from the group for data fetching
		if len(group.Stocks) == 0 {
			log.Warn("Stock group %s has no stocks, skipping", group.ID)
			continue
		}

		stock := group.Stocks[0] // Use first stock for the group

		// Fetch candles using the candle repository
		candles, err := engine.candleRepository.FindByInstrumentAndTimeRange(
			ctx, stock.StockID, "5min", startTime, currentTime,
		)
		if err != nil {
			log.Error("Failed to fetch candles for stock %s: %v", stock.StockID, err)
			continue
		}

		if len(candles) == 0 {
			log.Warn("No candles found for stock %s in time range", stock.StockID)
			continue
		}

		// Convert candles to DataFrame
		df, err := engine.convertCandlesToDataFrame(candles)
		if err != nil {
			log.Error("Failed to convert candles to DataFrame for stock %s: %v", stock.StockID, err)
			continue
		}

		stockDataFrames[group.ID] = df
		log.Info("Fetched %d candles for stock group %s (stock: %s)", len(candles), group.ID, stock.StockID)
	}

	return stockDataFrames, nil
}

// convertCandlesToDataFrame converts candles to DataFrame
func (engine *ProductionReadyEngineV2) convertCandlesToDataFrame(candles []domain.Candle) (*dataframe.DataFrame, error) {
	if len(candles) == 0 {
		return &dataframe.DataFrame{}, nil
	}

	// Prepare data slices
	timestamps := make([]string, len(candles))
	opens := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	volumes := make([]int64, len(candles))

	// Populate data slices
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
		series.Strings(timestamps),
		series.Floats(opens),
		series.Floats(highs),
		series.Floats(lows),
		series.Floats(closes),
		series.Ints(volumes),
	)

	return &df, nil
}

// getMaxRequiredHistory gets the maximum required history across all strategies
func (engine *ProductionReadyEngineV2) getMaxRequiredHistory() int {
	strategies := engine.registry.ListActiveStrategies()
	maxHistory := 0

	for _, strategy := range strategies {
		required := strategy.GetRequiredHistory()
		if required > maxHistory {
			maxHistory = required
		}
	}

	return maxHistory
}

// updateStateManagement updates state management with processing results
func (engine *ProductionReadyEngineV2) updateStateManagement(
	stockGroups []domain.StockGroup,
	strategies []StrategyV2,
	results map[string]map[string]*StrategyResult,
) {
	// Update state management metrics
	stateMetrics := engine.stateManager.GetMetrics()
	engine.metrics.TotalStatesManaged = stateMetrics.TotalStatesManaged
	engine.metrics.StateValidationErrors = stateMetrics.ValidationErrors

	// Calculate cache hit rate
	if stateMetrics.CacheHits+stateMetrics.CacheMisses > 0 {
		engine.metrics.StateCacheHitRate = float64(stateMetrics.CacheHits) /
			float64(stateMetrics.CacheHits+stateMetrics.CacheMisses)
	}
}

// RegisterStrategy registers a strategy with the engine
func (engine *ProductionReadyEngineV2) RegisterStrategy(strategy StrategyV2) error {
	return engine.registry.Register(strategy)
}

// GetMetrics returns comprehensive engine metrics
func (engine *ProductionReadyEngineV2) GetMetrics() *ProductionEngineMetrics {
	engine.mu.RLock()
	defer engine.mu.RUnlock()

	// Update uptime
	engine.metrics.Uptime = time.Since(engine.metrics.StartTime)

	// Update parallel processor metrics
	parallelMetrics := engine.parallelProcessor.GetMetrics()
	engine.metrics.ActiveWorkers = parallelMetrics.ActiveWorkers
	engine.metrics.QueueLength = parallelMetrics.QueueLength

	// Update health status
	engine.metrics.IsHealthy = engine.healthChecker.IsHealthy()
	engine.metrics.LastHealthCheck = engine.healthChecker.GetLastCheckTime()

	// Create a copy to avoid race conditions
	metrics := *engine.metrics
	return &metrics
}

// ResetMetrics resets all metrics
func (engine *ProductionReadyEngineV2) ResetMetrics() {
	engine.mu.Lock()
	defer engine.mu.Unlock()

	engine.metrics = &ProductionEngineMetrics{
		StartTime: time.Now(),
	}
}

// GetHealthStatus returns the current health status
func (engine *ProductionReadyEngineV2) GetHealthStatus() *HealthStatus {
	return engine.healthChecker.GetHealthStatus()
}

// Shutdown gracefully shuts down the engine
func (engine *ProductionReadyEngineV2) Shutdown() {
	log.Info("Shutting down production-ready V2 strategy engine")

	// Signal shutdown
	close(engine.shutdownChan)

	// Shutdown components
	if engine.parallelProcessor != nil {
		engine.parallelProcessor.Shutdown()
	}

	if engine.stateManager != nil {
		engine.stateManager.Shutdown()
	}

	if engine.monitoring != nil {
		engine.monitoring.Shutdown()
	}

	log.Info("Production-ready V2 strategy engine shutdown complete")
}

// Background workers

func (engine *ProductionReadyEngineV2) healthMonitoringWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			engine.healthChecker.CheckHealth()
		case <-engine.shutdownChan:
			return
		}
	}
}

func (engine *ProductionReadyEngineV2) metricsCollectionWorker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			engine.updateMetrics()
		case <-engine.shutdownChan:
			return
		}
	}
}

func (engine *ProductionReadyEngineV2) performanceMonitoringWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			engine.updatePerformanceMetrics()
		case <-engine.shutdownChan:
			return
		}
	}
}

func (engine *ProductionReadyEngineV2) autoScalingWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			engine.autoScale()
		case <-engine.shutdownChan:
			return
		}
	}
}

func (engine *ProductionReadyEngineV2) updateMetrics() {
	engine.mu.Lock()
	defer engine.mu.Unlock()

	// Update error rate
	if engine.metrics.TotalProcessingRuns > 0 {
		engine.metrics.ErrorRate = float64(engine.metrics.TotalErrors) /
			float64(engine.metrics.TotalProcessingRuns)
	}
}

func (engine *ProductionReadyEngineV2) updatePerformanceMetrics() {
	engine.mu.Lock()
	defer engine.mu.Unlock()

	// Update memory usage
	engine.metrics.MemoryUsageMB = engine.parallelProcessor.getCurrentMemoryUsage()

	// Update CPU usage (simplified)
	engine.metrics.CPUUsagePercent = 50.0 // Placeholder

	// Update goroutine count
	engine.metrics.ActiveGoroutines = 100 // Placeholder
}

func (engine *ProductionReadyEngineV2) autoScale() {
	// Auto-scale parallel processor
	if engine.parallelProcessor != nil {
		engine.parallelProcessor.workerPool.AutoScale()
	}
}

func (engine *ProductionReadyEngineV2) recordError(err error) {
	engine.mu.Lock()
	defer engine.mu.Unlock()

	engine.metrics.TotalErrors++
	engine.metrics.LastErrorTime = time.Now()

	log.Error("Production engine error: %v", err)
}

// Placeholder implementations for components

type ProductionMonitoringV2 struct {
	config *config.StrategyEngineV2Config
}

func NewProductionMonitoringV2(config *config.StrategyEngineV2Config) *ProductionMonitoringV2 {
	return &ProductionMonitoringV2{config: config}
}

func (m *ProductionMonitoringV2) Shutdown() {
	// Implementation would shutdown monitoring
}

type HealthCheckerV2 struct {
	config *config.StrategyEngineV2Config
}

func NewHealthCheckerV2(config *config.StrategyEngineV2Config) *HealthCheckerV2 {
	return &HealthCheckerV2{config: config}
}

func (h *HealthCheckerV2) IsHealthy() bool {
	// Implementation would check health
	return true
}

func (h *HealthCheckerV2) GetLastCheckTime() time.Time {
	// Implementation would return last check time
	return time.Now()
}

func (h *HealthCheckerV2) CheckHealth() {
	// Implementation would perform health check
}

func (h *HealthCheckerV2) GetHealthStatus() *HealthStatus {
	// Implementation would return health status
	return &HealthStatus{IsHealthy: true}
}

type HealthStatus struct {
	IsHealthy bool
	Details   map[string]interface{}
}
