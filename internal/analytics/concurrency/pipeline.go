package concurrency

import (
	"context"
	"fmt"
	"sync"
	"time"

	"setbull_trader/internal/analytics/cache"
	"setbull_trader/internal/domain"
	"setbull_trader/pkg/log"
)

// Pipeline represents a parallel processing pipeline for technical indicators
type Pipeline struct {
	workerPool     *WorkerPool
	cache          *cache.IndicatorCache
	processingPool *cache.ProcessingPool

	// Configuration
	batchSize      int
	maxConcurrency int
	timeout        time.Duration

	// Metrics
	mu               sync.RWMutex
	processedBatches int64
	totalProcessTime time.Duration
	avgBatchTime     time.Duration
}

// PipelineConfig configures the processing pipeline
type PipelineConfig struct {
	WorkerPoolConfig WorkerPoolConfig `json:"worker_pool"`
	BatchSize        int              `json:"batch_size"`
	MaxConcurrency   int              `json:"max_concurrency"`
	Timeout          time.Duration    `json:"timeout"`
	CacheSize        int              `json:"cache_size_mb"`
}

// NewPipeline creates a new parallel processing pipeline
func NewPipeline(config PipelineConfig) *Pipeline {
	if config.BatchSize <= 0 {
		config.BatchSize = 10
	}
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = config.WorkerPoolConfig.MaxWorkers
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.CacheSize <= 0 {
		config.CacheSize = 100 // 100MB default
	}

	workerPool := NewWorkerPool(config.WorkerPoolConfig)
	indicatorCache := cache.NewIndicatorCache(config.CacheSize)
	processingPool := cache.NewProcessingPool()

	return &Pipeline{
		workerPool:     workerPool,
		cache:          indicatorCache,
		processingPool: processingPool,
		batchSize:      config.BatchSize,
		maxConcurrency: config.MaxConcurrency,
		timeout:        config.Timeout,
	}
}

// Start starts the pipeline
func (p *Pipeline) Start() error {
	log.Info("Starting parallel processing pipeline")
	p.workerPool.Start()
	return nil
}

// ProcessBatch processes a batch of instruments with specified indicators
func (p *Pipeline) ProcessBatch(
	ctx context.Context,
	instruments []string,
	indicators []IndicatorRequest,
	candleData map[string][]domain.Candle,
) (*BatchProcessResult, error) {

	startTime := time.Now()

	result := &BatchProcessResult{
		Results:   make(map[string]interface{}),
		Errors:    make(map[string]error),
		StartTime: startTime,
		Timing:    make(map[string]time.Duration),
	}

	// Create tasks for each instrument
	var tasks []Task
	for _, instrument := range instruments {
		candles, exists := candleData[instrument]
		if !exists {
			result.Errors[instrument] = fmt.Errorf("no candle data found for instrument %s", instrument)
			continue
		}

		taskID := fmt.Sprintf("batch_%s_%d", instrument, time.Now().UnixNano())
		task := NewBatchIndicatorTask(taskID, instrument, candles, indicators, 1)
		tasks = append(tasks, task)
	}

	if len(tasks) == 0 {
		return result, fmt.Errorf("no valid tasks to process")
	}

	// Submit tasks to worker pool
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	if err := p.workerPool.SubmitBatch(ctx, tasks); err != nil {
		return result, fmt.Errorf("failed to submit batch tasks: %w", err)
	}

	// Collect results
	resultsCollected := 0
	expectedResults := len(tasks)

	for resultsCollected < expectedResults {
		select {
		case taskResult := <-p.workerPool.Results():
			instrumentKey := p.extractInstrumentFromTaskID(taskResult.TaskID)

			if taskResult.Error != nil {
				result.Errors[instrumentKey] = taskResult.Error
			} else {
				result.Results[instrumentKey] = taskResult.Data
				result.Timing[instrumentKey] = taskResult.Timing.Duration
			}

			resultsCollected++

		case <-ctx.Done():
			result.EndTime = time.Now()
			result.TotalDuration = result.EndTime.Sub(result.StartTime)
			return result, fmt.Errorf("timeout waiting for batch results: %w", ctx.Err())
		}
	}

	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	// Update metrics
	p.mu.Lock()
	p.processedBatches++
	p.totalProcessTime += result.TotalDuration
	p.avgBatchTime = time.Duration(int64(p.totalProcessTime) / p.processedBatches)
	p.mu.Unlock()

	log.Info("Processed batch of %d instruments in %v", len(instruments), result.TotalDuration)
	return result, nil
}

