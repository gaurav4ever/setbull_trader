package postgres

import (
	"context"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"time"

	"gorm.io/gorm"
)

// MasterDataProcessRepository implements repository.MasterDataProcessRepository using PostgreSQL
type MasterDataProcessRepository struct {
	db *gorm.DB
}

// NewMasterDataProcessRepository creates a new MasterDataProcessRepository
func NewMasterDataProcessRepository(db *gorm.DB) repository.MasterDataProcessRepository {
	return &MasterDataProcessRepository{db: db}
}

// Create creates a new master data process
func (r *MasterDataProcessRepository) Create(ctx context.Context, processDate time.Time, numberOfPastDays int) (*domain.MasterDataProcess, error) {
	// Check if a process already exists for this date
	var existingProcess domain.MasterDataProcess
	err := r.db.WithContext(ctx).
		Where("process_date = ? AND active = ?", processDate.Format("2006-01-02"), true).
		First(&existingProcess).Error

	if err == nil {
		// Process already exists for this date
		return &existingProcess, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check for existing process: %w", err)
	}

	process := &domain.MasterDataProcess{
		ProcessDate:      processDate,
		NumberOfPastDays: numberOfPastDays,
		Status:           domain.ProcessStatusRunning,
		Active:           true,
	}

	// Use a transaction to create process and its steps
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create the process
	if err := tx.Create(process).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create process: %w", err)
	}

	// Create the three steps
	steps := []domain.MasterDataProcessStep{
		{
			ProcessID:  process.ID,
			StepNumber: 1,
			StepName:   domain.StepNameDailyIngestion,
			Status:     domain.StepStatusPending,
			Active:     true,
		},
		{
			ProcessID:  process.ID,
			StepNumber: 2,
			StepName:   domain.StepNameFilterPipeline,
			Status:     domain.StepStatusPending,
			Active:     true,
		},
		{
			ProcessID:  process.ID,
			StepNumber: 3,
			StepName:   domain.StepNameMinuteIngestion,
			Status:     domain.StepStatusPending,
			Active:     true,
		},
	}

	for _, step := range steps {
		if err := tx.Create(&step).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create step %d: %w", step.StepNumber, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return process, nil
}

// GetByDate retrieves a process by its date
func (r *MasterDataProcessRepository) GetByDate(ctx context.Context, processDate time.Time) (*domain.MasterDataProcess, error) {
	var process domain.MasterDataProcess

	err := r.db.WithContext(ctx).
		Where("process_date = ? AND active = ?", processDate.Format("2006-01-02"), true).
		Preload("Steps", "active = ?", true).
		First(&process).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get process by date: %w", err)
	}

	return &process, nil
}

// GetByID retrieves a process by its ID
func (r *MasterDataProcessRepository) GetByID(ctx context.Context, processID int64) (*domain.MasterDataProcess, error) {
	var process domain.MasterDataProcess

	err := r.db.WithContext(ctx).
		Where("id = ? AND active = ?", processID, true).
		Preload("Steps", "active = ?", true).
		First(&process).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get process by ID: %w", err)
	}

	return &process, nil
}

// UpdateStatus updates the status of a process
func (r *MasterDataProcessRepository) UpdateStatus(ctx context.Context, processID int64, status string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == domain.ProcessStatusCompleted {
		now := time.Now()
		updates["completed_at"] = &now
	}

	result := r.db.WithContext(ctx).
		Model(&domain.MasterDataProcess{}).
		Where("id = ? AND active = ?", processID, true).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update process status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("process with ID %d not found or not active", processID)
	}

	return nil
}

// CompleteProcess marks a process as completed
func (r *MasterDataProcessRepository) CompleteProcess(ctx context.Context, processID int64) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&domain.MasterDataProcess{}).
		Where("id = ? AND active = ?", processID, true).
		Updates(map[string]interface{}{
			"status":       domain.ProcessStatusCompleted,
			"completed_at": &now,
			"updated_at":   now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to complete process: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("process with ID %d not found or not active", processID)
	}

	return nil
}

// CreateStep creates a new step for a process
func (r *MasterDataProcessRepository) CreateStep(ctx context.Context, processID int64, stepNumber int, stepName string) error {
	step := &domain.MasterDataProcessStep{
		ProcessID:  processID,
		StepNumber: stepNumber,
		StepName:   stepName,
		Status:     domain.StepStatusPending,
		Active:     true,
	}

	if err := r.db.WithContext(ctx).Create(step).Error; err != nil {
		return fmt.Errorf("failed to create step: %w", err)
	}

	return nil
}

// GetStep retrieves a step by process ID and step number
func (r *MasterDataProcessRepository) GetStep(ctx context.Context, processID int64, stepNumber int) (*domain.MasterDataProcessStep, error) {
	var step domain.MasterDataProcessStep

	err := r.db.WithContext(ctx).
		Where("process_id = ? AND step_number = ? AND active = ?", processID, stepNumber, true).
		First(&step).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get step: %w", err)
	}

	return &step, nil
}

// UpdateStepStatus updates the status of a step
func (r *MasterDataProcessRepository) UpdateStepStatus(ctx context.Context, processID int64, stepNumber int, status string, errorMessage ...string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	// Set started_at if step is starting
	if status == domain.StepStatusRunning {
		now := time.Now()
		updates["started_at"] = &now
	}

	// Set completed_at if step is completed
	if status == domain.StepStatusCompleted {
		now := time.Now()
		updates["completed_at"] = &now
	}

	// Set error message if provided
	if len(errorMessage) > 0 && errorMessage[0] != "" {
		updates["error_message"] = &errorMessage[0]
	}

	result := r.db.WithContext(ctx).
		Model(&domain.MasterDataProcessStep{}).
		Where("process_id = ? AND step_number = ? AND active = ?", processID, stepNumber, true).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update step status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("step with process ID %d and step number %d not found or not active", processID, stepNumber)
	}

	return nil
}

// GetFilteredStocks retrieves filtered stocks for a specific date
func (r *MasterDataProcessRepository) GetFilteredStocks(ctx context.Context, processDate time.Time) ([]domain.FilteredStockRecord, error) {
	var stocks []domain.FilteredStockRecord

	err := r.db.WithContext(ctx).
		Where("filter_date = ?", processDate.Format("2006-01-02")).
		Find(&stocks).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get filtered stocks: %w", err)
	}

	return stocks, nil
}

// GetProcessHistory retrieves recent process history
func (r *MasterDataProcessRepository) GetProcessHistory(ctx context.Context, limit int) ([]domain.MasterDataProcess, error) {
	var processes []domain.MasterDataProcess

	err := r.db.WithContext(ctx).
		Where("active = ?", true).
		Preload("Steps", "active = ?", true).
		Order("created_at DESC").
		Limit(limit).
		Find(&processes).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get process history: %w", err)
	}

	return processes, nil
}
