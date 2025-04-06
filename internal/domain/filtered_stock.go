package domain

import (
	"time"
)

type FilteredStockRecord struct {
	ID                int64       `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol            string      `gorm:"type:varchar(20);not null" json:"symbol"`
	InstrumentKey     string      `gorm:"type:varchar(50);not null" json:"instrument_key"`
	ExchangeToken     string      `gorm:"type:varchar(20);not null" json:"exchange_token"`
	Trend             string      `gorm:"type:varchar(10);not null" json:"trend"`
	CurrentPrice      float64     `gorm:"type:decimal(10,2);not null" json:"current_price"`
	MambaCount        int         `gorm:"type:int;not null" json:"mamba_count"`
	BullishMambaCount int         `gorm:"type:int;not null" json:"bullish_mamba_count"`
	BearishMambaCount int         `gorm:"type:int;not null" json:"bearish_mamba_count"`
	AvgMambaMove      float64     `gorm:"type:decimal(10,2);not null;default:0" json:"avg_mamba_move"`
	AvgNonMambaMove   float64     `gorm:"type:decimal(10,2);not null;default:0" json:"avg_non_mamba_move"`
	MambaSeries       interface{} `gorm:"type:json" json:"mamba_series"`
	NonMambaSeries    interface{} `gorm:"type:json" json:"non_mamba_series"`
	FilterDate        time.Time   `gorm:"type:datetime;not null" json:"filter_date"`
	CreatedAt         time.Time   `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt         time.Time   `gorm:"type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}
