package service

import (
	"context"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"setbull_trader/internal/domain"
	"sort"
	"time"
)

type StockReport struct {
	Timestamp           time.Time
	PipelineMetrics     PipelineMetrics
	BullishStocks       []StockAnalysis
	BearishStocks       []StockAnalysis
	ConsolidatedStats   map[string]*MoveStats
	PatternDistribution PatternDistribution
}

type StockAnalysis struct {
	Symbol        string
	InstrumentKey string
	Price         float64
	Volume        int64
	EMA50         float64
	RSI14         float64
	FilterResults map[string]string
	MambaSeries   []DayMove
}

type DayMove struct {
	Date     time.Time
	MoveType string
	Change   float64
	Open     float64
	High     float64
	Low      float64
	Close    float64
}

type PatternDistribution struct {
	StocksWithThreeMoves   int
	StocksWithFiveMoves    int
	DominantBullishPattern int
	DominantBearishPattern int
}

type ReportGenerator struct {
	outputDir string
}

type PipelineReport struct {
	ExecutionTime   time.Duration
	FilterMetrics   map[string]*FilterMetric
	SequenceMetrics map[string]domain.SequenceMetrics
	FilteredStocks  []domain.FilteredStock
	GeneratedAt     time.Time
}

func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{
		outputDir: "reports",
	}
}

func (rg *ReportGenerator) GenerateReport(report *PipelineReport) error {
	// Ensure reports directory exists
	if err := os.MkdirAll(rg.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create reports directory: %w", err)
	}

	// Generate report filename with timestamp
	filename := filepath.Join(rg.outputDir,
		fmt.Sprintf("stock_analysis_%s.html",
			report.GeneratedAt.Format("2006-01-02_15-04-05")))

	// Create the report file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	// Execute template with report data
	return rg.executeTemplate(file, report)
}

func (rg *ReportGenerator) executeTemplate(file *os.File, report *PipelineReport) error {
	// Sort stocks by sequence quality score
	sortedStocks := make([]StockAnalysis, 0, len(report.FilteredStocks))
	for _, stock := range report.FilteredStocks {
		//metrics := report.SequenceMetrics[stock.Stock.Symbol]
		analysis := StockAnalysis{
			Symbol:        stock.Stock.Symbol,
			InstrumentKey: stock.Stock.InstrumentKey,
			Price:         stock.ClosePrice,
			Volume:        stock.DailyVolume,
			EMA50:         stock.EMA50,
			RSI14:         stock.RSI14,
			FilterResults: stock.FilterReasons,
			MambaSeries:   make([]DayMove, 0),
		}
		sortedStocks = append(sortedStocks, analysis)
	}

	sort.Slice(sortedStocks, func(i, j int) bool {
		return sortedStocks[i].Price > sortedStocks[j].Price
	})

	data := struct {
		Report        *PipelineReport
		StockAnalysis []StockAnalysis
		Summary       ReportSummary
	}{
		Report:        report,
		StockAnalysis: sortedStocks,
		Summary:       generateReportSummary(report),
	}

	tmpl := template.Must(template.New("report").Parse(reportTemplate))
	return tmpl.Execute(file, data)
}

type ReportSummary struct {
	TotalStocks         int
	HighQualityStocks   int
	AverageQuality      float64
	StrongTrends        int
	HighMomentum        int
	PatternDistribution map[string]int
}

func generateReportSummary(report *PipelineReport) ReportSummary {
	summary := ReportSummary{
		TotalStocks:         len(report.FilteredStocks),
		PatternDistribution: make(map[string]int),
	}

	var totalQuality float64
	for _, metrics := range report.SequenceMetrics {
		totalQuality += metrics.SequenceQuality

		if metrics.SequenceQuality >= 0.8 {
			summary.HighQualityStocks++
		}
		if math.Abs(metrics.TrendStrength) >= 0.7 {
			summary.StrongTrends++
		}
		if math.Abs(metrics.MomentumScore) >= 0.6 {
			summary.HighMomentum++
		}

		// Count pattern types
		for _, pattern := range metrics.PricePatterns {
			summary.PatternDistribution[pattern.Type]++
		}
	}

	if summary.TotalStocks > 0 {
		summary.AverageQuality = totalQuality / float64(summary.TotalStocks)
	}

	return summary
}

const reportTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Stock Analysis Report - {{.Report.GeneratedAt.Format "2006-01-02 15:04:05"}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f8f9fa; padding: 20px; margin-bottom: 20px; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .metric-card { background: white; padding: 15px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .stock-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 20px; }
        .stock-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .quality-high { color: #28a745; }
        .quality-medium { color: #ffc107; }
        .quality-low { color: #dc3545; }
        .sequence-chart { height: 100px; background: #f8f9fa; margin: 10px 0; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        .pattern-tag { display: inline-block; padding: 4px 8px; border-radius: 4px; margin: 2px; }
        .pattern-bullish { background: #d4edda; color: #155724; }
        .pattern-bearish { background: #f8d7da; color: #721c24; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Stock Analysis Report</h1>
        <p>Generated: {{.Report.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        <p>Execution Time: {{.Report.ExecutionTime}}</p>
    </div>

    <div class="metrics">
        <div class="metric-card">
            <h3>Summary Statistics</h3>
            <p>Total Stocks: {{.Summary.TotalStocks}}</p>
            <p>High Quality Stocks: {{.Summary.HighQualityStocks}}</p>
            <p>Average Quality: {{printf "%.2f" .Summary.AverageQuality}}</p>
            <p>Strong Trends: {{.Summary.StrongTrends}}</p>
            <p>High Momentum: {{.Summary.HighMomentum}}</p>
        </div>

        <div class="metric-card">
            <h3>Pattern Distribution</h3>
            {{range $type, $count := .Summary.PatternDistribution}}
            <p>{{$type}}: {{$count}}</p>
            {{end}}
        </div>
    </div>

    <h2>Stock Analysis</h2>
    <div class="stock-grid">
        {{range .StockAnalysis}}
        <div class="stock-card">
            <h3>{{.Symbol}}</h3>
            <div class="metrics">
                <p>Quality Score: 
                    <span class="{{if ge .Price 0.8}}quality-high
                        {{else if ge .Price 0.5}}quality-medium
                        {{else}}quality-low{{end}}">
                        {{printf "%.2f" .Price}}
                    </span>
                </p>
                <p>Momentum: {{printf "%.2f" .RSI14}}</p>
                <p>Trend Strength: {{printf "%.2f" .EMA50}}</p>
                <p>Volatility: {{printf "%.2f" .Price}}</p>
            </div>

            <h4>Price Patterns</h4>
            {{range .FilterResults}}
            <span class="pattern-tag pattern-{{lower .}}">
                {{.}}
            </span>
            {{end}}

            <h4>Sequence Statistics</h4>
            <table>
                <tr>
                    <th>Metric</th>
                    <th>Value</th>
                </tr>
                <tr>
                    <td>Avg Sequence Length</td>
                    <td>{{printf "%.2f" .EMA50}}</td>
                </tr>
                <tr>
                    <td>Consistency Score</td>
                    <td>{{printf "%.2f" .RSI14}}</td>
                </tr>
                <tr>
                    <td>Predictability Score</td>
                    <td>{{printf "%.2f" .Price}}</td>
                </tr>
            </table>
        </div>
        {{end}}
    </div>
</body>
</html>
`

func (p *StockFilterPipeline) GenerateReport(bullish, bearish []domain.FilteredStock, metrics PipelineMetrics) error {
	// Create reports directory if it doesn't exist
	reportsDir := "reports"
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return fmt.Errorf("failed to create reports directory: %w", err)
	}

	// Generate report data
	report := StockReport{
		Timestamp:       time.Now(),
		PipelineMetrics: metrics,
		BullishStocks:   make([]StockAnalysis, 0, len(bullish)),
		BearishStocks:   make([]StockAnalysis, 0, len(bearish)),
	}

	// Process bullish stocks
	for _, stock := range bullish {
		analysis := p.createStockAnalysis(stock)
		report.BullishStocks = append(report.BullishStocks, analysis)
	}

	// Process bearish stocks
	for _, stock := range bearish {
		analysis := p.createStockAnalysis(stock)
		report.BearishStocks = append(report.BearishStocks, analysis)
	}

	// Generate consolidated stats
	report.ConsolidatedStats = p.generateConsolidatedStats(append(bullish, bearish...))
	report.PatternDistribution = p.calculatePatternDistribution(report.ConsolidatedStats)

	// Generate HTML report
	return p.generateHTMLReport(report)
}

func (p *StockFilterPipeline) createStockAnalysis(stock domain.FilteredStock) StockAnalysis {
	analysis := StockAnalysis{
		Symbol:        stock.Stock.Symbol,
		InstrumentKey: stock.Stock.InstrumentKey,
		Price:         stock.ClosePrice,
		Volume:        stock.DailyVolume,
		EMA50:         stock.EMA50,
		RSI14:         stock.RSI14,
		FilterResults: stock.FilterReasons,
		MambaSeries:   make([]DayMove, 0),
	}

	// Get Mamba series data
	candles, err := p.candleRepo.GetNDailyCandlesByTimeframe(
		context.Background(),
		stock.Stock.InstrumentKey,
		"day",
		21,
	)
	if err == nil {
		for _, candle := range candles {
			movePerc := ((candle.High - candle.Low) / candle.Low) * 100
			moveType := "Non-Mamba"

			if movePerc >= 5.0 && candle.Close > candle.Open {
				moveType = "BULL-MAMBA"
			} else if movePerc >= 3.0 && candle.Close < candle.Open {
				moveType = "BEAR-MAMBA"
			}

			analysis.MambaSeries = append(analysis.MambaSeries, DayMove{
				Date:     candle.Timestamp,
				MoveType: moveType,
				Change:   movePerc,
				Open:     candle.Open,
				High:     candle.High,
				Low:      candle.Low,
				Close:    candle.Close,
			})
		}
	}

	return analysis
}

func (p *StockFilterPipeline) generateHTMLReport(report StockReport) error {
	// Create the HTML template
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Stock Filter Pipeline Report - {{.Timestamp.Format "2006-01-02 15:04:05"}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .section { margin: 20px 0; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        table { width: 100%; border-collapse: collapse; margin: 10px 0; }
        th, td { padding: 8px; text-align: left; border: 1px solid #ddd; }
        th { background-color: #f5f5f5; }
        .bullish { color: green; }
        .bearish { color: red; }
        .mamba-series { font-size: 0.9em; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 10px; }
        .metric-card { padding: 10px; background: #f9f9f9; border-radius: 5px; }
        .bull-mamba { background-color: #e6ffe6; }
        .bear-mamba { background-color: #ffe6e6; }
    </style>
</head>
<body>
    <h1>Stock Filter Pipeline Report</h1>
    <p>Generated: {{.Timestamp.Format "2006-01-02 15:04:05"}}</p>

    <div class="section">
        <h2>Pipeline Metrics</h2>
        <div class="metrics">
            <div class="metric-card">
                <h3>Total Stocks</h3>
                <p>{{.PipelineMetrics.TotalStocks}}</p>
            </div>
            {{range $name, $metric := .PipelineMetrics.FilterMetrics}}
            <div class="metric-card">
                <h3>{{$name}} Filter</h3>
                <p>Processed: {{$metric.Processed}}</p>
                <p>Passed: {{$metric.Passed}} ({{percentage $metric.Passed $metric.Processed}}%)</p>
                <p>Bullish: {{$metric.Bullish}}</p>
                <p>Bearish: {{$metric.Bearish}}</p>
            </div>
            {{end}}
        </div>
    </div>

    <div class="section">
        <h2>Bullish Stocks ({{len .BullishStocks}})</h2>
        {{template "stockTable" .BullishStocks}}
    </div>

    <div class="section">
        <h2>Bearish Stocks ({{len .BearishStocks}})</h2>
        {{template "stockTable" .BearishStocks}}
    </div>

    <div class="section">
        <h2>Pattern Distribution</h2>
        <div class="metrics">
            <div class="metric-card">
                <p>Stocks with >3 Mamba moves: {{.PatternDistribution.StocksWithThreeMoves}}</p>
                <p>Stocks with >5 Mamba moves: {{.PatternDistribution.StocksWithFiveMoves}}</p>
                <p>Dominant Bullish Pattern: {{.PatternDistribution.DominantBullishPattern}}</p>
                <p>Dominant Bearish Pattern: {{.PatternDistribution.DominantBearishPattern}}</p>
            </div>
        </div>
    </div>
</body>
</html>

{{define "stockTable"}}
<table>
    <tr>
        <th>Symbol</th>
        <th>Price</th>
        <th>Volume</th>
        <th>EMA50</th>
        <th>RSI14</th>
        <th>Mamba Series</th>
    </tr>
    {{range .}}
    <tr>
        <td>{{.Symbol}}</td>
        <td>{{printf "%.2f" .Price}}</td>
        <td>{{.Volume}}</td>
        <td>{{printf "%.2f" .EMA50}}</td>
        <td>{{printf "%.2f" .RSI14}}</td>
        <td>
            <table class="mamba-series">
                {{range .MambaSeries}}
                <tr class="{{if eq .MoveType "BULL-MAMBA"}}bull-mamba{{else if eq .MoveType "BEAR-MAMBA"}}bear-mamba{{end}}">
                    <td>{{.Date.Format "2006-01-02"}}</td>
                    <td>{{.MoveType}}</td>
                    <td>{{printf "%.2f%%" .Change}}</td>
                </tr>
                {{end}}
            </table>
        </td>
    </tr>
    {{end}}
</table>
{{end}}
`

	// Create a new template and parse the HTML
	t, err := template.New("report").Funcs(template.FuncMap{
		"percentage": func(n, total int) float64 {
			if total == 0 {
				return 0
			}
			return float64(n) * 100 / float64(total)
		},
	}).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create the output file
	filename := filepath.Join("reports", fmt.Sprintf("stock_report_%s.html",
		report.Timestamp.Format("20060102_150405")))
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer f.Close()

	// Execute the template
	if err := t.Execute(f, report); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func (p *StockFilterPipeline) generateConsolidatedStats(stocks []domain.FilteredStock) map[string]*MoveStats {
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

		stockStats := &MoveStats{
			Stock: stock.Stock.Symbol,
		}

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

	return stats
}

func (p *StockFilterPipeline) calculatePatternDistribution(stats map[string]*MoveStats) PatternDistribution {
	distribution := PatternDistribution{}

	for _, stat := range stats {
		// Count stocks with more than 3 Mamba moves
		if stat.TotalMambaMoves > 3 {
			distribution.StocksWithThreeMoves++
		}

		// Count stocks with more than 5 Mamba moves
		if stat.TotalMambaMoves > 5 {
			distribution.StocksWithFiveMoves++
		}

		// Count stocks with dominant patterns
		// Bullish dominance: More than twice as many bullish moves as bearish
		if stat.BullishMambaMoves > stat.BearishMambaMoves*2 {
			distribution.DominantBullishPattern++
		}
		// Bearish dominance: More than twice as many bearish moves as bullish
		if stat.BearishMambaMoves > stat.BullishMambaMoves*2 {
			distribution.DominantBearishPattern++
		}
	}

	return distribution
}
