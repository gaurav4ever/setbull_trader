package repository

import (
	"context"
	"time"

	"setbull_trader/internal/domain"
)

// StockRepository defines operations for managing stocks
type StockRepository interface {
	// Create creates a new stock
	Create(ctx context.Context, stock *domain.Stock) error

	// GetByID retrieves a stock by its ID
	GetByID(ctx context.Context, id string) (*domain.Stock, error)

	// GetBySymbol retrieves a stock by its symbol
	GetBySymbol(ctx context.Context, symbol string) (*domain.Stock, error)

	// GetBySecurityID retrieves a stock by its security ID
	GetBySecurityID(ctx context.Context, securityID string) (*domain.Stock, error)

	// GetAll retrieves all stocks
	GetAll(ctx context.Context) ([]*domain.Stock, error)

	// GetSelected retrieves all selected stocks
	GetSelected(ctx context.Context) ([]*domain.Stock, error)

	// Update updates a stock
	Update(ctx context.Context, stock *domain.Stock) error

	// ToggleSelection toggles the selection status of a stock
	ToggleSelection(ctx context.Context, id string, isSelected bool) error

	// Delete deletes a stock
	Delete(ctx context.Context, id string) error
}

// TradeParametersRepository defines operations for managing trade parameters
type TradeParametersRepository interface {
	// Create creates new trade parameters
	Create(ctx context.Context, params *domain.TradeParameters) error

	// GetByID retrieves trade parameters by their ID
	GetByID(ctx context.Context, id string) (*domain.TradeParameters, error)

	// GetByStockID retrieves trade parameters for a specific stock
	GetByStockID(ctx context.Context, stockID string) (*domain.TradeParameters, error)

	// Update updates trade parameters
	Update(ctx context.Context, params *domain.TradeParameters) error

	// Delete deletes trade parameters
	Delete(ctx context.Context, id string) error
}

// ExecutionPlanRepository defines operations for managing execution plans
type ExecutionPlanRepository interface {
	// Create creates a new execution plan
	Create(ctx context.Context, plan *domain.ExecutionPlan) error

	// GetByID retrieves an execution plan by its ID
	GetByID(ctx context.Context, id string) (*domain.ExecutionPlan, error)

	// GetByStockID retrieves the latest execution plan for a stock
	GetByStockID(ctx context.Context, stockID string) (*domain.ExecutionPlan, error)

	// GetAll retrieves all execution plans
	GetAll(ctx context.Context) ([]*domain.ExecutionPlan, error)

	// Delete deletes an execution plan
	Delete(ctx context.Context, id string) error
}

// LevelEntryRepository defines operations for managing level entries
type LevelEntryRepository interface {
	// CreateMany creates multiple level entries for an execution plan
	CreateMany(ctx context.Context, entries []domain.LevelEntry) error

	// GetByExecutionPlanID retrieves all level entries for an execution plan
	GetByExecutionPlanID(ctx context.Context, planID string) ([]domain.LevelEntry, error)

	// DeleteByExecutionPlanID deletes all level entries for an execution plan
	DeleteByExecutionPlanID(ctx context.Context, planID string) error
}

// OrderExecutionRepository defines operations for managing order executions
type OrderExecutionRepository interface {
	// Create creates a new order execution
	Create(ctx context.Context, execution *domain.OrderExecution) error

	// GetByID retrieves an order execution by its ID
	GetByID(ctx context.Context, id string) (*domain.OrderExecution, error)

	// GetByExecutionPlanID retrieves order executions for an execution plan
	GetByExecutionPlanID(ctx context.Context, planID string) ([]*domain.OrderExecution, error)

	// UpdateStatus updates the status of an order execution
	UpdateStatus(ctx context.Context, id string, status string, errorMessage string) error
}

