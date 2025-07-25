package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"

	"github.com/cinar/indicator"
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

// CalculateRSIV2 calculates the Relative Strength Index for the given period
func (s *TechnicalIndicatorService) CalculateRSIV2(candles []domain.Candle, period int) []domain.IndicatorValue {
	// Note: The cinar/indicator Rsi function uses a default period of 14.
	// The period parameter is ignored for now to use the library's high-level function.
	const rsiPeriod = 14
	if len(candles) <= rsiPeriod {
		return nil
	}
	// reverse the candles
	reverseCandles := make([]domain.Candle, len(candles))
	for i, c := range candles {
		reverseCandles[len(candles)-1-i] = c
	}
	closePrices, _, _, _, _ := candlesToFloat64Slices(reverseCandles)
	rsiValues, _ := indicator.Rsi(closePrices)
	// round to 2 decimal places
	for i, v := range rsiValues {
		rsiValues[i] = math.Round(v*100) / 100
	}

	indicatorValues := make([]domain.IndicatorValue, len(candles))
	offset := len(candles) - len(rsiValues)
	for i, v := range rsiValues {
		if i+offset < len(candles) {
			indicatorValues[i+offset] = domain.IndicatorValue{
				Timestamp: reverseCandles[i+offset].Timestamp,
				Value:     v,
			}
		}
	}
	reverseIndicatorValues := make([]domain.IndicatorValue, len(indicatorValues))
	for i, v := range indicatorValues {
		reverseIndicatorValues[len(indicatorValues)-1-i] = v
	}
	return reverseIndicatorValues
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

// CalculateATRV2 calculates the Average True Range for the given period
func (s *TechnicalIndicatorService) CalculateATRV2(candles []domain.Candle, period int) []domain.IndicatorValue {
	if period <= 0 || len(candles) <= period {
		return nil
	}
	// reverse the candles
	reverseCandles := make([]domain.Candle, len(candles))
	for i, c := range candles {
		reverseCandles[len(candles)-1-i] = c
	}
	closePrices, _, highPrices, lowPrices, _ := candlesToFloat64Slices(reverseCandles)
	atrValues, _ := indicator.Atr(period, highPrices, lowPrices, closePrices)

	indicatorValues := make([]domain.IndicatorValue, len(candles))
	offset := len(candles) - len(atrValues)
	for i, v := range atrValues {
		if i+offset < len(candles) {
			indicatorValues[i+offset] = domain.IndicatorValue{
				Timestamp: reverseCandles[i+offset].Timestamp,
				Value:     v,
			}
		}
	}
	reverseIndicatorValues := make([]domain.IndicatorValue, len(indicatorValues))
	for i, v := range indicatorValues {
		reverseIndicatorValues[len(indicatorValues)-1-i] = v
	}
	return reverseIndicatorValues
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

	// Calculate Bollinger Bands (20, 2)
	bbUpper, bbMiddle, bbLower, err := s.CalculateBollingerBandsForRange(ctx, instrumentKey, 20, 2, interval, start, end)
	if err != nil {
		log.Warn("Failed to calculate Bollinger Bands: %v", err)
	} else {
		indicators.BBUpper = bbUpper
		indicators.BBMiddle = bbMiddle
		indicators.BBLower = bbLower

		// Calculate BBWidth from the results
		bbWidth, err := s.CalculateBBWidthForRange(bbUpper, bbLower, bbMiddle)
		if err != nil {
			log.Warn("Failed to calculate BBWidth: %v", err)
		} else {
			indicators.BBWidth = bbWidth
		}
	}

	log.Info("Calculated all indicators for %s (%s)", instrumentKey, interval)
	return indicators, nil
}

// CalculateBollingerBandsForRange fetches the required candles and calculates Bollinger Bands.
// It handles fetching extra "warm-up" data to ensure indicators are present for the requested range.
func (s *TechnicalIndicatorService) CalculateBollingerBandsForRange(
	ctx context.Context,
	instrumentKey string,
	period int,
	stddev float64,
	interval string,
	start, end time.Time,
) (upper, middle, lower []domain.IndicatorValue, err error) {
	if period <= 0 {
		return nil, nil, nil, fmt.Errorf("period must be positive")
	}

	// Fetch extended candle data for warm-up
	// Note: This estimation is not perfect for daily intervals with weekends/holidays.
	// A more robust solution might involve fetching by limit/count rather than time.
	estimatedLookbackDuration := time.Duration(period-1) * 24 * time.Hour // Simple estimation for 'day'
	extendedStart := start.Add(-estimatedLookbackDuration)

	log.Info("Calculating Bollinger Bands for %s, interval: %s, extendedStart: %s, end: %s", instrumentKey, interval, extendedStart, end)

	candles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, interval, extendedStart, end)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get candles for BBands: %w", err)
	}

	if len(candles) < period {
		return nil, nil, nil, fmt.Errorf("not enough data to calculate BBands, need at least %d candles, got %d", period, len(candles))
	}

	// Calculate indicators on the full (warm-up + requested) data
	bbUpperAll, bbMiddleAll, bbLowerAll := s.CalculateBollingerBands(candles, period, stddev)

	// Find the starting index of our requested time range
	startIndex := 0
	for i, c := range candles {
		if !c.Timestamp.Before(start) {
			startIndex = i
			break
		}
	}

	// The 'All' slices are already padded and aligned with the 'candles' slice.
	// We just need to slice them from the calculated startIndex to get the requested range.
	if startIndex >= len(bbMiddleAll) {
		return []domain.IndicatorValue{}, []domain.IndicatorValue{}, []domain.IndicatorValue{}, nil
	}

	upper = bbUpperAll[startIndex:]
	middle = bbMiddleAll[startIndex:]
	lower = bbLowerAll[startIndex:]

	return upper, middle, lower, nil
}

