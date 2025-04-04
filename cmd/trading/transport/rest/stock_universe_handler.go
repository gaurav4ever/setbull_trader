package rest

import (
	"encoding/json"
	"net/http"
	"setbull_trader/internal/domain"
	"setbull_trader/pkg/log"
	"strconv"

	"github.com/gorilla/mux"
)

// GetStockBySymbol gets a stock from the universe by its symbol
func (s *Server) GetStockBySymbol(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	if symbol == "" {
		respondWithError(w, http.StatusBadRequest, "Symbol is required")
		return
	}

	stock, err := s.stockUniverseService.GetStockBySymbol(ctx, symbol)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Stock not found: "+err.Error())
		return
	}

	respondSuccess(w, stock)
}

// IngestStocks handles the request to ingest stocks from the Upstox JSON file
func (s *Server) IngestStocks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get file path from request if provided, otherwise use default
	var request struct {
		FilePath string `json:"filePath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		// If no body or invalid JSON, use default path
		request.FilePath = "" // Default path will be used by the service
	}

	// Call the service to ingest stocks
	created, updated, err := s.stockUniverseService.IngestStocksFromFile(ctx, request.FilePath)
	if err != nil {
		log.Error("Failed to ingest stocks: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to ingest stocks: "+err.Error())
		return
	}

	// Prepare the response
	response := struct {
		Created int `json:"created"`
		Updated int `json:"updated"`
		Total   int `json:"total"`
	}{
		Created: created,
		Updated: updated,
		Total:   created + updated,
	}

	respondSuccess(w, response)
}

// GetAllStocksFromUniverse gets all stocks from the universe with optional filtering
func (s *Server) GetAllStocksFromUniverse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	queryParams := r.URL.Query()

	// Check if only selected stocks should be returned
	onlySelected := queryParams.Get("selected") == "true"

	// Parse pagination parameters
	page := 1
	if pageStr := queryParams.Get("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	size := 50 // Default page size
	if sizeStr := queryParams.Get("size"); sizeStr != "" {
		if parsedSize, err := strconv.Atoi(sizeStr); err == nil && parsedSize > 0 {
			size = parsedSize
		}
	}

	// Get stocks from service
	stocks, total, err := s.stockUniverseService.GetAllStocks(ctx, onlySelected, page, size)
	if err != nil {
		log.Error("Failed to get stocks: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get stocks: "+err.Error())
		return
	}

	// Prepare the response
	response := struct {
		Stocks []domain.StockUniverse `json:"stocks"`
		Total  int64                  `json:"total"`
		Page   int                    `json:"page"`
		Size   int                    `json:"size"`
	}{
		Stocks: stocks,
		Total:  total,
		Page:   page,
		Size:   size,
	}

	// Add this before respondSuccess
	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Error("Failed to marshal response: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to marshal response: "+err.Error())
		return
	}
	log.Info("Response marshaled successfully, size: %d bytes", len(responseBytes))

	respondSuccess(w, response)
}

// ToggleStockSelectionInUniverse toggles the selection status of a stock in the universe
func (s *Server) ToggleStockSelectionInUniverse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	if symbol == "" {
		respondWithError(w, http.StatusBadRequest, "Symbol is required")
		return
	}

	// Parse the request body
	var request struct {
		IsSelected bool `json:"isSelected"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Toggle selection
	stock, err := s.stockUniverseService.ToggleStockSelection(ctx, symbol, request.IsSelected)
	if err != nil {
		log.Error("Failed to toggle selection for stock %s: %v", symbol, err)
		respondWithError(w, http.StatusInternalServerError, "Failed to toggle selection: "+err.Error())
		return
	}

	respondSuccess(w, stock)
}

// DeleteStockFromUniverse deletes a stock from the universe
func (s *Server) DeleteStockFromUniverse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	if symbol == "" {
		respondWithError(w, http.StatusBadRequest, "Symbol is required")
		return
	}

	// Delete stock
	err := s.stockUniverseService.DeleteStock(ctx, symbol)
	if err != nil {
		log.Error("Failed to delete stock %s: %v", symbol, err)
		respondWithError(w, http.StatusInternalServerError, "Failed to delete stock: "+err.Error())
		return
	}

	respondNoContent(w)
}
