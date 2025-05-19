package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"setbull_trader/internal/core/dto/response"
	dto "setbull_trader/internal/core/dto/response"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/log"
	"strconv"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

type StockBackTestAnalysis struct {
	StockID   string `json:"STOCK_ID"`
	Symbol    string `json:"SYMBOL"`
	Trend     string `json:"TREND"`
	Direction string `json:"DIRECTION"`
	Strategy  string `json:"STRATEGY"`
	EntryType string `json:"ENTRY_TYPE"`
	EntryTime string `json:"ENTRY_TIME"`
	SLPercent string `json:"SL%"`
	PSType    string `json:"PS_TYPE"`
}

type GroupExecutionService struct {
	StockGroupService         *StockGroupService
	MarketQuoteService        *MarketQuoteService
	TradeParametersService    *TradeParametersService
	ExecutionPlanService      *ExecutionPlanService
	OrderExecutionService     *OrderExecutionService
	Config                    *config.Config
	StockUniverseService      *StockUniverseService
	TechnicalIndicatorService *TechnicalIndicatorService
}

func NewGroupExecutionService(
	stockGroupSvc *StockGroupService,
	marketQuoteSvc *MarketQuoteService,
	tradeParamsSvc *TradeParametersService,
	execPlanSvc *ExecutionPlanService,
	orderExecSvc *OrderExecutionService,
	cfg *config.Config,
	stockUnivSvc *StockUniverseService,
	technicalIndicatorSvc *TechnicalIndicatorService,
) *GroupExecutionService {
	return &GroupExecutionService{
		StockGroupService:         stockGroupSvc,
		MarketQuoteService:        marketQuoteSvc,
		TradeParametersService:    tradeParamsSvc,
		ExecutionPlanService:      execPlanSvc,
		OrderExecutionService:     orderExecSvc,
		Config:                    cfg,
		StockUniverseService:      stockUnivSvc,
		TechnicalIndicatorService: technicalIndicatorSvc,
	}
}

