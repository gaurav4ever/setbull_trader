package domain

import "context"

// FilteredStock represents a stock that has passed through the filtering pipeline
type FilteredStock struct {
	Stock         StockUniverse
	LastCandle    Candle
	ClosePrice    float64
	DailyVolume   int64
	EMA50         float64
	RSI14         float64
	IsBullish     bool
	IsBearish     bool
	FilterResults map[string]bool
	FilterReasons map[string]string
}

// StockFilter interface defines the contract for all filters
type StockFilter interface {
	Filter(ctx context.Context, stocks interface{}) (bullish, bearish []FilteredStock, err error)
}

// Add to domain package
type FilterConfig struct {
	BasicFilter struct {
		MinPrice  float64
		MaxPrice  float64
		MinVolume int64
	}
	EMAFilter struct {
		Period    int
		Threshold float64
	}
	RSIFilter struct {
		Period           int
		BullishThreshold float64
		BearishThreshold float64
	}
}
