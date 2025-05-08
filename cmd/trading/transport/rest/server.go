package rest

import (
	"encoding/json"
	"net/http"
	"setbull_trader/internal/core/adapters/client/upstox"
	"setbull_trader/internal/core/dto/request"
	"setbull_trader/internal/core/dto/response"
	"setbull_trader/internal/core/service/orders"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/service"
	"setbull_trader/pkg/apperrors"
	"setbull_trader/pkg/log"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

// Server represents the REST API server
type Server struct {
	router                  *mux.Router
	orderService            *orders.Service
	stockService            *service.StockService
	paramsService           *service.TradeParametersService
	planService             *service.ExecutionPlanService
	executeService          *service.OrderExecutionService
	utilityService          *service.UtilityService
	upstoxAuthService       *upstox.AuthService
	candleAggService        *service.CandleAggregationService
	batchFetchService       *service.BatchFetchService
	validator               *validator.Validate
	stockUniverseService    *service.StockUniverseService
	candleProcessingService *service.CandleProcessingService
	stockFilterPipeline     *service.StockFilterPipeline
	marketQuoteService      *service.MarketQuoteService
}

// NewServer creates a new REST API server
func NewServer(
	orderService *orders.Service,
	stockService *service.StockService,
	paramsService *service.TradeParametersService,
	planService *service.ExecutionPlanService,
	executeService *service.OrderExecutionService,
	utilityService *service.UtilityService,
	upstoxAuthService *upstox.AuthService,
	candleAggService *service.CandleAggregationService,
	batchFetchService *service.BatchFetchService,
	stockUniverseService *service.StockUniverseService,
	candleProcessingService *service.CandleProcessingService,
	stockFilterPipeline *service.StockFilterPipeline,
	marketQuoteService *service.MarketQuoteService,
) *Server {
	s := &Server{
		router:                  mux.NewRouter(),
		orderService:            orderService,
		stockService:            stockService,
		paramsService:           paramsService,
		planService:             planService,
		executeService:          executeService,
		utilityService:          utilityService,
		upstoxAuthService:       upstoxAuthService,
		candleAggService:        candleAggService,
		batchFetchService:       batchFetchService,
		stockUniverseService:    stockUniverseService,
		candleProcessingService: candleProcessingService,
		validator:               validator.New(),
		stockFilterPipeline:     stockFilterPipeline,
		marketQuoteService:      marketQuoteService,
	}

	s.setupRoutes()
	return s
}

// setupRoutes sets up the routes for the API server
func (s *Server) setupRoutes() {
	// API v1 router
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Stock universe routes (must come before generic stock routes to avoid conflicts)
	api.HandleFunc("/stocks/universe/ingest", s.IngestStocks).Methods(http.MethodPost)
	api.HandleFunc("/stocks/universe/daily-candles", s.FetchUniverseDailyCandles).Methods(http.MethodPost)
	api.HandleFunc("/stocks/universe/{symbol}/toggle-selection", s.ToggleStockSelectionInUniverse).Methods(http.MethodPatch)
	api.HandleFunc("/stocks/universe/{symbol}", s.GetStockBySymbol).Methods(http.MethodGet)
	api.HandleFunc("/stocks/universe/{symbol}", s.DeleteStockFromUniverse).Methods(http.MethodDelete)
	api.HandleFunc("/stocks/universe", s.GetAllStocksFromUniverse).Methods(http.MethodGet)

	// Stock selection routes (specific endpoints)
	api.HandleFunc("/stocks/selected/enriched", s.GetSelectedStocksEnriched).Methods(http.MethodGet)
	api.HandleFunc("/stocks/selected", s.GetSelectedStocks).Methods(http.MethodGet)

	// Stock by security ID route (specific endpoint)
	api.HandleFunc("/stocks/security/{securityId}", s.GetStockBySecurityID).Methods(http.MethodGet)

	// Generic stock routes (with ID parameter)
	api.HandleFunc("/stocks/{id}/toggle-selection", s.ToggleStockSelection).Methods(http.MethodPatch)
	api.HandleFunc("/stocks/{id}", s.GetStockByID).Methods(http.MethodGet)
	api.HandleFunc("/stocks/{id}", s.UpdateStock).Methods(http.MethodPut)
	api.HandleFunc("/stocks/{id}", s.DeleteStock).Methods(http.MethodDelete)

	// Root stock routes (must come last to avoid conflicts)
	api.HandleFunc("/stocks", s.CreateStock).Methods(http.MethodPost)
	api.HandleFunc("/stocks", s.GetAllStocks).Methods(http.MethodGet)

	// Trade parameters routes
	api.HandleFunc("/parameters/stock/{stockId}", s.GetTradeParametersByStockID).Methods(http.MethodGet)
	api.HandleFunc("/parameters", s.CreateOrUpdateTradeParameters).Methods(http.MethodPost)
	api.HandleFunc("/parameters/{id}", s.DeleteTradeParameters).Methods(http.MethodDelete)

	// Execution plan routes
	api.HandleFunc("/plans", s.GetAllExecutionPlans).Methods(http.MethodGet)
	api.HandleFunc("/plans/{id}", s.GetExecutionPlanByID).Methods(http.MethodGet)
	api.HandleFunc("/plans/stock/{stockId}", s.GetExecutionPlanByStockID).Methods(http.MethodGet)
	api.HandleFunc("/plans/stock/{stockId}", s.CreateExecutionPlan).Methods(http.MethodPost)
	api.HandleFunc("/plans/{id}", s.DeleteExecutionPlan).Methods(http.MethodDelete)

	// Order execution routes
	api.HandleFunc("/execute/stock/{stockId}", s.ExecuteOrdersForStock).Methods(http.MethodPost)
	api.HandleFunc("/execute/all", s.ExecuteOrdersForAllSelectedStocks).Methods(http.MethodPost)
	api.HandleFunc("/executions/{id}", s.GetOrderExecutionByID).Methods(http.MethodGet)
	api.HandleFunc("/executions/plan/{planId}", s.GetOrderExecutionsByPlanID).Methods(http.MethodGet)

	// Utility routes
	api.HandleFunc("/fibonacci/calculate", s.CalculateFibonacciLevels).Methods(http.MethodGet)
	api.HandleFunc("/health", s.HealthCheck).Methods(http.MethodGet)

	// Orders endpoints (from http.go)
	api.HandleFunc("/orders", s.PlaceOrder).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/orders/{orderID}", s.ModifyOrder).Methods(http.MethodPut, http.MethodOptions)
	api.HandleFunc("/orders/{orderID}", s.CancelOrder).Methods(http.MethodDelete, http.MethodOptions)

	// Trades endpoints (from http.go)
	api.HandleFunc("/trades", s.GetAllTrades).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/trades/history", s.GetTradeHistory).Methods(http.MethodGet, http.MethodOptions)

	// Upstox routes
	api.HandleFunc("/upstox/login", s.InitiateUpstoxLogin).Methods("GET")
	api.HandleFunc("/upstox/callback", s.HandleUpstoxCallback).Methods("GET")
	api.HandleFunc("/upstox/historical/{instrument}/{interval}/{to_date}/{from_date}", s.GetHistoricalCandleDataWithRange).Methods("GET")
	api.HandleFunc("/historical-data/batch-store", s.BatchStoreHistoricalData).Methods(http.MethodPost)

	// Candle routes
	api.HandleFunc("/candles/{instrument_key}/{timeframe}", s.GetCandles).Methods(http.MethodGet)
	api.HandleFunc("/candles/{instrument_key}/multi", s.GetMultiTimeframeCandles).Methods(http.MethodPost)

	// Filter pipeline routes
	api.HandleFunc("/filter-pipeline/run", s.RunFilterPipeline).Methods(http.MethodPost)
	api.HandleFunc("/filter-pipeline/fetch/top-10", s.GetTop10FilteredStocks).Methods(http.MethodGet)

	// Post market quotes
	api.HandleFunc("/market/quotes", s.PostMarketQuotes).Methods(http.MethodPost)
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request body for POST, PUT, PATCH
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			r.ParseForm()
			log.Info("Request: %s %s | Body: %v", r.Method, r.URL.Path, r.Form)
		}

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log response time
		duration := time.Since(start)
		log.Info("Request: %s %s | Completed in %v", r.Method, r.URL.Path, duration)
	})
}

