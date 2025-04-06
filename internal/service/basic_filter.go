package service

import (
	"context"
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
			// Get latest candle
			lastCandle, err := f.candleRepo.GetLatestCandle(ctx, stock.InstrumentKey, "day")
			if err != nil {
				log.Warn("Failed to get latest candle for %s: %v", stock.InstrumentKey, err)
				skippedStocks++
				continue
			}

			// Check if lastCandle is nil
			if lastCandle == nil {
				log.Debug("No candle data available for %s", stock.InstrumentKey)
				skippedStocks++
				continue
			}

			// Apply basic filters
			if lastCandle.Close >= f.minPrice &&
				lastCandle.Close <= f.maxPrice &&
				lastCandle.Volume >= f.minVolume {

				filteredStock := domain.FilteredStock{
					Stock:       stock,
					LastCandle:  *lastCandle,
					ClosePrice:  lastCandle.Close,
					DailyVolume: lastCandle.Volume,
					FilterResults: map[string]bool{
						"basic_filter": true,
					},
				}

				filteredStocks = append(filteredStocks, filteredStock)
				log.Info("Stock %s passed basic filter: Price=%.2f, Volume=%d",
					stock.Symbol, lastCandle.Close, lastCandle.Volume)
			} else {
				log.Debug("Stock %s failed basic filter: Price=%.2f, Volume=%d",
					stock.Symbol, lastCandle.Close, lastCandle.Volume)
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
