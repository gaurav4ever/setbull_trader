package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"github.com/google/uuid"
)

// OrderExecutionService handles the execution of orders based on execution plans
type OrderExecutionService struct {
	orderExecutionRepo repository.OrderExecutionRepository
	executionPlanRepo  repository.ExecutionPlanRepository
	stockRepo          repository.StockRepository
	levelEntryRepo     repository.LevelEntryRepository
	// In a real system, you'd have a broker client here
	// brokerClient BrokerClient
}

// NewOrderExecutionService creates a new OrderExecutionService
func NewOrderExecutionService(
	orderExecutionRepo repository.OrderExecutionRepository,
	executionPlanRepo repository.ExecutionPlanRepository,
	stockRepo repository.StockRepository,
	levelEntryRepo repository.LevelEntryRepository,
) *OrderExecutionService {
	return &OrderExecutionService{
		orderExecutionRepo: orderExecutionRepo,
		executionPlanRepo:  executionPlanRepo,
		stockRepo:          stockRepo,
		levelEntryRepo:     levelEntryRepo,
	}
}

// ExecuteOrdersForStock executes trades for a single stock based on its execution plan
func (s *OrderExecutionService) ExecuteOrdersForStock(ctx context.Context, stockID string) (*domain.OrderExecution, error) {
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

	// Get the latest execution plan
	plan, err := s.executionPlanRepo.GetByStockID(ctx, stockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution plan: %w", err)
	}
	if plan == nil {
		return nil, errors.New("no execution plan found for this stock")
	}

	// Get level entries
	levelEntries, err := s.levelEntryRepo.GetByExecutionPlanID(ctx, plan.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get level entries: %w", err)
	}

	// Create order execution record
	orderExecution := &domain.OrderExecution{
		ID:              uuid.New().String(),
		ExecutionPlanID: plan.ID,
		Status:          domain.OrderStatusPending,
		ExecutedAt:      time.Now(),
	}

	err = s.orderExecutionRepo.Create(ctx, orderExecution)
	if err != nil {
		return nil, fmt.Errorf("failed to create order execution record: %w", err)
	}

	// Start execution - in a real system, this might be async
	err = s.executeOrders(ctx, orderExecution.ID, levelEntries)
	if err != nil {
		// Update status to failed
		_ = s.orderExecutionRepo.UpdateStatus(ctx, orderExecution.ID, domain.OrderStatusFailed, err.Error())
		return nil, fmt.Errorf("failed to execute orders: %w", err)
	}

	// Update status to completed
	err = s.orderExecutionRepo.UpdateStatus(ctx, orderExecution.ID, domain.OrderStatusCompleted, "")
	if err != nil {
		return nil, fmt.Errorf("failed to update order execution status: %w", err)
	}

	// Get the updated order execution
	updatedExecution, err := s.orderExecutionRepo.GetByID(ctx, orderExecution.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated order execution: %w", err)
	}

	return updatedExecution, nil
}

// ExecuteOrdersForAllSelectedStocks executes trades for all selected stocks
func (s *OrderExecutionService) ExecuteOrdersForAllSelectedStocks(ctx context.Context) ([]*domain.OrderExecution, error) {
	// Get all selected stocks
	selectedStocks, err := s.stockRepo.GetSelected(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get selected stocks: %w", err)
	}

	if len(selectedStocks) == 0 {
		return nil, errors.New("no stocks are selected for trading")
	}

	// Execute orders for each stock
	var executions []*domain.OrderExecution

	for _, stock := range selectedStocks {
		execution, err := s.ExecuteOrdersForStock(ctx, stock.ID)
		if err != nil {
			// Log the error but continue with other stocks
			fmt.Printf("Error executing orders for stock %s: %v\n", stock.Symbol, err)
			continue
		}

		executions = append(executions, execution)
	}

	if len(executions) == 0 {
		return nil, errors.New("failed to execute orders for any selected stocks")
	}

	return executions, nil
}

// executeOrders performs the actual order execution
// In a real system, this would interact with a broker's API
func (s *OrderExecutionService) executeOrders(
	ctx context.Context,
	executionID string,
	levelEntries []domain.LevelEntry,
) error {
	// Update status to executing
	err := s.orderExecutionRepo.UpdateStatus(ctx, executionID, domain.OrderStatusExecuting, "")
	if err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	// In a real implementation, this would place the orders with your broker
	// For each level entry (except stop loss, which would be at index 0)
	for i, entry := range levelEntries {
		if i == 0 || entry.Quantity <= 0 {
			// Skip stop loss level or entries with no quantity
			continue
		}

		// Placeholder for broker API calls
		// Replace with actual broker integration
		fmt.Printf(
			"Placing order: Level=%s, Price=%f, Quantity=%d\n",
			entry.Description,
			entry.Price,
			entry.Quantity,
		)

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// GetOrderExecutionByID retrieves an order execution by ID
func (s *OrderExecutionService) GetOrderExecutionByID(ctx context.Context, id string) (*domain.OrderExecution, error) {
	return s.orderExecutionRepo.GetByID(ctx, id)
}

// GetOrderExecutionsByPlanID retrieves all order executions for an execution plan
func (s *OrderExecutionService) GetOrderExecutionsByPlanID(ctx context.Context, planID string) ([]*domain.OrderExecution, error) {
	return s.orderExecutionRepo.GetByExecutionPlanID(ctx, planID)
}
