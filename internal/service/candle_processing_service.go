package service

import (
	"context"
	"fmt"
	"time"

	"setbull_trader/internal/core/adapters/client/upstox"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
	swagger "setbull_trader/upstox/go_api_client"
)

// CandleProcessingService handles fetching and processing of candle data
type CandleProcessingService struct {
	authService    *upstox.AuthService
	candleRepo     repository.CandleRepository
	candle5MinRepo repository.Candle5MinRepository
	maxConcurrent  int
	userID         string // User ID for authentication with Upstox
}

// NewCandleProcessingService creates a new candle processing service
func NewCandleProcessingService(
	authService *upstox.AuthService,
	candleRepo repository.CandleRepository,
	candle5MinRepo repository.Candle5MinRepository,
	maxConcurrent int,
	userID string,
) *CandleProcessingService {
	if maxConcurrent <= 0 {
		maxConcurrent = 5 // Default to 5 concurrent requests
	}

	return &CandleProcessingService{
		authService:    authService,
		candleRepo:     candleRepo,
		candle5MinRepo: candle5MinRepo,
		maxConcurrent:  maxConcurrent,
		userID:         userID,
	}
}

// ProcessHistoricalCandles fetches and processes historical candle data for a specific instrument
func (s *CandleProcessingService) ProcessHistoricalCandles(
	ctx context.Context,
	instrumentKey string,
	interval string,
	fromDate string,
	toDate string,
) (int, error) {
	// Fetch historical candle data
	response, err := s.authService.GetHistoricalCandleDataWithDateRange(
		ctx, s.userID, instrumentKey, interval, toDate, fromDate,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch historical candle data: %w", err)
	}

	// Process and store the candle data
	count, err := s.processCandleResponse(ctx, response, instrumentKey, interval)
	if err != nil {
		return 0, fmt.Errorf("failed to process candle data: %w", err)
	}

	return count, nil
}

// ProcessIntraDayCandles fetches and processes intra-day candle data for a specific instrument
func (s *CandleProcessingService) ProcessIntraDayCandles(
	ctx context.Context,
	instrumentKey string,
	interval string,
) (int, error) {
	// Fetch intra-day candle data
	response, err := s.authService.GetIntraDayCandleData(ctx, s.userID, instrumentKey, interval)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch intra-day candle data: %w", err)
	}
	// Convert response to domain model (latest candles)
	latestCandles, err := s.convertIntraDayCandles(response, instrumentKey, interval)
	if err != nil {
		return 0, fmt.Errorf("failed to convert intra-day candle data: %w", err)
	}
	// Ensure sufficient historical data for indicator calculation
	allCandles, err := s.ensureSufficientHistoricalData(ctx, instrumentKey, latestCandles, interval)
	if err != nil {
		return 0, fmt.Errorf("failed to ensure sufficient historical data: %w", err)
	}
	// Calculate indicators on the full set (historical + latest)
	candlesWithIndicators, err := s.calculateIndicatorsWithHistory(allCandles, latestCandles)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate indicators: %w", err)
	}
	// Store only the latest candles (with indicators)
	count, err := s.candleRepo.StoreBatch(ctx, candlesWithIndicators)
	if err != nil {
		return 0, fmt.Errorf("failed to store candle data: %w", err)
	}

	// Check if we need to trigger 5-minute aggregation
	if interval == "1minute" && len(latestCandles) > 0 {
		latestTimestamp := latestCandles[len(latestCandles)-1].Timestamp
		if s.IsFiveMinBoundarySinceMarketOpen(latestTimestamp) {
			log.Info("Triggering 5-minute aggregation for %s at %s", instrumentKey, latestTimestamp.Format("15:04"))
			if err := s.AggregateAndStore5MinCandles(ctx, instrumentKey, latestTimestamp); err != nil {
				log.Error("Failed to aggregate 5-minute candles for %s: %v", instrumentKey, err)
				// Don't return error here as 1-minute processing was successful
			}
		}
	}

	return count, nil
}

// processCandleResponse processes a historical candle response and stores the data
func (s *CandleProcessingService) processCandleResponse(
	ctx context.Context,
	response *swagger.GetHistoricalCandleResponse,
	instrumentKey string,
	interval string,
) (int, error) {
	if response == nil || response.Data == nil || response.Data.Candles == nil {
		return 0, nil
	}

	// Convert response to domain model
	candles, err := s.convertHistoricalCandles(response, instrumentKey, interval)
	if err != nil {
		return 0, err
	}

	if len(candles) == 0 {
		return 0, nil
	}

	// Store candles in the database
	count, err := s.candleRepo.StoreBatch(ctx, candles)
	if err != nil {
		return 0, fmt.Errorf("failed to store candle data: %w", err)
	}

	return count, nil
}

