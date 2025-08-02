package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/log"

	"gorm.io/gorm"
)

// StrategyResultPersistence handles database operations for V2 strategy results
type StrategyResultPersistence struct {
	db     *gorm.DB
	config *config.StrategyEngineV2Config
}

// StrategyResultRecord represents a strategy result record in the database
type StrategyResultRecord struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	StockGroupID    string    `gorm:"type:varchar(36);not null;index" json:"stock_group_id"`
	InstrumentKey   string    `gorm:"type:varchar(50);not null;index" json:"instrument_key"`
	StrategyName    string    `gorm:"type:varchar(100);not null;index" json:"strategy_name"`
	StrategyVersion string    `gorm:"type:varchar(20);not null" json:"strategy_version"`
	ProcessingTime  int64     `gorm:"type:bigint;not null" json:"processing_time_ms"`
	MemoryUsed      int64     `gorm:"type:bigint;not null" json:"memory_used_bytes"`
	RowsProcessed   int       `gorm:"type:int;not null" json:"rows_processed"`
	ColumnsAdded    string    `gorm:"type:text" json:"columns_added"`   // JSON array
	ResultData      string    `gorm:"type:longtext" json:"result_data"` // JSON object
	Error           string    `gorm:"type:text" json:"error,omitempty"`
	ProcessedAt     time.Time `gorm:"type:timestamp;not null;index" json:"processed_at"`
	CreatedAt       time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName returns the table name for the strategy result record
func (StrategyResultRecord) TableName() string {
	return "strategy_results_v2"
}

// NewStrategyResultPersistence creates a new persistence handler
func NewStrategyResultPersistence(db *gorm.DB, config *config.StrategyEngineV2Config) *StrategyResultPersistence {
	return &StrategyResultPersistence{
		db:     db,
		config: config,
	}
}

// SaveStrategyResults saves strategy results to the database
func (p *StrategyResultPersistence) SaveStrategyResults(
	ctx context.Context,
	results map[string]map[string]*StrategyResult,
	processedAt time.Time,
) error {
	if !p.config.Persistence.BatchInsert {
		return p.saveResultsIndividually(ctx, results, processedAt)
	}

	return p.saveResultsBatch(ctx, results, processedAt)
}

// saveResultsBatch saves results using batch insert for better performance
func (p *StrategyResultPersistence) saveResultsBatch(
	ctx context.Context,
	results map[string]map[string]*StrategyResult,
	processedAt time.Time,
) error {
	var records []StrategyResultRecord
	batchSize := p.config.Persistence.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for groupID, groupResults := range results {
		for resultKey, result := range groupResults {
			// Parse result key to extract instrument key and strategy name
			instrumentKey, strategyName := p.parseResultKey(resultKey)
			if instrumentKey == "" || strategyName == "" {
				log.Warn("Invalid result key format: %s", resultKey)
				continue
			}

			// Convert columns added to JSON
			columnsAddedJSON, err := json.Marshal(result.ColumnsAdded)
			if err != nil {
				log.Error("Failed to marshal columns added: %v", err)
				continue
			}

			// Convert metadata to JSON
			resultDataJSON, err := json.Marshal(result.Metadata)
			if err != nil {
				log.Error("Failed to marshal result metadata: %v", err)
				continue
			}

			record := StrategyResultRecord{
				StockGroupID:    groupID,
				InstrumentKey:   instrumentKey,
				StrategyName:    strategyName,
				StrategyVersion: "1.0.0", // TODO: Get from strategy
				ProcessingTime:  int64(result.ProcessingTime.Milliseconds()),
				MemoryUsed:      result.MemoryUsed,
				RowsProcessed:   result.RowsProcessed,
				ColumnsAdded:    string(columnsAddedJSON),
				ResultData:      string(resultDataJSON),
				Error:           p.getErrorMessage(result.Error),
				ProcessedAt:     processedAt,
			}

			records = append(records, record)

			// Insert batch when size is reached
			if len(records) >= batchSize {
				if err := p.insertBatch(ctx, records); err != nil {
					return fmt.Errorf("batch insert failed: %w", err)
				}
				records = records[:0] // Reset slice
			}
		}
	}

	// Insert remaining records
	if len(records) > 0 {
		if err := p.insertBatch(ctx, records); err != nil {
			return fmt.Errorf("final batch insert failed: %w", err)
		}
	}

	log.Info("Saved %d strategy results to database", len(records))
	return nil
}

