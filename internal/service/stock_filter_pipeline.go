package service

import (
	"context"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
	"time"
)

// StockFilterPipeline orchestrates the stock filtering process
type StockFilterPipeline struct {
	stockUniverseService *StockUniverseService
	candleRepo           repository.CandleRepository
	technicalIndicators  *TechnicalIndicatorService
	tradingCalendar      *TradingCalendarService
	filters              []domain.StockFilter
}

// NewStockFilterPipeline creates a new instance of StockFilterPipeline
func NewStockFilterPipeline(
	stockUniverseService *StockUniverseService,
	candleRepo repository.CandleRepository,
	technicalIndicators *TechnicalIndicatorService,
	tradingCalendar *TradingCalendarService,
) *StockFilterPipeline {
	return &StockFilterPipeline{
		stockUniverseService: stockUniverseService,
		candleRepo:           candleRepo,
		technicalIndicators:  technicalIndicators,
		tradingCalendar:      tradingCalendar,
	}
}

// Add to StockFilterPipeline
type PipelineMetrics struct {
	TotalStocks     int
	BasicFilterPass int
	EMAFilterPass   int
	RSIFilterPass   int
	BullishStocks   int
	BearishStocks   int
	ProcessingTime  time.Duration
}

// RunPipeline executes the complete filtering process
func (p *StockFilterPipeline) RunPipeline(ctx context.Context) (bullish, bearish []domain.FilteredStock, metrics PipelineMetrics, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pipeline panic: %v", r)
			log.Error("Pipeline panic: %v", r)
		}
	}()

	startTime := time.Now()
	log.Info("Starting stock filter pipeline")

	// Get all stocks from universe
	stocks, totalCount, err := p.stockUniverseService.GetAllStocks(ctx, false, 1, 10000)
	if err != nil {
		return nil, nil, PipelineMetrics{}, fmt.Errorf("failed to get stocks from universe: %w", err)
	}
	log.Info("Retrieved %d stocks from universe (total: %d)", len(stocks), totalCount)

	// Initialize filters
	basicFilter := NewBasicFilter(p.candleRepo)
	emaFilter := NewEMAFilter(p.technicalIndicators, p.tradingCalendar)
	rsiFilter := NewRSIFilter(p.technicalIndicators, p.tradingCalendar)
	p.filters = []domain.StockFilter{basicFilter, emaFilter, rsiFilter}

	var currentStocks interface{} = stocks
	var basicFilterResults, emaFilterResults, rsiFilterResults []domain.FilteredStock

	// Run through each filter
	for i, filter := range p.filters {
		log.Info("Running filter %d/%d", i+1, len(p.filters))

		bullish, bearish, err = filter.Filter(ctx, currentStocks)
		if err != nil {
			return nil, nil, PipelineMetrics{}, fmt.Errorf("filter %d failed: %w", i+1, err)
		}

		// Store results for metrics
		switch i {
		case 0:
			basicFilterResults = append(bullish, bearish...)
		case 1:
			emaFilterResults = append(bullish, bearish...)
		case 2:
			rsiFilterResults = append(bullish, bearish...)
		}

		// Update stocks for next filter
		currentStocks = append(bullish, bearish...)

		log.Info("Filter %d completed. Bullish: %d, Bearish: %d", i+1, len(bullish), len(bearish))

	}

	// Log final results
	log.Info("Pipeline completed. Results:")
	log.Info("- Total stocks processed: %d", len(stocks))
	log.Info("- Bullish candidates: %d", len(bullish))
	log.Info("- Bearish candidates: %d", len(bearish))

	// Calculate metrics
	metrics = PipelineMetrics{
		TotalStocks:     len(stocks),
		BasicFilterPass: len(basicFilterResults),
		EMAFilterPass:   len(emaFilterResults),
		RSIFilterPass:   len(rsiFilterResults),
		BullishStocks:   len(bullish),
		BearishStocks:   len(bearish),
		ProcessingTime:  time.Since(startTime),
	}

	// Log detailed results
	p.logDetailedResults(bullish)
	p.logDetailedResults(bearish)

	// Validate results
	if err := p.validateResults(bullish); err != nil {
		log.Warn("Validation warning for bullish stocks: %v", err)
	}
	if err := p.validateResults(bearish); err != nil {
		log.Warn("Validation warning for bearish stocks: %v", err)
	}

	return bullish, bearish, metrics, nil
}

// Add to StockFilterPipeline
func (p *StockFilterPipeline) validateResults(stocks []domain.FilteredStock) error {
	for _, stock := range stocks {
		if !stock.FilterResults["basic_filter"] {
			return fmt.Errorf("stock %s failed basic filter", stock.Stock.Symbol)
		}
		if !stock.FilterResults["ema_filter"] {
			return fmt.Errorf("stock %s failed EMA filter", stock.Stock.Symbol)
		}
		if !stock.FilterResults["rsi_filter"] {
			return fmt.Errorf("stock %s failed RSI filter", stock.Stock.Symbol)
		}
	}
	return nil
}

// Add to StockFilterPipeline
func (p *StockFilterPipeline) logDetailedResults(stocks []domain.FilteredStock) {
	for _, stock := range stocks {
		log.Info("Stock %s details:", stock.Stock.Symbol)
		log.Info("- Price: %.2f", stock.ClosePrice)
		log.Info("- Volume: %d", stock.DailyVolume)
		log.Info("- EMA50: %.2f", stock.EMA50)
		log.Info("- RSI14: %.2f", stock.RSI14)
		log.Info("- Filter Results:")
		for filter, reason := range stock.FilterReasons {
			log.Info("  %s: %s", filter, reason)
		}
	}
}