// CalculateSMA calculates the Simple Moving Average for the given period
func (s *TechnicalIndicatorService) CalculateSMA(candles []domain.Candle, period int) []domain.IndicatorValue {
	if period <= 0 || len(candles) < period {
		return nil
	}
	closePrices, _, _, _, _ := candlesToFloat64Slices(candles)
	smaValues := indicator.Sma(period, closePrices)

	indicatorValues := make([]domain.IndicatorValue, len(candles))
	offset := len(candles) - len(smaValues)
	for i, v := range smaValues {
		if i+offset < len(candles) {
			indicatorValues[i+offset] = domain.IndicatorValue{
				Timestamp: candles[i+offset].Timestamp,
				Value:     v,
			}
		}
	}
	return indicatorValues
}

// CalculateEMAV2 calculates the Exponential Moving Average for the given period
func (s *TechnicalIndicatorService) CalculateEMAV2(candles []domain.Candle, period int) []domain.IndicatorValue {
	if period <= 0 || len(candles) < period {
		return nil
	}
	// reverse the candles
	reverseCandles := make([]domain.Candle, len(candles))
	for i, c := range candles {
		reverseCandles[len(candles)-1-i] = c
	}
	closePrices, _, _, _, _ := candlesToFloat64Slices(reverseCandles)
	emaValues := indicator.Ema(period, closePrices)

	indicatorValues := make([]domain.IndicatorValue, len(candles))
	offset := len(candles) - len(emaValues)
	for i, v := range emaValues {
		if i+offset < len(candles) {
			indicatorValues[i+offset] = domain.IndicatorValue{
				Timestamp: reverseCandles[i+offset].Timestamp,
				Value:     v,
			}
		}
	}
	reverseIndicatorValues := make([]domain.IndicatorValue, len(indicatorValues))
	for i, v := range indicatorValues {
		reverseIndicatorValues[len(indicatorValues)-1-i] = v
	}
	return reverseIndicatorValues
}

