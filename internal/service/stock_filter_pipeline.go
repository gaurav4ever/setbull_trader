package service

import (
	"context"
	"fmt"
	"reflect"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/log"
	"sort"
	"strings"
	"time"
)

// StockFilterPipeline orchestrates the stock filtering process
type StockFilterPipeline struct {
	stockUniverseService *StockUniverseService
	candleRepo           repository.CandleRepository
	technicalIndicators  *TechnicalIndicatorService
	tradingCalendar      *TradingCalendarService
	filters              []domain.StockFilter
	sequenceAnalyzer     *SequenceAnalyzer
	reportGenerator      *ReportGenerator
	metrics              *PipelineMetrics
	filteredStockRepo    repository.FilteredStockRepository
	config               *config.Config
}

// NewStockFilterPipeline creates a new instance of StockFilterPipeline
func NewStockFilterPipeline(
	stockUniverseService *StockUniverseService,
	candleRepo repository.CandleRepository,
	technicalIndicators *TechnicalIndicatorService,
	tradingCalendar *TradingCalendarService,
	filteredStockRepo repository.FilteredStockRepository,
	config *config.Config,
) *StockFilterPipeline {
	return &StockFilterPipeline{
		stockUniverseService: stockUniverseService,
		candleRepo:           candleRepo,
		technicalIndicators:  technicalIndicators,
		tradingCalendar:      tradingCalendar,
		sequenceAnalyzer:     NewSequenceAnalyzer(),
		reportGenerator:      NewReportGenerator(),
		filteredStockRepo:    filteredStockRepo,
		config:               config,
		metrics: &PipelineMetrics{
			FilterMetrics:   make(map[string]*FilterMetric),
			SequenceMetrics: make(map[string]domain.SequenceMetrics),
		},
	}
}

// Add to StockFilterPipeline
type PipelineMetrics struct {
	TotalStocks     int
	BasicFilterPass int
	EMAFilterPass   int
	RSIFilterPass   int
	MambaFilterPass int
	BullishStocks   int
	BearishStocks   int
	ProcessingTime  time.Duration
	FilterMetrics   map[string]*FilterMetric
	SequenceMetrics map[string]domain.SequenceMetrics
	StartTime       time.Time
	EndTime         time.Time
}

// Add new struct for detailed filter metrics
type FilterMetric struct {
	Processed  int
	Passed     int
	Failed     int
	Bullish    int
	Bearish    int
	Duration   time.Duration
	ErrorCount int
}

type MoveStats struct {
	TotalMambaMoves    int
	BullishMambaMoves  int
	BearishMambaMoves  int
	NonMambaMoves      int
	LargestBullishMove float64
	LargestBearishMove float64
	Stock              string
	Date               time.Time
}

// GetTop10FilteredStocks retrieves the top 10 filtered stocks
func (p *StockFilterPipeline) GetTop10FilteredStocks(ctx context.Context) ([]domain.FilteredStockRecord, error) {
	return p.filteredStockRepo.GetTop10FilteredStocks(ctx)
}

