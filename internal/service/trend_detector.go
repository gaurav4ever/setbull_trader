package service

import (
	"context"
	"fmt"
	"setbull_trader/internal/domain"
)

type TrendDetector struct {
	technicalIndicators *TechnicalIndicatorService
	emaPeriod           int
	tradingCalendar     *TradingCalendarService
}

func NewTrendDetector(technicalIndicators *TechnicalIndicatorService, tradingCalendar *TradingCalendarService) *TrendDetector {
	return &TrendDetector{
		technicalIndicators: technicalIndicators,
		emaPeriod:           50,
		tradingCalendar:     tradingCalendar,
	}
}

// GetCandleEMAs calculates 50 EMA for each candle by looking back 50 trading days
func (td *TrendDetector) GetCandleEMAs(
	ctx context.Context,
	candles []domain.Candle) ([]domain.IndicatorValue, error) {

	emaValuesFinal := make([]domain.IndicatorValue, len(candles))

	// For each candle, calculate its EMA using previous 50 trading days
	for i, currentCandle := range candles {
		// Get start date (50 trading days back) for this candle
		endDate := currentCandle.Timestamp
		startDate := td.tradingCalendar.SubtractTradingDays(endDate, td.emaPeriod+10)

		// Calculate EMA for this specific candle using its historical data
		emaValues, err := td.technicalIndicators.CalculateEMA(
			ctx,
			currentCandle.InstrumentKey,
			td.emaPeriod,
			"day",
			startDate,
			endDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate EMA for %v: %w",
				currentCandle.Timestamp, err)
		}

		emaValuesFinal[i] = emaValues[len(emaValues)-1]
	}

	return emaValuesFinal, nil
}

// CalculateEMA calculates Exponential Moving Average
func (td *TrendDetector) CalculateEMA(candles []domain.Candle) []float64 {
	if len(candles) == 0 {
		return []float64{}
	}

	// Initialize EMA array
	emaValues := make([]float64, len(candles))

	// Calculate multiplier
	multiplier := 2.0 / float64(td.emaPeriod+1)

	// Calculate first SMA as initial EMA
	sum := 0.0
	for i := 0; i < td.emaPeriod && i < len(candles); i++ {
		sum += candles[i].Close
	}

	// Set initial EMA
	if len(candles) < td.emaPeriod {
		emaValues[0] = sum / float64(len(candles))
	} else {
		emaValues[0] = sum / float64(td.emaPeriod)
	}

	// Calculate EMA for remaining periods
	for i := 1; i < len(candles); i++ {
		closePrice := candles[i].Close
		previousEMA := emaValues[i-1]
		emaValues[i] = (closePrice-previousEMA)*multiplier + previousEMA
	}

	return emaValues
}

// AnalyzeTrend determines the overall trend based on price position relative to EMA
func (td *TrendDetector) AnalyzeTrend(ctx context.Context, candles []domain.Candle) domain.TrendAnalysis {
	if len(candles) == 0 {
		return domain.TrendAnalysis{Type: domain.NeutralTrend}
	}

	emaValues, err := td.GetCandleEMAs(ctx, candles)
	if err != nil {
		return domain.TrendAnalysis{Type: domain.NeutralTrend}
	}

	analysis := domain.TrendAnalysis{
		EMAValues: emaValues,
		StartDate: candles[0].Timestamp,
		EndDate:   candles[len(candles)-1].Timestamp,
	}

	// Count positions relative to EMA
	for i, candle := range candles {
		if i < len(emaValues) {
			if candle.Close < emaValues[i].Value {
				analysis.BelowEMACount++
			} else {
				analysis.AboveEMACount++
			}
		}
	}

	// Calculate percentage
	totalCandles := float64(analysis.BelowEMACount + analysis.AboveEMACount)
	if totalCandles > 0 {
		belowPercentage := float64(analysis.BelowEMACount) / totalCandles * 100
		abovePercentage := float64(analysis.AboveEMACount) / totalCandles * 100

		// Store the dominant percentage
		if belowPercentage > abovePercentage {
			analysis.TrendPercentage = belowPercentage
		} else {
			analysis.TrendPercentage = abovePercentage
		}

		// Determine trend based on 60% threshold
		if belowPercentage >= 60 {
			analysis.Type = domain.BearishTrend
		} else if abovePercentage >= 60 {
			analysis.Type = domain.BullishTrend
		} else {
			analysis.Type = domain.NeutralTrend
		}
	}

	return analysis
}

// ValidateTrendStrength ensures the trend is strong enough for analysis
func (td *TrendDetector) ValidateTrendStrength(analysis domain.TrendAnalysis) bool {
	return analysis.TrendPercentage >= 60
}
