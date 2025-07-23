package service

import (
	"context"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
	"time"
)

// MasterDataService orchestrates the master data ingestion pipeline
type MasterDataService interface {
	// StartProcess starts a new master data ingestion process
	StartProcess(ctx context.Context, req MasterDataRequest) (*MasterDataResponse, error)

	// GetProcessStatus retrieves the status of a process
	GetProcessStatus(ctx context.Context, processID int) (*ProcessStatusResponse, error)

	// GetProcessHistory retrieves recent process history
	GetProcessHistory(ctx context.Context, limit int) ([]ProcessStatusResponse, error)
}

// MasterDataRequest represents the request to start a master data process
type MasterDataRequest struct {
	NumberOfPastDays int `json:"numberOfPastDays"`
}

// MasterDataResponse represents the response from starting a process
type MasterDataResponse struct {
	ProcessID   int    `json:"processId"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	ProcessDate string `json:"processDate"`
}

// ProcessStatusResponse represents the status of a process with its steps
type ProcessStatusResponse struct {
	ProcessID   int                 `json:"processId"`
	Status      string              `json:"status"`
	Steps       []ProcessStepStatus `json:"steps"`
	ProcessDate string              `json:"processDate"`
	CreatedAt   string              `json:"createdAt"`
	CompletedAt string              `json:"completedAt,omitempty"`
}

// ProcessStepStatus represents the status of a step within a process
type ProcessStepStatus struct {
	StepNumber   int    `json:"stepNumber"`
	StepName     string `json:"stepName"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	StartedAt    string `json:"startedAt,omitempty"`
	CompletedAt  string `json:"completedAt,omitempty"`
}

// PipelineStep represents a step in the pipeline
type PipelineStep struct {
	Number  int
	Name    string
	Handler func(ctx context.Context, process *domain.MasterDataProcess) error
}

// masterDataService implements MasterDataService
type masterDataService struct {
	processRepo     repository.MasterDataProcessRepository
	tradingCalendar *TradingCalendarService
	dailyService    DailyDataService
	filterService   FilterPipelineService
	minuteService   MinuteDataService
}

// DailyDataService interface for daily data operations
type DailyDataService interface {
	InsertDailyCandles(ctx context.Context, days int) error
}

// FilterPipelineService interface for filter pipeline operations
type FilterPipelineService interface {
	RunFilterPipeline(ctx context.Context) error
}

// MinuteDataService interface for minute data operations
type MinuteDataService interface {
	BatchStore(ctx context.Context, instrumentKeys []string, fromDate, toDate time.Time, interval string) error
}

// NewMasterDataService creates a new MasterDataService
func NewMasterDataService(
	processRepo repository.MasterDataProcessRepository,
	tradingCalendar *TradingCalendarService,
	dailyService DailyDataService,
	filterService FilterPipelineService,
	minuteService MinuteDataService,
) MasterDataService {
	return &masterDataService{
		processRepo:     processRepo,
		tradingCalendar: tradingCalendar,
		dailyService:    dailyService,
		filterService:   filterService,
		minuteService:   minuteService,
	}
}

// StartProcess starts a new master data ingestion process
func (s *masterDataService) StartProcess(ctx context.Context, req MasterDataRequest) (*MasterDataResponse, error) {
	log.Info("Starting master data process with %d past days", req.NumberOfPastDays)

	// 1. Determine target date based on numberOfPastDays
	targetDate := s.getTargetDate(req.NumberOfPastDays)

	// 2. Check if process already exists for this date
	existingProcess, err := s.processRepo.GetByDate(ctx, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing process: %w", err)
	}

	if existingProcess != nil {
		log.Info("Found existing process for date %s, resuming...", targetDate.Format("2006-01-02"))
		return s.resumeProcess(ctx, existingProcess)
	}

	// 3. Create new process record
	process, err := s.processRepo.Create(ctx, targetDate, req.NumberOfPastDays)
	if err != nil {
		return nil, fmt.Errorf("failed to create process: %w", err)
	}

	log.Info("Created new process with ID %d for date %s", process.ID, targetDate.Format("2006-01-02"))

	// 4. Execute pipeline steps sequentially
	return s.executePipeline(ctx, process)
}

// getTargetDate determines the target date based on numberOfPastDays
func (s *masterDataService) getTargetDate(numberOfPastDays int) time.Time {
	if numberOfPastDays == 0 {
		// For current day, get the most recent trading day
		return s.tradingCalendar.PreviousTradingDay(time.Now())
	}

	// For past days, subtract the specified number of trading days
	return s.tradingCalendar.SubtractTradingDays(time.Now(), numberOfPastDays)
}

