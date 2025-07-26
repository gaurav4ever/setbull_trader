package service

import (
	"context"
	"time"

	"setbull_trader/internal/domain"
)

// Interface definitions for backward compatibility between V1 and V2 services

// TechnicalIndicatorServiceInterface defines the common interface for V1 and V2 services
type TechnicalIndicatorServiceInterface interface {
	CalculateEMA(ctx context.Context, instrumentKey string, period int, interval string, start, end time.Time) ([]domain.IndicatorValue, error)
	CalculateRSI(ctx context.Context, instrumentKey string, period int, interval string, start, end time.Time) ([]domain.IndicatorValue, error)
	CalculateATR(ctx context.Context, instrumentKey string, period int, interval string, start, end time.Time) ([]domain.IndicatorValue, error)
	CalculateBollingerBands(ctx context.Context, instrumentKey string, period int, multiplier float64, interval string, start, end time.Time) (upper, middle, lower []domain.IndicatorValue, err error)
	CalculateVWAP(ctx context.Context, instrumentKey string, interval string, start, end time.Time) ([]domain.IndicatorValue, error)
}

// CandleAggregationServiceInterface defines the common interface for V1 and V2 services
type CandleAggregationServiceInterface interface {
	Get5MinCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error)
	Get5MinCandlesWithIndicators(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error)
	GetDailyCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error)
	Aggregate5MinCandlesWithIndicators(ctx context.Context, instrumentKey string, start, end time.Time, callback func(ctx context.Context, instrumentKey string, candle domain.AggregatedCandle)) error
	GetMultiTimeframeCandles(ctx context.Context, instrumentKey string, timeframes []string, start, end time.Time) (map[string][]domain.AggregatedCandle, error)
	Store5MinCandles(ctx context.Context, candles []domain.AggregatedCandle) error
	RegisterCandleCloseListener(listener CandleCloseListener)
}

// SequenceAnalyzerInterface defines the common interface for V1 and V2 services
type SequenceAnalyzerInterface interface {
	AnalyzeSequences(analysis domain.SequenceAnalysis) domain.SequenceMetrics
}

// TechnicalIndicatorServiceWrapper provides backward compatibility for technical indicators
type TechnicalIndicatorServiceWrapper struct {
	v1Service TechnicalIndicatorServiceInterface
	v2Service TechnicalIndicatorServiceInterface
	useV2     bool
}

// NewTechnicalIndicatorServiceWrapper creates a wrapper that can switch between V1 and V2
func NewTechnicalIndicatorServiceWrapper(v1Service, v2Service TechnicalIndicatorServiceInterface, useV2 bool) TechnicalIndicatorServiceInterface {
	return &TechnicalIndicatorServiceWrapper{
		v1Service: v1Service,
		v2Service: v2Service,
		useV2:     useV2,
	}
}

func (w *TechnicalIndicatorServiceWrapper) CalculateEMA(ctx context.Context, instrumentKey string, period int, interval string, start, end time.Time) ([]domain.IndicatorValue, error) {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.CalculateEMA(ctx, instrumentKey, period, interval, start, end)
	}
	return w.v1Service.CalculateEMA(ctx, instrumentKey, period, interval, start, end)
}

func (w *TechnicalIndicatorServiceWrapper) CalculateRSI(ctx context.Context, instrumentKey string, period int, interval string, start, end time.Time) ([]domain.IndicatorValue, error) {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.CalculateRSI(ctx, instrumentKey, period, interval, start, end)
	}
	return w.v1Service.CalculateRSI(ctx, instrumentKey, period, interval, start, end)
}

func (w *TechnicalIndicatorServiceWrapper) CalculateATR(ctx context.Context, instrumentKey string, period int, interval string, start, end time.Time) ([]domain.IndicatorValue, error) {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.CalculateATR(ctx, instrumentKey, period, interval, start, end)
	}
	return w.v1Service.CalculateATR(ctx, instrumentKey, period, interval, start, end)
}

func (w *TechnicalIndicatorServiceWrapper) CalculateBollingerBands(ctx context.Context, instrumentKey string, period int, multiplier float64, interval string, start, end time.Time) (upper, middle, lower []domain.IndicatorValue, err error) {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.CalculateBollingerBands(ctx, instrumentKey, period, multiplier, interval, start, end)
	}
	return w.v1Service.CalculateBollingerBands(ctx, instrumentKey, period, multiplier, interval, start, end)
}

