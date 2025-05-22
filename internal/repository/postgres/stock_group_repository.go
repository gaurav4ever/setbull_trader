package postgres

import (
	"context"
	"errors"
	"setbull_trader/internal/domain"

	"gorm.io/gorm"
)

type StockGroupRepository struct {
	db *gorm.DB
}

func NewStockGroupRepository(db *gorm.DB) *StockGroupRepository {
	return &StockGroupRepository{db: db}
}

func (r *StockGroupRepository) Create(ctx context.Context, group *domain.StockGroup) error {
	return r.db.WithContext(ctx).Create(group).Error
}

func (r *StockGroupRepository) GetByID(ctx context.Context, id string) (*domain.StockGroup, error) {
	var group domain.StockGroup
	err := r.db.WithContext(ctx).Preload("Stocks").First(&group, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &group, err
}

func (r *StockGroupRepository) List(ctx context.Context, entryType string, status domain.StockGroupStatus) ([]domain.StockGroup, error) {
	var groups []domain.StockGroup
	query := r.db.WithContext(ctx).Preload("Stocks")
	if entryType != "" {
		query = query.Where("entry_type = ?", entryType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&groups).Error
	return groups, err
}

func (r *StockGroupRepository) Update(ctx context.Context, group *domain.StockGroup) error {
	return r.db.WithContext(ctx).Save(group).Error
}

func (r *StockGroupRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.StockGroup{}, "id = ?", id).Error
}

func (r *StockGroupRepository) GetActiveOrExecutingGroup(ctx context.Context) (*domain.StockGroup, error) {
	var group domain.StockGroup
	err := r.db.WithContext(ctx).Preload("Stocks").Where("status IN ?", []domain.StockGroupStatus{domain.GroupPending, domain.GroupExecuting}).First(&group).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &group, err
}

func (r *StockGroupRepository) GetByInstrumentKeyAndEntryType(ctx context.Context, instrumentKey string, entryType string) (*domain.StockGroup, error) {
	var group domain.StockGroup
	err := r.db.WithContext(ctx).Preload("Stocks").Where("instrument_key = ? AND entry_type = ?", instrumentKey, entryType).First(&group).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &group, err
}