// ExecuteGroup is a stub for the group execution orchestration logic
func (s *GroupExecutionService) ExecuteGroup(ctx context.Context, groupID string) error {
	// Fetch the group by ID
	group, err := s.StockGroupService.GetGroupByID(ctx, groupID, s.StockUniverseService)
	if err != nil {
		log.Error("failed to fetch group: %w", err)
		return fmt.Errorf("failed to fetch group: %w", err)
	}
	if len(group.Stocks) == 0 {
		log.Error("group is empty or has no stocks")
		return fmt.Errorf("group is empty or has no stocks")
	}

	// For each stock, fetch the latest price
	stockInstrumentKeys := make([]string, 0, len(group.Stocks))
	for _, stockRef := range group.Stocks {
		stockInstrumentKeys = append(stockInstrumentKeys, stockRef.InstrumentKey)
	}

	tradingMetadata, err := parseBackTestAnalysisFile(group.Stocks)
	if err != nil {
		log.Error("failed to parse backtest analysis file: %w", err)
		return fmt.Errorf("failed to parse backtest analysis file: %w", err)
	}

	quotesResp := s.MarketQuoteService.GetQuotes(
		ctx,
		"upstox_session",
		stockInstrumentKeys,
		"1min",
		"instrument_key",
		s.StockUniverseService,
	)
	if quotesResp == nil || quotesResp.Status != "success" {
		log.Error("failed to fetch market quotes: %v", quotesResp.Errors)
		return fmt.Errorf("failed to fetch market quotes: %v", quotesResp.Errors)
	}

	// Risk config
	var riskPerTrade int
	if group.EntryType == "1ST_ENTRY" {
		riskPerTrade = s.Config.GetFirstEntryRiskPerTrade()
	} else {
		riskPerTrade = s.Config.GetSecondEntryRiskPerTrade()
	}
	if riskPerTrade <= 0 {
		riskPerTrade = 50
	}

	type stockExecutionResult struct {
		StockID string
		Symbol  string
		Success bool
		Error   string
	}
	results := make([]stockExecutionResult, 0, len(group.Stocks))
	var anyFailed bool

	for stockInstrumentKey, ohlc := range quotesResp.Data {
		tradingMetadata, ok := tradingMetadata[stockInstrumentKey]
		if !ok {
			results = append(results, stockExecutionResult{StockID: stockInstrumentKey, Success: false, Error: "no trading metadata found"})
			anyFailed = true
			continue
		}
		currentPrice := ohlc.Close
		if currentPrice <= 0 {
			results = append(results, stockExecutionResult{StockID: stockInstrumentKey, Success: false, Error: "invalid price"})
			anyFailed = true
			continue
		}
		// DECIDE STOP LOSS
		slPercent, err := strconv.ParseFloat(tradingMetadata.SLPercent, 64)
		if err != nil {
			results = append(results, stockExecutionResult{StockID: stockInstrumentKey, Success: false, Error: "invalid SL: " + err.Error()})
			anyFailed = true
			continue
		}
		sl := currentPrice * slPercent / 100.0
		if sl <= 0 {
			results = append(results, stockExecutionResult{StockID: stockInstrumentKey, Success: false, Error: "invalid SL"})
			anyFailed = true
			continue
		}
		// DECIDE POSITION SIZE
		positionSize := int(float64(riskPerTrade) / sl)
		if positionSize <= 0 {
			results = append(results, stockExecutionResult{StockID: stockInstrumentKey, Success: false, Error: "invalid position size"})
			anyFailed = true
			continue
		}
		var tradeSide domain.TradeSide
		if tradingMetadata.Direction == "LONG" {
			tradeSide = domain.Buy
		} else {
			tradeSide = domain.Sell
		}
		// CREATE TRADE PARAMETERS
		params := &domain.TradeParameters{
			StockID:            tradingMetadata.StockID,
			StartingPrice:      currentPrice,
			StopLossPercentage: slPercent,
			RiskAmount:         float64(riskPerTrade),
			TradeSide:          tradeSide,
			PSType:             tradingMetadata.PSType,
			EntryType:          group.EntryType,
			Active:             true,
		}
		err = s.TradeParametersService.CreateOrUpdateTradeParameters(ctx, params)
		if err != nil {
			results = append(results, stockExecutionResult{
				StockID: stockInstrumentKey,
				Symbol:  tradingMetadata.Symbol,
				Success: false,
				Error:   "trade param error: " + err.Error(),
			})
			anyFailed = true
			continue
		}
		// CREATE EXECUTION PLAN
		_, err = s.ExecutionPlanService.CreateExecutionPlan(ctx, tradingMetadata.StockID)
		if err != nil {
			results = append(results, stockExecutionResult{
				StockID: stockInstrumentKey,
				Symbol:  tradingMetadata.Symbol,
				Success: false,
				Error:   "exec plan error: " + err.Error(),
			})
			anyFailed = true
			continue
		}
		// EXECUTE ORDERS
		_, _, err = s.OrderExecutionService.ExecuteOrdersForStock(ctx, tradingMetadata.StockID)
		if err != nil {
			results = append(results, stockExecutionResult{
				StockID: stockInstrumentKey,
				Symbol:  tradingMetadata.Symbol,
				Success: false,
				Error:   "order exec error: " + err.Error(),
			})
			anyFailed = true
			continue
		}
		// SUCCESS
		results = append(results, stockExecutionResult{
			StockID: stockInstrumentKey,
			Symbol:  tradingMetadata.Symbol,
			Success: true,
			Error:   "",
		})
	}

	// Log summary
	for _, res := range results {
		if res.Success {
			log.Info("GroupExec] Stock %s: SUCCESS\n", res.StockID)
		} else {
			log.Error("GroupExec] Stock %s: FAIL (%s)\n", res.StockID, res.Error)
		}
	}

	if anyFailed {
		return fmt.Errorf("one or more stocks failed in group execution; see logs for details")
	}
	return nil
}

