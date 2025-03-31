package domain

import (
	"time"
)

// Candle represents a single candlestick data point
type Candle struct {
	ID            uint      `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	InstrumentKey string    `json:"instrumentKey" gorm:"column:instrument_key;index"`
	Timestamp     time.Time `json:"timestamp" gorm:"column:timestamp;index"`
	Open          float64   `json:"open" gorm:"column:open"`
	High          float64   `json:"high" gorm:"column:high"`
	Low           float64   `json:"low" gorm:"column:low"`
	Close         float64   `json:"close" gorm:"column:close"`
	Volume        int64     `json:"volume" gorm:"column:volume"`
	OpenInterest  int64     `json:"openInterest" gorm:"column:open_interest"`
	TimeInterval  string    `json:"timeInterval" gorm:"column:time_interval;index"`
	CreatedAt     time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
}

// TableName specifies the database table name for the Candle model
func (Candle) TableName() string {
	return "stock_candle_data"
}

// CandleBatch represents a batch of candle data for bulk operations
type CandleBatch struct {
	Candles []Candle
}

// CandleRepository defines the interface for candle data persistence operations
type CandleRepository interface {
	// Store stores a single candle
	Store(candle *Candle) error

	// StoreBatch stores multiple candles in a batch operation
	StoreBatch(candles *CandleBatch) (int, error)

	// FindByInstrumentAndTimeRange retrieves candles for an instrument within a time range
	FindByInstrumentAndTimeRange(instrumentKey string, interval string, fromTime, toTime time.Time) ([]Candle, error)

	// DeleteByInstrumentAndTimeRange deletes candles for an instrument within a time range
	DeleteByInstrumentAndTimeRange(instrumentKey string, interval string, fromTime, toTime time.Time) (int, error)
}
