package rest

import (
	"encoding/json"
	"net/http"

	"setbull_trader/internal/domain"

	"github.com/gorilla/mux"
)

// GetTradeParametersByStockID gets trade parameters for a specific stock
func (s *Server) GetTradeParametersByStockID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	stockID := vars["stockId"]

	params, err := s.paramsService.GetTradeParametersByStockID(ctx, stockID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get trade parameters: "+err.Error())
		return
	}

	if params == nil {
		respondWithError(w, http.StatusNotFound, "Trade parameters not found for this stock")
		return
	}

	respondSuccess(w, params)
}

// CreateOrUpdateTradeParameters creates or updates trade parameters for a stock
func (s *Server) CreateOrUpdateTradeParameters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var params domain.TradeParameters
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	if params.StockID == "" {
		respondWithError(w, http.StatusBadRequest, "Stock ID is required")
		return
	}

	// Default risk amount to 30 if not provided
	if params.RiskAmount <= 0 {
		params.RiskAmount = 30.0
	}

	if err := s.paramsService.CreateOrUpdateTradeParameters(ctx, &params); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create/update trade parameters: "+err.Error())
		return
	}

	// Get the updated parameters to return
	updatedParams, err := s.paramsService.GetTradeParametersByStockID(ctx, params.StockID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve updated parameters: "+err.Error())
		return
	}

	respondSuccess(w, updatedParams)
}

// DeleteTradeParameters deletes trade parameters
func (s *Server) DeleteTradeParameters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.paramsService.DeleteTradeParameters(ctx, id); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete trade parameters: "+err.Error())
		return
	}

	respondNoContent(w)
}
