package service

import (
	"context"
	"errors"
	"fmt"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
)

// StockService provides operations on stocks
type StockService struct {
	stockRepo repository.StockRepository
}

// NewStockService creates a new StockService
func NewStockService(stockRepo repository.StockRepository) *StockService {
	return &StockService{
		stockRepo: stockRepo,
	}
}

// CreateStock creates a new stock
func (s *StockService) CreateStock(ctx context.Context, stock *domain.Stock) error {
	// Check if stock with the same symbol already exists
	existingStock, err := s.stockRepo.GetBySymbol(ctx, stock.Symbol)
	if err != nil {
		return fmt.Errorf("failed to check for existing stock: %w", err)
	}

	if existingStock != nil {
		return errors.New("stock with the same symbol already exists")
	}

	// Create new stock
	return s.stockRepo.Create(ctx, stock)
}

// GetStockByID retrieves a stock by its ID
func (s *StockService) GetStockByID(ctx context.Context, id string) (*domain.Stock, error) {
	return s.stockRepo.GetByID(ctx, id)
}

// GetStockBySymbol retrieves a stock by its symbol
func (s *StockService) GetStockBySymbol(ctx context.Context, symbol string) (*domain.Stock, error) {
	return s.stockRepo.GetBySymbol(ctx, symbol)
}

// GetAllStocks retrieves all stocks
func (s *StockService) GetAllStocks(ctx context.Context) ([]*domain.Stock, error) {
	return s.stockRepo.GetAll(ctx)
}

// GetSelectedStocks retrieves all selected stocks
func (s *StockService) GetSelectedStocks(ctx context.Context) ([]*domain.Stock, error) {
	return s.stockRepo.GetSelected(ctx)
}

// UpdateStock updates a stock
func (s *StockService) UpdateStock(ctx context.Context, stock *domain.Stock) error {
	// Check if stock exists
	existingStock, err := s.stockRepo.GetByID(ctx, stock.ID)
	if err != nil {
		return fmt.Errorf("failed to check for existing stock: %w", err)
	}

	if existingStock == nil {
		return errors.New("stock not found")
	}

	// Check if updating the symbol would create a duplicate
	if stock.Symbol != existingStock.Symbol {
		stockWithSameSymbol, err := s.stockRepo.GetBySymbol(ctx, stock.Symbol)
		if err != nil {
			return fmt.Errorf("failed to check for duplicate symbol: %w", err)
		}

		if stockWithSameSymbol != nil && stockWithSameSymbol.ID != stock.ID {
			return errors.New("another stock with the same symbol already exists")
		}
	}

	return s.stockRepo.Update(ctx, stock)
}

// ToggleStockSelection toggles the selection status of a stock
func (s *StockService) ToggleStockSelection(ctx context.Context, id string, isSelected bool) error {
	// Check if stock exists
	existingStock, err := s.stockRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check for existing stock: %w", err)
	}

	if existingStock == nil {
		return errors.New("stock not found")
	}

	// If trying to select, check if we've reached the limit of 3 selected stocks
	if isSelected && existingStock.IsSelected != isSelected {
		selectedStocks, err := s.stockRepo.GetSelected(ctx)
		if err != nil {
			return fmt.Errorf("failed to get selected stocks: %w", err)
		}

		if len(selectedStocks) >= 3 {
			return errors.New("maximum of 3 stocks can be selected at a time")
		}
	}

	return s.stockRepo.ToggleSelection(ctx, id, isSelected)
}

// DeleteStock deletes a stock
func (s *StockService) DeleteStock(ctx context.Context, id string) error {
	// Check if stock exists
	existingStock, err := s.stockRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check for existing stock: %w", err)
	}

	if existingStock == nil {
		return errors.New("stock not found")
	}

	return s.stockRepo.Delete(ctx, id)
}