// RunPipeline executes the complete filtering process
func (p *StockFilterPipeline) RunPipeline(
	ctx context.Context,
	instrumentKeys []string) (bullish, bearish []domain.FilteredStock, metrics PipelineMetrics, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pipeline panic: %v", r)
			log.Error("Pipeline panic: %v", r)
		}
	}()

	p.metrics.StartTime = time.Now()
	defer func() {
		p.metrics.EndTime = time.Now()
	}()

	startTime := time.Now()
	metrics.FilterMetrics = make(map[string]*FilterMetric)

	log.Info("Starting stock filter pipeline")

	var stocks []domain.StockUniverse
	var totalCount int64

	// Get all stocks from universe
	if len(instrumentKeys) > 0 {
		stocks, err = p.stockUniverseService.GetStocksByInstrumentKeys(ctx, instrumentKeys)
	} else {
		stocks, totalCount, err = p.stockUniverseService.GetAllStocks(ctx, false, 1, 10000)
	}

	if err != nil {
		return nil, nil, metrics, fmt.Errorf("failed to get stocks from universe: %w", err)
	}
	log.Info("Retrieved %d stocks from universe (total: %d)", len(stocks), totalCount)

	// Initialize filters

	// Basic Filter - 50,1000 price, 400000 volume
	basicFilter := NewBasicFilter(p.candleRepo)
	// EMA Filter - 3% above 40EMA
	emaFilter := NewEMAFilter(p.technicalIndicators, p.tradingCalendar)
	// RSI Filter - 60 Good for bullish, 40 Good for bearish
	rsiFilter := NewRSIFilter(p.technicalIndicators, p.tradingCalendar)
	// Mamba Filter - 5% bullish, 3% bearish
	mambaFilter := NewMambaFilter(p.candleRepo, p.config.MambaFilter, p.technicalIndicators, p.tradingCalendar)

	p.filters = []domain.StockFilter{basicFilter, emaFilter, rsiFilter, mambaFilter}

	var currentStocks interface{} = stocks
	var basicFilterResults, emaFilterResults, rsiFilterResults, mambaFilterResults []domain.FilteredStock

	// Run through each filter
	for i, filter := range p.filters {
		filterStartTime := time.Now()
		filterName := getFilterName(filter)

		log.Info("Running filter %d/%d: %s", i+1, len(p.filters), filterName)

		bullish, bearish, err = filter.Filter(ctx, currentStocks)
		if err != nil {
			return nil, nil, metrics, fmt.Errorf("filter %s failed: %w", filterName, err)
		}

		// Store results for metrics
		switch i {
		case 0:
			basicFilterResults = append(bullish, bearish...)
			metrics.BasicFilterPass = len(basicFilterResults)
		case 1:
			emaFilterResults = append(bullish, bearish...)
			metrics.EMAFilterPass = len(emaFilterResults)
		case 2:
			rsiFilterResults = append(bullish, bearish...)
			metrics.RSIFilterPass = len(rsiFilterResults)
		case 3:
			mambaFilterResults = append(bullish, bearish...)
			metrics.MambaFilterPass = len(mambaFilterResults)
		}

		// Update filter metrics
		var processedCount int
		switch v := currentStocks.(type) {
		case []domain.StockUniverse:
			processedCount = len(v)
		case []domain.FilteredStock:
			processedCount = len(v)
		}

		metrics.FilterMetrics[filterName] = &FilterMetric{
			Processed: processedCount,
			Passed:    len(bullish) + len(bearish),
			Failed:    processedCount - (len(bullish) + len(bearish)),
			Bullish:   len(bullish),
			Bearish:   len(bearish),
			Duration:  time.Since(filterStartTime),
		}

		// Update stocks for next filter
		currentStocks = append(bullish, bearish...)

		log.Info("Filter %s completed in %v. Bullish: %d, Bearish: %d",
			filterName,
			metrics.FilterMetrics[filterName].Duration,
			len(bullish),
			len(bearish))

		// Perform sequence analysis for MambaFilter
		if _, isMambaFilter := filter.(*MambaFilter); isMambaFilter {
			p.analyzeSequences(append(bullish, bearish...))
		}
	}

	// Calculate final metrics
	metrics.TotalStocks = len(stocks)
	metrics.BullishStocks = len(bullish)
	metrics.BearishStocks = len(bearish)
	metrics.ProcessingTime = time.Since(startTime)

	// Log detailed results
	p.logDetailedResults(bullish, bearish, metrics)

	// Generate HTML report
	if err := p.GenerateReport(bullish, bearish, metrics); err != nil {
		log.Warn("Failed to generate HTML report: %v", err)
	}

	// After filtering is complete and before returning results
	allStocks := append(bullish, bearish...)
	if err := p.storeFilteredStocks(ctx, allStocks); err != nil {
		log.Error("Failed to store filtered stocks: %v", err)
		// Don't return error here as it's not critical to pipeline operation
	}

	return bullish, bearish, metrics, nil
}

// Add helper function to get filter name
func getFilterName(filter domain.StockFilter) string {
	return strings.TrimSuffix(reflect.TypeOf(filter).Elem().Name(), "Filter")
}

// Update logDetailedResults method
func (p *StockFilterPipeline) logDetailedResults(bullish, bearish []domain.FilteredStock, metrics PipelineMetrics) {
	log.Info("Pipeline completed. Results:")
	log.Info("- Total processing time: %v", metrics.ProcessingTime)
	log.Info("- Total stocks processed: %d", metrics.TotalStocks)

	// Log filter-specific metrics
	for filterName, metric := range metrics.FilterMetrics {
		log.Info("- %s Filter:", filterName)
		log.Info("  • Processed: %d", metric.Processed)
		log.Info("  • Passed: %d (%.1f%%)", metric.Passed, float64(metric.Passed)/float64(metric.Processed)*100)
		log.Info("  • Bullish: %d, Bearish: %d", metric.Bullish, metric.Bearish)
		log.Info("  • Duration: %v", metric.Duration)
	}

	// Log stock details
	log.Info("Bullish Stocks:")
	for _, stock := range bullish {
		p.logStockDetails(stock)
		p.logMambaSeries(stock)
	}

	log.Info("Bearish Stocks:")
	for _, stock := range bearish {
		p.logStockDetails(stock)
		p.logMambaSeries(stock)
	}

	// Log consolidated move series
	p.logConsolidatedMoveSeries(append(bullish, bearish...))
}

