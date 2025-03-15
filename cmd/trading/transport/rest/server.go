package rest

import (
	"net/http"
	"setbull_trader/internal/service"

	"github.com/gorilla/mux"
)

// Server represents the REST API server
type Server struct {
	router         *mux.Router
	stockService   *service.StockService
	paramsService  *service.TradeParametersService
	planService    *service.ExecutionPlanService
	executeService *service.OrderExecutionService
	utilityService *service.UtilityService
}

// NewServer creates a new REST API server
func NewServer(
	stockService *service.StockService,
	paramsService *service.TradeParametersService,
	planService *service.ExecutionPlanService,
	executeService *service.OrderExecutionService,
	utilityService *service.UtilityService,
) *Server {
	s := &Server{
		router:         mux.NewRouter(),
		stockService:   stockService,
		paramsService:  paramsService,
		planService:    planService,
		executeService: executeService,
		utilityService: utilityService,
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
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