func (w *TechnicalIndicatorServiceWrapper) CalculateVWAP(ctx context.Context, instrumentKey string, interval string, start, end time.Time) ([]domain.IndicatorValue, error) {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.CalculateVWAP(ctx, instrumentKey, interval, start, end)
	}
	return w.v1Service.CalculateVWAP(ctx, instrumentKey, interval, start, end)
}

// CandleAggregationServiceWrapper provides backward compatibility for candle aggregation
type CandleAggregationServiceWrapper struct {
	v1Service CandleAggregationServiceInterface
	v2Service CandleAggregationServiceInterface
	useV2     bool
}

// NewCandleAggregationServiceWrapper creates a wrapper that can switch between V1 and V2
func NewCandleAggregationServiceWrapper(v1Service, v2Service CandleAggregationServiceInterface, useV2 bool) CandleAggregationServiceInterface {
	return &CandleAggregationServiceWrapper{
		v1Service: v1Service,
		v2Service: v2Service,
		useV2:     useV2,
	}
}

func (w *CandleAggregationServiceWrapper) Get5MinCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.Get5MinCandles(ctx, instrumentKey, start, end)
	}
	return w.v1Service.Get5MinCandles(ctx, instrumentKey, start, end)
}

func (w *CandleAggregationServiceWrapper) Get5MinCandlesWithIndicators(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.Get5MinCandlesWithIndicators(ctx, instrumentKey, start, end)
	}
	return w.v1Service.Get5MinCandlesWithIndicators(ctx, instrumentKey, start, end)
}

func (w *CandleAggregationServiceWrapper) GetDailyCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.GetDailyCandles(ctx, instrumentKey, start, end)
	}
	return w.v1Service.GetDailyCandles(ctx, instrumentKey, start, end)
}

func (w *CandleAggregationServiceWrapper) Aggregate5MinCandlesWithIndicators(ctx context.Context, instrumentKey string, start, end time.Time, callback func(ctx context.Context, instrumentKey string, candle domain.AggregatedCandle)) error {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.Aggregate5MinCandlesWithIndicators(ctx, instrumentKey, start, end, callback)
	}
	return w.v1Service.Aggregate5MinCandlesWithIndicators(ctx, instrumentKey, start, end, callback)
}

func (w *CandleAggregationServiceWrapper) GetMultiTimeframeCandles(ctx context.Context, instrumentKey string, timeframes []string, start, end time.Time) (map[string][]domain.AggregatedCandle, error) {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.GetMultiTimeframeCandles(ctx, instrumentKey, timeframes, start, end)
	}
	return w.v1Service.GetMultiTimeframeCandles(ctx, instrumentKey, timeframes, start, end)
}

func (w *CandleAggregationServiceWrapper) Store5MinCandles(ctx context.Context, candles []domain.AggregatedCandle) error {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.Store5MinCandles(ctx, candles)
	}
	return w.v1Service.Store5MinCandles(ctx, candles)
}

func (w *CandleAggregationServiceWrapper) RegisterCandleCloseListener(listener CandleCloseListener) {
	if w.useV2 && w.v2Service != nil {
		w.v2Service.RegisterCandleCloseListener(listener)
	} else {
		w.v1Service.RegisterCandleCloseListener(listener)
	}
}

// SequenceAnalyzerWrapper provides backward compatibility for sequence analysis
type SequenceAnalyzerWrapper struct {
	v1Service SequenceAnalyzerInterface
	v2Service SequenceAnalyzerInterface
	useV2     bool
}

// NewSequenceAnalyzerWrapper creates a wrapper that can switch between V1 and V2
func NewSequenceAnalyzerWrapper(v1Service, v2Service SequenceAnalyzerInterface, useV2 bool) SequenceAnalyzerInterface {
	return &SequenceAnalyzerWrapper{
		v1Service: v1Service,
		v2Service: v2Service,
		useV2:     useV2,
	}
}

func (w *SequenceAnalyzerWrapper) AnalyzeSequences(analysis domain.SequenceAnalysis) domain.SequenceMetrics {
	if w.useV2 && w.v2Service != nil {
		return w.v2Service.AnalyzeSequences(analysis)
	}
	return w.v1Service.AnalyzeSequences(analysis)
}
