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
	ID           string  `gorm:"column:id" json:"id"`                      // Changed from db to gorm
	Symbol       string  `gorm:"column:symbol" json:"symbol"`              // Changed from db to gorm
	Name         string  `gorm:"column:name" json:"name"`                  // Changed from db to gorm
	CurrentPrice float64 `gorm:"column:current_price" json:"currentPrice"` // Changed from db to gorm
	IsSelected   bool    `gorm:"column:is_selected" json:"isSelected"`     // Changed from db to gorm
}

// TradeParameters represents the configuration for a trade
type TradeParameters struct {
	ID                 string    `gorm:"column:id" json:"id"`                            // Changed from db to gorm
	StockID            string    `gorm:"column:stock_id" json:"stockId"`                 // Changed from db to gorm
	StartingPrice      float64   `gorm:"column:starting_price" json:"startingPrice"`     // Changed from db to gorm
	StopLossPercentage float64   `gorm:"column:sl_percentage" json:"stopLossPercentage"` // Changed from db to gorm
	RiskAmount         float64   `gorm:"column:risk_amount" json:"riskAmount"`           // Changed from db to gorm
	TradeSide          TradeSide `gorm:"column:trade_side" json:"tradeSide"`             // Changed from db to gorm
}

// ExecutionLevel represents a price level for trade execution
type ExecutionLevel struct {
	Level       float64 `json:"level"`       // No change
	Price       float64 `json:"price"`       // No change
	Description string  `json:"description"` // No change
}

// LevelEntry represents a single level in an execution plan
type LevelEntry struct {
	ID              string  `gorm:"column:id" json:"id"`                             // Changed from db to gorm
	ExecutionPlanID string  `gorm:"column:execution_plan_id" json:"executionPlanId"` // Changed from db to gorm
	FibLevel        float64 `gorm:"column:fib_level" json:"fibLevel"`                // Changed from db to gorm
	Price           float64 `gorm:"column:price" json:"price"`                       // Changed from db to gorm
	Quantity        int     `gorm:"column:quantity" json:"quantity"`                 // Changed from db to gorm
	Description     string  `gorm:"column:description" json:"description"`           // Changed from db to gorm
}

// ExecutionPlan represents a complete trading plan for a stock
type ExecutionPlan struct {
	ID            string           `gorm:"column:id" json:"id"`                        // Changed from db to gorm
	StockID       string           `gorm:"column:stock_id" json:"stockId"`             // Changed from db to gorm
	ParametersID  string           `gorm:"column:parameters_id" json:"parametersId"`   // Changed from db to gorm
	TotalQuantity int              `gorm:"column:total_quantity" json:"totalQuantity"` // Changed from db to gorm
	CreatedAt     time.Time        `gorm:"column:created_at" json:"createdAt"`         // Changed from db to gorm
	Stock         *Stock           `gorm:"-" json:"stock,omitempty"`                   // No change
	Parameters    *TradeParameters `gorm:"-" json:"parameters,omitempty"`              // No change
	LevelEntries  []LevelEntry     `gorm:"-" json:"levelEntries,omitempty"`            // No change
}

// OrderExecution represents the execution status of an order
type OrderExecution struct {
	ID              string    `gorm:"column:id" json:"id"`                             // Changed from db to gorm
	ExecutionPlanID string    `gorm:"column:execution_plan_id" json:"executionPlanId"` // Changed from db to gorm
	Status          string    `gorm:"column:status" json:"status"`                     // Changed from db to gorm
	ExecutedAt      time.Time `gorm:"column:executed_at" json:"executedAt"`            // Changed from db to gorm
	ErrorMessage    string    `gorm:"column:error_message" json:"errorMessage"`        // Changed from db to gorm
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
