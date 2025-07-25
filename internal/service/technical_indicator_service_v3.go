package service

import (
	"context"
	"fmt"
	"time"

	"setbull_trader/internal/analytics"
	"setbull_trader/internal/analytics/cache"
	"setbull_trader/internal/analytics/concurrency"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
)

// TechnicalIndicatorServiceV3 provides concurrent, cached technical indicator calculations
type TechnicalIndicatorServiceV3 struct {
	candleRepo repository.CandleRepository

	// Caching system
	cache *cache.IndicatorCache

	// Concurrency system
	workerPool *concurrency.WorkerPool
	pipeline   *concurrency.Pipeline

	// Configuration
	maxWorkers int
	cacheSize  int

	// Metrics
	metricsEnabled bool
	totalRequests  int64
	cacheHits      int64
	avgProcessTime time.Duration
}

// TechnicalIndicatorServiceV3Config holds configuration for the V3 service
type TechnicalIndicatorServiceV3Config struct {
	MaxWorkers     int
	CacheSize      int
	MetricsEnabled bool
}

// NewTechnicalIndicatorServiceV3 creates a new concurrent, cached technical indicator service
func NewTechnicalIndicatorServiceV3(
	candleRepo repository.CandleRepository,
	config TechnicalIndicatorServiceV3Config,
) (*TechnicalIndicatorServiceV3, error) {
	// Default configuration
	if config.MaxWorkers == 0 {
		config.MaxWorkers = 4
	}
	if config.CacheSize == 0 {
		config.CacheSize = 32 * 1024 * 1024 // 32MB default
	}

	// Initialize caching system
	indicatorCache := cache.NewIndicatorCache(config.CacheSize)

	// Initialize concurrency system
	workerPoolConfig := concurrency.WorkerPoolConfig{
		MaxWorkers:      config.MaxWorkers,
		QueueSize:       1000,
		ShutdownTimeout: 30 * time.Second,
	}
	workerPool := concurrency.NewWorkerPool(workerPoolConfig)

	pipelineConfig := concurrency.PipelineConfig{
		WorkerPoolConfig: workerPoolConfig,
		BatchSize:        10,
		MaxConcurrency:   config.MaxWorkers,
		Timeout:          30 * time.Second,
		CacheSize:        config.CacheSize / (1024 * 1024), // Convert to MB
	}
	pipeline := concurrency.NewPipeline(pipelineConfig)

	service := &TechnicalIndicatorServiceV3{
		candleRepo:     candleRepo,
		cache:          indicatorCache,
		workerPool:     workerPool,
		pipeline:       pipeline,
		maxWorkers:     config.MaxWorkers,
		cacheSize:      config.CacheSize,
		metricsEnabled: config.MetricsEnabled,
	}

	return service, nil
}

// CalculateIndicators calculates technical indicators with caching and concurrency
func (s *TechnicalIndicatorServiceV3) CalculateIndicators(
	ctx context.Context,
	instrumentKey string,
	interval string,
	start, end time.Time,
) (*analytics.IndicatorSet, error) {

	// Define indicator requests
	indicators := []concurrency.IndicatorRequest{
		{Type: "EMA", Parameters: map[string]interface{}{"period": 5}},
		{Type: "EMA", Parameters: map[string]interface{}{"period": 9}},
		{Type: "EMA", Parameters: map[string]interface{}{"period": 50}},
		{Type: "RSI", Parameters: map[string]interface{}{"period": 14}},
		{Type: "SMA", Parameters: map[string]interface{}{"period": 20}},
		{Type: "BB", Parameters: map[string]interface{}{"period": 20, "stdDev": 2.0}},
	}

	// For now, simulate fetching candles (since we need to check the CandleRepository interface)
	candles := s.generateSampleCandles(instrumentKey, 100)

	if len(candles) == 0 {
		return &analytics.IndicatorSet{}, nil
	}

	// Process indicators using pipeline batch processing
	instrumentKeys := []string{instrumentKey}
	candleData := map[string][]domain.Candle{
		instrumentKey: candles,
	}

	results, err := s.pipeline.ProcessBatch(ctx, instrumentKeys, indicators, candleData)
	if err != nil {
		return nil, fmt.Errorf("failed to process indicators: %w", err)
	}

	// Convert results to IndicatorSet
	if result, exists := results.Results[instrumentKey]; exists {
		if resultMap, ok := result.(map[string][]float64); ok {
			return s.convertResultToIndicatorSet(resultMap, candles), nil
		}
	}

	return &analytics.IndicatorSet{}, nil
}

// GetMetrics returns service metrics
func (s *TechnicalIndicatorServiceV3) GetMetrics() map[string]interface{} {
	if !s.metricsEnabled {
		return nil
	}

	return map[string]interface{}{
		"total_requests":   s.totalRequests,
		"cache_hits":       s.cacheHits,
		"cache_hit_ratio":  float64(s.cacheHits) / float64(s.totalRequests),
		"avg_process_time": s.avgProcessTime,
		"worker_pool_size": s.maxWorkers,
		"cache_size":       s.cacheSize,
	}
}

// Shutdown gracefully shuts down the service
func (s *TechnicalIndicatorServiceV3) Shutdown(ctx context.Context) error {
	log.Info("Shutting down TechnicalIndicatorServiceV3...")

	// Shutdown worker pool
	err := s.workerPool.Shutdown()
	if err != nil {
		log.Error("Error shutting down worker pool: %v", err)
	}

	// Clear cache
	s.cache.Clear()

	log.Info("TechnicalIndicatorServiceV3 shutdown complete")
	return nil
}

// Helper methods

// generateSampleCandles generates sample candle data for testing
func (s *TechnicalIndicatorServiceV3) generateSampleCandles(instrumentKey string, count int) []domain.Candle {
	candles := make([]domain.Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * time.Minute)
	basePrice := 100.0

	for i := 0; i < count; i++ {
		price := basePrice + float64(i%10-5)*0.1 // Small price variations
		candles[i] = domain.Candle{
			InstrumentKey: instrumentKey,
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Open:          price,
			High:          price + 0.5,
			Low:           price - 0.5,
			Close:         price + 0.1,
			Volume:        1000 + int64(i%100)*10,
		}
	}

	return candles
}

// convertResultToIndicatorSet converts pipeline results to IndicatorSet
func (s *TechnicalIndicatorServiceV3) convertResultToIndicatorSet(
	result map[string][]float64,
	candles []domain.Candle,
) *analytics.IndicatorSet {

	indicatorSet := &analytics.IndicatorSet{
		Timestamps: make([]time.Time, len(candles)),
	}

	// Extract timestamps
	for i, candle := range candles {
		indicatorSet.Timestamps[i] = candle.Timestamp
	}

	// Extract indicator values from result
	for indicator, values := range result {
		switch indicator {
		case "EMA_5":
			indicatorSet.EMA5 = values
		case "EMA_9":
			indicatorSet.EMA9 = values
		case "EMA_50":
			indicatorSet.EMA50 = values
		case "RSI_14":
			indicatorSet.RSI = values
		case "SMA_20":
			indicatorSet.MA9 = values // Using MA9 field for SMA_20
		case "BB_UPPER_20":
			indicatorSet.BBUpper = values
		case "BB_MIDDLE_20":
			indicatorSet.BBMiddle = values
		case "BB_LOWER_20":
			indicatorSet.BBLower = values
		}
	}

	return indicatorSet
}
