package repository

import (
	"context"

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