// convertHistoricalCandles converts a historical candle response to domain candles
func (s *CandleProcessingService) convertHistoricalCandles(
	response *swagger.GetHistoricalCandleResponse,
	instrumentKey string,
	interval string,
) ([]domain.Candle, error) {
	if response == nil || response.Data.Candles == nil {
		return []domain.Candle{}, nil
	}

	candles := make([]domain.Candle, 0, len(response.Data.Candles))

	for _, rawCandle := range response.Data.Candles {
		if len(rawCandle) < 7 {
			log.Warn("Skipping invalid candle data for %s: insufficient elements", instrumentKey)
			continue
		}

		// Parse timestamp
		timestampStr, ok := rawCandle[0].(string)
		if !ok {
			log.Warn("Skipping invalid candle data for %s: invalid timestamp format", instrumentKey)
			continue
		}

		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			log.Warn("Skipping invalid candle data for %s: %v", instrumentKey, err)
			continue
		}

		// Parse price data with type assertions and conversions
		open, err := parseFloat64(rawCandle[1])
		if err != nil {
			log.Warn("Invalid open price for %s, skipping: %v", instrumentKey, err)
			continue
		}

		high, err := parseFloat64(rawCandle[2])
		if err != nil {
			log.Warn("Invalid high price for %s, skipping: %v", instrumentKey, err)
			continue
		}

		low, err := parseFloat64(rawCandle[3])
		if err != nil {
			log.Warn("Invalid low price for %s, skipping: %v", instrumentKey, err)
			continue
		}

		closePrice, err := parseFloat64(rawCandle[4])
		if err != nil {
			log.Warn("Invalid close price for %s, skipping: %v", instrumentKey, err)
			continue
		}

		volume, err := parseInt64(rawCandle[5])
		if err != nil {
			log.Warn("Invalid volume for %s, skipping: %v", instrumentKey, err)
			continue
		}

		openInterest, err := parseInt64(rawCandle[6])
		if err != nil {
			log.Warn("Invalid open interest for %s, defaulting to 0: %v", instrumentKey, err)
			openInterest = 0
		}

		candle := domain.Candle{
			InstrumentKey: instrumentKey,
			Timestamp:     timestamp,
			Open:          open,
			High:          high,
			Low:           low,
			Close:         closePrice,
			Volume:        volume,
			OpenInterest:  openInterest,
			TimeInterval:  interval,
		}

		candles = append(candles, candle)
	}

	// --- Indicator Calculation Integration ---
	tis := NewTechnicalIndicatorService(s.candleRepo)
	// 9-period SMA
	ma9 := tis.CalculateSMA(candles, 9)
	// Bollinger Bands (20, 2.0)
	bbUpper, bbMiddle, bbLower := tis.CalculateBollingerBands(candles, 20, 2.0)
	// BB Width
	bbWidth := tis.CalculateBBWidth(bbUpper, bbLower, bbMiddle)
	// VWAP
	vwap := tis.CalculateVWAP(candles)
	// EMA
	ema5 := tis.CalculateEMAV2(candles, 5)
	ema9 := tis.CalculateEMAV2(candles, 9)
	ema50 := tis.CalculateEMAV2(candles, 50)
	// ATR (14)
	atr := tis.CalculateATRV2(candles, 14)
	// RSI (14)
	rsi := tis.CalculateRSIV2(candles, 14)

	// Map indicator values to candles by timestamp
	ma9Idx := 0
	bbIdx := 0
	bbWidthIdx := 0
	vwapIdx := 0
	ema5Idx := 0
	ema9Idx := 0
	ema50Idx := 0
	atrIdx := 0
	rsiIdx := 0
	for i := range candles {
		// MA9
		if ma9Idx < len(ma9) && candles[i].Timestamp.Equal(ma9[ma9Idx].Timestamp) {
			candles[i].MA9 = ma9[ma9Idx].Value
			ma9Idx++
		}
		// BB
		if bbIdx < len(bbMiddle) && candles[i].Timestamp.Equal(bbMiddle[bbIdx].Timestamp) {
			candles[i].BBUpper = bbUpper[bbIdx].Value
			candles[i].BBMiddle = bbMiddle[bbIdx].Value
			candles[i].BBLower = bbLower[bbIdx].Value
			bbIdx++
		}
		// BB Width
		if bbWidthIdx < len(bbWidth) && candles[i].Timestamp.Equal(bbWidth[bbWidthIdx].Timestamp) {
			candles[i].BBWidth = bbWidth[bbWidthIdx].Value
			bbWidthIdx++
		}
		// VWAP
		if vwapIdx < len(vwap) && candles[i].Timestamp.Equal(vwap[vwapIdx].Timestamp) {
			candles[i].VWAP = vwap[vwapIdx].Value
			vwapIdx++
		}
		// EMA5
		if ema5Idx < len(ema5) && candles[i].Timestamp.Equal(ema5[ema5Idx].Timestamp) {
			candles[i].EMA5 = ema5[ema5Idx].Value
			ema5Idx++
		}
		// EMA9
		if ema9Idx < len(ema9) && candles[i].Timestamp.Equal(ema9[ema9Idx].Timestamp) {
			candles[i].EMA9 = ema9[ema9Idx].Value
			ema9Idx++
		}
		// EMA50
		if ema50Idx < len(ema50) && candles[i].Timestamp.Equal(ema50[ema50Idx].Timestamp) {
			candles[i].EMA50 = ema50[ema50Idx].Value
			ema50Idx++
		}
		// ATR
		if atrIdx < len(atr) && candles[i].Timestamp.Equal(atr[atrIdx].Timestamp) {
			candles[i].ATR = atr[atrIdx].Value
			atrIdx++
		}
		// RSI
		if rsiIdx < len(rsi) && candles[i].Timestamp.Equal(rsi[rsiIdx].Timestamp) {
			candles[i].RSI = rsi[rsiIdx].Value
			rsiIdx++
		}
	}
	// --- End Indicator Integration ---

	return candles, nil
}