func parseBackTestAnalysisFile(stocks []response.StockGroupStockDTO) (map[string]StockBackTestAnalysis, error) {
	filePath := "/Users/gaurav/setbull_projects/setbull_trader/python_strategies/backtest_results/strategy_results/backtest_analysis.csv"
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 1 {
		return nil, nil // no data
	}

	header := records[0]
	idx := make(map[string]int)
	for i, col := range header {
		idx[col] = i
	}

	result := make(map[string]StockBackTestAnalysis)
	for _, row := range records[1:] {
		symbol := row[idx["SYMBOL"]]
		analysis := StockBackTestAnalysis{
			Trend:     row[idx["TREND"]],
			Direction: row[idx["DIRECTION"]],
			Strategy:  row[idx["STRATEGY"]],
			EntryType: row[idx["ENTRY_TYPE"]],
			EntryTime: row[idx["ENTRY_TIME"]],
			SLPercent: row[idx["SL%"]],
			PSType:    row[idx["PS_TYPE"]],
		}
		result[symbol] = analysis
	}
	finalResult := make(map[string]StockBackTestAnalysis)
	for _, stock := range stocks {
		analysis, ok := result[stock.Symbol]
		if !ok {
			return nil, fmt.Errorf("no backtest analysis found for stock: %s", stock.InstrumentKey)
		}
		finalResult[stock.InstrumentKey] = StockBackTestAnalysis{
			StockID:   stock.StockID,
			Symbol:    stock.Symbol,
			Trend:     analysis.Trend,
			Direction: analysis.Direction,
			Strategy:  analysis.Strategy,
			EntryType: analysis.EntryType,
			EntryTime: analysis.EntryTime,
			SLPercent: analysis.SLPercent,
			PSType:    analysis.PSType,
		}
	}
	return finalResult, nil
}

// ExecuteGroupWithCandle executes a group using the provided candle context (for scheduled execution)
func (s *GroupExecutionService) ExecuteGroupWithCandle(
	ctx context.Context,
	group dto.StockGroupResponse,
	candle domain.AggregatedCandle,
	candleTime string,
) error {
	if len(group.Stocks) == 0 {
		log.Error("group is empty or has no stocks")
		return fmt.Errorf("group is empty or has no stocks")
	}

	type stockExecutionResult struct {
		StockID string
		Symbol  string
		Success bool
		Error   string
	}
	results := make([]stockExecutionResult, 0, len(group.Stocks))
	var anyFailed bool

	tradingMetadata, err := parseBackTestAnalysisFile(group.Stocks) // You may need to adapt this to get correct metadata
	if err != nil {
		log.Error("failed to parse backtest analysis file: %w", err)
		return fmt.Errorf("failed to parse backtest analysis file: %w", err)
	}

	for _, stockRef := range group.Stocks {
		stock, err := s.getAndSelectStock(ctx, stockRef)
		if err != nil {
			results = append(results, stockExecutionResult{StockID: stockRef.StockID, Success: false, Error: err.Error()})
			anyFailed = true
			continue
		}
		shouldExecute := false
		if candleTime == "9:15" {
			shouldExecute = s.validateForMorningEntry(ctx, &stockRef, candle)
		} else if candleTime == "13:00" {
			shouldExecute = s.validateForAfternoonEntry(ctx, &stockRef, candle)
		}

		if !shouldExecute {
			continue
		}

		_, meta, err := s.getInstrumentKeyAndMetadata(ctx, stock, tradingMetadata)
		if err != nil {
			results = append(results, stockExecutionResult{StockID: stockRef.StockID, Symbol: stock.Symbol, Success: false, Error: err.Error()})
			anyFailed = true
			continue
		}

		entryPrice, err := s.calculateEntryPrice(ctx, meta, candle)
		if err != nil {
			results = append(results, stockExecutionResult{StockID: stockRef.StockID, Symbol: stock.Symbol, Success: false, Error: err.Error()})
			anyFailed = true
			continue
		}

		slPrice, _, positionSize, tradeSide, riskPerTrade, err := s.calculateSLAndPositionSize(ctx, meta, entryPrice, group.EntryType)
		if err != nil {
			results = append(results, stockExecutionResult{StockID: stockRef.StockID, Symbol: stock.Symbol, Success: false, Error: err.Error()})
			anyFailed = true
			continue
		}

		if slPrice <= 0 {
			results = append(results, stockExecutionResult{StockID: stockRef.StockID, Symbol: stock.Symbol, Success: false, Error: "invalid SL"})
			anyFailed = true
			continue
		}
		if positionSize <= 0 {
			results = append(results, stockExecutionResult{StockID: stockRef.StockID, Symbol: stock.Symbol, Success: false, Error: "invalid position size"})
			anyFailed = true
			continue
		}
		slPercent, _ := strconv.ParseFloat(meta.SLPercent, 64)
		params := &domain.TradeParameters{
			StockID:            meta.StockID,
			StartingPrice:      entryPrice,
			StopLossPercentage: slPercent,
			RiskAmount:         float64(riskPerTrade),
			TradeSide:          tradeSide,
			PSType:             meta.PSType,
			EntryType:          group.EntryType,
			Active:             true,
		}
		// Create trade parameters
		err = s.createTradeParameters(ctx, params)
		if err != nil {
			results = append(results, stockExecutionResult{
				StockID: stockRef.StockID,
				Symbol:  stock.Symbol,
				Success: false,
				Error:   "trade param error: " + err.Error(),
			})
			anyFailed = true
			continue
		}
		// Create execution plan
		err = s.createExecutionPlan(ctx, meta.StockID, stockRef)
		if err != nil {
			results = append(results, stockExecutionResult{
				StockID: stockRef.StockID,
				Symbol:  stock.Symbol,
				Success: false,
				Error:   "exec plan error: " + err.Error(),
			})
			anyFailed = true
			continue
		}
		// Execute orders
		err = s.executeOrders(ctx, meta.StockID, stockRef)
		if err != nil {
			results = append(results, stockExecutionResult{
				StockID: stockRef.StockID,
				Symbol:  stock.Symbol,
				Success: false,
				Error:   "order exec error: " + err.Error(),
			})
			anyFailed = true
			continue
		}
		results = append(results, stockExecutionResult{
			StockID: stockRef.StockID,
			Symbol:  stock.Symbol,
			Success: true,
			Error:   "",
		})
	}
	// Log summary
	for _, res := range results {
		s.logStockExecutionResult(res, candle)
	}
	if anyFailed {
		return fmt.Errorf("one or more stocks failed in group execution; see logs for details")
	}
	return nil
}