// CalculateBollingerBandsTradingViewCompatible implements TradingView-exact BB calculation
// Uses direct standard deviation formula √(Σ(x-μ)²/n) instead of cinar's √(Σ(x²)/n - μ²)
// This fixes the numerical precision issues causing the 20-candle delay
func (s *TechnicalIndicatorService) CalculateBollingerBandsTradingViewCompatible(candles []domain.Candle, period int, multiplier float64) (upper, middle, lower []domain.IndicatorValue) {
	if len(candles) < period {
		return nil, nil, nil
	}

	// STEP 1: Reverse candles to process from oldest to newest (chronological order)
	// Because the input candles are ordered from newest to oldest (2025-07-18 → 2025-07-14)
	// But BB calculation needs chronological order (2025-07-14 → 2025-07-18)
	reverseCandles := make([]domain.Candle, len(candles))
	for i, c := range candles {
		reverseCandles[len(candles)-1-i] = c
	}

	// STEP 2: Calculate BB on chronologically ordered candles
	tempUpper := make([]domain.IndicatorValue, len(reverseCandles))
	tempMiddle := make([]domain.IndicatorValue, len(reverseCandles))
	tempLower := make([]domain.IndicatorValue, len(reverseCandles))

	// Initialize all values with zero timestamps first
	for i := 0; i < len(reverseCandles); i++ {
		tempUpper[i] = domain.IndicatorValue{Timestamp: reverseCandles[i].Timestamp, Value: 0.0}
		tempMiddle[i] = domain.IndicatorValue{Timestamp: reverseCandles[i].Timestamp, Value: 0.0}
		tempLower[i] = domain.IndicatorValue{Timestamp: reverseCandles[i].Timestamp, Value: 0.0}
	}

	// Calculate BB for each position starting from period-1 (chronologically)
	for i := period - 1; i < len(reverseCandles); i++ {
		// Calculate SMA (Middle Band) with high precision
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += reverseCandles[j].Close
		}
		sma := sum / float64(period)

		// Calculate Standard Deviation using TradingView method: √(Σ(x-μ)²/n)
		// This avoids the precision loss in cinar's √(Σ(x²)/n - μ²) formula
		sumSquaredDiff := 0.0
		for j := i - period + 1; j <= i; j++ {
			diff := reverseCandles[j].Close - sma
			sumSquaredDiff += diff * diff
		}

		// Use population standard deviation (divide by n, not n-1)
		variance := sumSquaredDiff / float64(period)
		stdDev := math.Sqrt(variance)

		// Calculate Bollinger Bands
		upperBand := sma + (multiplier * stdDev)
		lowerBand := sma - (multiplier * stdDev)

		// Store results with proper timestamps
		tempUpper[i] = domain.IndicatorValue{
			Timestamp: reverseCandles[i].Timestamp,
			Value:     upperBand,
		}
		tempMiddle[i] = domain.IndicatorValue{
			Timestamp: reverseCandles[i].Timestamp,
			Value:     sma,
		}
		tempLower[i] = domain.IndicatorValue{
			Timestamp: reverseCandles[i].Timestamp,
			Value:     lowerBand,
		}
	}

	// STEP 3: Reverse the results back to match original candle order (newest to oldest)
	upper = make([]domain.IndicatorValue, len(candles))
	middle = make([]domain.IndicatorValue, len(candles))
	lower = make([]domain.IndicatorValue, len(candles))

	for i := 0; i < len(tempUpper); i++ {
		reverseIndex := len(tempUpper) - 1 - i
		upper[i] = tempUpper[reverseIndex]
		middle[i] = tempMiddle[reverseIndex]
		lower[i] = tempLower[reverseIndex]
	}

	return upper, middle, lower
}

// CalculateBollingerBands calculates Bollinger Bands for the given period and stddev
// UPDATED: Now uses TradingView-compatible calculation to fix 20-candle delay issue
func (s *TechnicalIndicatorService) CalculateBollingerBands(candles []domain.Candle, period int, stddev float64) (upper, middle, lower []domain.IndicatorValue) {
	// Use the new TradingView-compatible implementation instead of cinar/indicator
	return s.CalculateBollingerBandsTradingViewCompatible(candles, period, stddev)
}

