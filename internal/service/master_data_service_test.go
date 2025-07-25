package service

import (
	"context"
	"setbull_trader/internal/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing
type MockMasterDataProcessRepository struct {
	mock.Mock
}

func (m *MockMasterDataProcessRepository) Create(ctx context.Context, processDate time.Time, numberOfPastDays int) (*domain.MasterDataProcess, error) {
	args := m.Called(ctx, processDate, numberOfPastDays)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MasterDataProcess), args.Error(1)
}

func (m *MockMasterDataProcessRepository) GetByDate(ctx context.Context, processDate time.Time) (*domain.MasterDataProcess, error) {
	args := m.Called(ctx, processDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MasterDataProcess), args.Error(1)
}

func (m *MockMasterDataProcessRepository) GetByID(ctx context.Context, processID int) (*domain.MasterDataProcess, error) {
	args := m.Called(ctx, processID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MasterDataProcess), args.Error(1)
}

func (m *MockMasterDataProcessRepository) UpdateStatus(ctx context.Context, processID int, status string) error {
	args := m.Called(ctx, processID, status)
	return args.Error(0)
}

func (m *MockMasterDataProcessRepository) CompleteProcess(ctx context.Context, processID int) error {
	args := m.Called(ctx, processID)
	return args.Error(0)
}

func (m *MockMasterDataProcessRepository) CreateStep(ctx context.Context, processID int, stepNumber int, stepName string) error {
	args := m.Called(ctx, processID, stepNumber, stepName)
	return args.Error(0)
}

func (m *MockMasterDataProcessRepository) GetStep(ctx context.Context, processID int, stepNumber int) (*domain.MasterDataProcessStep, error) {
	args := m.Called(ctx, processID, stepNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MasterDataProcessStep), args.Error(1)
}

func (m *MockMasterDataProcessRepository) UpdateStepStatus(ctx context.Context, processID int, stepNumber int, status string, errorMessage ...string) error {
	args := m.Called(ctx, processID, stepNumber, status, errorMessage)
	return args.Error(0)
}

func (m *MockMasterDataProcessRepository) GetFilteredStocks(ctx context.Context, processDate time.Time) ([]domain.FilteredStockRecord, error) {
	args := m.Called(ctx, processDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.FilteredStockRecord), args.Error(1)
}

func (m *MockMasterDataProcessRepository) GetProcessHistory(ctx context.Context, limit int) ([]domain.MasterDataProcess, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.MasterDataProcess), args.Error(1)
}

type MockDailyDataService struct {
	mock.Mock
}

func (m *MockDailyDataService) InsertDailyCandles(ctx context.Context, days int) error {
	args := m.Called(ctx, days)
	return args.Error(0)
}

type MockFilterPipelineService struct {
	mock.Mock
}

