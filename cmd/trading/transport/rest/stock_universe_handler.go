package rest

import (
	"encoding/json"
	"net/http"
	"setbull_trader/internal/domain"
	"setbull_trader/pkg/log"
	"strconv"
	"time"

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

// FetchUniverseDailyCandles fetches and stores daily candle data for all stocks in the universe
func (s *Server) FetchUniverseDailyCandles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// Parse request body for optional parameters
	var request struct {
		Days int `json:"days"` // Number of days to fetch, default 100
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		// If there's an error parsing, just use default values
		request.Days = 100
	}

	// If days not specified or invalid, use default
	if request.Days <= 0 {
		request.Days = 100
	}

	log.Info("Starting to fetch daily candles for all stocks in universe (last %d days)", request.Days)

	// Get all stocks from universe
	stocks, _, err := s.stockUniverseService.GetAllStocks(ctx, false, 1, 10000)
	if err != nil {
		log.Error("Failed to get stocks from universe: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get stocks: "+err.Error())
		return
	}

	if len(stocks) == 0 {
		respondWithError(w, http.StatusNotFound, "No stocks found in universe")
		return
	}

	// Calculate date range
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -request.Days)

	log.Info("Fetching daily candles from %s to %s for %d stocks",
		startDate.Format(time.RFC3339), endDate.Format(time.RFC3339), len(stocks))

	// Initialize result tracking
	result := struct {
		TotalStocks      int      `json:"total_stocks"`
		ProcessedStocks  int      `json:"processed_stocks"`
		SkippedStocks    int      `json:"skipped_stocks"`
		SuccessfulStocks int      `json:"successful_stocks"`
		FailedStocks     int      `json:"failed_stocks"`
		FailedSymbols    []string `json:"failed_symbols,omitempty"`
	}{
		TotalStocks:   len(stocks),
		FailedSymbols: make([]string, 0),
	}

	// Get list of stocks that already have data for this date range
	existingStocks, err := s.candleAggService.GetStocksWithExistingDailyCandles(ctx, startDate, endDate)
	if err != nil {
		log.Error("Failed to check for existing candle data: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to check existing data: "+err.Error())
		return
	}

	log.Info("Found %d stocks with existing candle data in the date range", len(existingStocks))

	// Create a map for quick lookup
	existingStocksMap := make(map[string]bool)
	for _, stock := range existingStocks {
		existingStocksMap[stock] = true
	}

	// Create a batch request
	batchRequest := &domain.BatchStoreHistoricalDataRequest{
		InstrumentKeys: []string{},
		Interval:       "day",
		FromDate:       startDate.Format("2006-01-02"),
		ToDate:         endDate.Format("2006-01-02"),
	}

	// Add only stocks that don't have data
	for _, stock := range stocks {
		if stock.InstrumentKey == "" {
			continue
		}

		// Skip if this stock already has data for the date range
		if existingStocksMap[stock.InstrumentKey] {
			log.Info("Skipping %s (%s) - already has data for the date range",
				stock.Symbol, stock.InstrumentKey)
			result.SkippedStocks++
			continue
		}

		batchRequest.InstrumentKeys = append(batchRequest.InstrumentKeys, stock.InstrumentKey)
	}

	// If no stocks need data, return early
	if len(batchRequest.InstrumentKeys) == 0 {
		log.Info("No stocks need data - all %d stocks already have data for the date range", result.SkippedStocks)
		result.ProcessedStocks = result.TotalStocks
		respondSuccess(w, result)
		return
	}

	log.Info("Fetching data for %d stocks (skipped %d stocks with existing data)",
		len(batchRequest.InstrumentKeys), result.SkippedStocks)

	// Process the batch request
	batchResult, err := s.batchFetchService.ProcessBatchRequest(ctx, batchRequest)
	if err != nil {
		log.Error("Failed to process batch request: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to process batch request: "+err.Error())
		return
	}

	// Update result with batch processing results
	result.ProcessedStocks = result.SkippedStocks + batchResult.ProcessedItems
	result.SuccessfulStocks = result.SkippedStocks + batchResult.SuccessfulItems
	result.FailedStocks = batchResult.FailedItems
	// We could also add failed symbols from the batch result if available

	log.Info("Completed fetching daily candles in %v. Total: %d, Skipped: %d, Processed: %d, Success: %d, Failed: %d",
		time.Since(startTime), result.TotalStocks, result.SkippedStocks,
		batchResult.ProcessedItems, batchResult.SuccessfulItems, batchResult.FailedItems)

	respondSuccess(w, result)
}
