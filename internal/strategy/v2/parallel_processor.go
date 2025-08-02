package v2

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/pkg/log"

	"github.com/go-gota/gota/dataframe"
)

// ParallelProcessorV2 provides enhanced parallel processing capabilities for multiple strategies
type ParallelProcessorV2 struct {
	config     *ParallelProcessorConfig
	workerPool *WorkerPool
	metrics    *ParallelProcessorMetrics
	mu         sync.RWMutex
}

// ParallelProcessorConfig contains configuration for parallel processing
type ParallelProcessorConfig struct {
	// Worker pool configuration
	MaxWorkers    int           `yaml:"max_workers" json:"max_workers"`
	MinWorkers    int           `yaml:"min_workers" json:"min_workers"`
	WorkerTimeout time.Duration `yaml:"worker_timeout" json:"worker_timeout"`
	QueueSize     int           `yaml:"queue_size" json:"queue_size"`

	// Strategy execution configuration
	MaxConcurrentStrategies int           `yaml:"max_concurrent_strategies" json:"max_concurrent_strategies"`
	StrategyTimeout         time.Duration `yaml:"strategy_timeout" json:"strategy_timeout"`
	BatchSize               int           `yaml:"batch_size" json:"batch_size"`

	// Memory management
	MaxMemoryUsageMB    int           `yaml:"max_memory_usage_mb" json:"max_memory_usage_mb"`
	MemoryCheckInterval time.Duration `yaml:"memory_check_interval" json:"memory_check_interval"`

	// Error handling
	ErrorHandling string        `yaml:"error_handling" json:"error_handling"`
	MaxRetries    int           `yaml:"max_retries" json:"max_retries"`
	RetryDelay    time.Duration `yaml:"retry_delay" json:"retry_delay"`

	// Performance monitoring
	EnableMetrics   bool          `yaml:"enable_metrics" json:"enable_metrics"`
	MetricsInterval time.Duration `yaml:"metrics_interval" json:"metrics_interval"`
}

// ParallelProcessorMetrics contains metrics for parallel processing
type ParallelProcessorMetrics struct {
	TotalJobsProcessed      int64         `json:"total_jobs_processed"`
	TotalStrategiesExecuted int64         `json:"total_strategies_executed"`
	TotalStocksProcessed    int64         `json:"total_stocks_processed"`
	AverageProcessingTime   time.Duration `json:"average_processing_time"`
	TotalErrors             int64         `json:"total_errors"`
	TotalRetries            int64         `json:"total_retries"`
	MemoryUsageMB           int64         `json:"memory_usage_mb"`
	ActiveWorkers           int           `json:"active_workers"`
	QueueLength             int           `json:"queue_length"`
	LastProcessingTime      time.Time     `json:"last_processing_time"`
}

// ProcessingJob represents a job to be processed
type ProcessingJob struct {
	ID          string
	StockGroup  domain.StockGroup
	Strategies  []StrategyV2
	DataFrame   *dataframe.DataFrame
	CurrentTime time.Time
	Priority    int
	RetryCount  int
	CreatedAt   time.Time
}

// ProcessingResult represents the result of processing a job
type ProcessingResult struct {
	JobID          string
	StockGroupID   string
	Results        map[string]*StrategyResult
	Error          error
	ProcessingTime time.Duration
	MemoryUsed     int64
}

// NewParallelProcessorV2 creates a new parallel processor
func NewParallelProcessorV2(config *ParallelProcessorConfig) *ParallelProcessorV2 {
	if config == nil {
		config = DefaultParallelProcessorConfig()
	}

	processor := &ParallelProcessorV2{
		config:  config,
		metrics: &ParallelProcessorMetrics{},
	}

	// Create worker pool
	processor.workerPool = NewWorkerPool(config)

	// Start metrics collection if enabled
	if config.EnableMetrics {
		go processor.collectMetrics()
	}

	return processor
}

// DefaultParallelProcessorConfig returns default configuration
func DefaultParallelProcessorConfig() *ParallelProcessorConfig {
	return &ParallelProcessorConfig{
		// Worker pool configuration
		MaxWorkers:    runtime.NumCPU() * 2,
		MinWorkers:    runtime.NumCPU(),
		WorkerTimeout: 30 * time.Second,
		QueueSize:     1000,

		// Strategy execution configuration
		MaxConcurrentStrategies: 10,
		StrategyTimeout:         10 * time.Second,
		BatchSize:               50,

		// Memory management
		MaxMemoryUsageMB:    1024, // 1GB
		MemoryCheckInterval: 5 * time.Second,

		// Error handling
		ErrorHandling: "continue_on_error",
		MaxRetries:    3,
		RetryDelay:    1 * time.Second,

		// Performance monitoring
		EnableMetrics:   true,
		MetricsInterval: 10 * time.Second,
	}
}

