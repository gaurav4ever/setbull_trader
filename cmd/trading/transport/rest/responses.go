package rest

import (
	"encoding/json"
	"net/http"
)

// Response represents a standardized API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"error":"Failed to marshal response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, Response{
		Success: false,
		Error:   message,
	})
}

// respondSuccess sends a success response with data
func respondSuccess(w http.ResponseWriter, data interface{}) {
	respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// respondCreated sends a success response for resource creation
func respondCreated(w http.ResponseWriter, data interface{}) {
	respondWithJSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// respondNoContent sends a success response with no content
func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