// Health check handler
func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondSuccess(w, map[string]string{
		"status": "UP",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// PlaceOrder handles order placement
func (s *Server) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	var req request.PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format: "+err.Error())
		return
	}

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// Call service
	resp, err := s.orderService.PlaceOrder(&req)
	if err != nil {
		s.handleError(w, err)
		return
	}

	respondSuccess(w, resp)
}

// ModifyOrder handles order modification
func (s *Server) ModifyOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["orderID"]
	if orderID == "" {
		respondWithError(w, http.StatusBadRequest, "Order ID is required")
		return
	}

	var req request.ModifyOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format: "+err.Error())
		return
	}

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// Call service
	resp, err := s.orderService.ModifyOrder(orderID, &req)
	if err != nil {
		s.handleError(w, err)
		return
	}

	respondSuccess(w, resp)
}

// CancelOrder handles order cancellation
func (s *Server) CancelOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["orderID"]
	if orderID == "" {
		respondWithError(w, http.StatusBadRequest, "Order ID is required")
		return
	}

	// Call service
	resp, err := s.orderService.CancelOrder(orderID)
	if err != nil {
		s.handleError(w, err)
		return
	}

	respondSuccess(w, resp)
}

// GetAllTrades handles getting all trades
func (s *Server) GetAllTrades(w http.ResponseWriter, r *http.Request) {
	// Call service
	resp, err := s.orderService.GetAllTrades()
	if err != nil {
		s.handleError(w, err)
		return
	}

	respondSuccess(w, resp)
}

