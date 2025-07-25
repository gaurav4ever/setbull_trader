package rest

import (
	"encoding/json"
	"net/http"
	"setbull_trader/internal/service"
	"setbull_trader/pkg/log"
	"strconv"

	"github.com/gorilla/mux"
)

// MasterDataHandler handles master data API endpoints
type MasterDataHandler struct {
	masterDataService service.MasterDataService
}

// NewMasterDataHandler creates a new master data handler
func NewMasterDataHandler(masterDataService service.MasterDataService) *MasterDataHandler {
	return &MasterDataHandler{
		masterDataService: masterDataService,
	}
}

// StartProcess handles starting a new master data ingestion process
func (h *MasterDataHandler) StartProcess(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req service.MasterDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("Failed to decode request body: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.NumberOfPastDays < 0 {
		respondWithError(w, http.StatusBadRequest, "numberOfPastDays must be non-negative")
		return
	}

	// Start the process
	response, err := h.masterDataService.StartProcess(r.Context(), req)
	if err != nil {
		log.Error("Failed to start master data process: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to start process: "+err.Error())
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    response,
	})
}

// GetProcessStatus handles retrieving the status of a process
func (h *MasterDataHandler) GetProcessStatus(w http.ResponseWriter, r *http.Request) {
	// Extract process ID from URL
	vars := mux.Vars(r)
	processIDStr := vars["processId"]

	processID, err := strconv.ParseInt(processIDStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid process ID")
		return
	}

	// Get process status
	status, err := h.masterDataService.GetProcessStatus(r.Context(), processID)
	if err != nil {
		log.Error("Failed to get process status for ID %d: %v", processID, err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get process status: "+err.Error())
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    status,
	})
}

// GetProcessHistory handles retrieving recent process history
func (h *MasterDataHandler) GetProcessHistory(w http.ResponseWriter, r *http.Request) {
	// Get limit from query parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default limit

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get process history
	history, err := h.masterDataService.GetProcessHistory(r.Context(), limit)
	if err != nil {
		log.Error("Failed to get process history: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get process history: "+err.Error())
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    history,
	})
}