// saveResultsIndividually saves results one by one
func (p *StrategyResultPersistence) saveResultsIndividually(
	ctx context.Context,
	results map[string]map[string]*StrategyResult,
	processedAt time.Time,
) error {
	totalSaved := 0

	for groupID, groupResults := range results {
		for resultKey, result := range groupResults {
			// Parse result key to extract instrument key and strategy name
			instrumentKey, strategyName := p.parseResultKey(resultKey)
			if instrumentKey == "" || strategyName == "" {
				log.Warn("Invalid result key format: %s", resultKey)
				continue
			}

			// Convert columns added to JSON
			columnsAddedJSON, err := json.Marshal(result.ColumnsAdded)
			if err != nil {
				log.Error("Failed to marshal columns added: %v", err)
				continue
			}

			// Convert metadata to JSON
			resultDataJSON, err := json.Marshal(result.Metadata)
			if err != nil {
				log.Error("Failed to marshal result metadata: %v", err)
				continue
			}

			record := StrategyResultRecord{
				StockGroupID:    groupID,
				InstrumentKey:   instrumentKey,
				StrategyName:    strategyName,
				StrategyVersion: "1.0.0", // TODO: Get from strategy
				ProcessingTime:  int64(result.ProcessingTime.Milliseconds()),
				MemoryUsed:      result.MemoryUsed,
				RowsProcessed:   result.RowsProcessed,
				ColumnsAdded:    string(columnsAddedJSON),
				ResultData:      string(resultDataJSON),
				Error:           p.getErrorMessage(result.Error),
				ProcessedAt:     processedAt,
			}

			if err := p.db.WithContext(ctx).Create(&record).Error; err != nil {
				log.Error("Failed to save strategy result: %v", err)
				continue
			}

			totalSaved++
		}
	}

	log.Info("Saved %d strategy results to database", totalSaved)
	return nil
}

// insertBatch inserts a batch of records
func (p *StrategyResultPersistence) insertBatch(ctx context.Context, records []StrategyResultRecord) error {
	return p.db.WithContext(ctx).CreateInBatches(records, len(records)).Error
}

// parseResultKey parses the result key to extract instrument key and strategy name
func (p *StrategyResultPersistence) parseResultKey(resultKey string) (string, string) {
	// Expected format: "instrument_key_strategy_name"
	// For now, using a simple approach - can be enhanced based on actual key format
	if len(resultKey) == 0 {
		return "", ""
	}

	// TODO: Implement proper parsing based on actual result key format
	// For now, assuming the last underscore separates instrument key and strategy name
	lastUnderscore := -1
	for i := len(resultKey) - 1; i >= 0; i-- {
		if resultKey[i] == '_' {
			lastUnderscore = i
			break
		}
	}

	if lastUnderscore == -1 {
		return resultKey, "unknown"
	}

	instrumentKey := resultKey[:lastUnderscore]
	strategyName := resultKey[lastUnderscore+1:]

	return instrumentKey, strategyName
}

// getErrorMessage safely extracts error message
func (p *StrategyResultPersistence) getErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// GetStrategyResults retrieves strategy results for a given stock group and time range
func (p *StrategyResultPersistence) GetStrategyResults(
	ctx context.Context,
	stockGroupID string,
	startTime, endTime time.Time,
) ([]StrategyResultRecord, error) {
	var records []StrategyResultRecord

	err := p.db.WithContext(ctx).
		Where("stock_group_id = ? AND processed_at BETWEEN ? AND ?", stockGroupID, startTime, endTime).
		Order("processed_at DESC").
		Find(&records).Error

	return records, err
}

// GetStrategyResultsByInstrument retrieves strategy results for a given instrument
func (p *StrategyResultPersistence) GetStrategyResultsByInstrument(
	ctx context.Context,
	instrumentKey string,
	limit int,
) ([]StrategyResultRecord, error) {
	var records []StrategyResultRecord

	if limit <= 0 {
		limit = 100
	}

	err := p.db.WithContext(ctx).
		Where("instrument_key = ?", instrumentKey).
		Order("processed_at DESC").
		Limit(limit).
		Find(&records).Error

	return records, err
}

// CleanupOldResults removes old strategy results based on retention policy
func (p *StrategyResultPersistence) CleanupOldResults(ctx context.Context) error {
	retentionDays := p.config.Processing.DataRetentionDays
	if retentionDays <= 0 {
		retentionDays = 30 // Default 30 days
	}

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	result := p.db.WithContext(ctx).
		Where("processed_at < ?", cutoffDate).
		Delete(&StrategyResultRecord{})

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old results: %w", result.Error)
	}

	log.Info("Cleaned up %d old strategy results (older than %s)", result.RowsAffected, cutoffDate.Format("2006-01-02"))
	return nil
}

// GetMetrics retrieves persistence metrics
func (p *StrategyResultPersistence) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	var totalRecords int64
	var todayRecords int64
	var avgProcessingTime float64

	// Total records
	if err := p.db.WithContext(ctx).Model(&StrategyResultRecord{}).Count(&totalRecords).Error; err != nil {
		return nil, err
	}

	// Today's records
	today := time.Now().Truncate(24 * time.Hour)
	if err := p.db.WithContext(ctx).Model(&StrategyResultRecord{}).
		Where("processed_at >= ?", today).Count(&todayRecords).Error; err != nil {
		return nil, err
	}

	// Average processing time
	if err := p.db.WithContext(ctx).Model(&StrategyResultRecord{}).
		Select("AVG(processing_time)").Scan(&avgProcessingTime).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_records":       totalRecords,
		"today_records":       todayRecords,
		"avg_processing_time": avgProcessingTime,
		"retention_days":      p.config.Processing.DataRetentionDays,
	}, nil
}
