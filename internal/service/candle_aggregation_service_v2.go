package service

import (
	"context"
	"fmt"
	"time"

	"setbull_trader/internal/analytics"
	"setbull_trader/internal/core/dto/response"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
)

// CandleAggregationServiceV2 provides DataFrame-based candle aggregation operations
type CandleAggregationServiceV2 struct {
	candleRepo           repository.CandleRepository
	candle5MinRepo       repository.Candle5MinRepository
	analyticsEngine      analytics.AnalyticsEngine
	batchFetchService    *BatchFetchService
	tradingCalendar      *TradingCalendarService
	utilityService       *UtilityService
	candleCloseListeners []CandleCloseListener
}

// NewCandleAggregationServiceV2 creates a new DataFrame-based candle aggregation service
func NewCandleAggregationServiceV2(
	candleRepo repository.CandleRepository,
	candle5MinRepo repository.Candle5MinRepository,
	batchFetchService *BatchFetchService,
	tradingCalendar *TradingCalendarService,
	utilityService *UtilityService,
) *CandleAggregationServiceV2 {
	// Create analytics engine with optimized configuration
	config := &analytics.AnalyticsConfig{
		EnableCaching:   true,
		CacheSize:       256, // 256MB cache
		MaxMemoryUsage:  512, // 512MB max memory
		WorkerPoolSize:  4,
		TimeoutDuration: 30 * time.Second,
	}

	analyticsEngine := analytics.NewProcessor(config)

	return &CandleAggregationServiceV2{
		candleRepo:        candleRepo,
		candle5MinRepo:    candle5MinRepo,
		analyticsEngine:   analyticsEngine,
		batchFetchService: batchFetchService,
		tradingCalendar:   tradingCalendar,
		utilityService:    utilityService,
	}
}

// Aggregate5MinCandlesWithIndicators uses DataFrame-based processing to aggregate candles and calculate indicators
func (s *CandleAggregationServiceV2) Aggregate5MinCandlesWithIndicators(
	ctx context.Context,
	instrumentKey string,
	startTime, endTime time.Time,
	bbWidthCallback func(ctx context.Context, instrumentKey string, candle domain.AggregatedCandle),
) error {
	log.Info("Starting DataFrame-based 5-min aggregation for %s from %v to %v",
		instrumentKey, startTime, endTime)

	// 1. Fetch all 1-minute candles for the time range
	oneMinCandles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, "1minute", startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to fetch 1-min candles: %w", err)
	}
	if len(oneMinCandles) == 0 {
		return fmt.Errorf("no 1-min candles found for aggregation")
	}

	log.Info("Fetched %d 1-minute candles for %s", len(oneMinCandles), instrumentKey)

	// 2. Use analytics engine for DataFrame-based processing
	processingResult, err := s.analyticsEngine.ProcessCandles(ctx, oneMinCandles)
	if err != nil {
		return fmt.Errorf("failed to process candles with analytics engine: %w", err)
	}

	// 3. Aggregate to 5-minute timeframe using DataFrame operations
	candleData := &analytics.CandleData{Candles: oneMinCandles}
	aggregatedResult, err := s.analyticsEngine.AggregateTimeframes(ctx, candleData, "5m")
	if err != nil {
		return fmt.Errorf("failed to aggregate to 5-minute timeframe: %w", err)
	}

	if len(aggregatedResult.Candles) == 0 {
		return fmt.Errorf("aggregation to 5-min candles produced no results")
	}

	log.Info("Aggregated %d 1-minute candles to %d 5-minute candles for %s",
		len(oneMinCandles), len(aggregatedResult.Candles), instrumentKey)

	// 4. Calculate indicators on aggregated data
	indicators, err := s.analyticsEngine.CalculateIndicators(ctx, &analytics.CandleData{
		Candles: s.aggregatedCandlesToCandles(aggregatedResult.Candles),
	})
	if err != nil {
		return fmt.Errorf("failed to calculate indicators: %w", err)
	}

	// 5. Enrich aggregated candles with calculated indicators
	enrichedCandles := s.enrichCandlesWithIndicators(aggregatedResult.Candles, indicators)

	// 6. Trigger BB width callback for each enriched candle
	if bbWidthCallback != nil {
		for _, candle := range enrichedCandles {
			if candle.BBWidth > 0 { // Only trigger if BB width is calculated
				bbWidthCallback(ctx, instrumentKey, candle)
			}
		}
	}

	// 7. Store the enriched 5-minute candles
	if err := s.Store5MinCandles(ctx, enrichedCandles); err != nil {
		return fmt.Errorf("failed to store 5-min candles: %w", err)
	}

	log.Info("Successfully processed and stored %d 5-minute candles with indicators for %s. Cache hits: %d, Processing time: %v",
		len(enrichedCandles), instrumentKey, processingResult.CacheHits, processingResult.ProcessTime)

	return nil
}