// convertIntraDayCandles converts an intra-day candle response to domain candles
func (s *CandleProcessingService) convertIntraDayCandles(
	response *swagger.GetIntraDayCandleResponse,
	instrumentKey string,
	interval string,
) ([]domain.Candle, error) {
	if response == nil || response.Data.Candles == nil {
		return []domain.Candle{}, nil
	}

	log.Info("Total candles: %d for %s", len(response.Data.Candles), instrumentKey)
	candles := make([]domain.Candle, 0, len(response.Data.Candles))

	for _, rawCandle := range response.Data.Candles {
		if len(rawCandle) < 7 {
			log.Warn("Skipping invalid candle data for %s: insufficient elements", instrumentKey)
			continue
		}

		// Parse timestamp
		timestampStr, ok := rawCandle[0].(string)
		if !ok {
			log.Warn("Skipping invalid candle data for %s: invalid timestamp format", instrumentKey)
			continue
		}

		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			log.Warn("Skipping invalid candle data for %s: %v", instrumentKey, err)
			continue
		}

		// Parse price data with type assertions and conversions
		open, err := parseFloat64(rawCandle[1])
		if err != nil {
			log.Warn("Invalid open price for %s, skipping: %v", instrumentKey, err)
			continue
		}

		high, err := parseFloat64(rawCandle[2])
		if err != nil {
			log.Warn("Invalid high price for %s, skipping: %v", instrumentKey, err)
			continue
		}

		low, err := parseFloat64(rawCandle[3])
		if err != nil {
			log.Warn("Invalid low price for %s, skipping: %v", instrumentKey, err)
			continue
		}

		closePrice, err := parseFloat64(rawCandle[4])
		if err != nil {
			log.Warn("Invalid close price for %s, skipping: %v", instrumentKey, err)
			continue
		}

		volume, err := parseInt64(rawCandle[5])
		if err != nil {
			log.Warn("Invalid volume for %s, skipping: %v", instrumentKey, err)
			continue
		}

		openInterest, err := parseInt64(rawCandle[6])
		if err != nil {
			log.Warn("Invalid open interest for %s, defaulting to 0: %v", instrumentKey, err)
			openInterest = 0
		}

		candle := domain.Candle{
			InstrumentKey: instrumentKey,
			Timestamp:     timestamp,
			Open:          open,
			High:          high,
			Low:           low,
			Close:         closePrice,
			Volume:        volume,
			OpenInterest:  openInterest,
			TimeInterval:  interval,
		}

		candles = append(candles, candle)
	}

	// --- Indicator Calculation Integration ---
	tis := NewTechnicalIndicatorService(s.candleRepo)
	ma9 := tis.CalculateSMA(candles, 9)
	bbUpper, bbMiddle, bbLower := tis.CalculateBollingerBands(candles, 20, 2.0)
	bbWidth := tis.CalculateBBWidth(bbUpper, bbLower, bbMiddle)
	vwap := tis.CalculateVWAP(candles)
	ema5 := tis.CalculateEMAV2(candles, 5)
	ema9 := tis.CalculateEMAV2(candles, 9)
	ema50 := tis.CalculateEMAV2(candles, 50)
	atr := tis.CalculateATRV2(candles, 14)
	rsi := tis.CalculateRSIV2(candles, 14)

	ma9Idx := 0
	bbIdx := 0
	bbWidthIdx := 0
	vwapIdx := 0
	ema5Idx := 0
	ema9Idx := 0
	ema50Idx := 0
	atrIdx := 0
	rsiIdx := 0
	for i := range candles {
		if ma9Idx < len(ma9) && candles[i].Timestamp.Equal(ma9[ma9Idx].Timestamp) {
			candles[i].MA9 = ma9[ma9Idx].Value
			ma9Idx++
		}
		if bbIdx < len(bbMiddle) && candles[i].Timestamp.Equal(bbMiddle[bbIdx].Timestamp) {
			candles[i].BBUpper = bbUpper[bbIdx].Value
			candles[i].BBMiddle = bbMiddle[bbIdx].Value
			candles[i].BBLower = bbLower[bbIdx].Value
			bbIdx++
		}
		if bbWidthIdx < len(bbWidth) && candles[i].Timestamp.Equal(bbWidth[bbWidthIdx].Timestamp) {
			candles[i].BBWidth = bbWidth[bbWidthIdx].Value
			bbWidthIdx++
		}
		if vwapIdx < len(vwap) && candles[i].Timestamp.Equal(vwap[vwapIdx].Timestamp) {
			candles[i].VWAP = vwap[vwapIdx].Value
			vwapIdx++
		}
		if ema5Idx < len(ema5) && candles[i].Timestamp.Equal(ema5[ema5Idx].Timestamp) {
			candles[i].EMA5 = ema5[ema5Idx].Value
			ema5Idx++
		}
		if ema9Idx < len(ema9) && candles[i].Timestamp.Equal(ema9[ema9Idx].Timestamp) {
			candles[i].EMA9 = ema9[ema9Idx].Value
			ema9Idx++
		}
		if ema50Idx < len(ema50) && candles[i].Timestamp.Equal(ema50[ema50Idx].Timestamp) {
			candles[i].EMA50 = ema50[ema50Idx].Value
			ema50Idx++
		}
		if atrIdx < len(atr) && candles[i].Timestamp.Equal(atr[atrIdx].Timestamp) {
			candles[i].ATR = atr[atrIdx].Value
			atrIdx++
		}
		if rsiIdx < len(rsi) && candles[i].Timestamp.Equal(rsi[rsiIdx].Timestamp) {
			candles[i].RSI = rsi[rsiIdx].Value
			rsiIdx++
		}
	}
	// --- End Indicator Integration ---

	return candles, nil
}

