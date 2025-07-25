package analytics

import (
	"context"
	"time"

	"setbull_trader/internal/domain"

	"github.com/go-gota/gota/dataframe"
)

// AnalyticsEngine defines the main interface for analytics processing
type AnalyticsEngine interface {
	ProcessCandles(ctx context.Context, candles []domain.Candle) (*ProcessingResult, error)
	CalculateIndicators(ctx context.Context, data *CandleData) (*IndicatorSet, error)
	AggregateTimeframes(ctx context.Context, data *CandleData, timeframe string) (*AggregatedCandles, error)
}

// ProcessingResult contains the result of candle processing
type ProcessingResult struct {
	DataFrame   dataframe.DataFrame `json:"-"`
	Indicators  *IndicatorSet       `json:"indicators"`
	CacheHits   int                 `json:"cache_hits"`
	ProcessTime time.Duration       `json:"process_time"`
	MemoryUsage int64               `json:"memory_usage"`
}

// IndicatorSet contains all calculated technical indicators
type IndicatorSet struct {
	MA9                         []float64   `json:"ma9"`
	BBUpper                     []float64   `json:"bb_upper"`
	BBMiddle                    []float64   `json:"bb_middle"`
	BBLower                     []float64   `json:"bb_lower"`
	BBWidth                     []float64   `json:"bb_width"`
	BBWidthNormalized           []float64   `json:"bb_width_normalized"`
	BBWidthNormalizedPercentage []float64   `json:"bb_width_normalized_percentage"`
	VWAP                        []float64   `json:"vwap"`
	EMA5                        []float64   `json:"ema5"`
	EMA9                        []float64   `json:"ema9"`
	EMA50                       []float64   `json:"ema50"`
	ATR                         []float64   `json:"atr"`
	RSI                         []float64   `json:"rsi"`
	Timestamps                  []time.Time `json:"timestamps"`
}

// CandleData represents processed candle data for analytics
type CandleData struct {
	Candles   []domain.Candle     `json:"candles"`
	DataFrame dataframe.DataFrame `json:"-"`
	StartTime time.Time           `json:"start_time"`
	EndTime   time.Time           `json:"end_time"`
	Symbol    string              `json:"symbol"`
	Timeframe string              `json:"timeframe"`
}

// AggregatedCandles represents aggregated candle data with indicators
type AggregatedCandles struct {
	Candles     []domain.AggregatedCandle `json:"candles"`
	TotalCount  int                       `json:"total_count"`
	ProcessTime time.Duration             `json:"process_time"`
	CacheUsed   bool                      `json:"cache_used"`
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	MemoryUsage int64   `json:"memory_usage"`
	Keys        int64   `json:"keys"`
}

// AnalyticsConfig holds configuration for the analytics engine
type AnalyticsConfig struct {
	CacheSize       int           `json:"cache_size"`       // Cache size in MB
	MaxMemoryUsage  int64         `json:"max_memory_usage"` // Max memory usage in bytes
	WorkerPoolSize  int           `json:"worker_pool_size"` // Number of workers for parallel processing
	TimeoutDuration time.Duration `json:"timeout_duration"` // Processing timeout
	EnableCaching   bool          `json:"enable_caching"`   // Enable/disable caching
	EnableMetrics   bool          `json:"enable_metrics"`   // Enable/disable metrics collection
}

// DefaultAnalyticsConfig returns default configuration
func DefaultAnalyticsConfig() *AnalyticsConfig {
	return &AnalyticsConfig{
		CacheSize:       100,               // 100MB
		MaxMemoryUsage:  500 * 1024 * 1024, // 500MB
		WorkerPoolSize:  10,
		TimeoutDuration: 30 * time.Second,
		EnableCaching:   true,
		EnableMetrics:   true,
	}
}

// AnalyticsError represents analytics-specific errors
type AnalyticsError struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Operation string    `json:"operation"`
	Timestamp time.Time `json:"timestamp"`
	Cause     error     `json:"-"`
}

func (e *AnalyticsError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// NewAnalyticsError creates a new analytics error
func NewAnalyticsError(code, message, operation string, cause error) *AnalyticsError {
	return &AnalyticsError{
		Code:      code,
		Message:   message,
		Operation: operation,
		Timestamp: time.Now(),
		Cause:     cause,
	}
}
