package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"setbull_trader/internal/service"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMasterDataService is a mock implementation of MasterDataService
type MockMasterDataService struct {
	mock.Mock
}

func (m *MockMasterDataService) StartProcess(ctx context.Context, req service.MasterDataRequest) (*service.MasterDataResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.MasterDataResponse), args.Error(1)
}

func (m *MockMasterDataService) GetProcessStatus(ctx context.Context, processID int) (*service.ProcessStatusResponse, error) {
	args := m.Called(ctx, processID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ProcessStatusResponse), args.Error(1)
}

func (m *MockMasterDataService) GetProcessHistory(ctx context.Context, limit int) ([]service.ProcessStatusResponse, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]service.ProcessStatusResponse), args.Error(1)
}

func TestMasterDataHandler_StartProcess(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    service.MasterDataRequest
		mockResponse   *service.MasterDataResponse
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful process start",
			requestBody: service.MasterDataRequest{
				NumberOfPastDays: 0,
			},
			mockResponse: &service.MasterDataResponse{
				ProcessID:   123,
				Status:      "RUNNING",
				Message:     "Process started successfully",
				ProcessDate: "2025-01-22",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"processId":   float64(123),
					"status":      "RUNNING",
					"message":     "Process started successfully",
					"processDate": "2025-01-22",
				},
			},
		},
		{
			name: "invalid request - negative days",
			requestBody: service.MasterDataRequest{
				NumberOfPastDays: -1,
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "numberOfPastDays must be non-negative",
			},
		},
		{
			name: "service error",
			requestBody: service.MasterDataRequest{
				NumberOfPastDays: 0,
			},
			mockResponse:   nil,
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to start process: " + assert.AnError.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMasterDataService)
			if tt.mockError == nil && tt.expectedStatus == http.StatusOK {
				mockService.On("StartProcess", mock.Anything, tt.requestBody).Return(tt.mockResponse, nil)
			} else if tt.mockError != nil {
				mockService.On("StartProcess", mock.Anything, tt.requestBody).Return(nil, tt.mockError)
			}

			// Create handler
			handler := NewMasterDataHandler(mockService)

			// Create request
			requestBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/master-data/process/start", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler.StartProcess(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response body
			var responseBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &responseBody)
			assert.NoError(t, err)

			// Assert response body
			assert.Equal(t, tt.expectedBody, responseBody)

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}

func TestMasterDataHandler_GetProcessStatus(t *testing.T) {
	tests := []struct {
		name           string
		processID      string
		mockResponse   *service.ProcessStatusResponse
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:      "successful status retrieval",
			processID: "123",
			mockResponse: &service.ProcessStatusResponse{
				ProcessID:   123,
				Status:      "RUNNING",
				ProcessDate: "2025-01-22",
				CreatedAt:   "2025-01-22T10:00:00Z",
				Steps: []service.ProcessStepStatus{
					{
						StepNumber: 1,
						StepName:   "daily_ingestion",
						Status:     "COMPLETED",
					},
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"processId":   float64(123),
					"status":      "RUNNING",
					"processDate": "2025-01-22",
					"createdAt":   "2025-01-22T10:00:00Z",
					"steps": []interface{}{
						map[string]interface{}{
							"stepNumber": float64(1),
							"stepName":   "daily_ingestion",
							"status":     "COMPLETED",
						},
					},
				},
			},
		},
		{
			name:           "invalid process ID",
			processID:      "invalid",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Invalid process ID",
			},
		},
		{
			name:           "service error",
			processID:      "123",
			mockResponse:   nil,
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to get process status: " + assert.AnError.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMasterDataService)
			if tt.mockError == nil && tt.expectedStatus == http.StatusOK {
				processID, _ := strconv.Atoi(tt.processID)
				mockService.On("GetProcessStatus", mock.Anything, processID).Return(tt.mockResponse, nil)
			} else if tt.mockError != nil {
				processID, _ := strconv.Atoi(tt.processID)
				mockService.On("GetProcessStatus", mock.Anything, processID).Return(nil, tt.mockError)
			}

			// Create handler
			handler := NewMasterDataHandler(mockService)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/master-data/process/"+tt.processID+"/status", nil)

			// Set up router for URL parameters
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/master-data/process/{processId}/status", handler.GetProcessStatus).Methods(http.MethodGet)
			req = mux.SetURLVars(req, map[string]string{"processId": tt.processID})

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler.GetProcessStatus(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response body
			var responseBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &responseBody)
			assert.NoError(t, err)

			// Assert response body
			assert.Equal(t, tt.expectedBody, responseBody)

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}

func TestMasterDataHandler_GetProcessHistory(t *testing.T) {
	tests := []struct {
		name           string
		limit          string
		mockResponse   []service.ProcessStatusResponse
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:  "successful history retrieval",
			limit: "5",
			mockResponse: []service.ProcessStatusResponse{
				{
					ProcessID:   123,
					Status:      "COMPLETED",
					ProcessDate: "2025-01-22",
				},
				{
					ProcessID:   124,
					Status:      "FAILED",
					ProcessDate: "2025-01-21",
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"data": []interface{}{
					map[string]interface{}{
						"processId":   float64(123),
						"status":      "COMPLETED",
						"processDate": "2025-01-22",
						"steps":       []interface{}{},
					},
					map[string]interface{}{
						"processId":   float64(124),
						"status":      "FAILED",
						"processDate": "2025-01-21",
						"steps":       []interface{}{},
					},
				},
			},
		},
		{
			name:           "service error",
			limit:          "10",
			mockResponse:   nil,
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to get process history: " + assert.AnError.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMasterDataService)
			if tt.mockError == nil && tt.expectedStatus == http.StatusOK {
				limit := 10 // default
				if tt.limit != "" {
					limit, _ = strconv.Atoi(tt.limit)
				}
				mockService.On("GetProcessHistory", mock.Anything, limit).Return(tt.mockResponse, nil)
			} else if tt.mockError != nil {
				limit := 10 // default
				if tt.limit != "" {
					limit, _ = strconv.Atoi(tt.limit)
				}
				mockService.On("GetProcessHistory", mock.Anything, limit).Return(nil, tt.mockError)
			}

			// Create handler
			handler := NewMasterDataHandler(mockService)

			// Create request
			url := "/api/v1/master-data/process/history"
			if tt.limit != "" {
				url += "?limit=" + tt.limit
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler.GetProcessHistory(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response body
			var responseBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &responseBody)
			assert.NoError(t, err)

			// Assert response body
			assert.Equal(t, tt.expectedBody, responseBody)

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}