// Add ensureSufficientHistoricalData to CandleProcessingService
func (s *CandleProcessingService) ensureSufficientHistoricalData(
	ctx context.Context,
	instrumentKey string,
	latestCandles []domain.Candle,
	interval string,
) ([]domain.Candle, error) {
	const minCandlesForBB = 200 // 20 for BB + buffer for other indicators
	if len(latestCandles) == 0 {
		return nil, fmt.Errorf("no latest candles provided")
	}
	neededHistorical := minCandlesForBB - len(latestCandles)
	if neededHistorical <= 0 {
		return latestCandles, nil // Already have sufficient data
	}
	endTime := latestCandles[0].Timestamp
	startTime := endTime.Add(-time.Duration(neededHistorical) * time.Minute)
	historicalCandles, err := s.candleRepo.FindByInstrumentAndTimeRange(
		ctx, instrumentKey, interval, startTime, endTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical data: %w", err)
	}
	combinedCandles := append(historicalCandles, latestCandles...)
	return combinedCandles, nil
}

// Add calculateIndicatorsWithHistory to CandleProcessingService
func (s *CandleProcessingService) calculateIndicatorsWithHistory(
	allCandles []domain.Candle,
	latestCandles []domain.Candle,
) ([]domain.Candle, error) {
	if len(allCandles) == 0 || len(latestCandles) == 0 {
		return nil, fmt.Errorf("insufficient candles for indicator calculation")
	}
	tis := NewTechnicalIndicatorService(s.candleRepo)
	ma9 := tis.CalculateSMA(allCandles, 9)
	bbUpper, bbMiddle, bbLower := tis.CalculateBollingerBands(allCandles, 20, 2.0)
	bbWidth := tis.CalculateBBWidth(bbUpper, bbLower, bbMiddle)
	vwap := tis.CalculateVWAP(allCandles)
	ema5 := tis.CalculateEMAV2(allCandles, 5)
	ema9 := tis.CalculateEMAV2(allCandles, 9)
	ema50 := tis.CalculateEMAV2(allCandles, 50)
	atr := tis.CalculateATRV2(allCandles, 14)
	rsi := tis.CalculateRSIV2(allCandles, 14)
	// Map indicator values by timestamp for fast lookup
	ma9Map := make(map[time.Time]float64)
	for _, v := range ma9 {
		ma9Map[v.Timestamp] = v.Value
	}
	bbUpperMap := make(map[time.Time]float64)
	bbMiddleMap := make(map[time.Time]float64)
	bbLowerMap := make(map[time.Time]float64)
	for i := range bbUpper {
		bbUpperMap[bbUpper[i].Timestamp] = bbUpper[i].Value
		bbMiddleMap[bbMiddle[i].Timestamp] = bbMiddle[i].Value
		bbLowerMap[bbLower[i].Timestamp] = bbLower[i].Value
	}
	bbWidthMap := make(map[time.Time]float64)
	for _, v := range bbWidth {
		bbWidthMap[v.Timestamp] = v.Value
	}
	vwapMap := make(map[time.Time]float64)
	for _, v := range vwap {
		vwapMap[v.Timestamp] = v.Value
	}
	ema5Map := make(map[time.Time]float64)
	for _, v := range ema5 {
		ema5Map[v.Timestamp] = v.Value
	}
	ema9Map := make(map[time.Time]float64)
	for _, v := range ema9 {
		ema9Map[v.Timestamp] = v.Value
	}
	ema50Map := make(map[time.Time]float64)
	for _, v := range ema50 {
		ema50Map[v.Timestamp] = v.Value
	}
	atrMap := make(map[time.Time]float64)
	for _, v := range atr {
		atrMap[v.Timestamp] = v.Value
	}
	rsiMap := make(map[time.Time]float64)
	for _, v := range rsi {
		rsiMap[v.Timestamp] = v.Value
	}
	// Only return latestCandles, but with indicators from the full set
	for i := range latestCandles {
		ts := latestCandles[i].Timestamp
		if v, ok := ma9Map[ts]; ok {
			latestCandles[i].MA9 = v
		}
		if v, ok := bbUpperMap[ts]; ok {
			latestCandles[i].BBUpper = v
		}
		if v, ok := bbMiddleMap[ts]; ok {
			latestCandles[i].BBMiddle = v
		}
		if v, ok := bbLowerMap[ts]; ok {
			latestCandles[i].BBLower = v
		}
		if v, ok := bbWidthMap[ts]; ok {
			latestCandles[i].BBWidth = v
		}
		if v, ok := vwapMap[ts]; ok {
			latestCandles[i].VWAP = v
		}
		if v, ok := ema5Map[ts]; ok {
			latestCandles[i].EMA5 = v
		}
		if v, ok := ema9Map[ts]; ok {
			latestCandles[i].EMA9 = v
		}
		if v, ok := ema50Map[ts]; ok {
			latestCandles[i].EMA50 = v
		}
		if v, ok := atrMap[ts]; ok {
			latestCandles[i].ATR = v
		}
		if v, ok := rsiMap[ts]; ok {
			latestCandles[i].RSI = v
		}
	}
	return latestCandles, nil
}