// ProcessSingle processes a single instrument with specified indicators
func (p *Pipeline) ProcessSingle(
	ctx context.Context,
	instrumentKey string,
	indicators []IndicatorRequest,
	candles []domain.Candle,
) (*SingleProcessResult, error) {

	startTime := time.Now()

	// Check cache first
	cacheKey := cache.CacheKey{
		Symbol:      instrumentKey,
		Timeframe:   "concurrent",                   // Special marker for concurrent processing
		StartTime:   time.Now().Add(-1 * time.Hour), // Placeholder
		EndTime:     time.Now(),
		Indicators:  p.extractIndicatorTypes(indicators),
		DataVersion: "pipeline_v1.0",
	}

	// Try cache first
	if cached, found := p.cache.GetIndicators(cacheKey); found {
		log.Debug("Cache hit for instrument %s", instrumentKey)
		return &SingleProcessResult{
			InstrumentKey: instrumentKey,
			Data:          cached,
			StartTime:     startTime,
			EndTime:       time.Now(),
			Duration:      time.Since(startTime),
			CacheHit:      true,
		}, nil
	}

	// Create and submit task
	taskID := fmt.Sprintf("single_%s_%d", instrumentKey, time.Now().UnixNano())
	task := NewBatchIndicatorTask(taskID, instrumentKey, candles, indicators, 0) // High priority

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	if err := p.workerPool.Submit(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to submit task: %w", err)
	}

	// Wait for result
	select {
	case taskResult := <-p.workerPool.Results():
		endTime := time.Now()

		result := &SingleProcessResult{
			InstrumentKey: instrumentKey,
			StartTime:     startTime,
			EndTime:       endTime,
			Duration:      endTime.Sub(startTime),
			CacheHit:      false,
		}

		if taskResult.Error != nil {
			result.Error = taskResult.Error
		} else {
			result.Data = taskResult.Data

			// Cache the result for future use
			if batchResult, ok := taskResult.Data.(*BatchIndicatorResult); ok {
				technicalIndicators := p.convertBatchResultToTechnicalIndicators(batchResult)
				_ = p.cache.SetIndicators(cacheKey, technicalIndicators, time.Minute, 5*time.Minute)
			}
		}

		return result, nil

	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for single result: %w", ctx.Err())
	}
}

// Shutdown gracefully shuts down the pipeline
func (p *Pipeline) Shutdown() error {
	log.Info("Shutting down processing pipeline")

	// First wait for current tasks to complete
	if err := p.workerPool.Wait(30 * time.Second); err != nil {
		log.Warn("Timeout waiting for tasks to complete: %v", err)
	}

	// Then shutdown the worker pool
	return p.workerPool.Shutdown()
}

// GetMetrics returns pipeline metrics
func (p *Pipeline) GetMetrics() PipelineMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	workerMetrics := p.workerPool.GetMetrics()
	cacheMetrics := p.cache.GetMetrics()
	poolStats := p.processingPool.Stats()

	return PipelineMetrics{
		WorkerPool:       workerMetrics,
		Cache:            cacheMetrics,
		ProcessingPool:   poolStats,
		ProcessedBatches: p.processedBatches,
		TotalProcessTime: p.totalProcessTime,
		AvgBatchTime:     p.avgBatchTime,
		BatchSize:        p.batchSize,
		MaxConcurrency:   p.maxConcurrency,
	}
}

// BatchProcessResult represents the result of batch processing
type BatchProcessResult struct {
	Results       map[string]interface{}   `json:"results"`
	Errors        map[string]error         `json:"errors"`
	Timing        map[string]time.Duration `json:"timing"`
	StartTime     time.Time                `json:"start_time"`
	EndTime       time.Time                `json:"end_time"`
	TotalDuration time.Duration            `json:"total_duration"`
}

// SingleProcessResult represents the result of single instrument processing
type SingleProcessResult struct {
	InstrumentKey string        `json:"instrument_key"`
	Data          interface{}   `json:"data"`
	Error         error         `json:"error,omitempty"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Duration      time.Duration `json:"duration"`
	CacheHit      bool          `json:"cache_hit"`
}

// PipelineMetrics contains comprehensive pipeline metrics
type PipelineMetrics struct {
	WorkerPool       WorkerPoolMetrics      `json:"worker_pool"`
	Cache            cache.CacheMetrics     `json:"cache"`
	ProcessingPool   map[string]interface{} `json:"processing_pool"`
	ProcessedBatches int64                  `json:"processed_batches"`
	TotalProcessTime time.Duration          `json:"total_process_time"`
	AvgBatchTime     time.Duration          `json:"avg_batch_time"`
	BatchSize        int                    `json:"batch_size"`
	MaxConcurrency   int                    `json:"max_concurrency"`
}

// Helper methods
func (p *Pipeline) extractInstrumentFromTaskID(taskID string) string {
	// Extract instrument key from task ID format: "batch_INSTRUMENT_TIMESTAMP"
	// This is a simple implementation - in production you might want more robust parsing
	if len(taskID) > 6 && taskID[:6] == "batch_" {
		parts := taskID[6:] // Remove "batch_" prefix
		// Find the last underscore to separate instrument from timestamp
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] == '_' {
				return parts[:i]
			}
		}
	}
	return "unknown"
}

func (p *Pipeline) extractIndicatorTypes(indicators []IndicatorRequest) []string {
	types := make([]string, len(indicators))
	for i, indicator := range indicators {
		types[i] = indicator.Type
	}
	return types
}

func (p *Pipeline) convertBatchResultToTechnicalIndicators(batchResult *BatchIndicatorResult) *domain.TechnicalIndicators {
	// This is a simplified conversion - in practice you'd need more sophisticated mapping
	// based on the specific indicator types and results structure

	result := &domain.TechnicalIndicators{
		InstrumentKey: batchResult.InstrumentKey,
		Interval:      "pipeline",
		StartTime:     time.Now().Add(-1 * time.Hour),
		EndTime:       time.Now(),
	}

	// Map results to domain fields based on indicator types
	for indicatorID, data := range batchResult.Results {
		if indicatorValues, ok := data.([]domain.IndicatorValue); ok {
			// Map to appropriate field based on indicator type
			if len(indicatorID) >= 3 {
				switch indicatorID[:3] {
				case "EMA":
					if len(result.EMA9) == 0 {
						result.EMA9 = indicatorValues
					} else {
						result.EMA50 = indicatorValues
					}
				case "RSI":
					result.RSI14 = indicatorValues
				case "ATR":
					result.ATR14 = indicatorValues
				}
			}
		}
	}

	return result
}
