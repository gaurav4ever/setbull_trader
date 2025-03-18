package rest

import (
	"encoding/json"
	"net/http"

	"setbull_trader/internal/domain"

	"github.com/gorilla/mux"
)

// GetAllStocks gets a list of all stocks
func (s *Server) GetAllStocks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stocks, err := s.stockService.GetAllStocks(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get stocks: "+err.Error())
		return
	}

	respondSuccess(w, stocks)
}

// GetStockByID gets a stock by its ID
func (s *Server) GetStockByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	stock, err := s.stockService.GetStockByID(ctx, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get stock: "+err.Error())
		return
	}

	if stock == nil {
		respondWithError(w, http.StatusNotFound, "Stock not found")
		return
	}

	respondSuccess(w, stock)
}

// GetStockBySecurityID gets a stock by its security ID
func (s *Server) GetStockBySecurityID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	securityID := vars["securityId"]

	stock, err := s.stockService.GetStockBySecurityID(ctx, securityID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get stock: "+err.Error())
		return
	}

	if stock == nil {
		respondWithError(w, http.StatusNotFound, "Stock not found")
		return
	}

	respondSuccess(w, stock)
}

// GetSelectedStocks gets all selected stocks
func (s *Server) GetSelectedStocks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stocks, err := s.stockService.GetSelectedStocks(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get selected stocks: "+err.Error())
		return
	}

	respondSuccess(w, stocks)
}

// CreateStock creates a new stock
func (s *Server) CreateStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var stock domain.Stock
	if err := json.NewDecoder(r.Body).Decode(&stock); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	if err := s.stockService.CreateStock(ctx, &stock); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create stock: "+err.Error())
		return
	}

	respondCreated(w, stock)
}

// UpdateStock updates a stock
func (s *Server) UpdateStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	var stock domain.Stock
	if err := json.NewDecoder(r.Body).Decode(&stock); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Ensure the ID in the URL and body match
	stock.ID = id

	if err := s.stockService.UpdateStock(ctx, &stock); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update stock: "+err.Error())
		return
	}

	respondSuccess(w, stock)
}

// ToggleStockSelection toggles the selection status of a stock
func (s *Server) ToggleStockSelection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	// Parse the request body
	var request struct {
		IsSelected bool `json:"isSelected"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	if err := s.stockService.ToggleStockSelection(ctx, id, request.IsSelected); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to toggle stock selection: "+err.Error())
		return
	}

	respondSuccess(w, nil)
}

// DeleteStock deletes a stock
func (s *Server) DeleteStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.stockService.DeleteStock(ctx, id); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete stock: "+err.Error())
		return
	}

	respondNoContent(w)
}

// GetSelectedStocksEnriched handles GET /api/v1/stocks/selected/enriched
func (s *Server) GetSelectedStocksEnriched(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stocks, err := s.stockService.GetSelectedStocksEnriched(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get enriched selected stocks: "+err.Error())
		return
	}

	respondSuccess(w, stocks)
}
