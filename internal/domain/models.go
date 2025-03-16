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
	CurrentPrice float64   `gorm:"column:current_price" json:"currentPrice"`
	IsSelected   bool      `gorm:"column:is_selected" json:"isSelected"`
	Active       bool      `gorm:"column:active;default:1;index:idx_active"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
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