// GetTradeHistory handles getting trade history
func (s *Server) GetTradeHistory(w http.ResponseWriter, r *http.Request) {
	var req request.TradeHistoryRequest

	// Parse query parameters
	queryParams := r.URL.Query()

	// Set default values
	req.FromDate = queryParams.Get("fromDate")
	if req.FromDate == "" {
		req.FromDate = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}

	req.ToDate = queryParams.Get("toDate")
	if req.ToDate == "" {
		req.ToDate = time.Now().Format("2006-01-02")
	}

	// Parse page number with default 0
	pageNumber := 0
	if pageStr := queryParams.Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			pageNumber = page
		}
	}
	req.PageNumber = pageNumber

	// Log the extracted parameters
	log.Info("Processing trade history request | FromDate: %s, ToDate: %s, Page: %d",
		req.FromDate, req.ToDate, req.PageNumber)

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		log.Error("Trade history validation error: %v", err)
		respondWithError(w, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// Call service
	resp, err := s.orderService.GetTradeHistory(&req)
	if err != nil {
		s.handleError(w, err)
		return
	}

	// Log response summary
	log.Info("Trade history response | Count: %d", resp.Count)
	respondSuccess(w, resp)
}

// handleError handles common error responses
func (s *Server) handleError(w http.ResponseWriter, err error) {
	log.Error("API error: %v", err)

	// Check if it's an AppError
	if appErr, ok := err.(*apperrors.AppError); ok {
		respondWithError(w, appErr.Code, appErr.Message)
		return
	}

	// Default to internal server error
	respondWithError(w, http.StatusInternalServerError, "Internal server error")
}

// InitiateUpstoxLogin handles the login initiation for Upstox
func (s *Server) InitiateUpstoxLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	loginURL, sessionID, err := s.upstoxAuthService.InitiateLogin(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to initiate login: "+err.Error())
		return
	}

	response := struct {
		LoginURL  string `json:"login_url"`
		SessionID string `json:"session_id"`
	}{
		LoginURL:  loginURL,
		SessionID: sessionID,
	}

	respondSuccess(w, response)
}

