package postgres

import (
	"context"
	"errors"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExecutionPlanRepository implements repository.ExecutionPlanRepository using PostgreSQL
type ExecutionPlanRepository struct {
	db *gorm.DB
}

// NewExecutionPlanRepository creates a new ExecutionPlanRepository
func NewExecutionPlanRepository(db *gorm.DB) repository.ExecutionPlanRepository {
	return &ExecutionPlanRepository{db: db}
}

// Create creates a new execution plan
func (r *ExecutionPlanRepository) Create(ctx context.Context, plan *domain.ExecutionPlan) error {
	if plan.ID == "" {
		plan.ID = uuid.New().String()
	}

	return r.db.WithContext(ctx).Create(plan).Error
}

// GetByID retrieves an execution plan by its ID
func (r *ExecutionPlanRepository) GetByID(ctx context.Context, id string) (*domain.ExecutionPlan, error) {
	var plan domain.ExecutionPlan
	err := r.db.WithContext(ctx).First(&plan, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &plan, nil
}

// GetByStockID retrieves the latest execution plan for a stock
func (r *ExecutionPlanRepository) GetByStockID(ctx context.Context, stockID string) (*domain.ExecutionPlan, error) {
	var plan domain.ExecutionPlan
	err := r.db.WithContext(ctx).Where("stock_id = ?", stockID).Order("created_at DESC").First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &plan, nil
}

// GetAll retrieves all execution plans
func (r *ExecutionPlanRepository) GetAll(ctx context.Context) ([]*domain.ExecutionPlan, error) {
	var plans []*domain.ExecutionPlan
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&plans).Error
	if err != nil {
		return nil, err
	}

	return plans, nil
}

// Delete deletes an execution plan
func (r *ExecutionPlanRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.ExecutionPlan{}, id).Error
}