// DEPRECATED: Old cinar/indicator implementation - REMOVED due to data ordering issues
// This method has been deprecated in favor of CalculateBollingerBandsTradingViewCompatible
// which expects data in Past → Latest order (industry standard)
func (s *TechnicalIndicatorService) CalculateBollingerBandsOld(candles []domain.Candle, period int, stddev float64) (upper, middle, lower []domain.IndicatorValue) {
	// DEPRECATED: Use CalculateBollingerBandsTradingViewCompatible instead
	// This method had data ordering issues and has been removed
	log.Warn("CalculateBollingerBandsOld is deprecated - use CalculateBollingerBandsTradingViewCompatible")
	return s.CalculateBollingerBandsTradingViewCompatible(candles, period, stddev)
}

// CalculateVWAP calculates the Volume Weighted Average Price for the given candles (reset daily if needed)
func (s *TechnicalIndicatorService) CalculateVWAP(candles []domain.Candle) []domain.IndicatorValue {
	if len(candles) == 0 {
		return nil
	}
	values := make([]domain.IndicatorValue, len(candles))
	var cumPV, cumVol float64
	var currentDay int
	for i, c := range candles {
		// Reset VWAP at the start of a new day
		day := c.Timestamp.YearDay()
		if i == 0 || day != currentDay {
			cumPV = 0
			cumVol = 0
			currentDay = day
		}
		cumPV += c.Close * float64(c.Volume)
		cumVol += float64(c.Volume)
		vwap := 0.0
		if cumVol > 0 {
			vwap = cumPV / cumVol
		}
		values[i] = domain.IndicatorValue{
			Timestamp: c.Timestamp,
			Value:     vwap,
		}
	}
	return values
}

// AggregatedCandlesToCandles converts a slice of AggregatedCandle to a slice of Candle for indicator calculation reuse
func AggregatedCandlesToCandles(aggs []domain.AggregatedCandle) []domain.Candle {
	candles := make([]domain.Candle, len(aggs))
	for i, a := range aggs {
		candles[i] = domain.Candle{
			InstrumentKey: a.InstrumentKey,
			Timestamp:     a.Timestamp,
			Open:          a.Open,
			High:          a.High,
			Low:           a.Low,
			Close:         a.Close,
			Volume:        a.Volume,
			OpenInterest:  a.OpenInterest,
			TimeInterval:  a.TimeInterval,
		}
	}
	// reverse the candles
	reverseCandles := make([]domain.Candle, len(candles))
	for i, c := range candles {
		reverseCandles[len(candles)-1-i] = c
	}
	return reverseCandles
}