// HandleUpstoxCallback processes the callback from Upstox after authorization
func (s *Server) HandleUpstoxCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract code and state from query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		respondWithError(w, http.StatusBadRequest, "Authorization code is missing")
		return
	}

	if state == "" {
		respondWithError(w, http.StatusBadRequest, "State parameter is missing")
		return
	}

	// Validate the callback and exchange code for token
	token, err := s.upstoxAuthService.HandleCallback(ctx, code, state)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process callback: "+err.Error())
		return
	}

	respondSuccess(w, token)
}

// GetHistoricalCandleDataWithRange gets historical candle data for a specific date range
func (s *Server) GetHistoricalCandleDataWithRange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	// Extract path parameters
	instrument := vars["instrument"]
	interval := vars["interval"]
	toDate := vars["to_date"]
	fromDate := vars["from_date"]

	// Validate required parameters
	if instrument == "" {
		respondWithError(w, http.StatusBadRequest, "Instrument is required")
		return
	}
	if interval == "" {
		respondWithError(w, http.StatusBadRequest, "Interval is required")
		return
	}
	if toDate == "" {
		respondWithError(w, http.StatusBadRequest, "To date is required")
		return
	}
	if fromDate == "" {
		respondWithError(w, http.StatusBadRequest, "From date is required")
		return
	}

	// Get user ID from context or query parameter
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondWithError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	// Call the upstox service
	candleData, err := s.upstoxAuthService.GetHistoricalCandleDataWithDateRange(ctx, "upstox_session", instrument, interval, toDate, fromDate)
	if err != nil {
		log.Error("Failed to get historical candle data", "error", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch historical data: "+err.Error())
		return
	}

	respondSuccess(w, candleData)
}

func (s *Server) BatchStoreHistoricalData(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var request domain.BatchStoreHistoricalDataRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Error("Failed to decode request body: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request format: "+err.Error())
		return
	}

	if err := s.validator.Struct(request); err != nil {
		log.Error("Invalid request: %v", err)
		respondWithError(w, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	result, err := s.batchFetchService.ProcessBatchRequest(r.Context(), &request)
	if err != nil {
		log.Error("Failed to process batch request: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to process batch request: "+err.Error())
		return
	}

	response := domain.BatchStoreHistoricalDataResponse{
		Status: "success",
		Data:   *result,
	}

	log.Info("Batch request processed in %v. Processed: %d, Success: %d, Failed: %d",
		time.Since(startTime), result.ProcessedItems, result.SuccessfulItems, result.FailedItems)

	respondSuccess(w, response)
}

func (s *Server) GetCandles(w http.ResponseWriter, r *http.Request) {
	// Extract path parameters
	vars := mux.Vars(r)
	instrumentKey := vars["instrument_key"]
	timeframe := vars["timeframe"]

	if instrumentKey == "" {
		respondWithError(w, http.StatusBadRequest, "Instrument key is required")
		return
	}

	if timeframe == "" {
		respondWithError(w, http.StatusBadRequest, "Timeframe is required")
		return
	}

	// Parse query parameters for start and end times
	var start, end time.Time
	var err error

	startStr := r.URL.Query().Get("start")
	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			// Try parsing with the local timezone offset
			start, err = time.Parse("2006-01-02T15:04:05+05:30", startStr)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid start time format, use RFC3339")
				return
			}
		}
	}

	endStr := r.URL.Query().Get("end")
	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			// Try parsing with the local timezone offset
			end, err = time.Parse("2006-01-02T15:04:05+05:30", endStr)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid end time format, use RFC3339")
				return
			}
		}
	}

	// Process the request based on timeframe
	var candles interface{}

	switch timeframe {
	case "5minute":
		candles, err = s.candleAggService.Get5MinCandles(r.Context(), instrumentKey, start, end)
	case "day":
		candles, err = s.candleAggService.GetDailyCandles(r.Context(), instrumentKey, start, end)
	default:
		respondWithError(w, http.StatusBadRequest, "Unsupported timeframe")
		return
	}

	if err != nil {
		log.Error("Failed to get candles: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve candle data: "+err.Error())
		return
	}

	// Prepare response
	response := CandleResponse{
		Status: "success",
		Data:   candles,
	}

	// Write response
	respondSuccess(w, response)
}