// Helper functions for type conversion with error handling

// parseFloat64 converts an interface{} to float64
func parseFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return 0, fmt.Errorf("unexpected string value for numeric field")
	default:
		return 0, fmt.Errorf("unable to parse %T as float64", value)
	}
}

// parseInt64 converts an interface{} to int64
func parseInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case string:
		return 0, fmt.Errorf("unexpected string value for numeric field")
	default:
		return 0, fmt.Errorf("unable to parse %T as int64", value)
	}
}

// IsFiveMinBoundarySinceMarketOpen checks if a given time is a 5-minute boundary since market open (9:15 AM)
func (s *CandleProcessingService) IsFiveMinBoundarySinceMarketOpen(t time.Time) bool {
	marketOpenHour := 9
	marketOpenMinute := 15
	if t.Hour() < marketOpenHour || (t.Hour() == marketOpenHour && t.Minute() < marketOpenMinute) {
		return false
	}
	minutesSinceOpen := (t.Hour()-marketOpenHour)*60 + (t.Minute() - marketOpenMinute)
	return minutesSinceOpen >= 0 && minutesSinceOpen%5 == 0
}

// AggregateAndStore5MinCandles aggregates 1-minute candles to 5-minute candles and stores them
func (s *CandleProcessingService) AggregateAndStore5MinCandles(
	ctx context.Context,
	instrumentKey string,
	endTime time.Time,
) error {
	// Calculate the 5-minute window
	startTime := endTime.Add(-5 * time.Minute)

	// Fetch 1-minute candles for the 5-minute window
	oneMinCandles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, "1minute", startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to fetch 1-minute candles for aggregation: %w", err)
	}

	if len(oneMinCandles) == 0 {
		log.Debug("No 1-minute candles found for 5-minute aggregation for %s", instrumentKey)
		return nil
	}

	// Aggregate to 5-minute candles
	fiveMinCandles := aggregateTo5Min(oneMinCandles)
	if len(fiveMinCandles) == 0 {
		log.Debug("No 5-minute candles created from aggregation for %s", instrumentKey)
		return nil
	}

	// Calculate indicators for the 5-minute aggregated candles
	// First, get sufficient historical data for indicator calculation
	extendedStart := startTime.AddDate(0, 0, -1) // Include previous day for warm-up data
	historicalCandles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, "1minute", extendedStart, startTime)
	if err != nil {
		log.Warn("Failed to fetch historical candles for indicator calculation: %v", err)
		// Continue without indicators
	}

	log.Info("Fetched %d historical candles and %d current candles for %s", len(historicalCandles), len(oneMinCandles), instrumentKey)

	// Combine historical and current candles for indicator calculation
	allCandles := append(historicalCandles, oneMinCandles...)

	// Aggregate to 5-minute candles
	aggregatedCandles := aggregateTo5Min(allCandles)

	log.Info("Aggregated %d 1-minute candles to %d 5-minute candles for %s", len(allCandles), len(aggregatedCandles), instrumentKey)

	if len(aggregatedCandles) == 0 {
		log.Warn("No 5-minute candles generated for %s", instrumentKey)
		return nil
	}

	// Convert aggregated candles to domain.Candle for indicator calculation
	candlesForIndicators := AggregatedCandlesToCandles(aggregatedCandles)

	log.Info("Converted %d aggregated candles to domain.Candle format for %s", len(candlesForIndicators), instrumentKey)

	if len(candlesForIndicators) < 20 {
		log.Warn("Insufficient candles (%d) for indicator calculation for %s, need at least 20", len(candlesForIndicators), instrumentKey)
		// Store candles without indicators
		return s.store5MinCandlesWithoutIndicators(ctx, instrumentKey, aggregatedCandles)
	}

	// Create technical indicator service
	indicatorService := NewTechnicalIndicatorService(s.candleRepo)

	// Calculate Bollinger Bands
	bbUpper, bbMiddle, bbLower := indicatorService.CalculateBollingerBands(candlesForIndicators, 20, 2.0)

	log.Info("Calculated BB: upper=%d, middle=%d, lower=%d for %s", len(bbUpper), len(bbMiddle), len(bbLower), instrumentKey)

	if len(bbUpper) == 0 || len(bbMiddle) == 0 || len(bbLower) == 0 {
		log.Warn("BB calculation returned empty results for %s", instrumentKey)
		// Store candles without indicators
		return s.store5MinCandlesWithoutIndicators(ctx, instrumentKey, aggregatedCandles)
	}

	// Calculate BB Width
	bbWidth := indicatorService.CalculateBBWidth(bbUpper, bbLower, bbMiddle)

	log.Info("Calculated BB Width: %d values for %s", len(bbWidth), instrumentKey)

	if len(bbWidth) == 0 {
		log.Warn("BB Width calculation returned empty results for %s", instrumentKey)
		// Store candles without indicators
		return s.store5MinCandlesWithoutIndicators(ctx, instrumentKey, aggregatedCandles)
	}

	// Calculate BB Width Normalized
	bbWidthNormalized := indicatorService.CalculateBBWidthNormalized(bbUpper, bbLower, bbMiddle)

	log.Info("Calculated BB Width Normalized: %d values for %s", len(bbWidthNormalized), instrumentKey)

	// Calculate BB Width Normalized Percentage
	bbWidthNormalizedPercentage := indicatorService.CalculateBBWidthNormalizedPercentage(bbUpper, bbLower, bbMiddle)

	log.Info("Calculated BB Width Normalized Percentage: %d values for %s", len(bbWidthNormalizedPercentage), instrumentKey)

	// Calculate EMAs
	ema5 := indicatorService.CalculateEMAV2(candlesForIndicators, 5)
	ema9 := indicatorService.CalculateEMAV2(candlesForIndicators, 9)
	ema20 := indicatorService.CalculateEMAV2(candlesForIndicators, 20)
	ema50 := indicatorService.CalculateEMAV2(candlesForIndicators, 50)

	log.Info("Calculated EMAs: ema5=%d, ema9=%d, ema20=%d, ema50=%d for %s", len(ema5), len(ema9), len(ema20), len(ema50), instrumentKey)

	// Calculate ATR
	atr := indicatorService.CalculateATRV2(candlesForIndicators, 14)

	log.Info("Calculated ATR: %d values for %s", len(atr), instrumentKey)

	// Calculate RSI
	rsi := indicatorService.CalculateRSIV2(candlesForIndicators, 14)

	log.Info("Calculated RSI: %d values for %s", len(rsi), instrumentKey)

	// Calculate VWAP
	vwap := indicatorService.CalculateVWAP(candlesForIndicators)

	log.Info("Calculated VWAP: %d values for %s", len(vwap), instrumentKey)

	// Calculate MA9
	ma9 := indicatorService.CalculateSMA(candlesForIndicators, 9)

	log.Info("Calculated MA9: %d values for %s", len(ma9), instrumentKey)

	// Calculate lowest BB width for the period
	lowestBBWidth := s.calculateLowestBBWidth(bbWidth)

	log.Info("Calculated lowest BB width: %f for %s", lowestBBWidth, instrumentKey)

	// Create maps for quick lookup
	ma9Map := make(map[time.Time]float64)
	for _, v := range ma9 {
		ma9Map[v.Timestamp] = v.Value
	}
	bbUpperMap := make(map[time.Time]float64)
	for _, v := range bbUpper {
		bbUpperMap[v.Timestamp] = v.Value
	}
	bbMiddleMap := make(map[time.Time]float64)
	for _, v := range bbMiddle {
		bbMiddleMap[v.Timestamp] = v.Value
	}
	bbLowerMap := make(map[time.Time]float64)
	for _, v := range bbLower {
		bbLowerMap[v.Timestamp] = v.Value
	}
	bbWidthMap := make(map[time.Time]float64)
	for _, v := range bbWidth {
		bbWidthMap[v.Timestamp] = v.Value
	}
	bbWidthNormalizedMap := make(map[time.Time]float64)
	for _, v := range bbWidthNormalized {
		bbWidthNormalizedMap[v.Timestamp] = v.Value
	}
	bbWidthNormalizedPercentageMap := make(map[time.Time]float64)
	for _, v := range bbWidthNormalizedPercentage {
		bbWidthNormalizedPercentageMap[v.Timestamp] = v.Value
	}
	vwapMap := make(map[time.Time]float64)
	for _, v := range vwap {
		vwapMap[v.Timestamp] = v.Value
	}
	ema5Map := make(map[time.Time]float64)
	for _, v := range ema5 {
		ema5Map[v.Timestamp] = v.Value
	}
	ema9Map := make(map[time.Time]float64)
	for _, v := range ema9 {
		ema9Map[v.Timestamp] = v.Value
	}
	ema20Map := make(map[time.Time]float64)
	for _, v := range ema20 {
		ema20Map[v.Timestamp] = v.Value
	}
	ema50Map := make(map[time.Time]float64)
	for _, v := range ema50 {
		ema50Map[v.Timestamp] = v.Value
	}
	atrMap := make(map[time.Time]float64)
	for _, v := range atr {
		atrMap[v.Timestamp] = v.Value
	}
	rsiMap := make(map[time.Time]float64)
	for _, v := range rsi {
		rsiMap[v.Timestamp] = v.Value
	}

	// Convert AggregatedCandle to Candle5Min for storage with calculated indicators
	candlesForStorage := make([]domain.Candle5Min, len(fiveMinCandles))
	for i, aggCandle := range fiveMinCandles {
		ts := aggCandle.Timestamp

		candlesForStorage[i] = domain.Candle5Min{
			InstrumentKey: aggCandle.InstrumentKey,
			Timestamp:     aggCandle.Timestamp,
			Open:          aggCandle.Open,
			High:          aggCandle.High,
			Low:           aggCandle.Low,
			Close:         aggCandle.Close,
			Volume:        aggCandle.Volume,
			OpenInterest:  aggCandle.OpenInterest,
			TimeInterval:  "5minute",
			// Set calculated indicator values
			MA9:                         ma9Map[ts],
			BBUpper:                     bbUpperMap[ts],
			BBMiddle:                    bbMiddleMap[ts],
			BBLower:                     bbLowerMap[ts],
			BBWidth:                     bbWidthMap[ts],
			BBWidthNormalized:           bbWidthNormalizedMap[ts],
			BBWidthNormalizedPercentage: bbWidthNormalizedPercentageMap[ts],
			VWAP:                        vwapMap[ts],
			EMA5:                        ema5Map[ts],
			EMA9:                        ema9Map[ts],
			EMA20:                       ema20Map[ts],
			EMA50:                       ema50Map[ts],
			ATR:                         atrMap[ts],
			RSI:                         rsiMap[ts],
			LowestBBWidth:               0.0, // TODO: Calculate this if needed
		}
	}

	// Store 5-minute candles
	_, err = s.candle5MinRepo.StoreBatch(ctx, candlesForStorage)
	if err != nil {
		return fmt.Errorf("failed to store 5-minute candles: %w", err)
	}

	log.Info("Stored %d 5-minute candles with indicators for %s at %s", len(candlesForStorage), instrumentKey, endTime.Format("15:04"))
	return nil
}