// CandleRepository defines the interface for operations on candle data
type CandleRepository interface {
	// Store stores a single candle record
	Store(ctx context.Context, candle *domain.Candle) error

	// StoreBatch stores multiple candle records in a batch operation
	StoreBatch(ctx context.Context, candles []domain.Candle) (int, error)

	// FindByInstrumentKey retrieves all candles for a specific instrument
	FindByInstrumentKey(ctx context.Context, instrumentKey string) ([]domain.Candle, error)

	// FindByInstrumentAndInterval retrieves candles for an instrument with a specific interval
	FindByInstrumentAndInterval(ctx context.Context, instrumentKey, interval string) ([]domain.Candle, error)

	// FindByInstrumentAndTimeRange retrieves candles for an instrument within a time range
	FindByInstrumentAndTimeRange(
		ctx context.Context,
		instrumentKey string,
		interval string,
		fromTime,
		toTime time.Time,
	) ([]domain.Candle, error)

	// DeleteByInstrumentAndTimeRange deletes candles for an instrument within a time range
	DeleteByInstrumentAndTimeRange(
		ctx context.Context,
		instrumentKey string,
		interval string,
		fromTime,
		toTime time.Time,
	) (int, error)

	// CountByInstrumentAndTimeRange counts candles for an instrument within a time range
	CountByInstrumentAndTimeRange(
		ctx context.Context,
		instrumentKey string,
		interval string,
		fromTime,
		toTime time.Time,
	) (int, error)

	// DeleteOlderThan deletes candles older than a specified time
	DeleteOlderThan(ctx context.Context, olderThan time.Time) (int, error)

	// Core operations
	GetLatestCandle(ctx context.Context, instrumentKey, interval string) (*domain.Candle, error)
	// GetEarliestCandle retrieves the oldest candle for a specific instrument and interval
	GetEarliestCandle(ctx context.Context, instrumentKey string, interval string) (*domain.Candle, error)
	// GetCandleDateRange retrieves the earliest and latest timestamps for candles of a specific instrument and interval
	GetCandleDateRange(ctx context.Context, instrumentKey string, interval string) (earliest, latest time.Time, exists bool, err error)
	GetNDailyCandlesByTimeframe(ctx context.Context, instrumentKey string, interval string, n int) ([]domain.Candle, error)

	// Aggregation operations
	GetAggregated5MinCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error)
	GetAggregatedDailyCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error)

	GetDailyCandlesByTimeframe(ctx context.Context, instrumentKey string, startTime time.Time) ([]domain.Candle, error)

	// Optional: Store aggregated candles for future use
	StoreAggregatedCandles(ctx context.Context, candles []domain.CandleData) error

	// GetStocksWithExistingDailyCandles returns a list of instrument keys that already have daily candle data
	GetStocksWithExistingDailyCandles(ctx context.Context, startDate, endDate time.Time) ([]string, error)
}

// Candle5MinRepository defines operations for managing 5-minute candle data
type Candle5MinRepository interface {
	// Store stores a single 5-minute candle record
	Store(ctx context.Context, candle *domain.Candle5Min) error

	// StoreBatch stores multiple 5-minute candle records in a batch operation
	StoreBatch(ctx context.Context, candles []domain.Candle5Min) (int, error)

	// FindByInstrumentKey retrieves all 5-minute candles for a specific instrument
	FindByInstrumentKey(ctx context.Context, instrumentKey string) ([]domain.Candle5Min, error)

	// FindByInstrumentAndTimeRange retrieves 5-minute candles for an instrument within a time range
	FindByInstrumentAndTimeRange(
		ctx context.Context,
		instrumentKey string,
		fromTime,
		toTime time.Time,
	) ([]domain.Candle5Min, error)

	// DeleteByInstrumentAndTimeRange deletes 5-minute candles for an instrument within a time range
	DeleteByInstrumentAndTimeRange(
		ctx context.Context,
		instrumentKey string,
		fromTime,
		toTime time.Time,
	) (int, error)

	// CountByInstrumentAndTimeRange counts 5-minute candles for an instrument within a time range
	CountByInstrumentAndTimeRange(
		ctx context.Context,
		instrumentKey string,
		fromTime,
		toTime time.Time,
	) (int, error)

	// DeleteOlderThan deletes 5-minute candles older than a specified time
	DeleteOlderThan(ctx context.Context, olderThan time.Time) (int, error)

	// GetLatestCandle retrieves the most recent 5-minute candle for a specific instrument
	GetLatestCandle(ctx context.Context, instrumentKey string) (*domain.Candle5Min, error)

	// GetEarliestCandle retrieves the oldest 5-minute candle for a specific instrument
	GetEarliestCandle(ctx context.Context, instrumentKey string) (*domain.Candle5Min, error)

	// GetCandleDateRange retrieves the earliest and latest timestamps for 5-minute candles of a specific instrument
	GetCandleDateRange(ctx context.Context, instrumentKey string) (earliest, latest time.Time, exists bool, err error)

	// GetNLatestCandles retrieves the N most recent 5-minute candles for a specific instrument
	GetNLatestCandles(ctx context.Context, instrumentKey string, n int) ([]domain.Candle5Min, error)

	// UpdateCandlesInRangeCount updates the candles_in_range_count for the latest candle of a specific instrument
	UpdateCandlesInRangeCount(ctx context.Context, instrumentKey string, count int) error
}

