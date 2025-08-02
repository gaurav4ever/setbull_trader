package v2

import (
	"context"
	"testing"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/trading/config"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCandleRepository is a mock implementation of the candle repository
type MockCandleRepository struct {
	mock.Mock
}

func (m *MockCandleRepository) GetCandlesByInstrumentKeyAndTimeRange(
	ctx context.Context,
	instrumentKey string,
	startTime, endTime time.Time,
	interval string,
) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, startTime, endTime, interval)
	return args.Get(0).([]domain.Candle), args.Error(1)
}

// MockStrategyEngineV2 creates a strategy engine with mock repository
func NewMockStrategyEngineV2(config *config.StrategyEngineV2Config) *StrategyEngineV2 {
	mockRepo := &MockCandleRepository{}
	return &StrategyEngineV2{
		config:           config,
		registry:         NewStrategyRegistryV2(&RegistryConfig{}),
		candleRepository: nil, // Will be set to mockRepo in tests
		metrics:          &EngineMetrics{},
	}
}

// TestStrategyEngineV2_ProcessStockGroups tests the main processing functionality
func TestStrategyEngineV2_ProcessStockGroups(t *testing.T) {
	// Setup
	config := &config.StrategyEngineV2Config{
		Enabled: true,
	}
	config.Processing.ConcurrentWorkers = 2
	config.Processing.MaxHistoricalCandles = 20
	config.Processing.BatchSize = 10

	engine := NewMockStrategyEngineV2(config)

	// Register a test strategy
	testStrategy := &TestStrategyV2{
		BaseStrategy: NewBaseStrategy(StrategyMetadata{
			Name:        "test_strategy",
			Version:     "1.0.0",
			Description: "Test strategy for unit testing",
			Author:      "Test Author",
			Tags:        []string{"test", "unit"},
		}),
	}
	engine.RegisterStrategy(testStrategy)

	// Create test stock groups
	stockGroups := []domain.StockGroup{
		{
			ID: "group1",
			Stocks: []domain.StockGroupStock{
				{StockID: "NSE_EQ|INE005B01027"},
				{StockID: "NSE_EQ|INE349Y01013"},
			},
		},
		{
			ID: "group2",
			Stocks: []domain.StockGroupStock{
				{StockID: "NSE_EQ|INE123456789"},
			},
		},
	}

	// Create mock candle data
	mockCandles := createMockCandles(20)
	mockRepo.On("GetCandlesByInstrumentKeyAndTimeRange",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockCandles, nil)

	// Execute
	ctx := context.Background()
	currentTime := time.Now()
	results, err := engine.ProcessStockGroups(ctx, stockGroups, currentTime)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 2) // 2 groups

	// Check that results were generated for each stock
	for _, group := range stockGroups {
		groupResults, exists := results[group.ID]
		assert.True(t, exists, "Results should exist for group %s", group.ID)

		for _, stock := range group.Stocks {
			resultKey := stock.StockID + "_test_strategy"
			result, exists := groupResults[resultKey]
			assert.True(t, exists, "Result should exist for stock %s", stock.StockID)
			assert.NotNil(t, result)
			assert.Equal(t, "test_strategy", result.StrategyName)
			assert.Equal(t, 20, result.RowsProcessed) // 20 mock candles
			assert.Len(t, result.ColumnsAdded, 1)     // test_column
		}
	}

	// Verify metrics
	metrics := engine.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalProcessingRuns)
	assert.Equal(t, int64(2), metrics.TotalStocksProcessed)
	assert.Equal(t, int64(3), metrics.TotalStrategiesExecuted) // 3 stocks * 1 strategy
}

// TestStrategyEngineV2_FetchHistoricalData tests historical data fetching
func TestStrategyEngineV2_FetchHistoricalData(t *testing.T) {
	// Setup
	config := &config.StrategyEngineV2Config{
		Enabled: true,
	}
	config.Processing.ConcurrentWorkers = 2
	config.Processing.MaxHistoricalCandles = 10

	engine := NewMockStrategyEngineV2(config)

	stockGroups := []domain.StockGroup{
		{
			ID: "group1",
			Stocks: []domain.StockGroupStock{
				{StockID: "NSE_EQ|INE005B01027"},
			},
		},
	}

	// Create mock candle data
	mockCandles := createMockCandles(10)
	mockRepo.On("GetCandlesByInstrumentKeyAndTimeRange",
		mock.Anything, "NSE_EQ|INE005B01027", mock.Anything, mock.Anything, "5minute").
		Return(mockCandles, nil)

	// Execute
	ctx := context.Background()
	currentTime := time.Now()
	stockDataFrames, err := engine.fetchHistoricalData(ctx, stockGroups, currentTime)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, stockDataFrames)
	assert.Len(t, stockDataFrames, 1)

	df, exists := stockDataFrames["NSE_EQ|INE005B01027"]
	assert.True(t, exists)
	assert.Equal(t, 10, df.Nrow()) // 10 candles
	assert.Equal(t, 6, df.Ncol())  // timestamp, open, high, low, close, volume

	mockRepo.AssertExpectations(t)
}