// AggregateAndStore5MinCandlesForRange aggregates ALL 5-minute boundaries within a given time range
func (s *CandleProcessingService) AggregateAndStore5MinCandlesForRange(
	ctx context.Context,
	instrumentKey string,
	startTime time.Time,
	endTime time.Time,
) error {
	log.Info("[5MIN] Starting 5-minute aggregation for range %s to %s for %s",
		startTime.Format("2006-01-02 15:04"), endTime.Format("2006-01-02 15:04"), instrumentKey)

	// Find all 5-minute boundaries within the time range
	boundaries := s.findFiveMinBoundariesInRange(startTime, endTime)

	log.Info("[5MIN] Found %d 5-minute boundaries for %s", len(boundaries), instrumentKey)

	if len(boundaries) == 0 {
		log.Debug("No 5-minute boundaries found in range %s to %s for %s",
			startTime.Format("2006-01-02 15:04"), endTime.Format("2006-01-02 15:04"), instrumentKey)
		return nil
	}

	log.Info("Found %d 5-minute boundaries to aggregate for %s", len(boundaries), instrumentKey)

	// Aggregate each 5-minute boundary
	for i, boundary := range boundaries {
		log.Info("[5MIN] Processing boundary %d/%d: %s for %s", i+1, len(boundaries), boundary.Format("15:04"), instrumentKey)
		if err := s.AggregateAndStore5MinCandles(ctx, instrumentKey, boundary); err != nil {
			log.Error("Failed to aggregate 5-minute candles for %s at %s: %v",
				instrumentKey, boundary.Format("15:04"), err)
			// Continue with other boundaries, don't fail the entire operation
		} else {
			log.Info("[5MIN] Successfully processed boundary %s for %s", boundary.Format("15:04"), instrumentKey)
		}
	}

	log.Info("[5MIN] Completed 5-minute aggregation for %s", instrumentKey)
	return nil
}