// Get5MinCandles retrieves 5-minute candles using DataFrame operations where beneficial
func (s *CandleAggregationServiceV2) Get5MinCandles(
	ctx context.Context,
	instrumentKey string,
	startTime, endTime time.Time,
) ([]domain.AggregatedCandle, error) {
	log.Info("Fetching 5-min candles for %s from %v to %v using DataFrame operations",
		instrumentKey, startTime, endTime)

	// Try to get from 5-minute repository first
	candles, err := s.candle5MinRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, startTime, endTime)
	if err == nil && len(candles) > 0 {
		// Convert domain.Candle5Min to domain.AggregatedCandle
		return s.convertCandle5MinToAggregated(candles), nil
	}

	// If not available, aggregate from 1-minute data using analytics engine
	oneMinCandles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, "1minute", startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 1-min candles for aggregation: %w", err)
	}

	if len(oneMinCandles) == 0 {
		return []domain.AggregatedCandle{}, nil
	}

	// Use analytics engine for aggregation
	candleData := &analytics.CandleData{Candles: oneMinCandles}
	aggregatedResult, err := s.analyticsEngine.AggregateTimeframes(ctx, candleData, "5m")
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate to 5-minute timeframe: %w", err)
	}

	log.Info("Aggregated %d 1-minute candles to %d 5-minute candles for %s",
		len(oneMinCandles), len(aggregatedResult.Candles), instrumentKey)

	return aggregatedResult.Candles, nil
}

