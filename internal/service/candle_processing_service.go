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

	// Convert []AggregatedCandle to []Candle for indicator calculation
	candleSlice := AggregatedCandlesToCandles(fiveMinCandles)

	// Calculate indicators on 5-minute data
	indicatorService := NewTechnicalIndicatorService(s.candleRepo)
	ma9 := indicatorService.CalculateSMA(candleSlice, 9)
	bbUpper, bbMiddle, bbLower := indicatorService.CalculateBollingerBands(candleSlice, 20, 2.0)
	bbWidth := indicatorService.CalculateBBWidth(bbUpper, bbLower, bbMiddle)
	vwap := indicatorService.CalculateVWAP(candleSlice)
	ema5 := indicatorService.CalculateEMAV2(candleSlice, 5)
	ema9 := indicatorService.CalculateEMAV2(candleSlice, 9)
	ema20 := indicatorService.CalculateEMAV2(candleSlice, 20)
	ema50 := indicatorService.CalculateEMAV2(candleSlice, 50)
	atr := indicatorService.CalculateATRV2(candleSlice, 14)
	rsi := indicatorService.CalculateRSIV2(candleSlice, 14)

	// Map indicators by timestamp
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

	// Enrich 5-min candles with indicators
	for i := range fiveMinCandles {
		ts := fiveMinCandles[i].Timestamp
		if v, ok := ma9Map[ts]; ok {
			fiveMinCandles[i].MA9 = v
		}
		if v, ok := bbUpperMap[ts]; ok {
			fiveMinCandles[i].BBUpper = v
		}
		if v, ok := bbMiddleMap[ts]; ok {
			fiveMinCandles[i].BBMiddle = v
		}
		if v, ok := bbLowerMap[ts]; ok {
			fiveMinCandles[i].BBLower = v
		}
		if v, ok := bbWidthMap[ts]; ok {
			fiveMinCandles[i].BBWidth = v
		}
		if v, ok := vwapMap[ts]; ok {
			fiveMinCandles[i].VWAP = v
		}
		if v, ok := ema5Map[ts]; ok {
			fiveMinCandles[i].EMA5 = v
		}
		if v, ok := ema9Map[ts]; ok {
			fiveMinCandles[i].EMA9 = v
		}
		if v, ok := ema20Map[ts]; ok {
			fiveMinCandles[i].EMA20 = v
		}
		if v, ok := ema50Map[ts]; ok {
			fiveMinCandles[i].EMA50 = v
		}
		if v, ok := atrMap[ts]; ok {
			fiveMinCandles[i].ATR = v
		}
		if v, ok := rsiMap[ts]; ok {
			fiveMinCandles[i].RSI = v
		}
	}

	// Convert AggregatedCandle to Candle5Min for storage
	candlesForStorage := make([]domain.Candle5Min, len(fiveMinCandles))
	for i, aggCandle := range fiveMinCandles {
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
			// Copy calculated indicator values
			BBUpper:                     aggCandle.BBUpper,
			BBMiddle:                    aggCandle.BBMiddle,
			BBLower:                     aggCandle.BBLower,
			BBWidth:                     aggCandle.BBWidth,
			BBWidthNormalized:           aggCandle.BBWidthNormalized,
			BBWidthNormalizedPercentage: aggCandle.BBWidthNormalizedPercentage,
			EMA5:                        aggCandle.EMA5,
			EMA9:                        aggCandle.EMA9,
			EMA20:                       aggCandle.EMA20,
			EMA50:                       aggCandle.EMA50,
			ATR:                         aggCandle.ATR,
			RSI:                         aggCandle.RSI,
			VWAP:                        aggCandle.VWAP,
			MA9:                         aggCandle.MA9,
			LowestBBWidth:               aggCandle.LowestBBWidth,
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

// AggregateAndStoreHistorical5MinCandles aggregates all 1-minute candles to 5-minute candles for historical data
func (s *CandleProcessingService) AggregateAndStoreHistorical5MinCandles(
	ctx context.Context,
	instrumentKey string,
	startTime, endTime time.Time,
) error {
	log.Info("[HISTORICAL_AGG] Starting historical 5-minute aggregation for %s from %s to %s",
		instrumentKey, startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))

	// Fetch all 1-minute candles for the entire time range
	oneMinCandles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, "1minute", startTime, endTime)
	if err != nil {
		log.Error("[HISTORICAL_AGG] Failed to fetch 1-minute candles for %s: %v", instrumentKey, err)
		return fmt.Errorf("failed to fetch 1-minute candles for historical aggregation: %w", err)
	}

	if len(oneMinCandles) == 0 {
		log.Warn("[HISTORICAL_AGG] No 1-minute candles found for historical 5-minute aggregation for %s", instrumentKey)
		return nil
	}

	log.Info("[HISTORICAL_AGG] Found %d 1-minute candles for %s from %s to %s",
		len(oneMinCandles), instrumentKey, startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))

	// Aggregate to 5-minute candles
	fiveMinCandles := aggregateTo5Min(oneMinCandles)
	if len(fiveMinCandles) == 0 {
		log.Debug("No 5-minute candles created from historical aggregation for %s", instrumentKey)
		return nil
	}

	log.Info("Created %d 5-minute candles from %d 1-minute candles for %s",
		len(fiveMinCandles), len(oneMinCandles), instrumentKey)

	// Convert []AggregatedCandle to []Candle for indicator calculation
	candleSlice := AggregatedCandlesToCandles(fiveMinCandles)

	// Calculate indicators on 5-minute data
	indicatorService := NewTechnicalIndicatorService(s.candleRepo)
	ma9 := indicatorService.CalculateSMA(candleSlice, 9)
	bbUpper, bbMiddle, bbLower := indicatorService.CalculateBollingerBands(candleSlice, 20, 2.0)
	bbWidth := indicatorService.CalculateBBWidth(bbUpper, bbLower, bbMiddle)
	vwap := indicatorService.CalculateVWAP(candleSlice)
	ema5 := indicatorService.CalculateEMAV2(candleSlice, 5)
	ema9 := indicatorService.CalculateEMAV2(candleSlice, 9)
	ema20 := indicatorService.CalculateEMAV2(candleSlice, 20)
	ema50 := indicatorService.CalculateEMAV2(candleSlice, 50)
	atr := indicatorService.CalculateATRV2(candleSlice, 14)
	rsi := indicatorService.CalculateRSIV2(candleSlice, 14)

	// Map indicators by timestamp
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

	// Enrich 5-min candles with indicators
	for i := range fiveMinCandles {
		ts := fiveMinCandles[i].Timestamp
		if v, ok := ma9Map[ts]; ok {
			fiveMinCandles[i].MA9 = v
		}
		if v, ok := bbUpperMap[ts]; ok {
			fiveMinCandles[i].BBUpper = v
		}
		if v, ok := bbMiddleMap[ts]; ok {
			fiveMinCandles[i].BBMiddle = v
		}
		if v, ok := bbLowerMap[ts]; ok {
			fiveMinCandles[i].BBLower = v
		}
		if v, ok := bbWidthMap[ts]; ok {
			fiveMinCandles[i].BBWidth = v
		}
		if v, ok := vwapMap[ts]; ok {
			fiveMinCandles[i].VWAP = v
		}
		if v, ok := ema5Map[ts]; ok {
			fiveMinCandles[i].EMA5 = v
		}
		if v, ok := ema9Map[ts]; ok {
			fiveMinCandles[i].EMA9 = v
		}
		if v, ok := ema20Map[ts]; ok {
			fiveMinCandles[i].EMA20 = v
		}
		if v, ok := ema50Map[ts]; ok {
			fiveMinCandles[i].EMA50 = v
		}
		if v, ok := atrMap[ts]; ok {
			fiveMinCandles[i].ATR = v
		}
		if v, ok := rsiMap[ts]; ok {
			fiveMinCandles[i].RSI = v
		}
	}

	// Convert AggregatedCandle to Candle5Min for storage
	candlesForStorage := make([]domain.Candle5Min, len(fiveMinCandles))
	for i, aggCandle := range fiveMinCandles {
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
			// Copy calculated indicator values
			BBUpper:                     aggCandle.BBUpper,
			BBMiddle:                    aggCandle.BBMiddle,
			BBLower:                     aggCandle.BBLower,
			BBWidth:                     aggCandle.BBWidth,
			BBWidthNormalized:           aggCandle.BBWidthNormalized,
			BBWidthNormalizedPercentage: aggCandle.BBWidthNormalizedPercentage,
			EMA5:                        aggCandle.EMA5,
			EMA9:                        aggCandle.EMA9,
			EMA20:                       aggCandle.EMA20,
			EMA50:                       aggCandle.EMA50,
			ATR:                         aggCandle.ATR,
			RSI:                         aggCandle.RSI,
			VWAP:                        aggCandle.VWAP,
			MA9:                         aggCandle.MA9,
			LowestBBWidth:               aggCandle.LowestBBWidth,
		}
	}

	// Store 5-minute candles
	_, err = s.candle5MinRepo.StoreBatch(ctx, candlesForStorage)
	if err != nil {
		return fmt.Errorf("failed to store historical 5-minute candles: %w", err)
	}

	log.Info("Stored %d historical 5-minute candles with indicators for %s from %s to %s",
		len(candlesForStorage), instrumentKey, startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
	return nil
}

// GetLatestCandle retrieves the latest candle for a given instrument and interval
func (s *CandleProcessingService) GetLatestCandle(ctx context.Context, instrumentKey, interval string) (*domain.Candle, error) {
	return s.candleRepo.GetLatestCandle(ctx, instrumentKey, interval)
}