// findFiveMinBoundariesInRange finds all 5-minute boundaries within a given time range
func (s *CandleProcessingService) findFiveMinBoundariesInRange(startTime, endTime time.Time) []time.Time {
	var boundaries []time.Time

	// Start from the first 5-minute boundary after or at startTime
	current := startTime.Truncate(5 * time.Minute)
	if current.Before(startTime) {
		current = current.Add(5 * time.Minute)
	}

	// Generate all 5-minute boundaries within the range
	for current.Before(endTime) || current.Equal(endTime) {
		// For historical batch processing, include all 5-minute boundaries within trading hours
		// Market hours: 9:15 AM to 3:30 PM
		hour := current.Hour()
		minute := current.Minute()

		// Check if time is within trading hours
		if (hour > 9 || (hour == 9 && minute >= 15)) && (hour < 15 || (hour == 15 && minute <= 30)) {
			boundaries = append(boundaries, current)
		}

		current = current.Add(5 * time.Minute)
	}

	return boundaries
}

// GetLatestCandle retrieves the latest candle for a given instrument and interval
func (s *CandleProcessingService) GetLatestCandle(ctx context.Context, instrumentKey, interval string) (*domain.Candle, error) {
	return s.candleRepo.GetLatestCandle(ctx, instrumentKey, interval)
}

