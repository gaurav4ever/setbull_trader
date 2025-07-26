package analytics

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"time"

	"setbull_trader/internal/analytics/dataframe"
	"setbull_trader/internal/analytics/indicators"
	"setbull_trader/internal/domain"

	"github.com/VictoriaMetrics/fastcache"
)

// Processor implements the AnalyticsEngine interface
type Processor struct {
	config     *AnalyticsConfig
	cache      *fastcache.Cache
	aggregator *dataframe.Aggregator
	calculator *indicators.Calculator
	metrics    *ProcessorMetrics
}

// ProcessorMetrics tracks processor performance
type ProcessorMetrics struct {
	TotalProcessed  int64         `json:"total_processed"`
	TotalErrors     int64         `json:"total_errors"`
	AverageTime     time.Duration `json:"average_time"`
	CacheHitRate    float64       `json:"cache_hit_rate"`
	MemoryUsage     int64         `json:"memory_usage"`
	LastProcessedAt time.Time     `json:"last_processed_at"`
}

// NewProcessor creates a new analytics processor
func NewProcessor(config *AnalyticsConfig) *Processor {
	if config == nil {
		config = DefaultAnalyticsConfig()
	}

	var cache *fastcache.Cache
	if config.EnableCaching {
		cache = fastcache.New(config.CacheSize * 1024 * 1024) // Convert MB to bytes
	}

	aggregator := dataframe.NewAggregator(nil)
	calculator := indicators.NewCalculator()

	return &Processor{
		config:     config,
		cache:      cache,
		aggregator: aggregator,
		calculator: calculator,
		metrics:    &ProcessorMetrics{},
	}
}

// ProcessCandles processes candles through the DataFrame pipeline
func (p *Processor) ProcessCandles(ctx context.Context, candles []domain.Candle) (*ProcessingResult, error) {
	startTime := time.Now()

	// Validate input
	if len(candles) == 0 {
		return &ProcessingResult{
			DataFrame:   dataframe.NewCandleDataFrameWithContext([]domain.Candle{}, dataframe.TimestampFromDatabase).DataFrame(),
			Indicators:  &IndicatorSet{},
			CacheHits:   0,
			ProcessTime: time.Since(startTime),
		}, nil
	}

	// Check cache first
	cacheKey := p.generateCacheKey(candles)
	var cacheHits int

	if p.cache != nil {
		if cached := p.cache.Get(nil, []byte(cacheKey)); cached != nil {
			// TODO: Implement cache deserialization
			cacheHits = 1
		}
	}

	// Create DataFrame from candles - assume database source (IST timestamps)
	df := dataframe.NewCandleDataFrameWithContext(candles, dataframe.TimestampFromDatabase)
	if df.Empty() {
		return nil, fmt.Errorf("failed to create DataFrame from candles")
	}

	// Calculate indicators (placeholder for now)
	indicators := &IndicatorSet{
		Timestamps:                  make([]time.Time, len(candles)),
		MA9:                         make([]float64, len(candles)),
		BBUpper:                     make([]float64, len(candles)),
		BBMiddle:                    make([]float64, len(candles)),
		BBLower:                     make([]float64, len(candles)),
		BBWidth:                     make([]float64, len(candles)),
		BBWidthNormalized:           make([]float64, len(candles)),
		BBWidthNormalizedPercentage: make([]float64, len(candles)),
		VWAP:                        make([]float64, len(candles)),
		EMA5:                        make([]float64, len(candles)),
		EMA9:                        make([]float64, len(candles)),
		EMA50:                       make([]float64, len(candles)),
		ATR:                         make([]float64, len(candles)),
		RSI:                         make([]float64, len(candles)),
	}

	// Fill timestamps
	for i, candle := range candles {
		indicators.Timestamps[i] = candle.Timestamp
	}

	// Update metrics
	p.updateMetrics(startTime, len(candles), cacheHits, nil)

	// Cache result if caching is enabled
	if p.cache != nil {
		// TODO: Implement cache serialization
	}

	return &ProcessingResult{
		DataFrame:   df.DataFrame(),
		Indicators:  indicators,
		CacheHits:   cacheHits,
		ProcessTime: time.Since(startTime),
		MemoryUsage: p.getMemoryUsage(),
	}, nil
}

