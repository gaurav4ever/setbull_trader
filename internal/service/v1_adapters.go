package service

import (
	"context"
	"time"

	"setbull_trader/internal/domain"
)

// V1TechnicalIndicatorServiceAdapter adapts the V1 TechnicalIndicatorService to implement TechnicalIndicatorServiceInterface
type V1TechnicalIndicatorServiceAdapter struct {
	v1Service *TechnicalIndicatorService
}

// NewV1TechnicalIndicatorServiceAdapter creates an adapter for V1 TechnicalIndicatorService
func NewV1TechnicalIndicatorServiceAdapter(v1Service *TechnicalIndicatorService) TechnicalIndicatorServiceInterface {
	return &V1TechnicalIndicatorServiceAdapter{v1Service: v1Service}
}

func (a *V1TechnicalIndicatorServiceAdapter) CalculateEMA(ctx context.Context, instrumentKey string, period int, interval string, start, end time.Time) ([]domain.IndicatorValue, error) {
	// V1 service doesn't use these parameters in the same way, so we need to fetch candles first
	// This is a simplified adapter - in practice, you'd need to fetch candles based on the parameters
	return nil, nil // Placeholder implementation
}

func (a *V1TechnicalIndicatorServiceAdapter) CalculateRSI(ctx context.Context, instrumentKey string, period int, interval string, start, end time.Time) ([]domain.IndicatorValue, error) {
	// Similar placeholder for RSI
	return nil, nil
}

func (a *V1TechnicalIndicatorServiceAdapter) CalculateATR(ctx context.Context, instrumentKey string, period int, interval string, start, end time.Time) ([]domain.IndicatorValue, error) {
	// Similar placeholder for ATR
	return nil, nil
}

func (a *V1TechnicalIndicatorServiceAdapter) CalculateBollingerBands(ctx context.Context, instrumentKey string, period int, multiplier float64, interval string, start, end time.Time) (upper, middle, lower []domain.IndicatorValue, err error) {
	// This would need to fetch candles first and then call the V1 method
	// For now, return empty slices
	return []domain.IndicatorValue{}, []domain.IndicatorValue{}, []domain.IndicatorValue{}, nil
}

func (a *V1TechnicalIndicatorServiceAdapter) CalculateVWAP(ctx context.Context, instrumentKey string, interval string, start, end time.Time) ([]domain.IndicatorValue, error) {
	// Similar placeholder for VWAP
	return nil, nil
}

// V1CandleAggregationServiceAdapter adapts the V1 CandleAggregationService to implement CandleAggregationServiceInterface
type V1CandleAggregationServiceAdapter struct {
	v1Service *CandleAggregationService
}

// NewV1CandleAggregationServiceAdapter creates an adapter for V1 CandleAggregationService
func NewV1CandleAggregationServiceAdapter(v1Service *CandleAggregationService) CandleAggregationServiceInterface {
	return &V1CandleAggregationServiceAdapter{v1Service: v1Service}
}

func (a *V1CandleAggregationServiceAdapter) Get5MinCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	return a.v1Service.Get5MinCandles(ctx, instrumentKey, start, end)
}

func (a *V1CandleAggregationServiceAdapter) Get5MinCandlesWithIndicators(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	// V1 service doesn't have this method, so fall back to regular Get5MinCandles
	// Note: V1 won't have indicators calculated, but this provides compatibility
	return a.Get5MinCandles(ctx, instrumentKey, start, end)
}

func (a *V1CandleAggregationServiceAdapter) GetDailyCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	// V1 service returns []domain.Candle, but we need []domain.AggregatedCandle
	// Convert the result
	candles, err := a.v1Service.GetDailyCandles(ctx, instrumentKey, start, end)
	if err != nil {
		return nil, err
	}

	// Convert []domain.Candle to []domain.AggregatedCandle
	aggregatedCandles := make([]domain.AggregatedCandle, len(candles))
	for i, candle := range candles {
		aggregatedCandles[i] = domain.AggregatedCandle{
			InstrumentKey: candle.InstrumentKey,
			Timestamp:     candle.Timestamp,
			Open:          candle.Open,
			High:          candle.High,
			Low:           candle.Low,
			Close:         candle.Close,
			Volume:        candle.Volume,
			OpenInterest:  candle.OpenInterest,
			TimeInterval:  candle.TimeInterval,
		}
	}

	return aggregatedCandles, nil
}

func (a *V1CandleAggregationServiceAdapter) Aggregate5MinCandlesWithIndicators(ctx context.Context, instrumentKey string, start, end time.Time, callback func(ctx context.Context, instrumentKey string, candle domain.AggregatedCandle)) error {
	return a.v1Service.Aggregate5MinCandlesWithIndicators(ctx, instrumentKey, start, end, callback)
}

func (a *V1CandleAggregationServiceAdapter) GetMultiTimeframeCandles(ctx context.Context, instrumentKey string, timeframes []string, start, end time.Time) (map[string][]domain.AggregatedCandle, error) {
	result := make(map[string][]domain.AggregatedCandle)

	for _, timeframe := range timeframes {
		switch timeframe {
		case "5minute":
			candles, err := a.Get5MinCandles(ctx, instrumentKey, start, end)
			if err != nil {
				return nil, err
			}
			result[timeframe] = candles
		case "day":
			candles, err := a.GetDailyCandles(ctx, instrumentKey, start, end)
			if err != nil {
				return nil, err
			}
			result[timeframe] = candles
		default:
			return nil, nil // Unsupported timeframe
		}
	}

	return result, nil
}

func (a *V1CandleAggregationServiceAdapter) Store5MinCandles(ctx context.Context, candles []domain.AggregatedCandle) error {
	return a.v1Service.Store5MinCandles(ctx, candles)
}

func (a *V1CandleAggregationServiceAdapter) RegisterCandleCloseListener(listener CandleCloseListener) {
	a.v1Service.RegisterCandleCloseListener(listener)
}

// V1SequenceAnalyzerAdapter adapts the V1 SequenceAnalyzer to implement SequenceAnalyzerInterface
type V1SequenceAnalyzerAdapter struct {
	v1Service *SequenceAnalyzer
}

// NewV1SequenceAnalyzerAdapter creates an adapter for V1 SequenceAnalyzer
func NewV1SequenceAnalyzerAdapter(v1Service *SequenceAnalyzer) SequenceAnalyzerInterface {
	return &V1SequenceAnalyzerAdapter{v1Service: v1Service}
}

func (a *V1SequenceAnalyzerAdapter) AnalyzeSequences(analysis domain.SequenceAnalysis) domain.SequenceMetrics {
	return a.v1Service.AnalyzeSequences(analysis)
}