func (m *MockFilterPipelineService) RunFilterPipeline(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockMinuteDataService struct {
	mock.Mock
}

func (m *MockMinuteDataService) BatchStore(ctx context.Context, instrumentKeys []string, fromDate, toDate time.Time, interval string) error {
	args := m.Called(ctx, instrumentKeys, fromDate, toDate, interval)
	return args.Error(0)
}

// Test implementation for master data service
func TestMasterDataService_StartProcess_NewProcess(t *testing.T) {
	// Setup
	mockRepo := &MockMasterDataProcessRepository{}
	mockTradingCalendar := NewTradingCalendarService(true)
	mockDailyService := &MockDailyDataService{}
	mockFilterService := &MockFilterPipelineService{}
	mockMinuteService := &MockMinuteDataService{}

	service := NewMasterDataService(
		mockRepo,
		mockTradingCalendar,
		mockDailyService,
		mockFilterService,
		mockMinuteService,
	)

	ctx := context.Background()
	req := MasterDataRequest{NumberOfPastDays: 0}

	// Mock expectations
	processDate := time.Date(2025, 1, 22, 0, 0, 0, 0, time.UTC)

	// No existing process
	mockRepo.On("GetByDate", ctx, processDate).Return(nil, nil)

	// Create new process
	process := &domain.MasterDataProcess{
		ID:               123,
		ProcessDate:      processDate,
		NumberOfPastDays: 0,
		Status:           domain.ProcessStatusRunning,
	}
	mockRepo.On("Create", ctx, processDate, 0).Return(process, nil)

	// Step 1: Daily ingestion
	mockRepo.On("GetStep", ctx, 123, 1).Return(&domain.MasterDataProcessStep{
		StepNumber: 1,
		Status:     domain.StepStatusPending,
	}, nil)
	mockRepo.On("UpdateStepStatus", ctx, 123, 1, domain.StepStatusRunning).Return(nil)
	mockDailyService.On("InsertDailyCandles", ctx, 0).Return(nil)
	mockRepo.On("UpdateStepStatus", ctx, 123, 1, domain.StepStatusCompleted).Return(nil)

	// Step 2: Filter pipeline
	mockRepo.On("GetStep", ctx, 123, 2).Return(&domain.MasterDataProcessStep{
		StepNumber: 2,
		Status:     domain.StepStatusPending,
	}, nil)
	mockRepo.On("UpdateStepStatus", ctx, 123, 2, domain.StepStatusRunning).Return(nil)
	mockFilterService.On("RunFilterPipeline", ctx).Return(nil)
	mockRepo.On("UpdateStepStatus", ctx, 123, 2, domain.StepStatusCompleted).Return(nil)

	// Step 3: Minute ingestion
	mockRepo.On("GetStep", ctx, 123, 3).Return(&domain.MasterDataProcessStep{
		StepNumber: 3,
		Status:     domain.StepStatusPending,
	}, nil)
	mockRepo.On("UpdateStepStatus", ctx, 123, 3, domain.StepStatusRunning).Return(nil)

	// Mock filtered stocks
	filteredStocks := []domain.FilteredStockRecord{
		{InstrumentKey: "NSE_EQ|INE1234567890"},
		{InstrumentKey: "NSE_EQ|INE0987654321"},
	}
	mockRepo.On("GetFilteredStocks", ctx, processDate).Return(filteredStocks, nil)

	// Mock minute data service
	mockMinuteService.On("BatchStore", ctx, []string{"NSE_EQ|INE1234567890", "NSE_EQ|INE0987654321"}, processDate, processDate, "1minute").Return(nil)
	mockRepo.On("UpdateStepStatus", ctx, 123, 3, domain.StepStatusCompleted).Return(nil)

	// Complete process
	mockRepo.On("CompleteProcess", ctx, 123).Return(nil)

	// Execute
	response, err := service.StartProcess(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 123, response.ProcessID)
	assert.Equal(t, domain.ProcessStatusCompleted, response.Status)
	assert.Equal(t, "Process completed successfully", response.Message)
	assert.Equal(t, "2025-01-22", response.ProcessDate)

	// Verify all mocks were called as expected
	mockRepo.AssertExpectations(t)
	mockDailyService.AssertExpectations(t)
	mockFilterService.AssertExpectations(t)
	mockMinuteService.AssertExpectations(t)
}

func TestMasterDataService_StartProcess_ResumeExistingProcess(t *testing.T) {
	// Setup
	mockRepo := &MockMasterDataProcessRepository{}
	mockTradingCalendar := NewTradingCalendarService(true)
	mockDailyService := &MockDailyDataService{}
	mockFilterService := &MockFilterPipelineService{}
	mockMinuteService := &MockMinuteDataService{}

	service := NewMasterDataService(
		mockRepo,
		mockTradingCalendar,
		mockDailyService,
		mockFilterService,
		mockMinuteService,
	)

	ctx := context.Background()
	req := MasterDataRequest{NumberOfPastDays: 0}

	// Mock existing process
	processDate := time.Date(2025, 1, 22, 0, 0, 0, 0, time.UTC)
	existingProcess := &domain.MasterDataProcess{
		ID:               123,
		ProcessDate:      processDate,
		NumberOfPastDays: 0,
		Status:           domain.ProcessStatusRunning,
		Steps: []domain.MasterDataProcessStep{
			{StepNumber: 1, Status: domain.StepStatusCompleted},
			{StepNumber: 2, Status: domain.StepStatusPending},
			{StepNumber: 3, Status: domain.StepStatusPending},
		},
	}

	mockRepo.On("GetByDate", ctx, processDate).Return(existingProcess, nil)

	// Step 1 already completed, skip
	mockRepo.On("GetStep", ctx, 123, 1).Return(&domain.MasterDataProcessStep{
		StepNumber: 1,
		Status:     domain.StepStatusCompleted,
	}, nil)

	// Step 2: Filter pipeline
	mockRepo.On("GetStep", ctx, 123, 2).Return(&domain.MasterDataProcessStep{
		StepNumber: 2,
		Status:     domain.StepStatusPending,
	}, nil)
	mockRepo.On("UpdateStepStatus", ctx, 123, 2, domain.StepStatusRunning).Return(nil)
	mockFilterService.On("RunFilterPipeline", ctx).Return(nil)
	mockRepo.On("UpdateStepStatus", ctx, 123, 2, domain.StepStatusCompleted).Return(nil)

	// Step 3: Minute ingestion
	mockRepo.On("GetStep", ctx, 123, 3).Return(&domain.MasterDataProcessStep{
		StepNumber: 3,
		Status:     domain.StepStatusPending,
	}, nil)
	mockRepo.On("UpdateStepStatus", ctx, 123, 3, domain.StepStatusRunning).Return(nil)

	// Mock filtered stocks
	filteredStocks := []domain.FilteredStockRecord{
		{InstrumentKey: "NSE_EQ|INE1234567890"},
	}
	mockRepo.On("GetFilteredStocks", ctx, processDate).Return(filteredStocks, nil)

	// Mock minute data service
	mockMinuteService.On("BatchStore", ctx, []string{"NSE_EQ|INE1234567890"}, processDate, processDate, "1minute").Return(nil)
	mockRepo.On("UpdateStepStatus", ctx, 123, 3, domain.StepStatusCompleted).Return(nil)

	// Complete process
	mockRepo.On("CompleteProcess", ctx, 123).Return(nil)

	// Execute
	response, err := service.StartProcess(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 123, response.ProcessID)
	assert.Equal(t, domain.ProcessStatusCompleted, response.Status)

	// Verify all mocks were called as expected
	mockRepo.AssertExpectations(t)
	mockDailyService.AssertExpectations(t)
	mockFilterService.AssertExpectations(t)
	mockMinuteService.AssertExpectations(t)
}

func TestMasterDataService_GetProcessStatus(t *testing.T) {
	// Setup
	mockRepo := &MockMasterDataProcessRepository{}
	mockTradingCalendar := NewTradingCalendarService(true)
	mockDailyService := &MockDailyDataService{}
	mockFilterService := &MockFilterPipelineService{}
	mockMinuteService := &MockMinuteDataService{}

	service := NewMasterDataService(
		mockRepo,
		mockTradingCalendar,
		mockDailyService,
		mockFilterService,
		mockMinuteService,
	)

	ctx := context.Background()
	processID := int64(123)

	// Mock process
	processDate := time.Date(2025, 1, 22, 0, 0, 0, 0, time.UTC)
	process := &domain.MasterDataProcess{
		ID:               processID,
		ProcessDate:      processDate,
		NumberOfPastDays: 0,
		Status:           domain.ProcessStatusRunning,
		CreatedAt:        time.Date(2025, 1, 22, 10, 0, 0, 0, time.UTC),
		Steps: []domain.MasterDataProcessStep{
			{StepNumber: 1, StepName: domain.StepNameDailyIngestion, Status: domain.StepStatusCompleted},
			{StepNumber: 2, StepName: domain.StepNameFilterPipeline, Status: domain.StepStatusRunning},
			{StepNumber: 3, StepName: domain.StepNameMinuteIngestion, Status: domain.StepStatusPending},
		},
	}

	mockRepo.On("GetByID", ctx, processID).Return(process, nil)

	// Execute
	response, err := service.GetProcessStatus(ctx, processID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, processID, response.ProcessID)
	assert.Equal(t, domain.ProcessStatusRunning, response.Status)
	assert.Equal(t, "2025-01-22", response.ProcessDate)
	assert.Len(t, response.Steps, 3)
	assert.Equal(t, domain.StepStatusCompleted, response.Steps[0].Status)
	assert.Equal(t, domain.StepStatusRunning, response.Steps[1].Status)
	assert.Equal(t, domain.StepStatusPending, response.Steps[2].Status)

	mockRepo.AssertExpectations(t)
}

func TestMasterDataService_GetProcessHistory(t *testing.T) {
	// Setup
	mockRepo := &MockMasterDataProcessRepository{}
	mockTradingCalendar := NewTradingCalendarService(true)
	mockDailyService := &MockDailyDataService{}
	mockFilterService := &MockFilterPipelineService{}
	mockMinuteService := &MockMinuteDataService{}

	service := NewMasterDataService(
		mockRepo,
		mockTradingCalendar,
		mockDailyService,
		mockFilterService,
		mockMinuteService,
	)

	ctx := context.Background()
	limit := 10

	// Mock process history
	processDate := time.Date(2025, 1, 22, 0, 0, 0, 0, time.UTC)
	processes := []domain.MasterDataProcess{
		{
			ID:               123,
			ProcessDate:      processDate,
			NumberOfPastDays: 0,
			Status:           domain.ProcessStatusCompleted,
			CreatedAt:        time.Date(2025, 1, 22, 10, 0, 0, 0, time.UTC),
			CompletedAt:      &processDate,
		},
		{
			ID:               124,
			ProcessDate:      processDate.AddDate(0, 0, -1),
			NumberOfPastDays: 1,
			Status:           domain.ProcessStatusRunning,
			CreatedAt:        time.Date(2025, 1, 21, 10, 0, 0, 0, time.UTC),
		},
	}

	mockRepo.On("GetProcessHistory", ctx, limit).Return(processes, nil)

	// Execute
	response, err := service.GetProcessHistory(ctx, limit)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response, 2)
	assert.Equal(t, 123, response[0].ProcessID)
	assert.Equal(t, domain.ProcessStatusCompleted, response[0].Status)
	assert.Equal(t, 124, response[1].ProcessID)
	assert.Equal(t, domain.ProcessStatusRunning, response[1].Status)

	mockRepo.AssertExpectations(t)
}