// store5MinCandlesWithoutIndicators stores 5-minute candles without calculating indicators
func (s *CandleProcessingService) store5MinCandlesWithoutIndicators(
	ctx context.Context,
	instrumentKey string,
	aggregatedCandles []domain.AggregatedCandle,
) error {
	// Convert to 5-minute candles without indicators
	candles := make([]domain.Candle5Min, len(aggregatedCandles))
	for i, agg := range aggregatedCandles {
		candles[i] = domain.Candle5Min{
			InstrumentKey: agg.InstrumentKey,
			Timestamp:     agg.Timestamp,
			Open:          agg.Open,
			High:          agg.High,
			Low:           agg.Low,
			Close:         agg.Close,
			Volume:        agg.Volume,
			OpenInterest:  agg.OpenInterest,
			TimeInterval:  agg.TimeInterval,
			// All indicators will be 0.0000 (default values)
		}
	}

	// Store the candles
	_, err := s.candle5MinRepo.StoreBatch(ctx, candles)
	if err != nil {
		return fmt.Errorf("failed to store 5-minute candles without indicators: %w", err)
	}

	log.Info("Stored %d 5-minute candles without indicators for %s", len(candles), instrumentKey)
	return nil
}

// calculateLowestBBWidth calculates the lowest BB width value from the given BB width indicators
func (s *CandleProcessingService) calculateLowestBBWidth(bbWidth []domain.IndicatorValue) float64 {
	if len(bbWidth) == 0 {
		return 0.0
	}

	lowest := bbWidth[0].Value
	for _, width := range bbWidth {
		if width.Value > 0 && (lowest == 0 || width.Value < lowest) {
			lowest = width.Value
		}
	}

	return lowest
}