// CalculateBBWidth calculates the Bollinger Band width for each candle
// Uses custom implementation to avoid cinar/indicator library precision issues
func (s *TechnicalIndicatorService) CalculateBBWidth(bbUpper, bbLower, bbMiddle []domain.IndicatorValue) []domain.IndicatorValue {
	if len(bbUpper) != len(bbLower) || len(bbUpper) != len(bbMiddle) {
		return nil
	}

	widths := make([]domain.IndicatorValue, len(bbUpper))
	for i := range bbUpper {
		if bbMiddle[i].Timestamp.IsZero() || bbMiddle[i].Value == 0 {
			// Skip invalid data points
			widths[i] = domain.IndicatorValue{
				Timestamp: bbUpper[i].Timestamp,
				Value:     0.0,
			}
			continue
		}

		// Calculate BB Width using TradingView formula: (Upper - Lower) / Middle
		upper := bbUpper[i].Value
		lower := bbLower[i].Value
		middle := bbMiddle[i].Value

		// Validate that bands are in correct order
		if upper < lower {
			log.Warn("Invalid BB bands: upper (%f) < lower (%f) for timestamp %v", upper, lower, bbUpper[i].Timestamp)
			widths[i] = domain.IndicatorValue{
				Timestamp: bbUpper[i].Timestamp,
				Value:     0.0,
			}
			continue
		}

		// Calculate BB Width: upper - lower (absolute difference)
		bbWidth := upper - lower

		// Log the calculation for debugging
		// log.Info("BB Width calculation: upper=%f, lower=%f, middle=%f, bbWidth=%f for timestamp %v",
		// 	upper, lower, middle, bbWidth, bbUpper[i].Timestamp)

		// Validate the calculated value is reasonable
		if math.IsNaN(bbWidth) || math.IsInf(bbWidth, 0) || bbWidth < 0 {
			log.Warn("Invalid BB width calculated: %f for timestamp %v (upper: %f, lower: %f, middle: %f)",
				bbWidth, bbUpper[i].Timestamp, upper, lower, middle)
			bbWidth = 0.0
		}

		// Cap extremely large values to prevent database overflow
		if bbWidth > 100.0 { // 10000% BB width is extremely high
			log.Warn("BB width too large (%f), capping to 100.0 for timestamp %v", bbWidth, bbUpper[i].Timestamp)
			bbWidth = 100.0
		}

		// Additional safety check for extremely large values that might cause MySQL overflow
		if bbWidth > 1e15 { // 1 quadrillion is way beyond reasonable BB width
			log.Error("BB width extremely large (%f), setting to 0.0 for timestamp %v", bbWidth, bbUpper[i].Timestamp)
			bbWidth = 0.0
		}

		// MySQL DOUBLE limit check - prevent values larger than 1e308
		if bbWidth > 1e308 {
			log.Error("BB width exceeds MySQL DOUBLE limit (%f), setting to 0.0 for timestamp %v", bbWidth, bbUpper[i].Timestamp)
			bbWidth = 0.0
		}

		// Additional safety check - cap at a reasonable maximum value
		if bbWidth > 1e6 { // 1 million is way beyond reasonable BB width
			log.Error("BB width unreasonably large (%f), capping to 1.0 for timestamp %v", bbWidth, bbUpper[i].Timestamp)
			bbWidth = 1.0
		}

		widths[i] = domain.IndicatorValue{
			Timestamp: bbUpper[i].Timestamp,
			Value:     bbWidth,
		}
	}
	return widths
}

// CalculateBBWidthForRange calculates Bollinger Band Width from existing band values.
func (s *TechnicalIndicatorService) CalculateBBWidthForRange(bbUpper, bbLower, bbMiddle []domain.IndicatorValue) ([]domain.IndicatorValue, error) {
	if len(bbUpper) != len(bbLower) || len(bbUpper) != len(bbMiddle) {
		return nil, fmt.Errorf("Bollinger Band slices have different lengths")
	}
	return s.CalculateBBWidth(bbUpper, bbLower, bbMiddle), nil
}

// CalculateBBWidthNormalized calculates the normalized Bollinger Band width: (upper - lower) / middle
func (s *TechnicalIndicatorService) CalculateBBWidthNormalized(bbUpper, bbLower, bbMiddle []domain.IndicatorValue) []domain.IndicatorValue {
	if len(bbUpper) != len(bbLower) || len(bbUpper) != len(bbMiddle) {
		return nil
	}

	widths := make([]domain.IndicatorValue, len(bbUpper))
	for i := range bbUpper {
		if bbMiddle[i].Timestamp.IsZero() || bbMiddle[i].Value == 0 {
			// Skip invalid data points
			widths[i] = domain.IndicatorValue{
				Timestamp: bbUpper[i].Timestamp,
				Value:     0.0,
			}
			continue
		}

		upper := bbUpper[i].Value
		lower := bbLower[i].Value
		middle := bbMiddle[i].Value

		// Validate that bands are in correct order
		if upper < lower {
			log.Warn("Invalid BB bands: upper (%f) < lower (%f) for timestamp %v", upper, lower, bbUpper[i].Timestamp)
			widths[i] = domain.IndicatorValue{
				Timestamp: bbUpper[i].Timestamp,
				Value:     0.0,
			}
			continue
		}

		// Calculate normalized BB Width: (upper - lower) / middle
		bbWidth := (upper - lower) / middle

		// Validate the calculated value
		if math.IsNaN(bbWidth) || math.IsInf(bbWidth, 0) || bbWidth < 0 {
			log.Warn("Invalid normalized BB width calculated: %f for timestamp %v", bbWidth, bbUpper[i].Timestamp)
			bbWidth = 0.0
		}

		// Cap extremely large values
		if bbWidth > 10.0 { // 1000% normalized BB width is extremely high
			log.Warn("Normalized BB width too large (%f), capping to 10.0 for timestamp %v", bbWidth, bbUpper[i].Timestamp)
			bbWidth = 10.0
		}

		widths[i] = domain.IndicatorValue{
			Timestamp: bbUpper[i].Timestamp,
			Value:     bbWidth,
		}
	}
	return widths
}

