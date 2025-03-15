package postgres

import (
	"context"
	"database/sql"
	"errors"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ExecutionPlanRepository implements repository.ExecutionPlanRepository using PostgreSQL
type ExecutionPlanRepository struct {
	db *sqlx.DB
}

// NewExecutionPlanRepository creates a new ExecutionPlanRepository
func NewExecutionPlanRepository(db *sqlx.DB) repository.ExecutionPlanRepository {
	return &ExecutionPlanRepository{db: db}
}

// Create creates a new execution plan
func (r *ExecutionPlanRepository) Create(ctx context.Context, plan *domain.ExecutionPlan) error {
	if plan.ID == "" {
		plan.ID = uuid.New().String()
	}

	query := `
		INSERT INTO execution_plans (id, stock_id, parameters_id, total_quantity, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		plan.ID,
		plan.StockID,
		plan.ParametersID,
		plan.TotalQuantity,
	).Scan(&plan.CreatedAt)

	return err
}

// GetByID retrieves an execution plan by its ID
func (r *ExecutionPlanRepository) GetByID(ctx context.Context, id string) (*domain.ExecutionPlan, error) {
	var plan domain.ExecutionPlan

	query := `
		SELECT id, stock_id, parameters_id, total_quantity, created_at
		FROM execution_plans
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &plan, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}

	return &plan, nil
}

// GetByStockID retrieves the latest execution plan for a stock
func (r *ExecutionPlanRepository) GetByStockID(ctx context.Context, stockID string) (*domain.ExecutionPlan, error) {
	var plan domain.ExecutionPlan

	query := `
		SELECT id, stock_id, parameters_id, total_quantity, created_at
		FROM execution_plans
		WHERE stock_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &plan, query, stockID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}

	return &plan, nil
}

// GetAll retrieves all execution plans
func (r *ExecutionPlanRepository) GetAll(ctx context.Context) ([]*domain.ExecutionPlan, error) {
	var plans []*domain.ExecutionPlan

	query := `
		SELECT id, stock_id, parameters_id, total_quantity, created_at
		FROM execution_plans
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &plans, query)
	if err != nil {
		return nil, err
	}

	return plans, nil
}

// Delete deletes an execution plan
func (r *ExecutionPlanRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM execution_plans WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