// Add helper method for logging stock details
func (p *StockFilterPipeline) logStockDetails(stock domain.FilteredStock) {
	log.Info("Stock %s (%s):", stock.Stock.Symbol, stock.Stock.InstrumentKey)
	log.Info("- Price: %.2f", stock.ClosePrice)
	log.Info("- Volume: %d", stock.DailyVolume)
	log.Info("- EMA50: %.2f", stock.EMA50)
	log.Info("- RSI14: %.2f", stock.RSI14)
	log.Info("- Filter Results:")
	for filter, reason := range stock.FilterReasons {
		log.Info("  • %s: %s", filter, reason)
	}
}

// Add new method for logging Mamba series
func (p *StockFilterPipeline) logMambaSeries(stock domain.FilteredStock) {
	if _, ok := stock.FilterReasons["mamba_filter"]; ok {
		log.Info("Mamba Move Series for %s:", stock.Stock.Symbol)
		log.Info("------------------------")

		// Get the last 21 days of candles
		candles, err := p.candleRepo.GetNDailyCandlesByTimeframe(
			context.Background(),
			stock.Stock.InstrumentKey,
			"day",
			21,
		)
		if err != nil {
			log.Error("Failed to get candles for move series: %v", err)
			return
		}

		// Sort candles from oldest to newest
		sort.Slice(candles, func(i, j int) bool {
			return candles[i].Timestamp.Before(candles[j].Timestamp)
		})

		// Print the series
		log.Info("Date\t\tMove Type\tChange%%\tOpen\tHigh\tLow\tClose")
		log.Info("----\t\t---------\t-------\t----\t----\t---\t-----")

		for _, candle := range candles {
			movePerc := ((candle.High - candle.Low) / candle.Low) * 100
			moveType := "Non-Mamba"

			if movePerc >= 5.0 && candle.Close > candle.Open {
				moveType = "BULL-MAMBA"
			} else if movePerc >= 3.0 && candle.Close < candle.Open {
				moveType = "BEAR-MAMBA"
			}

			log.Info("%s\t%s\t%.2f%%\t%.2f\t%.2f\t%.2f\t%.2f",
				candle.Timestamp.Format("2006-01-02"),
				moveType,
				movePerc,
				candle.Open,
				candle.High,
				candle.Low,
				candle.Close,
			)
		}
		log.Info("------------------------")
	}
}

// Add method for consolidated move series analysis
func (p *StockFilterPipeline) logConsolidatedMoveSeries(stocks []domain.FilteredStock) {
	log.Info("\nConsolidated Mamba Move Analysis")
	log.Info("================================")

	stats := make(map[string]*MoveStats)

	for _, stock := range stocks {
		candles, err := p.candleRepo.GetNDailyCandlesByTimeframe(
			context.Background(),
			stock.Stock.InstrumentKey,
			"day",
			21,
		)
		if err != nil {
			continue
		}

		stockStats := &MoveStats{Stock: stock.Stock.Symbol}

		for _, candle := range candles {
			movePerc := ((candle.High - candle.Low) / candle.Low) * 100

			if movePerc >= 5.0 && candle.Close > candle.Open {
				stockStats.BullishMambaMoves++
				stockStats.TotalMambaMoves++
				if movePerc > stockStats.LargestBullishMove {
					stockStats.LargestBullishMove = movePerc
					stockStats.Date = candle.Timestamp
				}
			} else if movePerc >= 3.0 && candle.Close < candle.Open {
				stockStats.BearishMambaMoves++
				stockStats.TotalMambaMoves++
				if movePerc > stockStats.LargestBearishMove {
					stockStats.LargestBearishMove = movePerc
					stockStats.Date = candle.Timestamp
				}
			} else {
				stockStats.NonMambaMoves++
			}
		}

		stats[stock.Stock.Symbol] = stockStats
	}

	// Print consolidated statistics
	log.Info("\nMove Pattern Summary:")
	log.Info("Symbol\tTotal Mamba\tBull\tBear\tNon-Mamba\tLargest Bull\tLargest Bear")
	log.Info("------\t-----------\t----\t----\t---------\t------------\t------------")

	for _, stat := range stats {
		log.Info("%s\t%d\t\t%d\t%d\t%d\t\t%.2f%%\t\t%.2f%%",
			stat.Stock,
			stat.TotalMambaMoves,
			stat.BullishMambaMoves,
			stat.BearishMambaMoves,
			stat.NonMambaMoves,
			stat.LargestBullishMove,
			stat.LargestBearishMove,
		)
	}

	// Print pattern distribution
	log.Info("\nPattern Distribution Analysis:")
	log.Info("- Stocks with >3 Mamba moves: %d", countStocksWithMambaMoves(stats, 3))
	log.Info("- Stocks with >5 Mamba moves: %d", countStocksWithMambaMoves(stats, 5))
	log.Info("- Stocks with dominant bullish pattern: %d", countStocksWithDominantPattern(stats, true))
	log.Info("- Stocks with dominant bearish pattern: %d", countStocksWithDominantPattern(stats, false))
}