// ProcessStockGroups processes multiple stock groups with multiple strategies in parallel
func (p *ParallelProcessorV2) ProcessStockGroups(
	ctx context.Context,
	stockGroups []domain.StockGroup,
	strategies []StrategyV2,
	dataFrames map[string]*dataframe.DataFrame,
	currentTime time.Time,
) (map[string]map[string]*StrategyResult, error) {
	startTime := time.Now()
	p.metrics.LastProcessingTime = startTime
	p.metrics.TotalStocksProcessed += int64(len(stockGroups))

	log.Info("Starting parallel processing for %d stock groups with %d strategies",
		len(stockGroups), len(strategies))

	// Create processing jobs
	jobs := p.createProcessingJobs(stockGroups, strategies, dataFrames, currentTime)

	// Process jobs in parallel
	results := make(map[string]map[string]*StrategyResult)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var processingErrors []error

	// Create result channels
	resultChan := make(chan *ProcessingResult, len(jobs))
	errorChan := make(chan error, len(jobs))

	// Submit jobs to worker pool
	for _, job := range jobs {
		wg.Add(1)
		go func(job *ProcessingJob) {
			defer wg.Done()

			result := p.processJob(ctx, job)
			if result.Error != nil {
				errorChan <- result.Error
			} else {
				resultChan <- result
			}
		}(job)
	}

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Process results
	for result := range resultChan {
		mu.Lock()
		results[result.StockGroupID] = result.Results
		mu.Unlock()

		p.metrics.TotalJobsProcessed++
		p.metrics.TotalStrategiesExecuted += int64(len(result.Results))
	}

	// Process errors
	for err := range errorChan {
		processingErrors = append(processingErrors, err)
		p.metrics.TotalErrors++
	}

	// Calculate processing time
	processingTime := time.Since(startTime)
	p.metrics.AverageProcessingTime = (p.metrics.AverageProcessingTime + processingTime) / 2

	// Handle errors based on configuration
	if len(processingErrors) > 0 {
		if p.config.ErrorHandling == "fail_fast" {
			return results, fmt.Errorf("parallel processing failed: %v", processingErrors[0])
		}
		// For "continue_on_error", we log errors but don't fail
		for _, err := range processingErrors {
			log.Error("Parallel processing error: %v", err)
		}
	}

	log.Info("Parallel processing completed in %v for %d stock groups",
		processingTime, len(stockGroups))

	return results, nil
}

// createProcessingJobs creates processing jobs for stock groups
func (p *ParallelProcessorV2) createProcessingJobs(
	stockGroups []domain.StockGroup,
	strategies []StrategyV2,
	dataFrames map[string]*dataframe.DataFrame,
	currentTime time.Time,
) []*ProcessingJob {
	jobs := make([]*ProcessingJob, 0, len(stockGroups))

	for i, group := range stockGroups {
		job := &ProcessingJob{
			ID:          fmt.Sprintf("job_%d_%s", i, group.ID),
			StockGroup:  group,
			Strategies:  strategies,
			DataFrame:   dataFrames[group.ID],
			CurrentTime: currentTime,
			Priority:    1, // Default priority
			RetryCount:  0,
			CreatedAt:   time.Now(),
		}
		jobs = append(jobs, job)
	}

	return jobs
}

// processJob processes a single job
func (p *ParallelProcessorV2) processJob(ctx context.Context, job *ProcessingJob) *ProcessingResult {
	startTime := time.Now()
	result := &ProcessingResult{
		JobID:        job.ID,
		StockGroupID: job.StockGroup.ID,
		Results:      make(map[string]*StrategyResult),
	}

	// Check memory usage
	if p.shouldThrottleMemory() {
		time.Sleep(p.config.MemoryCheckInterval)
	}

	// Process strategies for this stock group
	strategyResults, err := p.processStrategies(ctx, job)
	if err != nil {
		result.Error = err
		// Retry logic
		if job.RetryCount < p.config.MaxRetries {
			job.RetryCount++
			p.metrics.TotalRetries++
			time.Sleep(p.config.RetryDelay)
			return p.processJob(ctx, job)
		}
	} else {
		result.Results = strategyResults
	}

	result.ProcessingTime = time.Since(startTime)
	result.MemoryUsed = p.getCurrentMemoryUsage()

	return result
}

