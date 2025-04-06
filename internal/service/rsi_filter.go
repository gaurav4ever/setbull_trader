package service

import (
	"context"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/pkg/log"
	"time"
)

// RSIFilter implements the RSI-based filtering criteria
type RSIFilter struct {
	technicalIndicators *TechnicalIndicatorService
	rsiPeriod           int
	bullishThreshold    float64
	bearishThreshold    float64
	tradingCalendar     *TradingCalendarService
}

// NewRSIFilter creates a new instance of RSIFilter
func NewRSIFilter(technicalIndicators *TechnicalIndicatorService,
	tradingCalendar *TradingCalendarService) *RSIFilter {
	return &RSIFilter{
		technicalIndicators: technicalIndicators,
		rsiPeriod:           14,
		bullishThreshold:    60.0,
		bearishThreshold:    40.0,
		tradingCalendar:     tradingCalendar,
	}
}

// Filter implements the StockFilter interface
func (f *RSIFilter) Filter(ctx context.Context, stocks interface{}) (bullish, bearish []domain.FilteredStock, err error) {
	// Handle different input types
	switch input := stocks.(type) {
	case []domain.StockUniverse:
		// Convert to FilteredStock
		var filteredStocks []domain.FilteredStock
		for _, stock := range input {
			filteredStocks = append(filteredStocks, domain.FilteredStock{
				Stock:         stock,
				FilterResults: make(map[string]bool),
			})
		}
		return f.processStocks(ctx, filteredStocks)

	case []domain.FilteredStock:
		return f.processStocks(ctx, input)
	}

	return nil, nil, fmt.Errorf("unsupported input type")
}

// processStocks handles the actual RSI filtering logic
func (f *RSIFilter) processStocks(ctx context.Context, stocks []domain.FilteredStock) (bullish, bearish []domain.FilteredStock, err error) {
	var bullishStocks, bearishStocks []domain.FilteredStock
	var skippedStocks int

	log.Info("Starting RSI filter for %d stocks", len(stocks))

	for _, stock := range stocks {
		// Get RSI values for the last 14 days
		endTime := time.Now()
		if !f.tradingCalendar.IsTradingDay(endTime) {
			endTime = f.tradingCalendar.PreviousTradingDay(endTime)
			log.Info("Adjusted end date to previous trading day: %s", endTime.Format("2006-01-02"))
		}

		// Calculate the requested start date based on trading days
		startTime := f.tradingCalendar.SubtractTradingDays(endTime, f.rsiPeriod)

		rsiValues, err := f.technicalIndicators.CalculateRSI(
			ctx,
			stock.Stock.InstrumentKey,
			f.rsiPeriod,
			"day",
			startTime,
			endTime,
		)
		if err != nil {
			log.Warn("Failed to calculate RSI for %s: %v", stock.Stock.InstrumentKey, err)
			skippedStocks++
			continue
		}

		if len(rsiValues) == 0 {
			log.Debug("No RSI values found for %s", stock.Stock.InstrumentKey)
			skippedStocks++
			continue
		}

		// Get the latest RSI value
		latestRSI := rsiValues[len(rsiValues)-1].Value
		stock.RSI14 = latestRSI

		// Apply bullish/bearish conditions
		if latestRSI >= f.bullishThreshold {
			stock.IsBullish = true
			stock.FilterResults["rsi_filter"] = true
			bullishStocks = append(bullishStocks, stock)
			log.Info("Stock %s passed bullish RSI filter: RSI=%.2f",
				stock.Stock.Symbol, latestRSI)
		} else if latestRSI <= f.bearishThreshold {
			stock.IsBearish = true
			stock.FilterResults["rsi_filter"] = true
			bearishStocks = append(bearishStocks, stock)
			log.Info("Stock %s passed bearish RSI filter: RSI=%.2f",
				stock.Stock.Symbol, latestRSI)
		} else {
			log.Debug("Stock %s failed RSI filter: RSI=%.2f",
				stock.Stock.Symbol, latestRSI)
			skippedStocks++
		}
	}

	log.Info("RSI filter completed. Bullish: %d, Bearish: %d, Skipped: %d",
		len(bullishStocks), len(bearishStocks), skippedStocks)

	return bullishStocks, bearishStocks, nil
}