// getAndSelectStock fetches a stock by ID, marks it as selected, and updates it
func (s *GroupExecutionService) getAndSelectStock(ctx context.Context, stockRef dto.StockGroupStockDTO) (*domain.Stock, error) {
	stock, err := s.StockGroupService.stockService.GetOnlyStockByID(ctx, stockRef.StockID)
	if err != nil || stock == nil {
		return nil, fmt.Errorf("stock not found or error")
	}
	stock.IsSelected = true
	err = s.StockGroupService.stockService.UpdateStock(ctx, stock)
	if err != nil {
		return nil, fmt.Errorf("stock update error: %w", err)
	}
	return stock, nil
}

// getInstrumentKeyAndMetadata retrieves the instrument key and trading metadata for a stock
func (s *GroupExecutionService) getInstrumentKeyAndMetadata(ctx context.Context, stock *domain.Stock, tradingMetadata map[string]StockBackTestAnalysis) (string, StockBackTestAnalysis, error) {
	univ, err := s.StockUniverseService.GetStockBySymbol(ctx, stock.Symbol)
	if err != nil || univ == nil {
		return "", StockBackTestAnalysis{}, fmt.Errorf("instrument key not found")
	}
	instrumentKey := univ.InstrumentKey
	meta, ok := tradingMetadata[instrumentKey]
	if !ok {
		return "", StockBackTestAnalysis{}, fmt.Errorf("no trading metadata found")
	}
	return instrumentKey, meta, nil
}

// calculateEntryPrice determines the entry price based on direction and candle
func (s *GroupExecutionService) calculateEntryPrice(ctx context.Context, meta StockBackTestAnalysis, candle domain.AggregatedCandle) (float64, error) {
	logger := ctxzap.Extract(ctx).Sugar()
	if meta.Direction == "LONG" {
		if candle.High <= 0 {
			return 0, fmt.Errorf("invalid entry price from candle")
		}
		logger.Info("Calculated entry price for stock %s: %f, time: %s", meta.StockID, candle.High, candle.Timestamp)
		return candle.High, nil
	} else {
		if candle.Low <= 0 {
			return 0, fmt.Errorf("invalid entry price from candle")
		}
		logger.Info("Calculated entry price for stock %s: %f, time: %s", meta.StockID, candle.Low, candle.Timestamp)
		return candle.Low, nil
	}
}

