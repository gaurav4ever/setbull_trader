package v2

import (
	"context"
	"testing"
	"time"

	"setbull_trader/internal/domain"

	"github.com/go-gota/gota/dataframe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParallelProcessorV2(t *testing.T) {
	processor := NewParallelProcessorV2(nil)

	assert.NotNil(t, processor)
	assert.NotNil(t, processor.config)
	assert.NotNil(t, processor.workerPool)
	assert.NotNil(t, processor.metrics)

	// Check default configuration
	assert.Equal(t, 10, processor.config.MaxConcurrentStrategies)
	assert.Equal(t, 10*time.Second, processor.config.StrategyTimeout)
	assert.Equal(t, 1024, processor.config.MaxMemoryUsageMB)
	assert.Equal(t, "continue_on_error", processor.config.ErrorHandling)
}

func TestParallelProcessorV2_ProcessStockGroups(t *testing.T) {
	processor := NewParallelProcessorV2(nil)

	// Create test stock groups
	stockGroups := []domain.StockGroup{
		{
			ID: "group1",
			Stocks: []domain.StockGroupStock{
				{ID: "stock1", GroupID: "group1", StockID: "STOCK1"},
			},
		},
		{
			ID: "group2",
			Stocks: []domain.StockGroupStock{
				{ID: "stock2", GroupID: "group2", StockID: "STOCK2"},
			},
		},
	}

	// Create test strategies
	strategies := []StrategyV2{
		&MockStrategyV2{name: "Strategy1"},
		&MockStrategyV2{name: "Strategy2"},
	}

	// Create test data frames
	dataFrames := map[string]*dataframe.DataFrame{
		"group1": &dataframe.DataFrame{},
		"group2": &dataframe.DataFrame{},
	}

	currentTime := time.Now()
	ctx := context.Background()

	// Process stock groups
	results, err := processor.ProcessStockGroups(ctx, stockGroups, strategies, dataFrames, currentTime)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 2)

	// Check that both groups were processed
	assert.Contains(t, results, "group1")
	assert.Contains(t, results, "group2")

	// Check metrics
	metrics := processor.GetMetrics()
	assert.Equal(t, int64(2), metrics.TotalStocksProcessed)
	assert.Equal(t, int64(4), metrics.TotalStrategiesExecuted) // 2 groups * 2 strategies
}

func TestParallelProcessorV2_CreateProcessingJobs(t *testing.T) {
	processor := NewParallelProcessorV2(nil)

	stockGroups := []domain.StockGroup{
		{ID: "group1", Stocks: []domain.StockGroupStock{{ID: "stock1", GroupID: "group1", StockID: "STOCK1"}}},
		{ID: "group2", Stocks: []domain.StockGroupStock{{ID: "stock2", GroupID: "group2", StockID: "STOCK2"}}},
	}

	strategies := []StrategyV2{
		&MockStrategyV2{name: "Strategy1"},
	}

	dataFrames := map[string]*dataframe.DataFrame{
		"group1": &dataframe.DataFrame{},
		"group2": &dataframe.DataFrame{},
	}

	currentTime := time.Now()

	jobs := processor.createProcessingJobs(stockGroups, strategies, dataFrames, currentTime)

	assert.Len(t, jobs, 2)
	assert.Equal(t, "job_0_group1", jobs[0].ID)
	assert.Equal(t, "job_1_group2", jobs[1].ID)
	assert.Equal(t, stockGroups[0], jobs[0].StockGroup)
	assert.Equal(t, stockGroups[1], jobs[1].StockGroup)
	assert.Equal(t, strategies, jobs[0].Strategies)
	assert.Equal(t, currentTime, jobs[0].CurrentTime)
}

func TestParallelProcessorV2_ProcessJob(t *testing.T) {
	processor := NewParallelProcessorV2(nil)

	job := &ProcessingJob{
		ID: "test_job",
		StockGroup: domain.StockGroup{
			ID:     "group1",
			Stocks: []domain.StockGroupStock{{Symbol: "STOCK1"}},
		},
		Strategies: []StrategyV2{
			&MockStrategyV2{name: "Strategy1"},
		},
		DataFrame:   &dataframe.DataFrame{},
		CurrentTime: time.Now(),
		Priority:    1,
		RetryCount:  0,
		CreatedAt:   time.Now(),
	}

	ctx := context.Background()
	result := processor.processJob(ctx, job)

	assert.NotNil(t, result)
	assert.Equal(t, "test_job", result.JobID)
	assert.Equal(t, "group1", result.StockGroupID)
	assert.NotNil(t, result.Results)
	assert.Nil(t, result.Error)
	assert.Greater(t, result.ProcessingTime, time.Duration(0))
}

func TestParallelProcessorV2_ProcessStrategies(t *testing.T) {
	processor := NewParallelProcessorV2(nil)

	job := &ProcessingJob{
		ID: "test_job",
		StockGroup: domain.StockGroup{
			ID:     "group1",
			Stocks: []domain.StockGroupStock{{ID: "stock1", GroupID: "group1", StockID: "STOCK1"}},
		},
		Strategies: []StrategyV2{
			&MockStrategyV2{name: "Strategy1"},
			&MockStrategyV2{name: "Strategy2"},
		},
		DataFrame:   &dataframe.DataFrame{},
		CurrentTime: time.Now(),
	}

	ctx := context.Background()
	results, err := processor.processStrategies(ctx, job)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 2)
	assert.Contains(t, results, "Strategy1")
	assert.Contains(t, results, "Strategy2")
}

