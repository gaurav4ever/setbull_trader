package service

import (
	"context"
	"fmt"
	"time"

	"setbull_trader/internal/analytics/indicators"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
)

// TechnicalIndicatorServiceV2 provides GoNum-powered technical indicator calculations
type TechnicalIndicatorServiceV2 struct {
	candleRepo repository.CandleRepository

	// GoNum-powered calculators
	emaCalculator       *indicators.EMACalculator
	rsiCalculator       *indicators.RSICalculator
	bollingerCalculator *indicators.BollingerCalculator
	vwapCalculator      *indicators.VWAPCalculator
	atrCalculator       *indicators.ATRCalculator
	baseCalculator      *indicators.Calculator
}

// NewTechnicalIndicatorServiceV2 creates a new GoNum-powered technical indicator service
func NewTechnicalIndicatorServiceV2(candleRepo repository.CandleRepository) *TechnicalIndicatorServiceV2 {
	return &TechnicalIndicatorServiceV2{
		candleRepo:          candleRepo,
		emaCalculator:       indicators.NewEMACalculator(),
		rsiCalculator:       indicators.NewRSICalculator(),
		bollingerCalculator: indicators.NewBollingerCalculator(),
		vwapCalculator:      indicators.NewVWAPCalculator(),
		atrCalculator:       indicators.NewATRCalculator(),
		baseCalculator:      indicators.NewCalculator(),
	}
}

// IndicatorSet holds all calculated indicators for a set of candles
type IndicatorSet struct {
	MA9        []domain.IndicatorValue
	BBUpper    []domain.IndicatorValue
	BBMiddle   []domain.IndicatorValue
	BBLower    []domain.IndicatorValue
	BBWidth    []domain.IndicatorValue
	VWAP       []domain.IndicatorValue
	EMA5       []domain.IndicatorValue
	EMA9       []domain.IndicatorValue
	EMA50      []domain.IndicatorValue
	ATR        []domain.IndicatorValue
	RSI        []domain.IndicatorValue
	Timestamps []time.Time
}

// CalculateAllIndicators calculates all technical indicators efficiently using GoNum
func (s *TechnicalIndicatorServiceV2) CalculateAllIndicators(
	ctx context.Context,
	instrumentKey string,
	interval string,
	start, end time.Time,
) (*IndicatorSet, error) {
	// Get candles for the instrument
	candles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get candles: %w", err)
	}

	if len(candles) == 0 {
		log.Warn("No candles found for instrument %s", instrumentKey)
		return &IndicatorSet{}, nil
	}

	return s.CalculateIndicatorsFromCandles(candles)
}

// CalculateIndicatorsFromCandles calculates all indicators from provided candles
func (s *TechnicalIndicatorServiceV2) CalculateIndicatorsFromCandles(candles []domain.Candle) (*IndicatorSet, error) {
	if len(candles) == 0 {
		return &IndicatorSet{}, nil
	}

	log.Info("Calculating indicators for %d candles using GoNum-optimized algorithms", len(candles))

	// Extract timestamps
	timestamps := make([]time.Time, len(candles))
	for i, candle := range candles {
		timestamps[i] = candle.Timestamp
	}

	// Calculate EMAs efficiently (multiple periods at once)
	emaResults := s.emaCalculator.CalculateMultipleEMAs(candles, []int{5, 9, 50})

	// Calculate Bollinger Bands (includes width calculation)
	bbResult := s.bollingerCalculator.CalculateBollingerBands(candles, 20, 2.0)

	// Calculate other indicators
	rsiValues := s.rsiCalculator.CalculateRSI(candles, 14)
	vwapValues := s.vwapCalculator.CalculateVWAP(candles)
	atrValues := s.atrCalculator.CalculateATR(candles, 14)

	// Calculate MA9 using SMA
	ma9Values := s.calculateSMA(candles, 9)

	indicatorSet := &IndicatorSet{
		MA9:        ma9Values,
		BBUpper:    bbResult.Upper,
		BBMiddle:   bbResult.Middle,
		BBLower:    bbResult.Lower,
		BBWidth:    bbResult.Width,
		VWAP:       vwapValues,
		EMA5:       emaResults[5],
		EMA9:       emaResults[9],
		EMA50:      emaResults[50],
		ATR:        atrValues,
		RSI:        rsiValues,
		Timestamps: timestamps,
	}

	log.Info("Successfully calculated all indicators using GoNum optimization")
	return indicatorSet, nil
}

// CalculateEMA calculates EMA with GoNum optimization
func (s *TechnicalIndicatorServiceV2) CalculateEMA(
	ctx context.Context,
	instrumentKey string,
	period int,
	interval string,
	start, end time.Time,
) ([]domain.IndicatorValue, error) {
	// Validate inputs
	if period <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}

	// Get candles for the instrument
	candles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get candles: %w", err)
	}

	if len(candles) < period {
		return nil, fmt.Errorf("not enough data to calculate EMA, need at least %d candles", period)
	}

	log.Info("Calculating EMA%d for %s using GoNum optimization", period, instrumentKey)
	return s.emaCalculator.CalculateEMA(candles, period), nil
}

