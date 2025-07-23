package domain

import (
	"time"
)

// MasterDataProcess represents a master data ingestion process
type MasterDataProcess struct {
	ID               int        `gorm:"primaryKey;autoIncrement" json:"id"`
	ProcessDate      time.Time  `gorm:"type:date;not null" json:"process_date"`
	NumberOfPastDays int        `gorm:"type:int;not null" json:"number_of_past_days"`
	Status           string     `gorm:"type:varchar(20);not null" json:"status"`
	CreatedAt        time.Time  `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	CompletedAt      *time.Time `gorm:"type:timestamp;null" json:"completed_at,omitempty"`
	Active           bool       `gorm:"type:boolean;not null;default:true" json:"active"`

	// Relationships
	Steps []MasterDataProcessStep `gorm:"foreignKey:ProcessID" json:"steps,omitempty"`
}

// MasterDataProcessStep represents a step within a master data process
type MasterDataProcessStep struct {
	ID           int        `gorm:"primaryKey;autoIncrement" json:"id"`
	ProcessID    int        `gorm:"type:int;not null" json:"process_id"`
	StepNumber   int        `gorm:"type:int;not null" json:"step_number"`
	StepName     string     `gorm:"type:varchar(50);not null" json:"step_name"`
	Status       string     `gorm:"type:varchar(20);not null" json:"status"`
	ErrorMessage *string    `gorm:"type:text;null" json:"error_message,omitempty"`
	StartedAt    *time.Time `gorm:"type:timestamp;null" json:"started_at,omitempty"`
	CompletedAt  *time.Time `gorm:"type:timestamp;null" json:"completed_at,omitempty"`
	CreatedAt    time.Time  `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	Active       bool       `gorm:"type:boolean;not null;default:true" json:"active"`

	// Relationships
	Process *MasterDataProcess `gorm:"foreignKey:ProcessID" json:"process,omitempty"`
}

// Process status constants
const (
	ProcessStatusRunning   = "RUNNING"
	ProcessStatusCompleted = "COMPLETED"
	ProcessStatusFailed    = "FAILED"
)

// Step status constants
const (
	StepStatusPending   = "PENDING"
	StepStatusRunning   = "RUNNING"
	StepStatusCompleted = "COMPLETED"
	StepStatusFailed    = "FAILED"
)

// Step names
const (
	StepNameDailyIngestion  = "daily_ingestion"
	StepNameFilterPipeline  = "filter_pipeline"
	StepNameMinuteIngestion = "minute_ingestion"
)

// TableName specifies the table name for MasterDataProcess
func (MasterDataProcess) TableName() string {
	return "master_data_process"
}

// TableName specifies the table name for MasterDataProcessStep
func (MasterDataProcessStep) TableName() string {
	return "master_data_process_steps"
}
