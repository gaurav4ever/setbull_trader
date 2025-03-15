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

// OrderExecutionRepository implements repository.OrderExecutionRepository using PostgreSQL
type OrderExecutionRepository struct {
	db *sqlx.DB
}

// NewOrderExecutionRepository creates a new OrderExecutionRepository
func NewOrderExecutionRepository(db *sqlx.DB) repository.OrderExecutionRepository {
	return &OrderExecutionRepository{db: db}
}

// Create creates a new order execution
func (r *OrderExecutionRepository) Create(ctx context.Context, execution *domain.OrderExecution) error {
	if execution.ID == "" {
		execution.ID = uuid.New().String()
	}

	query := `
		INSERT INTO order_executions (id, execution_plan_id, status, executed_at, error_message)
		VALUES ($1, $2, $3, NOW(), $4)
		RETURNING executed_at
	`

	err := r.db.QueryRowContext(ctx, query,
		execution.ID,
		execution.ExecutionPlanID,
		execution.Status,
		execution.ErrorMessage,
	).Scan(&execution.ExecutedAt)

	return err
}

// GetByID retrieves an order execution by its ID
func (r *OrderExecutionRepository) GetByID(ctx context.Context, id string) (*domain.OrderExecution, error) {
	var execution domain.OrderExecution

	query := `
		SELECT id, execution_plan_id, status, executed_at, error_message
		FROM order_executions
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &execution, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}

	return &execution, nil
}

// GetByExecutionPlanID retrieves order executions for an execution plan
func (r *OrderExecutionRepository) GetByExecutionPlanID(ctx context.Context, planID string) ([]*domain.OrderExecution, error) {
	var executions []*domain.OrderExecution

	query := `
		SELECT id, execution_plan_id, status, executed_at, error_message
		FROM order_executions
		WHERE execution_plan_id = $1
		ORDER BY executed_at DESC
	`

	err := r.db.SelectContext(ctx, &executions, query, planID)
	if err != nil {
		return nil, err
	}

	return executions, nil
}

// UpdateStatus updates the status of an order execution
func (r *OrderExecutionRepository) UpdateStatus(ctx context.Context, id string, status string, errorMessage string) error {
	query := `
		UPDATE order_executions
		SET status = $1, error_message = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, errorMessage, id)
	return err
}
