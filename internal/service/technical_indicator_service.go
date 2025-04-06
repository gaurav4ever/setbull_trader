package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
)

// TechnicalIndicatorService provides operations for calculating technical indicators
type TechnicalIndicatorService struct {
	candleRepo repository.CandleRepository
}

// NewTechnicalIndicatorService creates a new technical indicator service
func NewTechnicalIndicatorService(candleRepo repository.CandleRepository) *TechnicalIndicatorService {
	return &TechnicalIndicatorService{
		candleRepo: candleRepo,
	}
}

// CalculateEMA calculates the Exponential Moving Average for the given period
func (s *TechnicalIndicatorService) CalculateEMA(
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

	// Calculate EMA
	// Formula: EMA = (Close - Previous EMA) * (2 / (Period + 1)) + Previous EMA
	multiplier := 2.0 / float64(period+1)

	// Start with SMA for the first EMA value
	sma := 0.0
	for i := 0; i < period; i++ {
		sma += candles[i].Close
	}
	sma /= float64(period)

	// Calculate subsequent EMA values
	values := make([]domain.IndicatorValue, len(candles)-period+1)
	values[0] = domain.IndicatorValue{
		Timestamp: candles[period-1].Timestamp,
		Value:     sma,
	}

	// Calculate remaining EMAs
	for i := period; i < len(candles); i++ {
		ema := (candles[i].Close-values[i-period].Value)*multiplier + values[i-period].Value
		values[i-period+1] = domain.IndicatorValue{
			Timestamp: candles[i].Timestamp,
			Value:     ema,
		}
	}

	// log.Info("Calculated EMA-%d for %s, found %d values",
	// 	period, instrumentKey, len(values))

	return values, nil
}

// CalculateRSI calculates the Relative Strength Index for the given period
func (s *TechnicalIndicatorService) CalculateRSI(
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

	if len(candles) <= period {
		return nil, fmt.Errorf("not enough data to calculate RSI, need more than %d candles", period)
	}

	// Calculate RSI
	// Step 1: Calculate price changes
	changes := make([]float64, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		changes[i-1] = candles[i].Close - candles[i-1].Close
	}

	// Step 2: Separate gains and losses
	gains := make([]float64, len(changes))
	losses := make([]float64, len(changes))
	for i, change := range changes {
		if change > 0 {
			gains[i] = change
		} else {
			losses[i] = math.Abs(change)
		}
	}

	// Step 3: Calculate average gains and losses for the first period
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Step 4: Calculate RSI values
	values := make([]domain.IndicatorValue, len(candles)-period)

	// First RSI value
	var rs, rsi float64
	if avgLoss == 0 {
		rsi = 100 // Prevent division by zero
	} else {
		rs = avgGain / avgLoss
		rsi = 100 - (100 / (1 + rs))
	}

	values[0] = domain.IndicatorValue{
		Timestamp: candles[period].Timestamp,
		Value:     rsi,
	}

	// Subsequent RSI values using smoothed method
	for i := period; i < len(changes); i++ {
		// Update average gain and loss using smoothing formula
		avgGain = ((avgGain * float64(period-1)) + gains[i]) / float64(period)
		avgLoss = ((avgLoss * float64(period-1)) + losses[i]) / float64(period)

		if avgLoss == 0 {
			rsi = 100 // Prevent division by zero
		} else {
			rs = avgGain / avgLoss
			rsi = 100 - (100 / (1 + rs))
		}

		values[i-period+1] = domain.IndicatorValue{
			Timestamp: candles[i+1].Timestamp,
			Value:     rsi,
		}
	}

	// log.Info("Calculated RSI-%d for %s, found %d values",
	// 	period, instrumentKey, len(values))

	return values, nil
}

// CalculateATR calculates the Average True Range for the given period
func (s *TechnicalIndicatorService) CalculateATR(
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

	if len(candles) <= period {
		return nil, fmt.Errorf("not enough data to calculate ATR, need more than %d candles", period)
	}

	// Calculate ATR
	// Step 1: Calculate True Range for each candle
	trueRanges := make([]float64, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		high := candles[i].High
		low := candles[i].Low
		prevClose := candles[i-1].Close

		// True Range is the greatest of:
		// 1. Current High - Current Low
		// 2. |Current High - Previous Close|
		// 3. |Current Low - Previous Close|
		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trueRanges[i-1] = math.Max(tr1, math.Max(tr2, tr3))
	}

	// Step 2: Calculate initial ATR (simple average for first period)
	var sum float64
	for i := 0; i < period; i++ {
		sum += trueRanges[i]
	}
	atr := sum / float64(period)

	// Step 3: Calculate subsequent ATR values using smoothing
	values := make([]domain.IndicatorValue, len(candles)-period)
	values[0] = domain.IndicatorValue{
		Timestamp: candles[period].Timestamp,
		Value:     atr,
	}

	// Use smoothed method: ATR = ((Period-1) * Previous ATR + Current TR) / Period
	for i := period; i < len(trueRanges); i++ {
		atr = ((atr * float64(period-1)) + trueRanges[i]) / float64(period)
		values[i-period+1] = domain.IndicatorValue{
			Timestamp: candles[i+1].Timestamp,
			Value:     atr,
		}
	}

	// log.Info("Calculated ATR-%d for %s, found %d values",
	// 	period, instrumentKey, len(values))

	return values, nil
}

