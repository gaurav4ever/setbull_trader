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
	authService   *upstox.AuthService
	candleRepo    repository.CandleRepository
	maxConcurrent int
	userID        string // User ID for authentication with Upstox
}

// NewCandleProcessingService creates a new candle processing service
func NewCandleProcessingService(
	authService *upstox.AuthService,
	candleRepo repository.CandleRepository,
	maxConcurrent int,
	userID string,
) *CandleProcessingService {
	if maxConcurrent <= 0 {
		maxConcurrent = 5 // Default to 5 concurrent requests
	}

	return &CandleProcessingService{
		authService:   authService,
		candleRepo:    candleRepo,
		maxConcurrent: maxConcurrent,
		userID:        userID,
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

	// Convert response to domain model
	candles, err := s.convertIntraDayCandles(response, instrumentKey, interval)
	if err != nil {
		return 0, fmt.Errorf("failed to convert intra-day candle data: %w", err)
	}

	// Store candles in the database
	count, err := s.candleRepo.StoreBatch(ctx, candles)
	if err != nil {
		return 0, fmt.Errorf("failed to store candle data: %w", err)
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
	vwap := tis.CalculateVWAP(candles)
	ema5 := tis.CalculateEMAV2(candles, 5)
	ema9 := tis.CalculateEMAV2(candles, 9)
	ema50 := tis.CalculateEMAV2(candles, 50)
	atr := tis.CalculateATRV2(candles, 14)
	rsi := tis.CalculateRSIV2(candles, 14)

	ma9Idx := 0
	bbIdx := 0
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
