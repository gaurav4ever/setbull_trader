package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"gorm.io/gorm"
)

// StrategyParametersRecord represents the database record for strategy parameters
type StrategyParametersRecord struct {
	ID             string          `gorm:"primaryKey;type:varchar(36)"`
	StockID        string          `gorm:"not null;type:varchar(36);index:idx_stock_strategy,priority:1"`
	StrategyName   string          `gorm:"not null;type:varchar(100);index:idx_stock_strategy,priority:2"`
	Timestamp      time.Time       `gorm:"not null;index:idx_timestamp"`
	Parameters     json.RawMessage `gorm:"type:json;not null"`
	StateData      json.RawMessage `gorm:"type:json"`
	Metadata       json.RawMessage `gorm:"type:json"`
	Version        int             `gorm:"default:1"`
	IsValid        bool            `gorm:"default:true;index:idx_valid_sync,priority:1"`
	IsSynchronized bool            `gorm:"default:false;index:idx_valid_sync,priority:2"`
	CreatedAt      time.Time       `gorm:"autoCreateTime"`
	UpdatedAt      time.Time       `gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (StrategyParametersRecord) TableName() string {
	return "strategy_parameters_v2"
}

// StrategyParametersRepositoryV2 implements the repository.StrategyParametersRepository interface
type StrategyParametersRepositoryV2 struct {
	db *gorm.DB
}

// NewStrategyParametersRepositoryV2 creates a new instance of StrategyParametersRepositoryV2
func NewStrategyParametersRepositoryV2(db *gorm.DB) repository.StrategyParametersRepository {
	return &StrategyParametersRepositoryV2{
		db: db,
	}
}

// SaveParameters saves strategy parameters to the database
func (r *StrategyParametersRepositoryV2) SaveParameters(stockID, strategyName string, timestamp time.Time, params *domain.StrategyParameters) error {
	// Convert parameters to JSON
	parametersJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Convert state data to JSON
	stateDataJSON, err := json.Marshal(params.StrategyState)
	if err != nil {
		return fmt.Errorf("failed to marshal state data: %w", err)
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	recordID := fmt.Sprintf("%s_%s_%d", stockID, strategyName, timestamp.Unix())
	record := StrategyParametersRecord{
		ID:             recordID,
		StockID:        stockID,
		StrategyName:   strategyName,
		Timestamp:      timestamp,
		Parameters:     parametersJSON,
		StateData:      stateDataJSON,
		Metadata:       metadataJSON,
		Version:        1,
		IsValid:        true,
		IsSynchronized: false,
	}

	result := r.db.WithContext(context.Background()).Save(&record)
	if result.Error != nil {
		return fmt.Errorf("failed to save strategy parameters: %w", result.Error)
	}

	return nil
}

// GetParameters retrieves strategy parameters for a specific stock, strategy, and timestamp
func (r *StrategyParametersRepositoryV2) GetParameters(stockID, strategyName string, timestamp time.Time) (*domain.StrategyParameters, error) {
	var record StrategyParametersRecord

	result := r.db.WithContext(context.Background()).
		Where("stock_id = ? AND strategy_name = ? AND timestamp = ? AND is_valid = ?",
			stockID, strategyName, timestamp, true).
		Order("version DESC").
		First(&record)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // Return nil if no parameters found
		}
		return nil, fmt.Errorf("failed to get strategy parameters: %w", result.Error)
	}

	// Unmarshal parameters
	var params domain.StrategyParameters
	if err := json.Unmarshal(record.Parameters, &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	// Unmarshal state data
	if record.StateData != nil {
		if err := json.Unmarshal(record.StateData, &params.StrategyState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state data: %w", err)
		}
	}

	// Unmarshal metadata
	if record.Metadata != nil {
		if err := json.Unmarshal(record.Metadata, &params.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &params, nil
}

// GetParametersRange retrieves strategy parameters within a time range
func (r *StrategyParametersRepositoryV2) GetParametersRange(stockID, strategyName string, startTime, endTime time.Time) ([]*domain.StrategyParametersRecord, error) {
	var records []StrategyParametersRecord

	result := r.db.WithContext(context.Background()).
		Where("stock_id = ? AND strategy_name = ? AND timestamp BETWEEN ? AND ? AND is_valid = ?",
			stockID, strategyName, startTime, endTime, true).
		Order("timestamp ASC, version DESC").
		Find(&records)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get strategy parameters range: %w", result.Error)
	}

	resultRecords := make([]*domain.StrategyParametersRecord, len(records))
	for i, record := range records {
		// Unmarshal parameters
		var params domain.StrategyParameters
		if err := json.Unmarshal(record.Parameters, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters for record %d: %w", i, err)
		}

		// Unmarshal state data
		if record.StateData != nil {
			if err := json.Unmarshal(record.StateData, &params.StrategyState); err != nil {
				return nil, fmt.Errorf("failed to unmarshal state data for record %d: %w", i, err)
			}
		}

		// Unmarshal metadata
		if record.Metadata != nil {
			if err := json.Unmarshal(record.Metadata, &params.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata for record %d: %w", i, err)
			}
		}

		resultRecords[i] = &domain.StrategyParametersRecord{
			ID:                 0, // Will be set if needed
			StockID:            record.StockID,
			StrategyName:       record.StrategyName,
			CandleTimestamp:    record.Timestamp,
			CreatedAt:          record.CreatedAt,
			UpdatedAt:          record.UpdatedAt,
			Active:             record.IsValid,
			StrategyParameters: params,
		}
	}

	return resultRecords, nil
}

// GetLatestParameters retrieves the latest strategy parameters for a stock and strategy
func (r *StrategyParametersRepositoryV2) GetLatestParameters(stockID, strategyName string) (*domain.StrategyParameters, error) {
	var record StrategyParametersRecord

	result := r.db.WithContext(context.Background()).
		Where("stock_id = ? AND strategy_name = ? AND is_valid = ?",
			stockID, strategyName, true).
		Order("timestamp DESC, version DESC").
		First(&record)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // Return nil if no parameters found
		}
		return nil, fmt.Errorf("failed to get latest strategy parameters: %w", result.Error)
	}

	// Unmarshal parameters
	var params domain.StrategyParameters
	if err := json.Unmarshal(record.Parameters, &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	// Unmarshal state data
	if record.StateData != nil {
		if err := json.Unmarshal(record.StateData, &params.StrategyState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state data: %w", err)
		}
	}

	// Unmarshal metadata
	if record.Metadata != nil {
		if err := json.Unmarshal(record.Metadata, &params.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &params, nil
}

// UpdateParameters updates parameters for a specific record
func (r *StrategyParametersRepositoryV2) UpdateParameters(id int64, params *domain.StrategyParameters) error {
	// Convert parameters to JSON
	parametersJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Convert state data to JSON
	stateDataJSON, err := json.Marshal(params.StrategyState)
	if err != nil {
		return fmt.Errorf("failed to marshal state data: %w", err)
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result := r.db.WithContext(context.Background()).
		Model(&StrategyParametersRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"parameters": parametersJSON,
			"state_data": stateDataJSON,
			"metadata":   metadataJSON,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update strategy parameters: %w", result.Error)
	}

	return nil
}

// DeleteParameters deletes parameters for a specific stock, strategy, and candle
func (r *StrategyParametersRepositoryV2) DeleteParameters(stockID, strategyName string, timestamp time.Time) error {
	result := r.db.WithContext(context.Background()).
		Where("stock_id = ? AND strategy_name = ? AND timestamp = ?",
			stockID, strategyName, timestamp).
		Delete(&StrategyParametersRecord{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete strategy parameters: %w", result.Error)
	}

	return nil
}

// CleanupOldParameters deletes old parameters for data retention
func (r *StrategyParametersRepositoryV2) CleanupOldParameters(beforeTime time.Time) error {
	result := r.db.WithContext(context.Background()).
		Where("timestamp < ?", beforeTime).
		Delete(&StrategyParametersRecord{})

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old strategy parameters: %w", result.Error)
	}

	return nil
}