// CalculateVolumeMA calculates the Volume Moving Average for the given period
func (s *TechnicalIndicatorService) CalculateVolumeMA(
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
		return nil, fmt.Errorf("not enough data to calculate Volume MA, need at least %d candles", period)
	}

	// Calculate Volume MA using a simple moving average
	values := make([]domain.IndicatorValue, len(candles)-period+1)

	// Calculate first value
	var sum int64
	for i := 0; i < period; i++ {
		sum += candles[i].Volume
	}
	values[0] = domain.IndicatorValue{
		Timestamp: candles[period-1].Timestamp,
		Value:     float64(sum) / float64(period),
	}

	// Calculate subsequent values
	for i := period; i < len(candles); i++ {
		sum = sum - candles[i-period].Volume + candles[i].Volume
		values[i-period+1] = domain.IndicatorValue{
			Timestamp: candles[i].Timestamp,
			Value:     float64(sum) / float64(period),
		}
	}

	// log.Info("Calculated Volume MA-%d for %s, found %d values",
	// 	period, instrumentKey, len(values))

	return values, nil
}

// CalculateMorningRange calculates the Morning Range based on the first 5-minute candle
func (s *TechnicalIndicatorService) CalculateMorningRange(
	ctx context.Context,
	instrumentKey string,
	dateStr string,
	atrMultiplier float64,
) (float64, error) {
	// Parse date string
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0, fmt.Errorf("invalid date format: %w", err)
	}

	// Calculate 9:15 AM to 9:20 AM time range for the given date (assuming Indian market hours)
	// Adjust according to your market's opening time
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 9, 15, 0, 0, date.Location())
	endTime := time.Date(date.Year(), date.Month(), date.Day(), 9, 20, 0, 0, date.Location())

	// Get the first 5-minute candle
	candles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, "1minute", startTime, endTime)
	if err != nil {
		return 0, fmt.Errorf("failed to get morning candles: %w", err)
	}

	if len(candles) == 0 {
		return 0, fmt.Errorf("no morning candles found for %s on %s", instrumentKey, dateStr)
	}

	// Calculate the high-low range of the first 5 minutes
	var highestHigh, lowestLow float64
	highestHigh = candles[0].High
	lowestLow = candles[0].Low

	for _, candle := range candles {
		if candle.High > highestHigh {
			highestHigh = candle.High
		}
		if candle.Low < lowestLow {
			lowestLow = candle.Low
		}
	}

	morningRange := highestHigh - lowestLow

	// Get ATR if atrMultiplier is provided
	if atrMultiplier > 0 {
		// Get ATR for previous N days
		endDate := date.AddDate(0, 0, -1)       // Yesterday
		startDate := endDate.AddDate(0, 0, -14) // 14 days before yesterday

		atrValues, err := s.CalculateATR(ctx, instrumentKey, 14, "day", startDate, endDate)
		if err != nil {
			log.Warn("Failed to calculate ATR for morning range: %v", err)
		} else if len(atrValues) > 0 {
			latestATR := atrValues[len(atrValues)-1].Value

			// Calculate MR using formula: Morning Range / (ATR * multiplier)
			if latestATR > 0 {
				normalizedMR := morningRange / (latestATR * atrMultiplier)
				log.Info("Calculated Morning Range for %s: %f (normalized with ATR: %f)",
					instrumentKey, morningRange, normalizedMR)
				return normalizedMR, nil
			}
		}
	}

	log.Info("Calculated Morning Range for %s: %f", instrumentKey, morningRange)
	return morningRange, nil
}

// CalculateAllIndicators calculates all technical indicators for a given instrument
func (s *TechnicalIndicatorService) CalculateAllIndicators(
	ctx context.Context,
	instrumentKey string,
	interval string,
	start, end time.Time,
) (*domain.TechnicalIndicators, error) {
	indicators := &domain.TechnicalIndicators{
		InstrumentKey: instrumentKey,
		Interval:      interval,
		StartTime:     start,
		EndTime:       end,
	}

	// Calculate EMA-9
	ema9, err := s.CalculateEMA(ctx, instrumentKey, 9, interval, start, end)
	if err != nil {
		log.Warn("Failed to calculate EMA-9: %v", err)
	} else {
		indicators.EMA9 = ema9
	}

	// Calculate EMA-50
	ema50, err := s.CalculateEMA(ctx, instrumentKey, 50, interval, start, end)
	if err != nil {
		log.Warn("Failed to calculate EMA-50: %v", err)
	} else {
		indicators.EMA50 = ema50
	}

	// Calculate RSI-14
	rsi14, err := s.CalculateRSI(ctx, instrumentKey, 14, interval, start, end)
	if err != nil {
		log.Warn("Failed to calculate RSI-14: %v", err)
	} else {
		indicators.RSI14 = rsi14
	}

	// Calculate ATR-14
	atr14, err := s.CalculateATR(ctx, instrumentKey, 14, interval, start, end)
	if err != nil {
		log.Warn("Failed to calculate ATR-14: %v", err)
	} else {
		indicators.ATR14 = atr14
	}

	// Calculate Volume MA-10 (10-day average volume)
	volumeMA10, err := s.CalculateVolumeMA(ctx, instrumentKey, 10, interval, start, end)
	if err != nil {
		log.Warn("Failed to calculate Volume MA-10: %v", err)
	} else {
		indicators.VolumeMA10 = volumeMA10
	}

	log.Info("Calculated all indicators for %s (%s)", instrumentKey, interval)
	return indicators, nil
}