// NotifyOnNew5MinCandles notifies listeners about new 5-minute candles using DataFrame processing
func (s *CandleAggregationServiceV2) NotifyOnNew5MinCandles(ctx context.Context, stock response.StockGroupStockDTO, start, end time.Time) error {
	log.Info("Notifying listeners for new 5-min candles for %s using DataFrame processing", stock.InstrumentKey)

	// Use DataFrame-based aggregation for notification processing
	err := s.Aggregate5MinCandlesWithIndicators(ctx, stock.InstrumentKey, start, end, func(ctx context.Context, instrumentKey string, candle domain.AggregatedCandle) {
		// Notify all registered listeners with the callback signature they expect
		for _, listener := range s.candleCloseListeners {
			listener([]domain.AggregatedCandle{candle}, stock)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to aggregate and notify 5-min candles: %w", err)
	}

	return nil
}

// Store5MinCandles stores 5-minute candles with indicators
func (s *CandleAggregationServiceV2) Store5MinCandles(ctx context.Context, candles []domain.AggregatedCandle) error {
	if len(candles) == 0 {
		return nil
	}

	log.Info("Storing %d 5-minute candles", len(candles))

	for _, candle := range candles {
		// Convert AggregatedCandle to Candle5Min for storage
		candle5Min := s.convertAggregatedToCandle5Min(candle)
		if err := s.candle5MinRepo.Store(ctx, &candle5Min); err != nil {
			return fmt.Errorf("failed to save 5-min candle for %s at %v: %w",
				candle.InstrumentKey, candle.Timestamp, err)
		}
	}

	return nil
}

// RegisterCandleCloseListener registers a listener for candle close events
func (s *CandleAggregationServiceV2) RegisterCandleCloseListener(listener CandleCloseListener) {
	s.candleCloseListeners = append(s.candleCloseListeners, listener)
}

// GetAnalyticsMetrics returns performance metrics from the analytics engine
func (s *CandleAggregationServiceV2) GetAnalyticsMetrics() interface{} {
	if processor, ok := s.analyticsEngine.(*analytics.Processor); ok {
		return processor.GetMetrics()
	}
	return nil
}

// aggregatedCandlesToCandles converts AggregatedCandle slice to Candle slice for indicator calculation
func (s *CandleAggregationServiceV2) aggregatedCandlesToCandles(aggregated []domain.AggregatedCandle) []domain.Candle {
	candles := make([]domain.Candle, len(aggregated))
	for i, agg := range aggregated {
		candles[i] = domain.Candle{
			InstrumentKey: agg.InstrumentKey,
			Timestamp:     agg.Timestamp,
			Open:          agg.Open,
			High:          agg.High,
			Low:           agg.Low,
			Close:         agg.Close,
			Volume:        agg.Volume,
			OpenInterest:  agg.OpenInterest,
		}
	}
	return candles
}

// enrichCandlesWithIndicators enriches aggregated candles with calculated indicators
func (s *CandleAggregationServiceV2) enrichCandlesWithIndicators(candles []domain.AggregatedCandle, indicators *analytics.IndicatorSet) []domain.AggregatedCandle {
	if indicators == nil || len(indicators.Timestamps) == 0 {
		return candles
	}

	// Create a map for fast indicator lookup by timestamp
	indicatorMap := make(map[time.Time]int)
	for i, ts := range indicators.Timestamps {
		indicatorMap[ts] = i
	}

	// Enrich candles with indicators
	enriched := make([]domain.AggregatedCandle, len(candles))
	for i, candle := range candles {
		enriched[i] = candle

		if idx, exists := indicatorMap[candle.Timestamp]; exists {
			// Safely assign indicators if they exist and index is valid
			if idx < len(indicators.MA9) {
				enriched[i].MA9 = indicators.MA9[idx]
			}
			if idx < len(indicators.BBUpper) {
				enriched[i].BBUpper = indicators.BBUpper[idx]
			}
			if idx < len(indicators.BBMiddle) {
				enriched[i].BBMiddle = indicators.BBMiddle[idx]
			}
			if idx < len(indicators.BBLower) {
				enriched[i].BBLower = indicators.BBLower[idx]
			}
			if idx < len(indicators.BBWidth) {
				enriched[i].BBWidth = indicators.BBWidth[idx]
			}
			if idx < len(indicators.BBWidthNormalized) {
				enriched[i].BBWidthNormalized = indicators.BBWidthNormalized[idx]
			}
			if idx < len(indicators.BBWidthNormalizedPercentage) {
				enriched[i].BBWidthNormalizedPercentage = indicators.BBWidthNormalizedPercentage[idx]
			}
			if idx < len(indicators.VWAP) {
				enriched[i].VWAP = indicators.VWAP[idx]
			}
			if idx < len(indicators.EMA5) {
				enriched[i].EMA5 = indicators.EMA5[idx]
			}
			if idx < len(indicators.EMA9) {
				enriched[i].EMA9 = indicators.EMA9[idx]
			}
			if idx < len(indicators.EMA50) {
				enriched[i].EMA50 = indicators.EMA50[idx]
			}
			if idx < len(indicators.ATR) {
				enriched[i].ATR = indicators.ATR[idx]
			}
			if idx < len(indicators.RSI) {
				enriched[i].RSI = indicators.RSI[idx]
			}
		}
	}

	return enriched
}

// convertCandle5MinToAggregated converts domain.Candle5Min slice to domain.AggregatedCandle slice
func (s *CandleAggregationServiceV2) convertCandle5MinToAggregated(candles []domain.Candle5Min) []domain.AggregatedCandle {
	aggregated := make([]domain.AggregatedCandle, len(candles))
	for i, candle := range candles {
		aggregated[i] = domain.AggregatedCandle{
			InstrumentKey:               candle.InstrumentKey,
			Timestamp:                   candle.Timestamp,
			Open:                        candle.Open,
			High:                        candle.High,
			Low:                         candle.Low,
			Close:                       candle.Close,
			Volume:                      candle.Volume,
			OpenInterest:                candle.OpenInterest,
			TimeInterval:                candle.TimeInterval,
			MA9:                         candle.MA9,
			BBUpper:                     candle.BBUpper,
			BBMiddle:                    candle.BBMiddle,
			BBLower:                     candle.BBLower,
			BBWidth:                     candle.BBWidth,
			BBWidthNormalized:           candle.BBWidthNormalized,
			BBWidthNormalizedPercentage: candle.BBWidthNormalizedPercentage,
			VWAP:                        candle.VWAP,
			EMA5:                        candle.EMA5,
			EMA9:                        candle.EMA9,
			EMA50:                       candle.EMA50,
			ATR:                         candle.ATR,
			RSI:                         candle.RSI,
		}
	}
	return aggregated
}

// convertAggregatedToCandle5Min converts domain.AggregatedCandle to domain.Candle5Min
func (s *CandleAggregationServiceV2) convertAggregatedToCandle5Min(candle domain.AggregatedCandle) domain.Candle5Min {
	return domain.Candle5Min{
		InstrumentKey:               candle.InstrumentKey,
		Timestamp:                   candle.Timestamp,
		Open:                        candle.Open,
		High:                        candle.High,
		Low:                         candle.Low,
		Close:                       candle.Close,
		Volume:                      candle.Volume,
		OpenInterest:                candle.OpenInterest,
		TimeInterval:                candle.TimeInterval,
		MA9:                         candle.MA9,
		BBUpper:                     candle.BBUpper,
		BBMiddle:                    candle.BBMiddle,
		BBLower:                     candle.BBLower,
		BBWidth:                     candle.BBWidth,
		BBWidthNormalized:           candle.BBWidthNormalized,
		BBWidthNormalizedPercentage: candle.BBWidthNormalizedPercentage,
		VWAP:                        candle.VWAP,
		EMA5:                        candle.EMA5,
		EMA9:                        candle.EMA9,
		EMA50:                       candle.EMA50,
		ATR:                         candle.ATR,
		RSI:                         candle.RSI,
	}
}