// processStrategies processes multiple strategies for a stock group
func (p *ParallelProcessorV2) processStrategies(
	ctx context.Context,
	job *ProcessingJob,
) (map[string]*StrategyResult, error) {
	results := make(map[string]*StrategyResult)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	// Create semaphore for concurrent strategy execution
	semaphore := make(chan struct{}, p.config.MaxConcurrentStrategies)

	for _, strategy := range job.Strategies {
		wg.Add(1)
		go func(s StrategyV2) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			strategyResult := p.executeStrategy(ctx, s, job.DataFrame)

			mu.Lock()
			results[s.GetName()] = strategyResult
			if strategyResult.Error != nil {
				errors = append(errors, strategyResult.Error)
			}
			mu.Unlock()
		}(strategy)
	}

	wg.Wait()

	// Handle errors based on configuration
	if len(errors) > 0 {
		if p.config.ErrorHandling == "fail_fast" {
			return results, fmt.Errorf("strategy execution failed: %v", errors[0])
		}
		// For "continue_on_error", we log errors but don't fail
		for _, err := range errors {
			log.Error("Strategy execution error: %v", err)
		}
	}

	return results, nil
}

// executeStrategy executes a single strategy with timeout and memory monitoring
func (p *ParallelProcessorV2) executeStrategy(
	ctx context.Context,
	strategy StrategyV2,
	df *dataframe.DataFrame,
) *StrategyResult {
	startTime := time.Now()
	result := &StrategyResult{
		StrategyName:  strategy.GetName(),
		RowsProcessed: df.Nrow(),
		ColumnsAdded:  make([]string, 0),
	}

	// Create a context with timeout
	execCtx, cancel := context.WithTimeout(ctx, p.config.StrategyTimeout)
	defer cancel()

	// Execute strategy in a goroutine to handle timeout
	done := make(chan error, 1)
	go func() {
		processedDF, err := strategy.Process(df)
		if err != nil {
			done <- err
			return
		}

		// Calculate columns added
		originalCols := df.Names()
		newCols := processedDF.Names()
		for _, col := range newCols {
			found := false
			for _, origCol := range originalCols {
				if col == origCol {
					found = true
					break
				}
			}
			if !found {
				result.ColumnsAdded = append(result.ColumnsAdded, col)
			}
		}

		done <- nil
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		if err != nil {
			result.Error = err
		}
	case <-execCtx.Done():
		result.Error = fmt.Errorf("strategy execution timeout after %v", p.config.StrategyTimeout)
	}

	result.ProcessingTime = time.Since(startTime)
	return result
}

// shouldThrottleMemory checks if memory usage is too high and throttling is needed
func (p *ParallelProcessorV2) shouldThrottleMemory() bool {
	currentUsage := p.getCurrentMemoryUsage()
	return currentUsage > int64(p.config.MaxMemoryUsageMB)
}

// getCurrentMemoryUsage gets current memory usage in MB
func (p *ParallelProcessorV2) getCurrentMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc / 1024 / 1024) // Convert to MB
}

// collectMetrics collects performance metrics periodically
func (p *ParallelProcessorV2) collectMetrics() {
	ticker := time.NewTicker(p.config.MetricsInterval)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		p.metrics.MemoryUsageMB = p.getCurrentMemoryUsage()
		p.metrics.ActiveWorkers = p.workerPool.GetActiveWorkerCount()
		p.metrics.QueueLength = p.workerPool.GetQueueLength()
		p.mu.Unlock()

		log.Debug("Parallel processor metrics: Memory=%dMB, Workers=%d, Queue=%d",
			p.metrics.MemoryUsageMB, p.metrics.ActiveWorkers, p.metrics.QueueLength)
	}
}

// GetMetrics returns current metrics
func (p *ParallelProcessorV2) GetMetrics() *ParallelProcessorMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *p.metrics
	return &metrics
}

// ResetMetrics resets all metrics
func (p *ParallelProcessorV2) ResetMetrics() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.metrics = &ParallelProcessorMetrics{}
}

// Shutdown gracefully shuts down the parallel processor
func (p *ParallelProcessorV2) Shutdown() {
	log.Info("Shutting down parallel processor")

	// Shutdown worker pool
	if p.workerPool != nil {
		p.workerPool.Shutdown()
	}

	log.Info("Parallel processor shutdown complete")
}