// StockUniverseRepository defines the interface for stock universe operations
type StockUniverseRepository interface {
	Create(ctx context.Context, stock *domain.StockUniverse) (*domain.StockUniverse, error)
	BulkUpsert(ctx context.Context, stocks []domain.StockUniverse) (int, int, error)
	GetBySymbol(ctx context.Context, symbol string) (*domain.StockUniverse, error)
	GetAll(ctx context.Context, onlySelected bool, limit, offset int) ([]domain.StockUniverse, int64, error)
	ToggleSelection(ctx context.Context, symbol string, isSelected bool) (*domain.StockUniverse, error)
	DeleteBySymbol(ctx context.Context, symbol string) error
	// GetStocksByInstrumentKeys retrieves stocks by their instrument keys
	GetStocksByInstrumentKeys(ctx context.Context, instrumentKeys []string) ([]domain.StockUniverse, error)
}

// FilteredStockRepository defines operations for managing filtered stocks
type FilteredStockRepository interface {
	// Store stores a filtered stock record
	Store(ctx context.Context, record *domain.FilteredStockRecord) error

	// StoreBatch stores multiple filtered stock records
	StoreBatch(ctx context.Context, records []domain.FilteredStockRecord) error

	// GetBySymbol retrieves filtered stock records for a specific symbol
	GetBySymbol(ctx context.Context, symbol string) ([]domain.FilteredStockRecord, error)

	// GetByDate retrieves all filtered stocks for a specific date
	GetByDate(ctx context.Context, date time.Time) ([]domain.FilteredStockRecord, error)

	// GetByDateRange retrieves filtered stocks within a date range
	GetByDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.FilteredStockRecord, error)

	// DeleteOlderThan deletes filtered stock records older than the specified date
	DeleteOlderThan(ctx context.Context, date time.Time) (int, error)

	// GetLatestBySymbol retrieves the most recent filtered stock record for a symbol
	GetLatestBySymbol(ctx context.Context, symbol string) (*domain.FilteredStockRecord, error)

	// GetTop10FilteredStocks retrieves the top 10 filtered stocks
	GetTop10FilteredStocks(ctx context.Context) ([]domain.FilteredStockRecord, error)
}

// MasterDataProcessRepository defines operations for managing master data processes
type MasterDataProcessRepository interface {
	// Create creates a new master data process
	Create(ctx context.Context, processDate time.Time, numberOfPastDays int) (*domain.MasterDataProcess, error)

	// GetByDate retrieves a process by its date
	GetByDate(ctx context.Context, processDate time.Time) (*domain.MasterDataProcess, error)

	// GetByID retrieves a process by its ID
	GetByID(ctx context.Context, processID int64) (*domain.MasterDataProcess, error)

	// UpdateStatus updates the status of a process
	UpdateStatus(ctx context.Context, processID int64, status string) error

	// CompleteProcess marks a process as completed
	CompleteProcess(ctx context.Context, processID int64) error

	// CreateStep creates a new step for a process
	CreateStep(ctx context.Context, processID int64, stepNumber int, stepName string) error

	// GetStep retrieves a step by process ID and step number
	GetStep(ctx context.Context, processID int64, stepNumber int) (*domain.MasterDataProcessStep, error)

	// UpdateStepStatus updates the status of a step
	UpdateStepStatus(ctx context.Context, processID int64, stepNumber int, status string, errorMessage ...string) error

	// GetFilteredStocks retrieves filtered stocks for a specific date
	GetFilteredStocks(ctx context.Context, processDate time.Time) ([]domain.FilteredStockRecord, error)

	// GetProcessHistory retrieves recent process history
	GetProcessHistory(ctx context.Context, limit int) ([]domain.MasterDataProcess, error)
}
