package service

import (
	"context"
	"fmt"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
)

// CandleAggregationService provides operations for aggregating candles to different timeframes
type CandleAggregationService struct {
	candleRepo repository.CandleRepository
}

// NewCandleAggregationService creates a new candle aggregation service
func NewCandleAggregationService(candleRepo repository.CandleRepository) *CandleAggregationService {
	return &CandleAggregationService{
		candleRepo: candleRepo,
	}
}

// Get5MinCandles retrieves 5-minute candles for the given instrument and time range
func (s *CandleAggregationService) Get5MinCandles(
	ctx context.Context,
	instrumentKey string,
	start, end time.Time,
) ([]domain.AggregatedCandle, error) {
	// Validate inputs
	if instrumentKey == "" {
		return nil, fmt.Errorf("instrument key is required")
	}

	if end.Before(start) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	// If end time is zero, use current time
	if end.IsZero() {
		end = time.Now()
	}

	// If start time is zero, use end time minus 7 days
	if start.IsZero() {
		start = end.AddDate(0, 0, -7)
	}

	log.Info("Retrieving 5-minute candles for %s from %s to %s",
		instrumentKey, start.Format(time.RFC3339), end.Format(time.RFC3339))

	// Get the aggregated candles from the repository
	candles, err := s.candleRepo.GetAggregated5MinCandles(ctx, instrumentKey, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated 5-minute candles: %w", err)
	}

	return candles, nil
}

// GetDailyCandles retrieves daily candles for the given instrument and time range
func (s *CandleAggregationService) GetDailyCandles(
	ctx context.Context,
	instrumentKey string,
	start, end time.Time,
) ([]domain.AggregatedCandle, error) {
	// Validate inputs
	if instrumentKey == "" {
		return nil, fmt.Errorf("instrument key is required")
	}

	if end.Before(start) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	// If end time is zero, use current time
	if end.IsZero() {
		end = time.Now()
	}

	// If start time is zero, use end time minus 30 days
	if start.IsZero() {
		start = end.AddDate(0, 0, -30)
	}

	log.Info("Retrieving daily candles for %s from %s to %s",
		instrumentKey, start.Format(time.RFC3339), end.Format(time.RFC3339))

	// Get the aggregated candles from the repository
	candles, err := s.candleRepo.GetAggregatedDailyCandles(ctx, instrumentKey, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated daily candles: %w", err)
	}

	return candles, nil
}

// GetMultiTimeframeCandles retrieves candles for multiple timeframes for an instrument
func (s *CandleAggregationService) GetMultiTimeframeCandles(
	ctx context.Context,
	instrumentKey string,
	timeframes []string,
	start, end time.Time,
) (map[string][]domain.AggregatedCandle, error) {
	result := make(map[string][]domain.AggregatedCandle)

	for _, timeframe := range timeframes {
		var candles []domain.AggregatedCandle
		var err error

		switch timeframe {
		case "1minute":
			// For 1-minute candles, fetch directly from the database
			minuteCandles, err := s.candleRepo.FindByInstrumentAndTimeRange(ctx, instrumentKey, "1minute", start, end)
			if err != nil {
				return nil, fmt.Errorf("failed to get 1-minute candles: %w", err)
			}

			// Convert to AggregatedCandle format for consistency
			candles = make([]domain.AggregatedCandle, len(minuteCandles))
			for i, c := range minuteCandles {
				candles[i] = domain.AggregatedCandle{
					InstrumentKey: c.InstrumentKey,
					Timestamp:     c.Timestamp,
					Open:          c.Open,
					High:          c.High,
					Low:           c.Low,
					Close:         c.Close,
					Volume:        c.Volume,
					OpenInterest:  c.OpenInterest,
					TimeInterval:  c.TimeInterval,
				}
			}

		case "5minute":
			candles, err = s.Get5MinCandles(ctx, instrumentKey, start, end)
			if err != nil {
				return nil, fmt.Errorf("failed to get 5-minute candles: %w", err)
			}

		case "day":
			candles, err = s.GetDailyCandles(ctx, instrumentKey, start, end)
			if err != nil {
				return nil, fmt.Errorf("failed to get daily candles: %w", err)
			}

		default:
			log.Warn("Unsupported timeframe: %s", timeframe)
			continue
		}

		result[timeframe] = candles
	}

	return result, nil
}

// CacheAggregatedCandles caches aggregated candles for future use
// This is useful for timeframes that are queried frequently
func (s *CandleAggregationService) CacheAggregatedCandles(
	ctx context.Context,
	instrumentKey string,
	timeframe string,
	start, end time.Time,
) error {
	var candles []domain.AggregatedCandle
	var err error

	// Get the aggregated candles based on timeframe
	switch timeframe {
	case "5minute":
		candles, err = s.Get5MinCandles(ctx, instrumentKey, start, end)
	case "day":
		candles, err = s.GetDailyCandles(ctx, instrumentKey, start, end)
	default:
		return fmt.Errorf("unsupported timeframe for caching: %s", timeframe)
	}

	if err != nil {
		return fmt.Errorf("failed to get aggregated candles for caching: %w", err)
	}

	// Convert to CandleData for storage
	candleData := make([]domain.CandleData, len(candles))
	for i, c := range candles {
		candleData[i] = domain.CandleData{
			InstrumentKey: c.InstrumentKey,
			Timestamp:     c.Timestamp,
			Open:          c.Open,
			High:          c.High,
			Low:           c.Low,
			Close:         c.Close,
			Volume:        c.Volume,
			OpenInterest:  c.OpenInterest,
			Interval:      c.TimeInterval,
		}
	}

	// Store the converted data
	err = s.candleRepo.StoreAggregatedCandles(ctx, candleData)
	if err != nil {
		return fmt.Errorf("failed to store aggregated candles: %w", err)
	}

	log.Info("Cached %d %s candles for %s from %s to %s",
		len(candles), timeframe, instrumentKey,
		start.Format(time.RFC3339), end.Format(time.RFC3339))

	return nil
}

// GetStocksWithExistingDailyCandles returns a list of instrument keys that already have daily candle data
// for the specified date range
func (s *CandleAggregationService) GetStocksWithExistingDailyCandles(
	ctx context.Context,
	startDate, endDate time.Time,
) ([]string, error) {
	return s.candleRepo.GetStocksWithExistingDailyCandles(ctx, startDate, endDate)
}
