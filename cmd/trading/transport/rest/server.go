package rest

import (
	"encoding/json"
	"net/http"
	"setbull_trader/internal/core/dto/request"
	"setbull_trader/internal/core/service/orders"
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
	router         *mux.Router
	orderService   *orders.Service
	stockService   *service.StockService
	paramsService  *service.TradeParametersService
	planService    *service.ExecutionPlanService
	executeService *service.OrderExecutionService
	utilityService *service.UtilityService
	validator      *validator.Validate
}

// NewServer creates a new REST API server
func NewServer(
	orderService *orders.Service,
	stockService *service.StockService,
	paramsService *service.TradeParametersService,
	planService *service.ExecutionPlanService,
	executeService *service.OrderExecutionService,
	utilityService *service.UtilityService,
) *Server {
	s := &Server{
		router:         mux.NewRouter(),
		orderService:   orderService,
		stockService:   stockService,
		paramsService:  paramsService,
		planService:    planService,
		executeService: executeService,
		utilityService: utilityService,
		validator:      validator.New(),
	}

	s.setupRoutes()
	return s
}

// setupRoutes sets up the routes for the API server
func (s *Server) setupRoutes() {
	// API v1 router
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Stock management routes
	api.HandleFunc("/stocks", s.GetAllStocks).Methods(http.MethodGet)
	api.HandleFunc("/stocks/{id}", s.GetStockByID).Methods(http.MethodGet)
	api.HandleFunc("/stocks/selected", s.GetSelectedStocks).Methods(http.MethodGet)
	api.HandleFunc("/stocks", s.CreateStock).Methods(http.MethodPost)
	api.HandleFunc("/stocks/{id}", s.UpdateStock).Methods(http.MethodPut)
	api.HandleFunc("/stocks/{id}/toggle-selection", s.ToggleStockSelection).Methods(http.MethodPatch)
	api.HandleFunc("/stocks/{id}", s.DeleteStock).Methods(http.MethodDelete)

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
