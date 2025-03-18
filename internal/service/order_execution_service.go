package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"setbull_trader/internal/core/dto/request"
	"setbull_trader/internal/core/service/orders"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"

	"github.com/google/uuid"
)

// OrderExecutionService handles the execution of orders based on execution plans
type OrderExecutionService struct {
	orderExecutionRepo repository.OrderExecutionRepository
	executionPlanRepo  repository.ExecutionPlanRepository
	stockRepo          repository.StockRepository
	levelEntryRepo     repository.LevelEntryRepository
	orderService       orders.Service
	stockService       StockService
}

// NewOrderExecutionService creates a new OrderExecutionService
func NewOrderExecutionService(
	orderExecutionRepo repository.OrderExecutionRepository,
	executionPlanRepo repository.ExecutionPlanRepository,
	stockRepo repository.StockRepository,
	levelEntryRepo repository.LevelEntryRepository,
	orderService orders.Service,
	stockService StockService,
) *OrderExecutionService {
	return &OrderExecutionService{
		orderExecutionRepo: orderExecutionRepo,
		executionPlanRepo:  executionPlanRepo,
		stockRepo:          stockRepo,
		levelEntryRepo:     levelEntryRepo,
		orderService:       orderService,
		stockService:       stockService,
	}
}

// ExecuteOrdersForStock executes trades for a single stock based on its execution plan
func (s *OrderExecutionService) ExecuteOrdersForStock(ctx context.Context, stockID string) (*domain.OrderExecution, *domain.ExecutionResults, error) {
	// Verify stock exists and is selected
	// stock, err := s.stockRepo.GetByID(ctx, stockID)
	stock, err := s.stockService.GetStockByID(ctx, stockID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stock: %w", err)
	}
	if stock == nil {
		return nil, nil, errors.New("stock not found")
	}
	if !stock.IsSelected {
		return nil, nil, errors.New("stock is not selected for trading")
	}

	// Get the latest execution plan
	plan, err := s.executionPlanRepo.GetByStockID(ctx, stockID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get execution plan: %w", err)
	}
	if plan == nil {
		return nil, nil, errors.New("no execution plan found for this stock")
	}

	// Get level entries
	levelEntries, err := s.levelEntryRepo.GetByExecutionPlanID(ctx, plan.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get level entries: %w", err)
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
		return nil, nil, fmt.Errorf("failed to create order execution record: %w", err)
	}

	// Start execution - in a real system, this might be async
	executionResults, err := s.executeOrders(ctx, orderExecution.ID, levelEntries, stock, stock.Parameters.TradeSide)
	if err != nil {
		// Update status to failed
		_ = s.orderExecutionRepo.UpdateStatus(ctx, orderExecution.ID, domain.OrderStatusFailed, err.Error())
		return nil, nil, fmt.Errorf("failed to execute orders: %w", err)
	}

	// Update status based on execution results
	finalStatus := domain.OrderStatusCompleted
	var statusMessage string
	if !executionResults.Success {
		finalStatus = domain.OrderStatusFailed
		statusMessage = "Some orders failed to execute"
	}

	err = s.orderExecutionRepo.UpdateStatus(ctx, orderExecution.ID, finalStatus, statusMessage)
	if err != nil {
		return nil, executionResults, fmt.Errorf("failed to update order execution status: %w", err)
	}

	// Get the updated order execution
	updatedExecution, err := s.orderExecutionRepo.GetByID(ctx, orderExecution.ID)
	if err != nil {
		return nil, executionResults, fmt.Errorf("failed to get updated order execution: %w", err)
	}

	return updatedExecution, executionResults, nil
}

// ExecuteOrdersForAllSelectedStocks executes trades for all selected stocks
func (s *OrderExecutionService) ExecuteOrdersForAllSelectedStocks(ctx context.Context) ([]*domain.OrderExecution, []*domain.ExecutionResults, error) {
	// Get all selected stocks
	selectedStocks, err := s.stockRepo.GetSelected(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get selected stocks: %w", err)
	}

	if len(selectedStocks) == 0 {
		return nil, nil, errors.New("no stocks are selected for trading")
	}

	// Execute orders for each stock
	var executions []*domain.OrderExecution
	var allResults []*domain.ExecutionResults

	for _, stock := range selectedStocks {
		execution, results, err := s.ExecuteOrdersForStock(ctx, stock.ID)
		if err != nil {
			// Log the error but continue with other stocks
			fmt.Printf("Error executing orders for stock %s: %v\n", stock.Symbol, err)
			return nil, nil, err
		}

		executions = append(executions, execution)
		allResults = append(allResults, results)
	}

	if len(executions) == 0 {
		return nil, nil, errors.New("failed to execute orders for any selected stocks")
	}

	return executions, allResults, nil
}

// executeOrders performs the actual order execution
// In a real system, this would interact with a broker's API
func (s *OrderExecutionService) executeOrders(
	ctx context.Context,
	executionID string,
	levelEntries []domain.LevelEntry,
	stock *domain.Stock,
	tradeSide domain.TradeSide,
) (*domain.ExecutionResults, error) {
	// Update status to executing
	err := s.orderExecutionRepo.UpdateStatus(ctx, executionID, domain.OrderStatusExecuting, "")
	if err != nil {
		return nil, fmt.Errorf("failed to update execution status: %w", err)
	}

	results := &domain.ExecutionResults{
		ExecutionID: executionID,
		StockSymbol: stock.Symbol,
		Results:     make([]domain.OrderExecutionResult, 0),
		Success:     true, // Will be set to false if any order fails
	}

	// Place orders for each level entry (except stop loss at index 0)
	for i, entry := range levelEntries {
		if i == 0 || entry.Quantity <= 0 {
			// Skip stop loss level or entries with no quantity
			continue
		}

		// Create order request
		orderReq := &request.PlaceOrderRequest{
			TransactionType: string(tradeSide),
			ExchangeSegment: "NSE_EQ",
			ProductType:     "INTRADAY",
			OrderType:       "MARKET",
			SecurityID:      stock.SecurityID,
			Quantity:        entry.Quantity,
			Price:           entry.Price,
			Validity:        "DAY",
		}

		// Place the order with Dhan
		response, err := s.orderService.PlaceOrder(orderReq)

		result := domain.OrderExecutionResult{
			LevelDescription: entry.Description,
		}

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			results.Success = false
			log.Error("Failed to place order", "level", entry.Description, "error", err)
		} else {
			result.Success = true
			result.OrderID = response.OrderID
			result.OrderStatus = response.OrderStatus
			log.Info("Order placed successfully",
				"level", entry.Description,
				"orderID", response.OrderID,
				"status", response.OrderStatus)
		}

		results.Results = append(results.Results, result)
	}

	if len(results.Results) == 0 {
		return nil, errors.New("no orders were attempted")
	}

	return results, nil
}

// GetOrderExecutionByID retrieves an order execution by ID
func (s *OrderExecutionService) GetOrderExecutionByID(ctx context.Context, id string) (*domain.OrderExecution, error) {
	return s.orderExecutionRepo.GetByID(ctx, id)
}

// GetOrderExecutionsByPlanID retrieves all order executions for an execution plan
func (s *OrderExecutionService) GetOrderExecutionsByPlanID(ctx context.Context, planID string) ([]*domain.OrderExecution, error) {
	return s.orderExecutionRepo.GetByExecutionPlanID(ctx, planID)
}
