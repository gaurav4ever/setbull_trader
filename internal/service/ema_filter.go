package service

import (
	"context"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/pkg/log"
	"time"
)

// EMAFilter implements the EMA-based filtering criteria
type EMAFilter struct {
	technicalIndicators *TechnicalIndicatorService
	emaPeriod           int
	threshold           float64 // 3% threshold
	tradingCalendar     *TradingCalendarService
}

// NewEMAFilter creates a new instance of EMAFilter
func NewEMAFilter(technicalIndicators *TechnicalIndicatorService,
	tradingCalendar *TradingCalendarService) *EMAFilter {
	return &EMAFilter{
		technicalIndicators: technicalIndicators,
		emaPeriod:           50,
		threshold:           0.03, // 3%
		tradingCalendar:     tradingCalendar,
	}
}

// Filter implements the StockFilter interface
func (f *EMAFilter) Filter(ctx context.Context, stocks interface{}) (bullish, bearish []domain.FilteredStock, err error) {

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

// processStocks handles the actual EMA filtering logic
func (f *EMAFilter) processStocks(ctx context.Context, stocks []domain.FilteredStock) (bullish, bearish []domain.FilteredStock, err error) {
	var bullishStocks, bearishStocks []domain.FilteredStock
	var skippedStocks int

	log.Info("Starting EMA filter for %d stocks", len(stocks))

	for _, stock := range stocks {
		// Get EMA values for the last 50 days
		endTime := time.Now()
		if !f.tradingCalendar.IsTradingDay(endTime) {
			endTime = f.tradingCalendar.PreviousTradingDay(endTime)
			// log.Info("Adjusted end date to previous trading day: %s", endTime.Format("2006-01-02"))
		}
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 0, 0, 0, 0, endTime.Location())

		// Calculate the requested start date based on trading days
		startTime := f.tradingCalendar.SubtractTradingDays(endTime, f.emaPeriod+10)
		startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
		// log.Info("Start time and end time: %s, %s for stock %s", startTime, endTime, stock.Stock.InstrumentKey)

		emaValues, err := f.technicalIndicators.CalculateEMA(
			ctx,
			stock.Stock.InstrumentKey,
			f.emaPeriod,
			"day",
			startTime,
			endTime,
		)
		if err != nil {
			log.Warn("Failed to calculate EMA for %s: %v", stock.Stock.InstrumentKey, err)
			skippedStocks++
			continue
		}

		if len(emaValues) == 0 {
			log.Debug("No EMA values found for %s", stock.Stock.InstrumentKey)
			skippedStocks++
			continue
		}

		// Get the latest EMA value
		latestEMA := emaValues[len(emaValues)-1].Value
		stock.EMA50 = latestEMA

		// Calculate price difference percentage
		priceDiff := (stock.ClosePrice - latestEMA) / latestEMA

		// Apply bullish/bearish conditions
		if priceDiff > f.threshold {
			stock.IsBullish = true
			stock.FilterResults["ema_filter"] = true
			stock.FilterReasons["ema_filter"] = fmt.Sprintf("BULLISH: Price %.2f is %.2f%% above EMA50 %.2f (threshold: %.2f%%)",
				stock.ClosePrice, priceDiff*100, latestEMA, f.threshold*100)
			bullishStocks = append(bullishStocks, stock)
			// log.Info("Stock %s passed bullish EMA filter: Price=%.2f, EMA=%.2f, Diff=%.2f%%",
			// 	stock.Stock.Symbol, stock.ClosePrice, latestEMA, priceDiff*100)
		} else if priceDiff < -f.threshold {
			stock.IsBearish = true
			stock.FilterResults["ema_filter"] = true
			stock.FilterReasons["ema_filter"] = fmt.Sprintf("BEARISH: Price %.2f is %.2f%% below EMA50 %.2f (threshold: %.2f%%)",
				stock.ClosePrice, priceDiff*100, latestEMA, f.threshold*100)
			bearishStocks = append(bearishStocks, stock)
			// log.Info("Stock %s passed bearish EMA filter: Price=%.2f, EMA=%.2f, Diff=%.2f%%",
			// 	stock.Stock.Symbol, stock.ClosePrice, latestEMA, priceDiff*100)
		} else {
			stock.FilterReasons["ema_filter"] = fmt.Sprintf("REJECTED: Price %.2f deviation %.2f%% from EMA50 %.2f within threshold ±%.2f%%",
				stock.ClosePrice, priceDiff*100, latestEMA, f.threshold*100)
			// log.Debug("Stock %s failed EMA filter: Price=%.2f, EMA=%.2f, Diff=%.2f%%",
			// 	stock.Stock.Symbol, stock.ClosePrice, latestEMA, priceDiff*100)
			skippedStocks++
		}
	}

	log.Info("EMA filter completed. Bullish: %d, Bearish: %d, Skipped: %d",
		len(bullishStocks), len(bearishStocks), skippedStocks)

	return bullishStocks, bearishStocks, nil
}
