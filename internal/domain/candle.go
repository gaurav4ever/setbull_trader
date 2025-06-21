package domain

import (
	"context"
	"time"
)

// Candle represents a single OHLCV candle for a security
type Candle struct {
	ID            uint64    `json:"id" gorm:"primaryKey"`
	InstrumentKey string    `json:"instrument_key" gorm:"column:instrument_key;uniqueIndex:idx_stock_candle_unique"`
	Timestamp     time.Time `json:"timestamp" gorm:"column:timestamp;uniqueIndex:idx_stock_candle_unique"`
	Open          float64   `json:"open" gorm:"column:open"`
	High          float64   `json:"high" gorm:"column:high"`
	Low           float64   `json:"low" gorm:"column:low"`
	Close         float64   `json:"close" gorm:"column:close"`
	Volume        int64     `json:"volume" gorm:"column:volume"`
	OpenInterest  int64     `json:"open_interest" gorm:"column:open_interest"`
	TimeInterval  string    `json:"time_interval" gorm:"column:time_interval;uniqueIndex:idx_stock_candle_unique"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`

	// Indicator fields for integration
	MA9      float64 `json:"ma_9" gorm:"column:ma_9"`
	BBUpper  float64 `json:"bb_upper" gorm:"column:bb_upper"`
	BBMiddle float64 `json:"bb_middle" gorm:"column:bb_middle"`
	BBLower  float64 `json:"bb_lower" gorm:"column:bb_lower"`
	VWAP     float64 `json:"vwap" gorm:"column:vwap"`
	EMA5     float64 `json:"ema_5" gorm:"column:ema_5"`
	EMA9     float64 `json:"ema_9" gorm:"column:ema_9"`
	EMA50    float64 `json:"ema_50" gorm:"column:ema_50"`
	ATR      float64 `json:"atr" gorm:"column:atr"`
	RSI      float64 `json:"rsi" gorm:"column:rsi"`
	BBWidth  float64 `json:"bb_width" gorm:"column:bb_width"`
}

// TableName returns the table name for the Candle model
func (Candle) TableName() string {
	return "stock_candle_data"
}

// CandleBatch represents a batch of candles for a single instrument and interval
type CandleBatch struct {
	InstrumentKey string
	Interval      string
	Candles       []*Candle
}

// CandleRepository defines the interface for candle data operations
type CandleRepository interface {
	Save(ctx context.Context, candle *Candle) error
	FindByInstrumentAndTimeRange(ctx context.Context, instrumentKey, interval string, start, end time.Time) ([]Candle, error)
	FindCandlesBefore(ctx context.Context, instrumentKey, interval string, date time.Time, limit int) ([]Candle, error)
	GetLastCandle(ctx context.Context, instrumentKey, interval string) (*Candle, error)
	GetLastNCandles(ctx context.Context, instrumentKey, interval string, n int) ([]Candle, error)
	SaveAll(ctx context.Context, candles []Candle) error
}

// MinMaxTimestamp is a helper struct for aggregation operations
type MinMaxTimestamp struct {
	InstrumentKey  string    `