package domain

import (
	"context"
	"time"
)

// Candle represents a single OHLCV candle for a security (1-minute data)
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
	MA9           float64 `json:"ma_9" gorm:"column:ma_9"`
	BBUpper       float64 `json:"bb_upper" gorm:"column:bb_upper"`
	BBMiddle      float64 `json:"bb_middle" gorm:"column:bb_middle"`
	BBLower       float64 `json:"bb_lower" gorm:"column:bb_lower"`
	VWAP          float64 `json:"vwap" gorm:"column:vwap"`
	EMA5          float64 `json:"ema_5" gorm:"column:ema_5"`
	EMA9          float64 `json:"ema_9" gorm:"column:ema_9"`
	EMA50         float64 `json:"ema_50" gorm:"column:ema_50"`
	ATR           float64 `json:"atr" gorm:"column:atr"`
	RSI           float64 `json:"rsi" gorm:"column:rsi"`
	BBWidth       float64 `json:"bb_width" gorm:"column:bb_width"`
	LowestBBWidth float64 `json:"lowest_bb_width" gorm:"column:lowest_bb_width"`
}

// TableName returns the table name for the Candle model
func (Candle) TableName() string {
	return "stock_candle_data"
}

// Candle5Min represents a single OHLCV candle for a security (5-minute data)
type Candle5Min struct {
	ID            uint64    `json:"id" gorm:"primaryKey"`
	InstrumentKey string    `json:"instrument_key" gorm:"column:instrument_key;uniqueIndex:idx_stock_candle_5min_unique"`
	Timestamp     time.Time `json:"timestamp" gorm:"column:timestamp;uniqueIndex:idx_stock_candle_5min_unique"`
	Open          float64   `json:"open" gorm:"column:open"`
	High          float64   `json:"high" gorm:"column:high"`
	Low           float64   `json:"low" gorm:"column:low"`
	Close         float64   `json:"close" gorm:"column:close"`
	Volume        int64     `json:"volume" gorm:"column:volume"`
	OpenInterest  int64     `json:"open_interest" gorm:"column:open_interest"`
	TimeInterval  string    `json:"time_interval" gorm:"column:time_interval;uniqueIndex:idx_stock_candle_5min_unique"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	Active        bool      `json:"active" gorm:"column:active;default:true"`

	// Indicator fields for integration (including BB width normalized fields)
	MA9                         float64 `json:"ma_9" gorm:"column:ma_9"`
	BBUpper                     float64 `json:"bb_upper" gorm:"column:bb_upper"`
	BBMiddle                    float64 `json:"bb_middle" gorm:"column:bb_middle"`
	BBLower                     float64 `json:"bb_lower" gorm:"column:bb_lower"`
	VWAP                        float64 `json:"vwap" gorm:"column:vwap"`
	EMA5                        float64 `json:"ema_5" gorm:"column:ema_5"`
	EMA9                        float64 `json:"ema_9" gorm:"column:ema_9"`
	EMA20                       float64 `json:"ema_20" gorm:"column:ema_20"`
	EMA50                       float64 `json:"ema_50" gorm:"column:ema_50"`
	ATR                         float64 `json:"atr" gorm:"column:atr"`
	RSI                         float64 `json:"rsi" gorm:"column:rsi"`
	BBWidth                     float64 `json:"bb_width" gorm:"column:bb_width"`
	BBWidthNormalized           float64 `json:"bb_width_normalized" gorm:"column:bb_width_normalized"`
	BBWidthNormalizedPercentage float64 `json:"bb_width_normalized_percentage" gorm:"column:bb_width_normalized_percentage"`
	LowestBBWidth               float64 `json:"lowest_bb_width" gorm:"column:lowest_bb_width"`
}

// TableName returns the table name for the Candle5Min model
func (Candle5Min) TableName() string {
	return "stock_candle_data_5min"
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
	MA9                         float64 `json:"ma_9"`
	BBUpper                     float64 `json:"bb_upper"`
	BBMiddle                    float64 `json:"bb_middle"`
	BBLower                     float64 `json:"bb_lower"`
	VWAP                        float64 `json:"vwap"`
	EMA5                        float64 `json:"ema_5"`
	EMA9                        float64 `json:"ema_9"`
	EMA20                       float64 `json:"ema_20"`
	EMA50                       float64 `json:"ema_50"`
	ATR                         float64 `json:"atr"`
	RSI                         float64 `json:"rsi"`
	BBWidth                     float64 `json:"bb_width"`                       // upper - lower
	BBWidthNormalized           float64 `json:"bb_width_normalized"`            // (upper - lower) / middle
	BBWidthNormalizedPercentage float64 `json:"bb_width_normalized_percentage"` // ((upper - lower) / middle) * 100
	LowestBBWidth               float64 `json:"lowest_bb_width"`
}

// DailyCandelFetchResult represents the result of a batch operation to fetch daily candles
type DailyCandelFetchResult struct {
	TotalStocks      int      `json:"total_stocks"`
	ProcessedStocks  int      `json:"processed_stocks"`
	SuccessfulStocks int      `json:"successful_stocks"`
	FailedStocks     int      `json:"failed_stocks"`
	FailedSymbols    []string `json:"failed_symbols,omitempty"`
}
