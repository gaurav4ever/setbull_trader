package repository

import (
	"context"
	"time"

	"setbull_trader/internal/analytics"
	"setbull_trader/internal/analytics/cache"
	"setbull_trader/internal/domain"
)

// SequenceRepository interface for sequence analysis data persistence
type SequenceRepository interface {
	StoreSequenceAnalysis(ctx context.Context, analysis domain.SequenceAnalysis) error
	GetSequenceMetrics(ctx context.Context, stock string, timeRange TimeRange) (*domain.SequenceMetrics, error)
	GetSequencePatterns(ctx context.Context, filters PatternFilters) ([]domain.SequencePattern, error)
	GetSequenceHistory(ctx context.Context, stock string, limit int) ([]domain.SequenceAnalysis, error)
}

// IndicatorCacheRepository interface for persisting indicator cache data
type IndicatorCacheRepository interface {
	StoreCachedIndicators(ctx context.Context, key cache.CacheKey, indicators *domain.TechnicalIndicators) error
	GetCachedIndicators(ctx context.Context, key cache.CacheKey) (*domain.TechnicalIndicators, error)
	InvalidateCache(ctx context.Context, symbol string) error
	GetCacheMetrics(ctx context.Context) (*CacheMetrics, error)
	CleanupExpiredCache(ctx context.Context, ttl time.Duration) error
}

// AnalyticsRepository interface for analytics engine data persistence
type AnalyticsRepository interface {
	StoreProcessingResult(ctx context.Context, symbol string, result *analytics.ProcessingResult) error
	GetAggregatedCandles(ctx context.Context, symbol, timeframe string, start, end time.Time) (*analytics.AggregatedCandles, error)
	StoreAggregatedCandles(ctx context.Context, symbol, timeframe string, result *analytics.AggregatedCandles) error
	InvalidateAggregationCache(ctx context.Context, instrumentKey string) error
	GetAnalyticsMetrics(ctx context.Context) (*AnalyticsMetrics, error)
}

// Supporting types for repository interfaces

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// PatternFilters represents filters for pattern queries
type PatternFilters struct {
	PatternType    string
	MinFrequency   int
	MinSuccessRate float64
	TimeRange      TimeRange
}

// CacheMetrics represents cache performance metrics
type CacheMetrics struct {
	TotalKeys   int64
	HitRate     float64
	MissRate    float64
	MemoryUsage int64
	LastCleanup time.Time
}

// AnalyticsMetrics represents analytics engine performance metrics
type AnalyticsMetrics struct {
	TotalProcessedItems int64
	AverageProcessTime  time.Duration
	CacheHitRate        float64
	WorkerPoolUsage     float64
	LastProcessedAt     time.Time
}