// CalculateIndicators calculates technical indicators for the given data
func (p *Processor) CalculateIndicators(ctx context.Context, data *CandleData) (*IndicatorSet, error) {
	if data == nil || len(data.Candles) == 0 {
		return &IndicatorSet{}, nil
	}

	startTime := time.Now()

	// Extract price and volume data from candles
	n := len(data.Candles)
	closePrices := make([]float64, n)
	highPrices := make([]float64, n)
	lowPrices := make([]float64, n)
	volumes := make([]float64, n)
	timestamps := make([]time.Time, n)

	for i, candle := range data.Candles {
		closePrices[i] = candle.Close
		highPrices[i] = candle.High
		lowPrices[i] = candle.Low
		volumes[i] = float64(candle.Volume) // Convert int64 to float64
		timestamps[i] = candle.Timestamp
	}

	// Initialize indicator set with timestamps
	indicators := &IndicatorSet{
		Timestamps: timestamps,
	}

	// EMA calculations
	indicators.EMA5 = p.calculator.EMA(closePrices, 5)
	indicators.EMA9 = p.calculator.EMA(closePrices, 9)
	indicators.EMA50 = p.calculator.EMA(closePrices, 50)

	// MA9 (Simple Moving Average)
	indicators.MA9 = p.calculator.SMA(closePrices, 9)

	// Bollinger Bands (period=20, std dev multiplier=2.0)
	bbUpper, bbMiddle, bbLower := p.calculator.BollingerBands(closePrices, 20, 2.0)
	indicators.BBUpper = bbUpper
	indicators.BBMiddle = bbMiddle
	indicators.BBLower = bbLower

	// BB Width calculations
	indicators.BBWidth = p.calculator.BBWidth(bbUpper, bbMiddle, bbLower)

	// Normalized BB Width (already normalized by middle band in BBWidth calculation)
	indicators.BBWidthNormalized = make([]float64, len(indicators.BBWidth))
	copy(indicators.BBWidthNormalized, indicators.BBWidth)

	// BB Width as percentage
	indicators.BBWidthNormalizedPercentage = make([]float64, len(indicators.BBWidth))
	for i, width := range indicators.BBWidth {
		if !math.IsNaN(width) {
			indicators.BBWidthNormalizedPercentage[i] = width * 100
		} else {
			indicators.BBWidthNormalizedPercentage[i] = math.NaN()
		}
	}

	// VWAP calculation
	indicators.VWAP = p.calculator.VWAP(closePrices, volumes)

	// RSI calculation (period=14)
	indicators.RSI = p.calculator.RSI(closePrices, 14)

	// ATR calculation (period=14)
	indicators.ATR = p.calculator.ATR(highPrices, lowPrices, closePrices, 14)

	// Update metrics
	p.updateMetrics(startTime, len(data.Candles), 0, nil)

	return indicators, nil
}

// AggregateTimeframes aggregates candles to different timeframes
func (p *Processor) AggregateTimeframes(ctx context.Context, data *CandleData, timeframe string) (*AggregatedCandles, error) {
	startTime := time.Now()

	if data == nil || len(data.Candles) == 0 {
		return &AggregatedCandles{
			Candles:     []domain.AggregatedCandle{},
			TotalCount:  0,
			ProcessTime: time.Since(startTime),
			CacheUsed:   false,
		}, nil
	}

	// Validate timeframe
	if err := p.aggregator.ValidateTimeframe(timeframe); err != nil {
		return nil, err
	}

	// Parse timeframe duration
	interval, err := p.aggregator.ParseTimeframe(timeframe)
	if err != nil {
		return nil, err
	}

	// Aggregate with indicators
	df, err := p.aggregator.AggregateWithIndicators(data.Candles, interval)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate candles: %v", err)
	}

	// Convert to aggregated candles
	aggregatedCandles := df.ToAggregatedCandles()

	return &AggregatedCandles{
		Candles:     aggregatedCandles,
		TotalCount:  len(aggregatedCandles),
		ProcessTime: time.Since(startTime),
		CacheUsed:   false,
	}, nil
}

// GetMetrics returns current processor metrics
func (p *Processor) GetMetrics() *ProcessorMetrics {
	return p.metrics
}

// generateCacheKey generates a cache key for the given candles
func (p *Processor) generateCacheKey(candles []domain.Candle) string {
	if len(candles) == 0 {
		return ""
	}

	first := candles[0]
	last := candles[len(candles)-1]

	return fmt.Sprintf("candles:%s:%s:%s:%d",
		first.InstrumentKey,
		first.Timestamp.Format("2006-01-02T15:04:05"),
		last.Timestamp.Format("2006-01-02T15:04:05"),
		len(candles),
	)
}

// updateMetrics updates processor metrics
func (p *Processor) updateMetrics(startTime time.Time, candleCount, cacheHits int, err error) {
	p.metrics.TotalProcessed++
	p.metrics.LastProcessedAt = time.Now()

	if err != nil {
		p.metrics.TotalErrors++
	}

	// Update average time (simple moving average)
	processingTime := time.Since(startTime)
	if p.metrics.AverageTime == 0 {
		p.metrics.AverageTime = processingTime
	} else {
		// Simple exponential moving average with alpha = 0.1
		p.metrics.AverageTime = time.Duration(
			0.9*float64(p.metrics.AverageTime) + 0.1*float64(processingTime),
		)
	}

	// Update cache hit rate
	if p.metrics.TotalProcessed > 0 {
		p.metrics.CacheHitRate = float64(cacheHits) / float64(p.metrics.TotalProcessed)
	}

	// Update memory usage
	p.metrics.MemoryUsage = p.getMemoryUsage()
}

// getMemoryUsage returns current memory usage
func (p *Processor) getMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc)
}

// Reset resets processor metrics and cache
func (p *Processor) Reset() {
	p.metrics = &ProcessorMetrics{}
	if p.cache != nil {
		p.cache.Reset()
	}
}

// Close gracefully shuts down the processor
func (p *Processor) Close() error {
	// Currently no resources to clean up
	// This will be useful when we add worker pools in Phase 3
	return nil
}

// ValidateConfig validates the processor configuration
func (p *Processor) ValidateConfig() error {
	if p.config.CacheSize < 0 {
		return fmt.Errorf("cache size must be non-negative")
	}

	if p.config.MaxMemoryUsage < 0 {
		return fmt.Errorf("max memory usage must be non-negative")
	}

	if p.config.WorkerPoolSize <= 0 {
		return fmt.Errorf("worker pool size must be positive")
	}

	if p.config.TimeoutDuration <= 0 {
		return fmt.Errorf("timeout duration must be positive")
	}

	return nil
}

// GetConfig returns the current configuration
func (p *Processor) GetConfig() *AnalyticsConfig {
	return p.config
}

// SetConfig updates the processor configuration
func (p *Processor) SetConfig(config *AnalyticsConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	p.config = config
	return p.ValidateConfig()
}