// CalculateBBWidthNormalizedPercentage calculates the normalized percentage Bollinger Band width: ((upper - lower) / middle) * 100
func (s *TechnicalIndicatorService) CalculateBBWidthNormalizedPercentage(bbUpper, bbLower, bbMiddle []domain.IndicatorValue) []domain.IndicatorValue {
	if len(bbUpper) != len(bbLower) || len(bbUpper) != len(bbMiddle) {
		return nil
	}

	widths := make([]domain.IndicatorValue, len(bbUpper))
	for i := range bbUpper {
		if bbMiddle[i].Timestamp.IsZero() || bbMiddle[i].Value == 0 {
			// Skip invalid data points
			widths[i] = domain.IndicatorValue{
				Timestamp: bbUpper[i].Timestamp,
				Value:     0.0,
			}
			continue
		}

		upper := bbUpper[i].Value
		lower := bbLower[i].Value
		middle := bbMiddle[i].Value

		// Validate that bands are in correct order
		if upper < lower {
			log.Warn("Invalid BB bands: upper (%f) < lower (%f) for timestamp %v", upper, lower, bbUpper[i].Timestamp)
			widths[i] = domain.IndicatorValue{
				Timestamp: bbUpper[i].Timestamp,
				Value:     0.0,
			}
			continue
		}

		// Calculate normalized percentage BB Width: ((upper - lower) / middle) * 100
		bbWidth := (upper - lower) / middle * 100

		// Validate the calculated value
		if math.IsNaN(bbWidth) || math.IsInf(bbWidth, 0) || bbWidth < 0 {
			log.Warn("Invalid normalized percentage BB width calculated: %f for timestamp %v", bbWidth, bbUpper[i].Timestamp)
			bbWidth = 0.0
		}

		// Cap extremely large values
		if bbWidth > 1000.0 { // 1000% BB width is extremely high
			log.Warn("Normalized percentage BB width too large (%f), capping to 1000.0 for timestamp %v", bbWidth, bbUpper[i].Timestamp)
			bbWidth = 1000.0
		}

		widths[i] = domain.IndicatorValue{
			Timestamp: bbUpper[i].Timestamp,
			Value:     bbWidth,
		}
	}
	return widths
}

// ValidateDataOrdering checks if candles are in chronological order (Past → Latest)
// This helps ensure data consistency across the application
func ValidateDataOrdering(candles []domain.Candle) error {
	if len(candles) < 2 {
		return nil // Single candle or empty slice is always valid
	}

	for i := 1; i < len(candles); i++ {
		if !candles[i].Timestamp.After(candles[i-1].Timestamp) {
			return fmt.Errorf("data ordering violation: candle %d (%s) is not after candle %d (%s)",
				i, candles[i].Timestamp.Format(time.RFC3339),
				i-1, candles[i-1].Timestamp.Format(time.RFC3339))
		}
	}

	return nil
}

// candlesToFloat64Slices is a helper function to convert candle data to float slices for the indicator library.
func candlesToFloat64Slices(candles []domain.Candle) (closing, opening, high, low []float64, volume []int64) {
	closing = make([]float64, len(candles))
	opening = make([]float64, len(candles))
	high = make([]float64, len(candles))
	low = make([]float64, len(candles))
	volume = make([]int64, len(candles))
	for i, c := range candles {
		closing[i] = c.Close
		opening[i] = c.Open
		high[i] = c.High
		low[i] = c.Low
		volume[i] = c.Volume
	}
	return
}
