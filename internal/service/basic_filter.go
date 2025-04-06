package service

import (
	"context"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
)

// BasicFilter implements the basic price and volume filtering criteria
type BasicFilter struct {
	candleRepo repository.CandleRepository
	minPrice   float64
	maxPrice   float64
	minVolume  int64
}

// NewBasicFilter creates a new instance of BasicFilter
func NewBasicFilter(candleRepo repository.CandleRepository) *BasicFilter {
	return &BasicFilter{
		candleRepo: candleRepo,
		minPrice:   50.0,
		maxPrice:   1000.0,
		minVolume:  400000,
	}
}

// Filter implements the StockFilter interface
func (f *BasicFilter) Filter(ctx context.Context, stocks interface{}) (bullish, bearish []domain.FilteredStock, err error) {
	var filteredStocks []domain.FilteredStock
	var skippedStocks int

	// Handle different input types
	switch input := stocks.(type) {
	case []domain.StockUniverse:
		log.Info("Starting basic filter for %d stocks", len(input))
		for _, stock := range input {
			// Get last 10 days candles
			candles, err := f.candleRepo.GetNDailyCandlesByTimeframe(ctx, stock.InstrumentKey, "day", 10)
			if err != nil {
				log.Warn("Failed to get candles for %s: %v", stock.InstrumentKey, err)
				skippedStocks++
				continue
			}

			// Check if we have enough candle data
			if len(candles) < 10 {
				log.Debug("Insufficient candle data for %s: got %d days", stock.InstrumentKey, len(candles))
				skippedStocks++
				continue
			}

			// Calculate average volume
			var totalVolume int64
			for _, candle := range candles {
				totalVolume += candle.Volume
			}
			avgVolume := totalVolume / int64(len(candles))
			lastCandle := candles[0] // Most recent candle

			// Apply basic filters
			if lastCandle.Close >= f.minPrice &&
				lastCandle.Close <= f.maxPrice &&
				avgVolume >= f.minVolume {

				filteredStock := domain.FilteredStock{
					Stock:       stock,
					LastCandle:  lastCandle,
					ClosePrice:  lastCandle.Close,
					DailyVolume: avgVolume,
					FilterResults: map[string]bool{
						"basic_filter": true,
					},
					FilterReasons: map[string]string{
						"basic_filter": fmt.Sprintf("PASSED: Price=%.2f (min:%.2f,max:%.2f), Avg Volume=%d (min:%d)",
							lastCandle.Close, f.minPrice, f.maxPrice, avgVolume, f.minVolume),
					},
				}

				filteredStocks = append(filteredStocks, filteredStock)
				log.Info("Stock %s passed basic filter: Price=%.2f, Avg Volume=%d",
					stock.Symbol, lastCandle.Close, avgVolume)
			} else {
				var reason string
				switch {
				case lastCandle.Close < f.minPrice:
					reason = fmt.Sprintf("REJECTED: Price %.2f below minimum %.2f", lastCandle.Close, f.minPrice)
				case lastCandle.Close > f.maxPrice:
					reason = fmt.Sprintf("REJECTED: Price %.2f above maximum %.2f", lastCandle.Close, f.maxPrice)
				case avgVolume < f.minVolume:
					reason = fmt.Sprintf("REJECTED: Average volume %d below minimum %d", avgVolume, f.minVolume)
				}
				log.Debug("Stock %s failed basic filter: %s", stock.Symbol, reason)
				skippedStocks++
			}
		}

	case []domain.FilteredStock:
		log.Info("Basic filter received already filtered stocks: %d", len(input))
		filteredStocks = input
	}

	log.Info("Basic filter completed. Passed: %d, Skipped: %d",
		len(filteredStocks), skippedStocks)

	return filteredStocks, filteredStocks, nil
}
