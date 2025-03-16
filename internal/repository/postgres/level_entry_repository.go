package postgres

import (
	"context"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LevelEntryRepository implements repository.LevelEntryRepository using PostgreSQL
type LevelEntryRepository struct {
	db *gorm.DB
}

// NewLevelEntryRepository creates a new LevelEntryRepository
func NewLevelEntryRepository(db *gorm.DB) repository.LevelEntryRepository {
	return &LevelEntryRepository{db: db}
}

// CreateMany creates multiple level entries for an execution plan
func (r *LevelEntryRepository) CreateMany(ctx context.Context, entries []domain.LevelEntry) error {
	for i := range entries {
		if entries[i].ID == "" {
			entries[i].ID = uuid.New().String()
			entries[i].CreatedAt = time.Now()
			entries[i].UpdatedAt = time.Now()
		}
	}

	return r.db.WithContext(ctx).Create(&entries).Error
}

// GetByExecutionPlanID retrieves all level entries for an execution plan
func (r *LevelEntryRepository) GetByExecutionPlanID(ctx context.Context, planID string) ([]domain.LevelEntry, error) {
	var entries []domain.LevelEntry
	err := r.db.WithContext(ctx).Where("execution_plan_id = ? AND active = 1", planID).Order("fib_level").Find(&entries).Error
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// DeleteByExecutionPlanID deletes all level entries for an execution plan
func (r *LevelEntryRepository) DeleteByExecutionPlanID(ctx context.Context, planID string) error {
	return r.db.WithContext(ctx).Where("execution_plan_id = ? AND active = 1", planID).Update("active", false).Error
}
