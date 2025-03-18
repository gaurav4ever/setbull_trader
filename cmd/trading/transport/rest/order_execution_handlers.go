package rest

import (
	"net/http"
	"setbull_trader/internal/domain"

	"github.com/gorilla/mux"
)

// ExecuteOrdersForStock executes orders for a specific stock
func (s *Server) ExecuteOrdersForStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	stockID := vars["stockId"]

	execution, results, err := s.executeService.ExecuteOrdersForStock(ctx, stockID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to execute orders: "+err.Error())
		return
	}

	response := struct {
		Execution *domain.OrderExecution   `json:"execution"`
		Results   *domain.ExecutionResults `json:"results"`
	}{
		Execution: execution,
		Results:   results,
	}

	respondSuccess(w, response)
}

// ExecuteOrdersForAllSelectedStocks executes orders for all selected stocks
func (s *Server) ExecuteOrdersForAllSelectedStocks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	executions, results, err := s.executeService.ExecuteOrdersForAllSelectedStocks(ctx)
	if err != nil {
		respondWithError(w, http.StatusPreconditionFailed, "Failed to execute orders: "+err.Error())
		return
	}

	response := struct {
		Execution []*domain.OrderExecution   `json:"execution"`
		Results   []*domain.ExecutionResults `json:"results"`
	}{
		Execution: executions,
		Results:   results,
	}

	respondSuccess(w, response)
}

// GetOrderExecutionByID gets an order execution by ID
func (s *Server) GetOrderExecutionByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	execution, err := s.executeService.GetOrderExecutionByID(ctx, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get order execution: "+err.Error())
		return
	}

	if execution == nil {
		respondWithError(w, http.StatusNotFound, "Order execution not found")
		return
	}

	respondSuccess(w, execution)
}

// GetOrderExecutionsByPlanID gets all order executions for an execution plan
func (s *Server) GetOrderExecutionsByPlanID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	planID := vars["planId"]

	executions, err := s.executeService.GetOrderExecutionsByPlanID(ctx, planID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get order executions: "+err.Error())
		return
	}

	respondSuccess(w, executions)
}
