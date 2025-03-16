package postgres

import (
	"context"
	"errors"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderExecutionRepository implements repository.OrderExecutionRepository using PostgreSQL
type OrderExecutionRepository struct {
	db *gorm.DB
}

// NewOrderExecutionRepository creates a new OrderExecutionRepository
func NewOrderExecutionRepository(db *gorm.DB) repository.OrderExecutionRepository {
	return &OrderExecutionRepository{db: db}
}

// Create creates a new order execution
func (r *OrderExecutionRepository) Create(ctx context.Context, execution *domain.OrderExecution) error {
	if execution.ID == "" {
		execution.ID = uuid.New().String()
	}
	execution.CreatedAt = time.Now()
	execution.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).Create(execution).Error
}

// GetByID retrieves an order execution by its ID
func (r *OrderExecutionRepository) GetByID(ctx context.Context, id string) (*domain.OrderExecution, error) {
	var execution domain.OrderExecution
	err := r.db.WithContext(ctx).First(&execution, "id = ? AND active = 1", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &execution, nil
}

// GetByExecutionPlanID retrieves order executions for an execution plan
func (r *OrderExecutionRepository) GetByExecutionPlanID(ctx context.Context, planID string) ([]*domain.OrderExecution, error) {
	var executions []*domain.OrderExecution
	err := r.db.WithContext(ctx).Where("execution_plan_id = ? AND active = 1", planID).
		Order("executed_at DESC").Find(&executions).Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

// UpdateStatus updates the status of an order execution
func (r *OrderExecutionRepository) UpdateStatus(ctx context.Context, id string, status string, errorMessage string) error {
	return r.db.WithContext(ctx).Model(&domain.OrderExecution{}).Where("id = ? AND active = 1", id).Updates(map[string]interface{}{
		"status":        status,
		"error_message": errorMessage,
	}).Error
}
