package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/service"
	"setbull_trader/pkg/log"
	"strconv"
	"sync"
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
// or for specific stocks if instrumentKeys are provided
func (s *Server) FetchUniverseDailyCandles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// Parse request body for optional parameters
	var request struct {
		Days           int      `json:"days"`           // Number of days to fetch, default 100
		Parallel       bool     `json:"parallel"`       // Whether to process stocks in parallel, default false
		InstrumentKeys []string `json:"instrumentKeys"` // Optional list of instrument keys to process
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		// If there's an error parsing, just use default values
		request.Days = 100
		request.Parallel = false
	}

	// If days not specified or invalid, use default
	if request.Days <= 0 {
		request.Days = 100
	}

	var stocks []domain.StockUniverse
	var err error

	// Check if specific instrument keys were provided
	if len(request.InstrumentKeys) > 0 {
		log.Info("Starting to fetch daily candles for %d specific stocks in universe (last %d days)",
			len(request.InstrumentKeys), request.Days)

		// Get only the specified stocks from the universe
		stocks, err = s.stockUniverseService.GetStocksByInstrumentKeys(ctx, request.InstrumentKeys)
		if err != nil {
			log.Error("Failed to get specific stocks from universe: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to get specific stocks: "+err.Error())
			return
		}
	} else {
		log.Info("Starting to fetch daily candles for all stocks in universe (last %d days)", request.Days)

		// Get all stocks from universe
		stocks, _, err = s.stockUniverseService.GetAllStocks(ctx, false, 1, 10000)
		if err != nil {
			log.Error("Failed to get stocks from universe: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to get stocks: "+err.Error())
			return
		}
	}

	if len(stocks) == 0 {
		respondWithError(w, http.StatusNotFound, "No stocks found in universe")
		return
	}

	// Calculate date range
	endDate := time.Now()

	log.Info("Fetching daily candles for %d stocks with backfill support for the last %d days",
		len(stocks), request.Days)

	// Initialize result tracking
	result := &DailyCandles{
		TotalStocks:      len(stocks),
		ProcessedStocks:  0,
		SkippedStocks:    0,
		SuccessfulStocks: 0,
		FailedStocks:     0,
		StockResults:     make([]StockProcessResult, 0, len(stocks)),
		StartTime:        startTime,
		EndTime:          time.Time{},
		Duration:         "",
	}

	// Process stocks based on parallel flag
	if request.Parallel {
		result = s.processDailyCandlesParallel(ctx, stocks, endDate, request.Days)
	} else {
		result = s.processDailyCandlesSequential(ctx, stocks, endDate, request.Days)
	}

	// Set end time and duration
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime).String()

	log.Info("Completed fetching daily candles in %v. Total: %d, Processed: %d, Skipped: %d, Success: %d, Failed: %d",
		result.Duration, result.TotalStocks, result.ProcessedStocks,
		result.SkippedStocks, result.SuccessfulStocks, result.FailedStocks)

	respondSuccess(w, result)
}

// DailyCandles represents the result of fetching daily candles
type DailyCandles struct {
	TotalStocks      int                  `json:"total_stocks"`
	ProcessedStocks  int                  `json:"processed_stocks"`
	SkippedStocks    int                  `json:"skipped_stocks"`
	SuccessfulStocks int                  `json:"successful_stocks"`
	FailedStocks     int                  `json:"failed_stocks"`
	StockResults     []StockProcessResult `json:"stock_results,omitempty"`
	StartTime        time.Time            `json:"start_time"`
	EndTime          time.Time            `json:"end_time"`
	Duration         string               `json:"duration"`
}

// StockProcessResult represents the result of processing a single stock
type StockProcessResult struct {
	service.ProcessResult
	Duration string `json:"duration"`
}

