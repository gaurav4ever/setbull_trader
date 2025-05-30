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

// MinMaxTimestamp is a helper struct for aggregation operations
type MinMaxTimestamp struct {
	InstrumentKey  string    `json:"instrument_key"`
	IntervalTime   time.Time `json:"interval_time"` // 5-min interval or day
	FirstTimestamp time.Time `json:"first_timestamp"`
	LastTimestamp  time.Time `json:"last_timestamp"`
	HighPrice      float64   `json:"high_price"`
	LowPrice       float64   `json:"low_price"`
	TotalVolume    int64     `json:"total_volume"`
}

// OpenPriceData is a helper struct for holding open price data
type OpenPriceData struct {
	InstrumentKey string    `json:"instrument_key"`
	IntervalTime  time.Time `json:"interval_time"`
	OpenPrice     float64   `json:"open_price"`
}

// ClosePriceData is a helper struct for holding close price data
type ClosePriceData struct {
	InstrumentKey string    `json:"instrument_key"`
	IntervalTime  time.Time `json:"interval_time"`
	ClosePrice    float64   `json:"close_price"`
	OpenInterest  int64     `json:"open_interest"`
}

// AggregatedCandle represents a candle at an aggregated timeframe (5-min, daily)
type AggregatedCandle struct {
	InstrumentKey string    `json:"instrument_key"`
	Timestamp     time.Time `json:"timestamp"`
	Open          float64   `json:"open"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	Close         float64   `json:"close"`
	Volume        int64     `json:"volume"`
	OpenInterest  int64     `json:"open_interest"`
	TimeInterval  string    `json:"time_interval"`

	// Indicator fields for integration
	MA9      float64 `json:"ma_9"`
	BBUpper  float64 `json:"bb_upper"`
	BBMiddle float64 `json:"bb_middle"`
	BBLower  float64 `json:"bb_lower"`
	VWAP     float64 `json:"vwap"`
	EMA5     float64 `json:"ema_5"`
	EMA9     float64 `json:"ema_9"`
	EMA50    float64 `json:"ema_50"`
	ATR      float64 `json:"atr"`
	RSI      float64 `json:"rsi"`
}

// DailyCandelFetchResult represents the result of a batch operation to fetch daily candles
type DailyCandelFetchResult struct {
	TotalStocks      int      `json:"total_stocks"`
	ProcessedStocks  int      `json:"processed_stocks"`
	SuccessfulStocks int      `json:"successful_stocks"`
	FailedStocks     int      `json:"failed_stocks"`
	FailedSymbols    []string `json:"failed_symbols,omitempty"`
}