func TestParallelProcessorV2_ExecuteStrategy(t *testing.T) {
	processor := NewParallelProcessorV2(nil)

	strategy := &MockStrategyV2{name: "TestStrategy"}
	df := &dataframe.DataFrame{}
	ctx := context.Background()

	result := processor.executeStrategy(ctx, strategy, df)

	assert.NotNil(t, result)
	assert.Equal(t, "TestStrategy", result.StrategyName)
	assert.Equal(t, df.Nrow(), result.RowsProcessed)
	assert.Nil(t, result.Error)
	assert.Greater(t, result.ProcessingTime, time.Duration(0))
}

func TestParallelProcessorV2_MemoryManagement(t *testing.T) {
	processor := NewParallelProcessorV2(nil)

	// Test memory usage calculation
	memoryUsage := processor.getCurrentMemoryUsage()
	assert.GreaterOrEqual(t, memoryUsage, int64(0))

	// Test memory throttling
	shouldThrottle := processor.shouldThrottleMemory()
	assert.IsType(t, false, shouldThrottle)
}

func TestParallelProcessorV2_Metrics(t *testing.T) {
	processor := NewParallelProcessorV2(nil)

	// Get initial metrics
	initialMetrics := processor.GetMetrics()
	assert.NotNil(t, initialMetrics)
	assert.Equal(t, int64(0), initialMetrics.TotalJobsProcessed)

	// Process some jobs to update metrics
	stockGroups := []domain.StockGroup{
		{ID: "group1", Stocks: []domain.StockGroupStock{{Symbol: "STOCK1"}}},
	}
	strategies := []StrategyV2{&MockStrategyV2{name: "Strategy1"}}
	dataFrames := map[string]*dataframe.DataFrame{"group1": &dataframe.DataFrame{}}

	ctx := context.Background()
	currentTime := time.Now()

	_, err := processor.ProcessStockGroups(ctx, stockGroups, strategies, dataFrames, currentTime)
	require.NoError(t, err)

	// Check updated metrics
	updatedMetrics := processor.GetMetrics()
	assert.Equal(t, int64(1), updatedMetrics.TotalStocksProcessed)
	assert.Equal(t, int64(1), updatedMetrics.TotalStrategiesExecuted)
	assert.Greater(t, updatedMetrics.AverageProcessingTime, time.Duration(0))
}

func TestParallelProcessorV2_ErrorHandling(t *testing.T) {
	// Test with error-producing strategy
	processor := NewParallelProcessorV2(nil)

	stockGroups := []domain.StockGroup{
		{ID: "group1", Stocks: []domain.StockGroupStock{{Symbol: "STOCK1"}}},
	}
	strategies := []StrategyV2{&MockStrategyV2{name: "ErrorStrategy", shouldError: true}}
	dataFrames := map[string]*dataframe.DataFrame{"group1": &dataframe.DataFrame{}}

	ctx := context.Background()
	currentTime := time.Now()

	results, err := processor.ProcessStockGroups(ctx, stockGroups, strategies, dataFrames, currentTime)

	// With continue_on_error, should not return error
	assert.NoError(t, err)
	assert.NotNil(t, results)

	// Check that error was recorded in metrics
	metrics := processor.GetMetrics()
	assert.Greater(t, metrics.TotalErrors, int64(0))
}

func TestParallelProcessorV2_Shutdown(t *testing.T) {
	processor := NewParallelProcessorV2(nil)

	// Shutdown should not panic
	assert.NotPanics(t, func() {
		processor.Shutdown()
	})
}

func TestDefaultParallelProcessorConfig(t *testing.T) {
	config := DefaultParallelProcessorConfig()

	assert.NotNil(t, config)
	assert.Greater(t, config.MaxWorkers, 0)
	assert.Greater(t, config.MinWorkers, 0)
	assert.Equal(t, 30*time.Second, config.WorkerTimeout)
	assert.Equal(t, 1000, config.QueueSize)
	assert.Equal(t, 10, config.MaxConcurrentStrategies)
	assert.Equal(t, 10*time.Second, config.StrategyTimeout)
	assert.Equal(t, 50, config.BatchSize)
	assert.Equal(t, 1024, config.MaxMemoryUsageMB)
	assert.Equal(t, 5*time.Second, config.MemoryCheckInterval)
	assert.Equal(t, "continue_on_error", config.ErrorHandling)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.RetryDelay)
	assert.True(t, config.EnableMetrics)
	assert.Equal(t, 10*time.Second, config.MetricsInterval)
}

// Mock Strategy for testing
type MockStrategyV2 struct {
	name        string
	shouldError bool
}

func (m *MockStrategyV2) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
	if m.shouldError {
		return nil, assert.AnError
	}
	return df, nil
}

func (m *MockStrategyV2) GetRequiredHistory() int {
	return 10
}

func (m *MockStrategyV2) GetName() string {
	return m.name
}

func (m *MockStrategyV2) GetVersion() string {
	return "1.0.0"
}

func (m *MockStrategyV2) GetDescription() string {
	return "Mock strategy for testing"
}

func (m *MockStrategyV2) GetAuthor() string {
	return "Test Author"
}

func (m *MockStrategyV2) GetTags() []string {
	return []string{"test", "mock"}
}

func (m *MockStrategyV2) IsEnabled() bool {
	return true
}

func (m *MockStrategyV2) ValidateConfiguration() error {
	return nil
}

func (m *MockStrategyV2) Configure(config map[string]interface{}) error {
	return nil
}

func (m *MockStrategyV2) GetEstimatedProcessingTime() time.Duration {
	return 50 * time.Millisecond
}

func (m *MockStrategyV2) GetMemoryRequirements() int64 {
	return 1024 * 1024 // 1MB
}
