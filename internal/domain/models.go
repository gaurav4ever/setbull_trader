package domain

import (
	"time"
)

// TradeSide represents whether the trade is a buy or sell
type TradeSide string

const (
	// Buy represents a buy trade
	Buy TradeSide = "BUY"
	// Sell represents a sell trade
	Sell TradeSide = "SELL"
)

// Stock represents a trading stock/security
type Stock struct {
	ID           string  `db:"id" json:"id"`
	Symbol       string  `db:"symbol" json:"symbol"`
	Name         string  `db:"name" json:"name"`
	CurrentPrice float64 `db:"current_price" json:"currentPrice"`
	IsSelected   bool    `db:"is_selected" json:"isSelected"`
}

// TradeParameters represents the configuration for a trade
type TradeParameters struct {
	ID                 string    `db:"id" json:"id"`
	StockID            string    `db:"stock_id" json:"stockId"`
	StartingPrice      float64   `db:"starting_price" json:"startingPrice"`
	StopLossPercentage float64   `db:"sl_percentage" json:"stopLossPercentage"`
	RiskAmount         float64   `db:"risk_amount" json:"riskAmount"`
	TradeSide          TradeSide `db:"trade_side" json:"tradeSide"`
}

// ExecutionLevel represents a price level for trade execution
type ExecutionLevel struct {
	Level       float64 `json:"level"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

// LevelEntry represents a single level in an execution plan
type LevelEntry struct {
	ID              string  `db:"id" json:"id"`
	ExecutionPlanID string  `db:"execution_plan_id" json:"executionPlanId"`
	FibLevel        float64 `db:"fib_level" json:"fibLevel"`
	Price           float64 `db:"price" json:"price"`
	Quantity        int     `db:"quantity" json:"quantity"`
	Description     string  `db:"description" json:"description"`
}

// ExecutionPlan represents a complete trading plan for a stock
type ExecutionPlan struct {
	ID            string           `db:"id" json:"id"`
	StockID       string           `db:"stock_id" json:"stockId"`
	ParametersID  string           `db:"parameters_id" json:"parametersId"`
	TotalQuantity int              `db:"total_quantity" json:"totalQuantity"`
	CreatedAt     time.Time        `db:"created_at" json:"createdAt"`
	Stock         *Stock           `db:"-" json:"stock,omitempty"`
	Parameters    *TradeParameters `db:"-" json:"parameters,omitempty"`
	LevelEntries  []LevelEntry     `db:"-" json:"levelEntries,omitempty"`
}

// OrderExecution represents the execution status of an order
type OrderExecution struct {
	ID              string    `db:"id" json:"id"`
	ExecutionPlanID string    `db:"execution_plan_id" json:"executionPlanId"`
	Status          string    `db:"status" json:"status"`
	ExecutedAt      time.Time `db:"executed_at" json:"executedAt"`
	ErrorMessage    string    `db:"error_message" json:"errorMessage"`
}

// OrderStatus constants
const (
	OrderStatusPending   = "PENDING"
	OrderStatusExecuting = "EXECUTING"
	OrderStatusCompleted = "COMPLETED"
	OrderStatusFailed    = "FAILED"
	OrderStatusCancelled = "CANCELLED"
)

// ValidationError represents an error during parameter validation
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