// TestStrategyEngineV2_ConvertCandlesToDataFrame tests DataFrame conversion
func TestStrategyEngineV2_ConvertCandlesToDataFrame(t *testing.T) {
	// Setup
	config := &config.StrategyEngineV2Config{Enabled: true}
	engine := NewMockStrategyEngineV2(config)

	// Create test candles
	candles := createMockCandles(5)

	// Execute
	df, err := engine.convertCandlesToDataFrame(candles)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, df)
	assert.Equal(t, 5, df.Nrow())
	assert.Equal(t, 6, df.Ncol()) // timestamp, open, high, low, close, volume

	// Check column names
	expectedColumns := []string{"timestamp", "open", "high", "low", "close", "volume"}
	for i, col := range expectedColumns {
		assert.Equal(t, col, df.Names()[i])
	}

	// Check data types
	assert.Equal(t, series.String, df.Col("timestamp").Type())
	assert.Equal(t, series.Float, df.Col("open").Type())
	assert.Equal(t, series.Float, df.Col("high").Type())
	assert.Equal(t, series.Float, df.Col("low").Type())
	assert.Equal(t, series.Float, df.Col("close").Type())
	assert.Equal(t, series.Int, df.Col("volume").Type())
}

// TestStrategyEngineV2_GetMaxRequiredHistory tests history calculation
func TestStrategyEngineV2_GetMaxRequiredHistory(t *testing.T) {
	// Setup
	config := &config.StrategyEngineV2Config{Enabled: true}
	mockRepo := &MockCandleRepository{}
	engine := NewStrategyEngineV2(config, mockRepo)

	// Register strategies with different history requirements
	strategy1 := &TestStrategyV2{
		BaseStrategy:    &BaseStrategy{name: "strategy1"},
		requiredHistory: 20,
	}
	strategy2 := &TestStrategyV2{
		BaseStrategy:    &BaseStrategy{name: "strategy2"},
		requiredHistory: 50,
	}

	engine.RegisterStrategy(strategy1)
	engine.RegisterStrategy(strategy2)

	// Execute
	maxHistory := engine.getMaxRequiredHistory()

	// Assertions
	assert.Equal(t, 50, maxHistory) // Should return the maximum required history
}

// TestStrategyEngineV2_EmptyStockGroups tests processing with empty stock groups
func TestStrategyEngineV2_ProcessStockGroups_EmptyGroups(t *testing.T) {
	// Setup
	config := &config.StrategyEngineV2Config{Enabled: true}
	mockRepo := &MockCandleRepository{}
	engine := NewStrategyEngineV2(config, mockRepo)

	// Register a test strategy
	testStrategy := &TestStrategyV2{
		BaseStrategy: &BaseStrategy{name: "test_strategy"},
	}
	engine.RegisterStrategy(testStrategy)

	// Empty stock groups
	stockGroups := []domain.StockGroup{}

	// Execute
	ctx := context.Background()
	currentTime := time.Now()
	results, err := engine.ProcessStockGroups(ctx, stockGroups, currentTime)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 0) // No results for empty groups
}

// TestStrategyEngineV2_NoStrategies tests processing with no registered strategies
func TestStrategyEngineV2_ProcessStockGroups_NoStrategies(t *testing.T) {
	// Setup
	config := &config.StrategyEngineV2Config{Enabled: true}
	mockRepo := &MockCandleRepository{}
	engine := NewStrategyEngineV2(config, mockRepo)

	// No strategies registered

	stockGroups := []domain.StockGroup{
		{
			ID: "group1",
			Stocks: []domain.Stock{
				{InstrumentKey: "NSE_EQ|INE005B01027"},
			},
		},
	}

	// Create mock candle data
	mockCandles := createMockCandles(10)
	mockRepo.On("GetCandlesByInstrumentKeyAndTimeRange",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockCandles, nil)

	// Execute
	ctx := context.Background()
	currentTime := time.Now()
	results, err := engine.ProcessStockGroups(ctx, stockGroups, currentTime)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 1) // Group exists but no strategy results

	groupResults := results["group1"]
	assert.Len(t, groupResults, 0) // No strategy results

	mockRepo.AssertExpectations(t)
}

// Helper functions

// createMockCandles creates mock candle data for testing
func createMockCandles(count int) []domain.Candle {
	candles := make([]domain.Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		candles[i] = domain.Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      100.0 + float64(i),
			High:      105.0 + float64(i),
			Low:       95.0 + float64(i),
			Close:     102.0 + float64(i),
			Volume:    1000 + int64(i*100),
		}
	}

	return candles
}

// TestStrategyV2 is a test implementation of StrategyV2
type TestStrategyV2 struct {
	*BaseStrategy
	requiredHistory int
}

func (s *TestStrategyV2) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
	// Add a test column
	testSeries := series.New(make([]float64, df.Nrow()), series.Float, "test_column")
	for i := 0; i < df.Nrow(); i++ {
		testSeries.Set(i, float64(i))
	}

	result := df.Mutate(testSeries)
	return &result, nil
}

func (s *TestStrategyV2) GetRequiredHistory() int {
	if s.requiredHistory == 0 {
		return 20 // Default
	}
	return s.requiredHistory
}

func (s *TestStrategyV2) Configure(config map[string]interface{}) error {
	return nil
}

func (s *TestStrategyV2) ValidateConfiguration() error {
	return nil
}

func (s *TestStrategyV2) GetEstimatedProcessingTime() time.Duration {
	return 10 * time.Millisecond
}

func (s *TestStrategyV2) GetMemoryRequirements() int64 {
	return 1024 * 1024 // 1MB
}