func (s *Server) GetMultiTimeframeCandles(w http.ResponseWriter, r *http.Request) {
	// Extract instrument key from path
	vars := mux.Vars(r)
	instrumentKey := vars["instrument_key"]

	if instrumentKey == "" {
		respondWithError(w, http.StatusBadRequest, "Instrument key is required")
		return
	}

	// Parse request body
	var request GetMultiTimeframeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format: "+err.Error())
		return
	}

	// Validate request
	if len(request.Timeframes) == 0 {
		respondWithError(w, http.StatusBadRequest, "At least one timeframe is required")
		return
	}

	// Get multi-timeframe candles
	candles, err := s.candleAggService.GetMultiTimeframeCandles(
		r.Context(),
		instrumentKey,
		request.Timeframes,
		request.Start,
		request.End,
	)
	if err != nil {
		log.Error("Failed to get multi-timeframe candles: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve multi-timeframe candle data: "+err.Error())
		return
	}

	// Prepare response
	response := CandleResponse{
		Status: "success",
		Data:   candles,
	}

	// Write response
	respondSuccess(w, response)
}

// RunFilterPipeline handles running the filter pipeline
func (s *Server) RunFilterPipeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body for optional parameters
	var request struct {
		InstrumentKeys []string `json:"instrumentKeys"` // Optional list of instrument keys to process
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		// If there's an error parsing, just use default values
		request.InstrumentKeys = []string{}
	}

	bullish, bearish, metrics, err := s.stockFilterPipeline.RunPipeline(ctx, request.InstrumentKeys)
	if err != nil {
		log.Error("Failed to run filter pipeline: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to run filter pipeline: "+err.Error())
		return
	}

	response := response.FilterPipelineResponse{
		Status: "success",
		Data: response.FilterPipelineData{
			BullishStocks: bullish,
			BearishStocks: bearish,
			Metrics: response.PipelineMetrics{
				TotalStocks:     metrics.TotalStocks,
				BasicFilterPass: metrics.BasicFilterPass,
				EMAFilterPass:   metrics.EMAFilterPass,
				RSIFilterPass:   metrics.RSIFilterPass,
				BullishStocks:   metrics.BullishStocks,
				BearishStocks:   metrics.BearishStocks,
				ProcessingTime:  metrics.ProcessingTime,
			},
		},
	}

	respondSuccess(w, response)
}

// GetTop10FilteredStocks handles getting the top 10 filtered stocks
func (s *Server) GetTop10FilteredStocks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	top10Stocks, err := s.stockFilterPipeline.GetTop10FilteredStocks(ctx)
	if err != nil {
		log.Error("Failed to get top 10 filtered stocks: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get top 10 filtered stocks: "+err.Error())
		return
	}

	respondSuccess(w, top10Stocks)
}

// PostMarketQuotes handles POST /market/quotes
//
// Request body can include either instrumentKeys (for instrument_key lookup) or symbols (for symbol lookup).
// keyType can be 'instrument_key' or 'symbol'. If not provided, defaults to 'instrument_key'.
// Example:
//
//	{
//	  "instrumentKeys": ["NSE_EQ:RELIANCE"],
//	  "interval": "1min",
//	  "keyType": "instrument_key"
//	}
//
// or
//
//	{
//	  "symbols": ["RELIANCE"],
//	  "interval": "1min",
//	  "keyType": "symbol"
//	}
func (s *Server) PostMarketQuotes(w http.ResponseWriter, r *http.Request) {
	var req request.MarketQuotesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format: "+err.Error())
		return
	}
	if err := s.validator.Struct(req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}
	// Extract userID from header or context (assume header for now)
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		respondWithError(w, http.StatusBadRequest, "User ID is required in X-User-Id header")
		return
	}

	var keys []string
	keyType := req.KeyType
	if keyType == "symbol" {
		keys = req.Symbols
	} else {
		keyType = "instrument_key" // default
		keys = req.InstrumentKeys
	}

	resp := s.marketQuoteService.GetQuotes(r.Context(), userID, keys, req.Interval, keyType, s.stockUniverseService)
	respondSuccess(w, resp)
}