// SegmentDetail represents details about a processed segment
type SegmentDetail struct {
	Type      string `json:"type"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// processDailyCandlesSequential processes stocks sequentially
func (s *Server) processDailyCandlesSequential(
	ctx context.Context,
	stocks []domain.StockUniverse,
	endDate time.Time,
	maxDays int,
) *DailyCandles {
	result := &DailyCandles{
		TotalStocks:  len(stocks),
		StockResults: make([]StockProcessResult, 0, len(stocks)),
		StartTime:    time.Now(),
	}

	// Process each stock sequentially
	for _, stock := range stocks {
		stockStartTime := time.Now()

		// Skip stocks without instrument key
		if stock.InstrumentKey == "" {
			log.Warn("Stock %s has no instrument key, skipping", stock.Symbol)

			stockResult := StockProcessResult{
				ProcessResult: service.ProcessResult{
					Symbol:        stock.Symbol,
					InstrumentKey: "",
					Status:        "failed",
					Error:         "no instrument key",
				},
				Duration: time.Since(stockStartTime).String(),
			}

			result.StockResults = append(result.StockResults, stockResult)
			result.ProcessedStocks++
			result.FailedStocks++
			continue
		}

		// Process the stock
		processResult, err := s.candleAggService.ProcessStockDailyCandles(ctx, stock, endDate, maxDays)
		if err != nil {
			log.Error("Failed to process daily candles for stock %s: %v", stock.Symbol, err)
		}

		// Convert service result to handler result
		stockResult := StockProcessResult{
			ProcessResult: processResult,
			Duration:      time.Since(stockStartTime).String(),
		}

		result.StockResults = append(result.StockResults, stockResult)
		result.ProcessedStocks++

		// Update counters based on status
		switch processResult.Status {
		case "success":
			result.SuccessfulStocks++
		case "skipped":
			result.SkippedStocks++
		case "failed":
			result.FailedStocks++
		}
	}

	return result
}

// processDailyCandlesParallel processes stocks in parallel
func (s *Server) processDailyCandlesParallel(
	ctx context.Context,
	stocks []domain.StockUniverse,
	endDate time.Time,
	maxDays int,
) *DailyCandles {
	result := &DailyCandles{
		TotalStocks:  len(stocks),
		StockResults: make([]StockProcessResult, 0, len(stocks)),
		StartTime:    time.Now(),
	}

	// Use a mutex to protect concurrent access to the result
	var resultMutex sync.Mutex

	// Use a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Use a semaphore to limit concurrency
	maxConcurrency := 5 // Adjust based on your system capabilities and API rate limits
	semaphore := make(chan struct{}, maxConcurrency)

	// Process each stock in parallel
	for _, stock := range stocks {
		wg.Add(1)

		go func(stock domain.StockUniverse) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			stockStartTime := time.Now()

			// Skip stocks without instrument key
			if stock.InstrumentKey == "" {
				log.Warn("Stock %s has no instrument key, skipping", stock.Symbol)

				stockResult := StockProcessResult{
					ProcessResult: service.ProcessResult{
						Symbol:        stock.Symbol,
						InstrumentKey: "",
						Status:        "failed",
						Error:         "no instrument key",
					},
					Duration: time.Since(stockStartTime).String(),
				}

				// Update result with mutex protection
				resultMutex.Lock()
				result.StockResults = append(result.StockResults, stockResult)
				result.ProcessedStocks++
				result.FailedStocks++
				resultMutex.Unlock()

				return
			}

			// Process the stock using the server's candleAggService
			processResult, _ := s.candleAggService.ProcessStockDailyCandles(ctx, stock, endDate, maxDays)

			// Convert service result to handler result
			stockResult := StockProcessResult{
				ProcessResult: processResult,
				Duration:      time.Since(stockStartTime).String(),
			}

			// Update result with mutex protection
			resultMutex.Lock()
			result.StockResults = append(result.StockResults, stockResult)
			result.ProcessedStocks++

			// Update counters based on status
			switch processResult.Status {
			case "success":
				result.SuccessfulStocks++
			case "skipped":
				result.SkippedStocks++
			case "failed":
				result.FailedStocks++
			}
			resultMutex.Unlock()

		}(stock)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	return result
}