// CalculateRSI calculates RSI with GoNum optimization
func (s *TechnicalIndicatorServiceV2) CalculateRSI(
	ctx context.Context,
	instrumentKey string,
	period int,
	interval string,
	start, end time.Time,
) ([]domain.IndicatorValue, error) {
	// Validate inputs
	if period <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}

	// Get candles for the instrument
	candles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get candles: %w", err)
	}

	if len(candles) < period+1 {
		return nil, fmt.Errorf("not enough data to calculate RSI, need at least %d candles", period+1)
	}

	log.Info("Calculating RSI%d for %s using GoNum optimization", period, instrumentKey)
	return s.rsiCalculator.CalculateRSI(candles, period), nil
}

// CalculateATR calculates ATR with GoNum optimization
func (s *TechnicalIndicatorServiceV2) CalculateATR(
	ctx context.Context,
	instrumentKey string,
	period int,
	interval string,
	start, end time.Time,
) ([]domain.IndicatorValue, error) {
	// Validate inputs
	if period <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}

	// Get candles for the instrument
	candles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get candles: %w", err)
	}

	if len(candles) < period+1 {
		return nil, fmt.Errorf("not enough data to calculate ATR, need at least %d candles", period+1)
	}

	log.Info("Calculating ATR%d for %s using GoNum optimization", period, instrumentKey)
	return s.atrCalculator.CalculateATR(candles, period), nil
}

// CalculateBollingerBands calculates Bollinger Bands with GoNum optimization
func (s *TechnicalIndicatorServiceV2) CalculateBollingerBands(
	ctx context.Context,
	instrumentKey string,
	period int,
	multiplier float64,
	interval string,
	start, end time.Time,
) (upper, middle, lower []domain.IndicatorValue, err error) {
	// Validate inputs
	if period <= 0 {
		return nil, nil, nil, fmt.Errorf("period must be positive")
	}

	// Get candles for the instrument
	candles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, interval, start, end)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get candles: %w", err)
	}

	if len(candles) < period {
		return nil, nil, nil, fmt.Errorf("not enough data to calculate Bollinger Bands, need at least %d candles", period)
	}

	log.Info("Calculating Bollinger Bands (period=%d, multiplier=%.1f) for %s using GoNum optimization",
		period, multiplier, instrumentKey)

	upper, middle, lower = s.bollingerCalculator.CalculateBollingerBandsCompatible(candles, period, multiplier)
	return upper, middle, lower, nil
}

// CalculateVWAP calculates VWAP with GoNum optimization
func (s *TechnicalIndicatorServiceV2) CalculateVWAP(
	ctx context.Context,
	instrumentKey string,
	interval string,
	start, end time.Time,
) ([]domain.IndicatorValue, error) {
	// Get candles for the instrument
	candles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get candles: %w", err)
	}

	if len(candles) == 0 {
		return []domain.IndicatorValue{}, nil
	}

	log.Info("Calculating VWAP for %s using GoNum optimization", instrumentKey)
	return s.vwapCalculator.CalculateVWAP(candles), nil
}

// CalculateBBWidth calculates Bollinger Band Width
func (s *TechnicalIndicatorServiceV2) CalculateBBWidth(upper, middle, lower []domain.IndicatorValue) []domain.IndicatorValue {
	return s.bollingerCalculator.CalculateBBWidth(upper, middle, lower)
}

// CalculateBBWidthNormalized calculates normalized Bollinger Band Width
func (s *TechnicalIndicatorServiceV2) CalculateBBWidthNormalized(upper, middle, lower []domain.IndicatorValue) []domain.IndicatorValue {
	return s.bollingerCalculator.CalculateBBWidthNormalized(upper, middle, lower)
}

// Helper methods

// calculateSMA calculates Simple Moving Average using GoNum
func (s *TechnicalIndicatorServiceV2) calculateSMA(candles []domain.Candle, period int) []domain.IndicatorValue {
	if len(candles) == 0 {
		return []domain.IndicatorValue{}
	}

	// Extract close prices
	prices := make([]float64, len(candles))
	for i, candle := range candles {
		prices[i] = candle.Close
	}

	// Calculate SMA using base calculator
	smaValues := s.baseCalculator.SMA(prices, period)

	// Convert to domain model
	result := make([]domain.IndicatorValue, len(candles))
	for i, candle := range candles {
		result[i] = domain.IndicatorValue{
			Timestamp: candle.Timestamp,
			Value:     smaValues[i],
		}
	}

	return result
}

// GetServiceMetrics returns performance metrics for the service
func (s *TechnicalIndicatorServiceV2) GetServiceMetrics() map[string]interface{} {
	return map[string]interface{}{
		"service_type":         "GoNum-optimized",
		"version":              "v2",
		"supported_indicators": []string{"EMA", "SMA", "RSI", "ATR", "VWAP", "BollingerBands"},
		"optimization":         "vectorized_calculations",
		"backend":              "gonum.org/v1/gonum/stat",
	}
}
