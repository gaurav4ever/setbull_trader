package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"github.com/google/uuid"
)

// ExecutionPlanService provides operations for creating and managing execution plans
type ExecutionPlanService struct {
	executionPlanRepo repository.ExecutionPlanRepository
	levelEntryRepo    repository.LevelEntryRepository
	stockRepo         repository.StockRepository
	tradeParamsRepo   repository.TradeParametersRepository
	fibCalculator     *FibonacciCalculator
}

// NewExecutionPlanService creates a new ExecutionPlanService
func NewExecutionPlanService(
	executionPlanRepo repository.ExecutionPlanRepository,
	levelEntryRepo repository.LevelEntryRepository,
	stockRepo repository.StockRepository,
	tradeParamsRepo repository.TradeParametersRepository,
) *ExecutionPlanService {
	return &ExecutionPlanService{
		executionPlanRepo: executionPlanRepo,
		levelEntryRepo:    levelEntryRepo,
		stockRepo:         stockRepo,
		tradeParamsRepo:   tradeParamsRepo,
		fibCalculator:     NewFibonacciCalculator(),
	}
}

// CreateExecutionPlan creates a new execution plan for a stock
func (s *ExecutionPlanService) CreateExecutionPlan(ctx context.Context, stockID string) (*domain.ExecutionPlan, error) {
	// Verify stock exists and is selected
	stock, err := s.stockRepo.GetByID(ctx, stockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stock: %w", err)
	}
	if stock == nil {
		return nil, errors.New("stock not found")
	}
	if !stock.IsSelected {
		return nil, errors.New("stock is not selected for trading")
	}

	// Get trade parameters for the stock
	params, err := s.tradeParamsRepo.GetByStockID(ctx, stockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade parameters: %w", err)
	}
	if params == nil {
		return nil, errors.New("trade parameters not configured for this stock")
	}

	// Calculate Fibonacci levels
	fibLevels := s.fibCalculator.CalculateFibonacciLevels(
		params.StartingPrice,
		params.StopLossPercentage,
		params.TradeSide,
	)

	// Calculate SL points for position sizing
	var slPoints float64
	if params.TradeSide == domain.Buy {
		slPoints = params.StartingPrice - fibLevels[0].Price
	} else {
		slPoints = fibLevels[0].Price - params.StartingPrice
	}

	// Calculate total quantity based on risk
	totalQuantity := int(math.Floor(params.RiskAmount / slPoints))
	if totalQuantity <= 0 {
		return nil, errors.New("calculated quantity is too small, consider increasing risk amount or reducing stop loss distance")
	}

	// Create execution plan
	plan := &domain.ExecutionPlan{
		ID:            uuid.New().String(),
		StockID:       stockID,
		ParametersID:  params.ID,
		TotalQuantity: totalQuantity,
		Stock:         stock,
		Parameters:    params,
	}

	// Save execution plan
	err = s.executionPlanRepo.Create(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution plan: %w", err)
	}

	// Create level entries
	levelEntries := s.calculateLevelEntries(fibLevels, plan.ID, totalQuantity)
	err = s.levelEntryRepo.CreateMany(ctx, levelEntries)
	if err != nil {
		// Rollback execution plan creation
		_ = s.executionPlanRepo.Delete(ctx, plan.ID)
		return nil, fmt.Errorf("failed to create level entries: %w", err)
	}

	// Set level entries on the plan for return
	plan.LevelEntries = levelEntries

	return plan, nil
}

// calculateLevelEntries converts execution levels to level entries with quantities
func (s *ExecutionPlanService) calculateLevelEntries(
	fibLevels []domain.ExecutionLevel,
	planID string,
	totalQuantity int,
) []domain.LevelEntry {
	// Calculate quantity per leg (distribute across 5 entry legs)
	legCount := 5
	baseQtyPerLeg := totalQuantity / legCount

	// Distribute remainders
	remainder := totalQuantity % legCount

	// Create level entries with quantities
	levelEntries := make([]domain.LevelEntry, 0, len(fibLevels))

	for i, level := range fibLevels {
		entry := domain.LevelEntry{
			ID:              uuid.New().String(),
			ExecutionPlanID: planID,
			FibLevel:        level.Level,
			Price:           level.Price,
			Description:     level.Description,
		}

		if i == 0 {
			// Stop loss level has no quantity
			entry.Quantity = 0
		} else {
			// Calculate quantity for this leg
			entry.Quantity = baseQtyPerLeg

			// Distribute remainder (if any) to early legs
			if i-1 < remainder {
				entry.Quantity++
			}
		}

		levelEntries = append(levelEntries, entry)
	}

	return levelEntries
}

// GetExecutionPlanByID retrieves an execution plan by ID with all related data
func (s *ExecutionPlanService) GetExecutionPlanByID(ctx context.Context, id string) (*domain.ExecutionPlan, error) {
	// Get execution plan
	plan, err := s.executionPlanRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution plan: %w", err)
	}
	if plan == nil {
		return nil, errors.New("execution plan not found")
	}

	// Enrich with stock data
	stock, err := s.stockRepo.GetByID(ctx, plan.StockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stock: %w", err)
	}
	plan.Stock = stock

	// Enrich with trade parameters
	params, err := s.tradeParamsRepo.GetByID(ctx, plan.ParametersID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade parameters: %w", err)
	}
	plan.Parameters = params

	// Enrich with level entries
	levelEntries, err := s.levelEntryRepo.GetByExecutionPlanID(ctx, plan.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get level entries: %w", err)
	}
	plan.LevelEntries = levelEntries

	return plan, nil
}

// GetExecutionPlanByStockID retrieves the latest execution plan for a stock
func (s *ExecutionPlanService) GetExecutionPlanByStockID(ctx context.Context, stockID string) (*domain.ExecutionPlan, error) {
	// Get execution plan
	plan, err := s.executionPlanRepo.GetByStockID(ctx, stockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution plan: %w", err)
	}
	if plan == nil {
		return nil, nil // No plan exists yet
	}

	// Enrich with related data
	return s.GetExecutionPlanByID(ctx, plan.ID)
}

// GetAllExecutionPlans retrieves all execution plans with their related data
func (s *ExecutionPlanService) GetAllExecutionPlans(ctx context.Context) ([]*domain.ExecutionPlan, error) {
	plans, err := s.executionPlanRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution plans: %w", err)
	}

	// Enrich all plans with their related data
	for i, plan := range plans {
		enrichedPlan, err := s.GetExecutionPlanByID(ctx, plan.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to enrich execution plan %s: %w", plan.ID, err)
		}
		plans[i] = enrichedPlan
	}

	return plans, nil
}

// DeleteExecutionPlan deletes an execution plan and its level entries
func (s *ExecutionPlanService) DeleteExecutionPlan(ctx context.Context, id string) error {
	// Check if plan exists
	plan, err := s.executionPlanRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get execution plan: %w", err)
	}
	if plan == nil {
		return errors.New("execution plan not found")
	}

	// Delete level entries first
	err = s.levelEntryRepo.DeleteByExecutionPlanID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete level entries: %w", err)
	}

	// Delete execution plan
	err = s.executionPlanRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete execution plan: %w", err)
	}

	return nil
}
