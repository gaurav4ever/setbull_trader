package domain

import (
	"time"
)

type TradeSide string

const (
	Buy  TradeSide = "BUY"
	Sell TradeSide = "SELL"
)

type Stock struct {
	ID           string    `gorm:"column:id" json:"id"`
	Symbol       string    `gorm:"column:symbol" json:"symbol"`
	Name         string    `gorm:"column:name" json:"name"`
	SecurityID   string    `gorm:"column:security_id" json:"securityId"`
	CurrentPrice float64   `gorm:"column:current_price" json:"currentPrice"`
	IsSelected   bool      `gorm:"column:is_selected" json:"isSelected"`
	Active       bool      `gorm:"column:active;default:1;index:idx_active"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`

	// Non-persisted fields for API responses
	Parameters    *TradeParameters `gorm:"-" json:"parameters,omitempty"`
	ExecutionPlan *ExecutionPlan   `gorm:"-" json:"executionPlan,omitempty"`
}

type TradeParameters struct {
	ID                 string    `gorm:"column:id" json:"id"`
	StockID            string    `gorm:"column:stock_id" json:"stockId"`
	StartingPrice      float64   `gorm:"column:starting_price" json:"startingPrice"`
	StopLossPercentage float64   `gorm:"column:sl_percentage" json:"stopLossPercentage"`
	RiskAmount         float64   `gorm:"column:risk_amount" json:"riskAmount"`
	TradeSide          TradeSide `gorm:"column:trade_side" json:"tradeSide"`
	Active             bool      `gorm:"column:active;default:1;index:idx_active"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}
type ExecutionLevel struct {
	Level       float64   `json:"level"`
	Price       float64   `json:"price"`
	Description string    `json:"description"`
	Active      bool      `gorm:"column:active;default:1;index:idx_active"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

type LevelEntry struct {
	ID              string    `gorm:"column:id" json:"id"`
	ExecutionPlanID string    `gorm:"column:execution_plan_id" json:"executionPlanId"`
	FibLevel        float64   `gorm:"column:fib_level" json:"fibLevel"`
	Price           float64   `gorm:"column:price" json:"price"`
	Quantity        int       `gorm:"column:quantity" json:"quantity"`
	Description     string    `gorm:"column:description" json:"description"`
	Active          bool      `gorm:"column:active;default:1;index:idx_active"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}
type ExecutionPlan struct {
	ID            string           `gorm:"column:id" json:"id"`
	StockID       string           `gorm:"column:stock_id" json:"stockId"`
	ParametersID  string           `gorm:"column:parameters_id" json:"parametersId"`
	TotalQuantity int              `gorm:"column:total_quantity" json:"totalQuantity"`
	CreatedAt     time.Time        `gorm:"column:created_at" json:"createdAt"`
	Stock         *Stock           `gorm:"-" json:"stock,omitempty"`
	Parameters    *TradeParameters `gorm:"-" json:"parameters,omitempty"`
	LevelEntries  []LevelEntry     `gorm:"-" json:"levelEntries,omitempty"`
	Active        bool             `gorm:"column:active;default:1;index:idx_active"`
	UpdatedAt     time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}
type OrderExecution struct {
	ID              string    `gorm:"column:id" json:"id"`
	ExecutionPlanID string    `gorm:"column:execution_plan_id" json:"executionPlanId"`
	Status          string    `gorm:"column:status" json:"status"`
	ExecutedAt      time.Time `gorm:"column:executed_at" json:"executedAt"`
	ErrorMessage    string    `gorm:"column:error_message" json:"errorMessage"`
	Active          bool      `gorm:"column:active;default:1;index:idx_active"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

// ExecutionResults contains all results from an execution attempt
type ExecutionResults struct {
	ExecutionID string
	StockSymbol string
	Results     []OrderExecutionResult
	Success     bool
}

type OrderExecutionResult struct {
	LevelDescription string
	OrderID          string
	OrderStatus      string
	Success          bool
	Error            string
}

const (
	OrderStatusPending   = "PENDING"
	OrderStatusExecuting = "EXECUTING"
	OrderStatusCompleted = "COMPLETED"
	OrderStatusFailed    = "FAILED"
	OrderStatusCancelled = "CANCELLED"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// BatchStoreHistoricalDataRequest represents the request body for storing historical data
type BatchStoreHistoricalDataRequest struct {
	InstrumentKeys []string `json:"instrumentKeys" validate:"required,min=1"`
	Interval       string   `json:"interval" validate:"required,oneof=1minute 5minute 30minute day week month"`
	FromDate       string   `json:"fromDate" validate:"required,datetime=2006-01-02"`
	ToDate         string   `json:"toDate" validate:"required,datetime=2006-01-02"`
}

// BatchStoreHistoricalDataResponse represents the response for batch storing historical data
type BatchStoreHistoricalDataResponse struct {
	Status string                 `json:"status"`
	Data   BatchProcessResultData `json:"data"`
}

// BatchProcessResultData contains the summary of batch processing
type BatchProcessResultData struct {
	ProcessedItems  int                   `json:"processedItems"`
	SuccessfulItems int                   `json:"successfulItems"`
	FailedItems     int                   `json:"failedItems"`
	Details         []InstrumentProcessed `json:"details"`
}

// InstrumentProcessed contains details about the processing of a single instrument
type InstrumentProcessed struct {
	InstrumentKey string `json:"instrumentKey"`
	Status        string `json:"status"`
	RecordsStored int    `json:"recordsStored"`
	Message       string `json:"message"`
}

// HistoricalCandleRequest represents the request parameters for Upstox historical candle API
type HistoricalCandleRequest struct {
	InstrumentKey string
	Interval      string
	ToDate        string
	FromDate      string
}

// HistoricalCandleResponse represents the response from Upstox historical candle API
type HistoricalCandleResponse struct {
	Status string               `json:"status"`
	Data   HistoricalCandleData `json:"data"`
}

// HistoricalCandleData contains the candle data
type HistoricalCandleData struct {
	Candles [][]interface{} `json:"candles"`
}

// CandleData represents a processed candle data point
type CandleData struct {
	InstrumentKey string
	Timestamp     time.Time
	Open          float64
	High          float64
	Low           float64
	Close         float64
	Volume        int64
	OpenInterest  int64
	Interval      string
}

// ProcessingError represents an error that occurred during processing
type ProcessingError struct {
	InstrumentKey string
	ErrorType     string
	Message       string
	RawError      error
}

// ProcessingResult represents the result of processing an instrument
type ProcessingResult struct {
	InstrumentKey string
	Success       bool
	RecordsStored int
	Error         *ProcessingError
}