// resumeProcess resumes an existing process from where it left off
func (s *masterDataService) resumeProcess(ctx context.Context, process *domain.MasterDataProcess) (*MasterDataResponse, error) {
	log.Info("Resuming process %d with status %s", process.ID, process.Status)

	// If process is already completed, return success
	if process.Status == domain.ProcessStatusCompleted {
		return &MasterDataResponse{
			ProcessID:   process.ID,
			Status:      process.Status,
			Message:     "Process already completed",
			ProcessDate: process.ProcessDate.Format("2006-01-02"),
		}, nil
	}

	// If process failed, restart from the beginning
	if process.Status == domain.ProcessStatusFailed {
		log.Info("Restarting failed process %d", process.ID)
		err := s.processRepo.UpdateStatus(ctx, process.ID, domain.ProcessStatusRunning)
		if err != nil {
			return nil, fmt.Errorf("failed to update process status: %w", err)
		}
		process.Status = domain.ProcessStatusRunning
	}

	// Execute pipeline from where it left off
	return s.executePipeline(ctx, process)
}

// executePipeline executes the pipeline steps sequentially
func (s *masterDataService) executePipeline(ctx context.Context, process *domain.MasterDataProcess) (*MasterDataResponse, error) {
	steps := []PipelineStep{
		{Number: 1, Name: domain.StepNameDailyIngestion, Handler: s.executeDailyIngestion},
		{Number: 2, Name: domain.StepNameFilterPipeline, Handler: s.executeFilterPipeline},
		{Number: 3, Name: domain.StepNameMinuteIngestion, Handler: s.executeMinuteIngestion},
	}

	for _, step := range steps {
		if err := s.executeStep(ctx, process, step); err != nil {
			// Mark process as failed
			s.processRepo.UpdateStatus(ctx, process.ID, domain.ProcessStatusFailed)
			return nil, fmt.Errorf("step %d (%s) failed: %w", step.Number, step.Name, err)
		}
	}

	// Mark process as completed
	err := s.processRepo.CompleteProcess(ctx, process.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to complete process: %w", err)
	}

	log.Info("Process %d completed successfully", process.ID)

	return &MasterDataResponse{
		ProcessID:   process.ID,
		Status:      domain.ProcessStatusCompleted,
		Message:     "Process completed successfully",
		ProcessDate: process.ProcessDate.Format("2006-01-02"),
	}, nil
}

// executeStep executes a single step in the pipeline
func (s *masterDataService) executeStep(ctx context.Context, process *domain.MasterDataProcess, step PipelineStep) error {
	// 1. Check if step is already completed
	stepRecord, err := s.processRepo.GetStep(ctx, process.ID, step.Number)
	if err != nil {
		return fmt.Errorf("failed to get step %d: %w", step.Number, err)
	}

	if stepRecord != nil && stepRecord.Status == domain.StepStatusCompleted {
		log.Info("Step %d (%s) already completed, skipping", step.Number, step.Name)
		return nil
	}

	// 2. Mark step as running
	err = s.processRepo.UpdateStepStatus(ctx, process.ID, step.Number, domain.StepStatusRunning)
	if err != nil {
		return fmt.Errorf("failed to update step status to running: %w", err)
	}

	log.Info("Executing step %d (%s) for process %d", step.Number, step.Name, process.ID)

	// 3. Execute step with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	if err := step.Handler(ctx, process); err != nil {
		// Mark step as failed
		s.processRepo.UpdateStepStatus(ctx, process.ID, step.Number, domain.StepStatusFailed, err.Error())
		return fmt.Errorf("step %d (%s) execution failed: %w", step.Number, step.Name, err)
	}

	// 4. Mark step as completed
	err = s.processRepo.UpdateStepStatus(ctx, process.ID, step.Number, domain.StepStatusCompleted)
	if err != nil {
		return fmt.Errorf("failed to update step status to completed: %w", err)
	}

	log.Info("Step %d (%s) completed successfully", step.Number, step.Name)
	return nil
}

// executeDailyIngestion executes step 1: Daily data ingestion
func (s *masterDataService) executeDailyIngestion(ctx context.Context, process *domain.MasterDataProcess) error {
	log.Info("Starting daily data ingestion for process %d", process.ID)

	// Call existing daily candles service
	err := s.dailyService.InsertDailyCandles(ctx, process.NumberOfPastDays)
	if err != nil {
		return fmt.Errorf("daily data ingestion failed: %w", err)
	}

	log.Info("Daily data ingestion completed for process %d", process.ID)
	return nil
}

