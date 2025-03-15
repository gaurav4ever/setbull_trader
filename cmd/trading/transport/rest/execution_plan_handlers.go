package rest

import (
	"net/http"

	"github.com/gorilla/mux"
)

// GetAllExecutionPlans gets all execution plans
func (s *Server) GetAllExecutionPlans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	plans, err := s.planService.GetAllExecutionPlans(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get execution plans: "+err.Error())
		return
	}

	respondSuccess(w, plans)
}

// GetExecutionPlanByID gets an execution plan by ID
func (s *Server) GetExecutionPlanByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	plan, err := s.planService.GetExecutionPlanByID(ctx, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get execution plan: "+err.Error())
		return
	}

	if plan == nil {
		respondWithError(w, http.StatusNotFound, "Execution plan not found")
		return
	}

	respondSuccess(w, plan)
}

// GetExecutionPlanByStockID gets the latest execution plan for a stock
func (s *Server) GetExecutionPlanByStockID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	stockID := vars["stockId"]

	plan, err := s.planService.GetExecutionPlanByStockID(ctx, stockID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get execution plan: "+err.Error())
		return
	}

	if plan == nil {
		respondWithError(w, http.StatusNotFound, "No execution plan found for this stock")
		return
	}

	respondSuccess(w, plan)
}

// CreateExecutionPlan creates a new execution plan for a stock
func (s *Server) CreateExecutionPlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	stockID := vars["stockId"]

	plan, err := s.planService.CreateExecutionPlan(ctx, stockID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create execution plan: "+err.Error())
		return
	}

	respondCreated(w, plan)
}

// DeleteExecutionPlan deletes an execution plan
func (s *Server) DeleteExecutionPlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.planService.DeleteExecutionPlan(ctx, id); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete execution plan: "+err.Error())
		return
	}

	respondNoContent(w)
}