// calculateSLAndPositionSize computes stop loss price, points, position size, trade side, and risk per trade
func (s *GroupExecutionService) calculateSLAndPositionSize(ctx context.Context, meta StockBackTestAnalysis, entryPrice float64, groupEntryType string) (slPrice float64, slPoints float64, positionSize int, tradeSide domain.TradeSide, riskPerTrade int, err error) {
	slPercent, err := strconv.ParseFloat(meta.SLPercent, 64)
	slDecimal := slPercent / 100.0
	if err != nil {
		return 0, 0, 0, domain.TradeSide(0), 0, fmt.Errorf("invalid SL percent: %w", err)
	}
	if groupEntryType == "1ST_ENTRY" {
		riskPerTrade = s.Config.GetFirstEntryRiskPerTrade()
	} else {
		riskPerTrade = s.Config.GetSecondEntryRiskPerTrade()
	}
	if riskPerTrade <= 0 {
		riskPerTrade = 50
	}
	if meta.Direction == "LONG" {
		tradeSide = domain.Buy
		slPrice = entryPrice - (entryPrice * slDecimal)
		slPoints = entryPrice - slPrice
	} else {
		tradeSide = domain.Sell
		slPrice = entryPrice + (entryPrice * slDecimal)
		slPoints = slPrice - entryPrice
	}
	if slPoints <= 0 {
		return slPrice, slPoints, 0, tradeSide, riskPerTrade, fmt.Errorf("invalid SL")
	}
	positionSize = int(float64(riskPerTrade) / slPoints)
	if positionSize <= 0 {
		return slPrice, slPoints, 0, tradeSide, riskPerTrade, fmt.Errorf("invalid position size")
	}
	return slPrice, slPoints, positionSize, tradeSide, riskPerTrade, nil
}

// createTradeParameters creates or updates trade parameters
func (s *GroupExecutionService) createTradeParameters(ctx context.Context, params *domain.TradeParameters) error {
	return s.TradeParametersService.CreateOrUpdateTradeParameters(ctx, params)
}

// createExecutionPlan creates an execution plan for a stock
func (s *GroupExecutionService) createExecutionPlan(ctx context.Context, stockID string, stockRef dto.StockGroupStockDTO) error {
	_, err := s.ExecutionPlanService.CreateExecutionPlan(ctx, stockID)
	return err
}

// executeOrders executes orders for a stock
func (s *GroupExecutionService) executeOrders(ctx context.Context, stockID string, stockRef dto.StockGroupStockDTO) error {
	_, _, err := s.OrderExecutionService.ExecuteOrdersForStock(ctx, stockID)
	return err
}

// logStockExecutionResult logs the result of stock execution
func (s *GroupExecutionService) logStockExecutionResult(res struct {
	StockID string
	Symbol  string
	Success bool
	Error   string
}, candle domain.AggregatedCandle) {
	if res.Success {
		log.Info("GroupExec] Stock %s: SUCCESS (candle %v)", res.StockID, candle.Timestamp)
	} else {
		log.Error("GroupExec] Stock %s: FAIL (%s) (candle %v)", res.StockID, res.Error, candle.Timestamp)
	}
}

func (s *GroupExecutionService) validateForMorningEntry(ctx context.Context, stock *dto.StockGroupStockDTO, candle domain.AggregatedCandle) bool {
	logger := ctxzap.Extract(ctx).Sugar()
	candleHigh := candle.High
	candleLow := candle.Low
	startDate := candle.Timestamp.AddDate(0, 0, -1)
	endDate := candle.Timestamp

	atrValue, err := s.TechnicalIndicatorService.CalculateATR(
		ctx,
		stock.InstrumentKey,
		14,
		"day",
		startDate,
		endDate,
	)
	if err != nil {
		logger.Error("failed to calculate ATR: %w", err)
		return false
	}
	atr := atrValue[len(atrValue)-1].Value
	candleRange := candleHigh - candleLow
	mr := atr / candleRange
	if mr < 3 {
		logger.Info("Invalid morning range for stock %s: %f", stock.InstrumentKey, mr)
		return false
	}
	logger.Info("Valid morning range for stock %s: %f", stock.InstrumentKey, mr)
	return true
}

func (s *GroupExecutionService) validateForAfternoonEntry(ctx context.Context, stock *dto.StockGroupStockDTO, candle domain.AggregatedCandle) bool {
	return true
}
