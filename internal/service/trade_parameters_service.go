package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
)

// TradeParametersService provides operations on trade parameters
type TradeParametersService struct {
	tradeParamsRepo repository.TradeParametersRepository
	stockRepo       repository.StockRepository
}

// NewTradeParametersService creates a new TradeParametersService
func NewTradeParametersService(
	tradeParamsRepo repository.TradeParametersRepository,
	stockRepo repository.StockRepository,
) *TradeParametersService {
	return &TradeParametersService{
		tradeParamsRepo: tradeParamsRepo,
		stockRepo:       stockRepo,
	}
}

// CreateOrUpdateTradeParameters creates or updates trade parameters for a stock
func (s *TradeParametersService) CreateOrUpdateTradeParameters(ctx context.Context, params *domain.TradeParameters) error {
	// Validate parameters
	if err := s.validateParameters(params); err != nil {
		return err
	}

	// Check if stock exists
	stock, err := s.stockRepo.GetByID(ctx, params.StockID)
	if err != nil {
		return fmt.Errorf("failed to check for existing stock: %w", err)
	}

	if stock == nil {
		return errors.New("stock not found")
	}

	// Check if parameters already exist for this stock
	existingParams, err := s.tradeParamsRepo.GetByStockID(ctx, params.StockID)
	if err != nil {
		return fmt.Errorf("failed to check for existing parameters: %w", err)
	}

	if existingParams != nil {
		// Update existing parameters
		params.ID = existingParams.ID
		params.CreatedAt = existingParams.CreatedAt
		params.UpdatedAt = time.Now()
		return s.tradeParamsRepo.Update(ctx, params)
	}

	// Create new parameters
	return s.tradeParamsRepo.Create(ctx, params)
}

// GetTradeParametersByID retrieves trade parameters by ID
func (s *TradeParametersService) GetTradeParametersByID(ctx context.Context, id string) (*domain.TradeParameters, error) {
	return s.tradeParamsRepo.GetByID(ctx, id)
}

// GetTradeParametersByStockID retrieves trade parameters for a stock
func (s *TradeParametersService) GetTradeParametersByStockID(ctx context.Context, stockID string) (*domain.TradeParameters, error) {
	return s.tradeParamsRepo.GetByStockID(ctx, stockID)
}

// DeleteTradeParameters deletes trade parameters
func (s *TradeParametersService) DeleteTradeParameters(ctx context.Context, id string) error {
	// Check if parameters exist
	existingParams, err := s.tradeParamsRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check for existing parameters: %w", err)
	}

	if existingParams == nil {
		return errors.New("trade parameters not found")
	}

	return s.tradeParamsRepo.Delete(ctx, id)
}

// validateParameters validates trade parameters
func (s *TradeParametersService) validateParameters(params *domain.TradeParameters) error {
	if params.StartingPrice <= 0 {
		return errors.New("starting price must be positive")
	}

	if params.StopLossPercentage < 0 || params.StopLossPercentage > 5 {
		return errors.New("stop loss percentage must be between 0 and 5")
	}

	if params.RiskAmount <= 0 {
		return errors.New("risk amount must be positive")
	}

	if params.TradeSide != domain.Buy && params.TradeSide != domain.Sell {
		return errors.New("trade side must be either BUY or SELL")
	}

	return nil
}