// executeFilterPipeline executes step 2: Filter pipeline
func (s *masterDataService) executeFilterPipeline(ctx context.Context, process *domain.MasterDataProcess) error {
	log.Info("Starting filter pipeline for process %d", process.ID)

	// Call existing filter pipeline service
	err := s.filterService.RunFilterPipeline(ctx)
	if err != nil {
		return fmt.Errorf("filter pipeline failed: %w", err)
	}

	log.Info("Filter pipeline completed for process %d", process.ID)
	return nil
}

// executeMinuteIngestion executes step 3: 1-minute data ingestion
func (s *masterDataService) executeMinuteIngestion(ctx context.Context, process *domain.MasterDataProcess) error {
	log.Info("Starting minute data ingestion for process %d", process.ID)

	// 1. Fetch filtered stocks from database
	filteredStocks, err := s.processRepo.GetFilteredStocks(ctx, process.ProcessDate)
	if err != nil {
		return fmt.Errorf("failed to get filtered stocks: %w", err)
	}

	if len(filteredStocks) == 0 {
		log.Warn("No filtered stocks found for date %s, skipping minute data ingestion", process.ProcessDate.Format("2006-01-02"))
		return nil
	}

	// 2. Extract instrument keys
	instrumentKeys := s.extractInstrumentKeys(filteredStocks)

	log.Info("Found %d filtered stocks for minute data ingestion", len(instrumentKeys))

	// 3. Call batch store API for 1-minute data
	err = s.minuteService.BatchStore(ctx, instrumentKeys, process.ProcessDate, process.ProcessDate, "1minute")
	if err != nil {
		return fmt.Errorf("minute data ingestion failed: %w", err)
	}

	log.Info("Minute data ingestion completed for process %d", process.ID)
	return nil
}

// extractInstrumentKeys extracts instrument keys from filtered stocks
func (s *masterDataService) extractInstrumentKeys(stocks []domain.FilteredStockRecord) []string {
	instrumentKeys := make([]string, len(stocks))
	for i, stock := range stocks {
		instrumentKeys[i] = stock.InstrumentKey
	}
	return instrumentKeys
}

// GetProcessStatus retrieves the status of a process
func (s *masterDataService) GetProcessStatus(ctx context.Context, processID int) (*ProcessStatusResponse, error) {
	process, err := s.processRepo.GetByID(ctx, processID)
	if err != nil {
		return nil, fmt.Errorf("failed to get process: %w", err)
	}

	if process == nil {
		return nil, fmt.Errorf("process with ID %d not found", processID)
	}

	return s.convertToProcessStatusResponse(process), nil
}

// GetProcessHistory retrieves recent process history
func (s *masterDataService) GetProcessHistory(ctx context.Context, limit int) ([]ProcessStatusResponse, error) {
	processes, err := s.processRepo.GetProcessHistory(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get process history: %w", err)
	}

	responses := make([]ProcessStatusResponse, len(processes))
	for i, process := range processes {
		responses[i] = *s.convertToProcessStatusResponse(&process)
	}

	return responses, nil
}

// convertToProcessStatusResponse converts a domain process to a response
func (s *masterDataService) convertToProcessStatusResponse(process *domain.MasterDataProcess) *ProcessStatusResponse {
	steps := make([]ProcessStepStatus, len(process.Steps))
	for i, step := range process.Steps {
		steps[i] = ProcessStepStatus{
			StepNumber:   step.StepNumber,
			StepName:     step.StepName,
			Status:       step.Status,
			ErrorMessage: s.getStringValue(step.ErrorMessage),
			StartedAt:    s.formatTime(step.StartedAt),
			CompletedAt:  s.formatTime(step.CompletedAt),
		}
	}

	return &ProcessStatusResponse{
		ProcessID:   process.ID,
		Status:      process.Status,
		Steps:       steps,
		ProcessDate: process.ProcessDate.Format("2006-01-02"),
		CreatedAt:   process.CreatedAt.Format(time.RFC3339),
		CompletedAt: s.formatTime(process.CompletedAt),
	}
}

// getStringValue safely gets string value from pointer
func (s *masterDataService) getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// formatTime safely formats time from pointer
func (s *masterDataService) formatTime(ptr *time.Time) string {
	if ptr == nil {
		return ""
	}
	return ptr.Format(time.RFC3339)
}
