package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"setbull_trader/internal/service"
	"setbull_trader/pkg/log"

	"github.com/gorilla/mux"
)

// CandleHandler handles HTTP requests for candle data
type CandleHandler struct {
	candleAggService *service.CandleAggregationService
}

// NewCandleHandler creates a new candle handler
func NewCandleHandler(candleAggService *service.CandleAggregationService) *CandleHandler {
	return &CandleHandler{
		candleAggService: candleAggService,
	}
}

// RegisterRoutes registers the handler routes
func (h *CandleHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/candles/{instrument_key}/{timeframe}", h.GetCandles).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/candles/{instrument_key}/multi", h.GetMultiTimeframeCandles).Methods(http.MethodPost)
}

// GetMultiTimeframeRequest represents a request for multiple timeframe candle data
type GetMultiTimeframeRequest struct {
	Timeframes []string  `json:"timeframes" validate:"required,min=1"`
	Start      time.Time `json:"start"` // Optional start time
	End        time.Time `json:"end"`   // Optional end time
}

// CandleResponse represents the response for candle data
type CandleResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// GetCandles handles requests for candle data at a specific timeframe
func (h *CandleHandler) GetCandles(w http.ResponseWriter, r *http.Request) {
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
			respondWithError(w, http.StatusBadRequest, "Invalid start time format, use RFC3339")
			return
		}
	}

	endStr := r.URL.Query().Get("end")
	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid end time format, use RFC3339")
			return
		}
	}

	// Process the request based on timeframe
	var candles interface{}

	switch timeframe {
	case "5minute":
		candles, err = h.candleAggService.Get5MinCandles(r.Context(), instrumentKey, start, end)
	case "day":
		candles, err = h.candleAggService.GetDailyCandles(r.Context(), instrumentKey, start, end)
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

// GetMultiTimeframeCandles handles requests for multiple timeframe candle data
func (h *CandleHandler) GetMultiTimeframeCandles(w http.ResponseWriter, r *http.Request) {
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
	candles, err := h.candleAggService.GetMultiTimeframeCandles(
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