// Helper functions for move analysis
func countStocksWithMambaMoves(stats map[string]*MoveStats, threshold int) int {
	count := 0
	for _, stat := range stats {
		if stat.TotalMambaMoves > threshold {
			count++
		}
	}
	return count
}

func countStocksWithDominantPattern(stats map[string]*MoveStats, bullish bool) int {
	count := 0
	for _, stat := range stats {
		if bullish && stat.BullishMambaMoves > stat.BearishMambaMoves*2 {
			count++
		} else if !bullish && stat.BearishMambaMoves > stat.BullishMambaMoves*2 {
			count++
		}
	}
	return count
}

func (p *StockFilterPipeline) analyzeSequences(stocks []domain.FilteredStock) {
	for _, stock := range stocks {
		// Perform sequence analysis
		metrics := p.sequenceAnalyzer.AnalyzeSequences(stock.SequenceAnalysis)
		p.metrics.SequenceMetrics[stock.Stock.Symbol] = domain.SequenceMetrics{
			Stock:           stock.Stock,
			SequenceQuality: metrics.SequenceQuality,
			ContinuityScore: metrics.ContinuityScore,
			PredictiveScore: metrics.PredictiveScore,
			MomentumScore:   metrics.MomentumScore,
			VolumeProfile: domain.VolumeProfile{
				AverageVolume:     metrics.VolumeProfile.AverageVolume,
				VolumeStrength:    metrics.VolumeProfile.VolumeStrength,
				VolumeTrend:       metrics.VolumeProfile.VolumeTrend,
				VolumeConsistency: metrics.VolumeProfile.VolumeConsistency,
			},
		}
	}
}

// Add new method to store filtered stocks
func (p *StockFilterPipeline) storeFilteredStocks(ctx context.Context, stocks []domain.FilteredStock) error {
	var records []domain.FilteredStockRecord
	filterDate := time.Now().Truncate(24 * time.Hour)

	for _, stock := range stocks {
		bullishCount := 0
		bearishCount := 0
		if stock.SequenceAnalysis.Trend.Type == domain.BullishTrend {
			bullishCount = stock.SequenceAnalysis.TotalMambaDays
		} else {
			bearishCount = stock.SequenceAnalysis.TotalMambaDays
		}

		// Use config value for minimum Mamba days
		if stock.SequenceAnalysis.TotalMambaDays >= p.config.MambaFilter.MinMambaDays {
			records = append(records, domain.FilteredStockRecord{
				Symbol:            stock.Stock.Symbol,
				InstrumentKey:     stock.Stock.InstrumentKey,
				ExchangeToken:     stock.Stock.ExchangeToken,
				CurrentPrice:      stock.ClosePrice,
				MambaCount:        stock.SequenceAnalysis.TotalMambaDays,
				BullishMambaCount: bullishCount,
				BearishMambaCount: bearishCount,
				AvgMambaMove:      stock.SequenceAnalysis.AverageMambaLen,
				AvgNonMambaMove:   stock.SequenceAnalysis.AverageNonMambaLen,
				MambaSeries:       stock.SequenceAnalysis.MambaSequences,
				NonMambaSeries:    stock.SequenceAnalysis.NonMambaSequences,
				FilterDate:        filterDate,
				Trend:             string(stock.SequenceAnalysis.Trend.Type),
			})
		}
	}

	if len(records) > 0 {
		if err := p.filteredStockRepo.StoreBatch(ctx, records); err != nil {
			return fmt.Errorf("failed to store filtered stocks: %w", err)
		}
		log.Info("Stored %d filtered stocks with significant Mamba moves", len(records))
	}

	return nil
}
