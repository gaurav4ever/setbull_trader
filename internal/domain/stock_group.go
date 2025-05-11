package domain

import (
	"time"
)

type StockGroupStatus string

const (
	GroupPending   StockGroupStatus = "PENDING"
	GroupExecuting StockGroupStatus = "EXECUTING"
	GroupCompleted StockGroupStatus = "COMPLETED"
	GroupFailed    StockGroupStatus = "FAILED"
)

type StockGroup struct {
	ID        string            `gorm:"column:id;primaryKey" json:"id"`
	EntryType string            `gorm:"column:entry_type" json:"entryType"`
	Status    StockGroupStatus  `gorm:"column:status" json:"status"`
	CreatedAt time.Time         `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time         `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	Stocks    []StockGroupStock `gorm:"foreignKey:GroupID;references:ID" json:"stocks"`
}

type StockGroupStock struct {
	ID      string `gorm:"column:id;primaryKey" json:"id"`
	GroupID string `gorm:"column:group_id" json:"groupId"`
	StockID string `gorm:"column:stock_id" json:"stockId"`
}
