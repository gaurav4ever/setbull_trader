package service

import (
	"context"
	"errors"
	"fmt"
	"math"
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

	// Debug: Log the stock details
	log.Info("[OrderExecution] Retrieved stock: ID=%s, Symbol=%s, SecurityID=%s",
		stock.ID, stock.Symbol, stock.SecurityID)

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
	log.Info("Starting order execution for stock %s", stockID)
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

	// Add debug log to show how many level entries we're processing
	log.Info("Processing level entries for executionID: %s, stockSymbol: %s, entryCount: %d, tradeSide: %s",
		executionID, stock.Symbol, len(levelEntries), tradeSide)

	// Track which levels we've already processed to avoid duplicates
	processedLevels := make(map[string]bool)

	// Place orders for each level entry (except stop loss at index 0)
	for i, entry := range levelEntries {

		// Add detailed logging for each entry
		log.Info("Processing entry index: %d, description: %s, price: %f, quantity: %d",
			i, entry.Description, entry.Price, entry.Quantity)

		if i == 0 || entry.Quantity <= 0 {
			// Skip stop loss level or entries with no quantity
			continue
		}

		// Check if we've already processed this level (using description as identifier)
		if processedLevels[entry.Description] {
			log.Warn("Duplicate level entry detected - skipping description: %s, price: %f",
				entry.Description, entry.Price)
			continue
		}

		// Mark this level as processed
		processedLevels[entry.Description] = true

		var triggerPrice float64
		if tradeSide == "SELL" {
			triggerPrice = math.Round((entry.Price+0.05)*100) / 100
		} else if tradeSide == "BUY" {
			triggerPrice = math.Round((entry.Price-0.05)*100) / 100
		}

		// Create order request
		orderReq := &request.PlaceOrderRequest{
			TransactionType: string(tradeSide),
			ExchangeSegment: "NSE_EQ",
			ProductType:     "INTRADAY",
			OrderType:       "STOP_LOSS",
			SecurityID:      stock.SecurityID,
			Quantity:        entry.Quantity,
			Price:           entry.Price,
			TriggerPrice:    triggerPrice,
			Validity:        "DAY",
		}

		// Debug: Log the SecurityID being used
		log.Info("[OrderExecution] Using SecurityID: %s for stock %s", stock.SecurityID, stock.Symbol)

		// Log the order request details
		log.Info("Placing order level: %s, price: %f, triggerPrice: %f, quantity: %d",
			entry.Description, entry.Price, triggerPrice, entry.Quantity)

		// Place the order with Dhan
		response, err := s.orderService.PlaceOrder(orderReq)

		result := domain.OrderExecutionResult{
			LevelDescription: entry.Description,
		}

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			results.Success = false
			log.Error("Failed to place order level: %s, error: %v", entry.Description, err)
		} else {
			result.Success = true
			result.OrderID = response.OrderID
			result.OrderStatus = response.OrderStatus
			log.Info("Order placed successfully level: %s, orderID: %s, status: %s",
				entry.Description, response.OrderID, response.OrderStatus)
		}

		results.Results = append(results.Results, result)
	}

	// Log summary of execution
	log.Info("Order execution completed executionID: %s, stockSymbol: %s, orderCount: %d, success: %t",
		executionID, stock.Symbol, len(results.Results), results.Success)

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
