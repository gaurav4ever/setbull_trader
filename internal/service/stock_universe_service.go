package service

import (
	"context"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/internal/service/normalizer"
	"setbull_trader/internal/service/parser"
	"setbull_trader/pkg/log"
)

// StockUniverseService handles business logic for the stock universe
type StockUniverseService struct {
	repo       repository.StockUniverseRepository
	parser     *parser.UpstoxParser
	normalizer *normalizer.StockNormalizer
	filePath   string // Default file path for Upstox data
}

// NewStockUniverseService creates a new instance of StockUniverseService
func NewStockUniverseService(
	repo repository.StockUniverseRepository,
	parser *parser.UpstoxParser,
	normalizer *normalizer.StockNormalizer,
	filePath string,
) *StockUniverseService {
	return &StockUniverseService{
		repo:       repo,
		parser:     parser,
		normalizer: normalizer,
		filePath:   filePath,
	}
}

// IngestStocksFromFile reads stocks from the Upstox JSON file, normalizes them,
// and stores them in the database
// Parameters:
// - ctx: Context for the operation
// - customFilePath: Optional custom file path (if empty, default path is used)
// Returns:
// - Number of stocks created
// - Number of stocks updated
// - Error if any occurred
func (s *StockUniverseService) IngestStocksFromFile(ctx context.Context, customFilePath string) (int, int, error) {
	// Use custom file path if provided, otherwise use default
	filePath := s.filePath
	if customFilePath != "" {
		filePath = customFilePath
	}

	// Set the file path in the parser
	s.parser.SetFilePath(filePath)

	// Parse the file
	log.Info("Starting stock ingestion from file: %s", filePath)
	stocks, err := s.parser.ParseFile()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse stock file: %w", err)
	}
	log.Info("Parsed %d stocks from file", len(stocks))

	// Normalize the stocks
	normalizedStocks := s.normalizer.NormalizeStocks(stocks)
	log.Info("Normalized to %d stocks after filtering", len(normalizedStocks))

	// Store in the database
	created, updated, err := s.repo.BulkUpsert(ctx, normalizedStocks)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to store stocks: %w", err)
	}

	log.Info("Stock ingestion completed. Created: %d, Updated: %d", created, updated)
	return created, updated, nil
}

// GetAllStocks retrieves all stocks with optional filtering
func (s *StockUniverseService) GetAllStocks(
	ctx context.Context,
	onlySelected bool,
	page, pageSize int,
) ([]domain.StockUniverse, int64, error) {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50 // Default page size
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get stocks from repository
	return s.repo.GetAll(ctx, onlySelected, pageSize, offset)
}

// GetStockBySymbol retrieves a stock by its symbol
func (s *StockUniverseService) GetStockBySymbol(ctx context.Context, symbol string) (*domain.StockUniverse, error) {
	return s.repo.GetBySymbol(ctx, symbol)
}

// ToggleStockSelection toggles the is_selected flag for a stock
func (s *StockUniverseService) ToggleStockSelection(ctx context.Context, symbol string, isSelected bool) (*domain.StockUniverse, error) {
	return s.repo.ToggleSelection(ctx, symbol, isSelected)
}

// DeleteStock deletes a stock by its symbol
func (s *StockUniverseService) DeleteStock(ctx context.Context, symbol string) error {
	return s.repo.DeleteBySymbol(ctx, symbol)
}

// GetStocksByInstrumentKeys retrieves stocks from the universe by their instrument keys
func (s *StockUniverseService) GetStocksByInstrumentKeys(ctx context.Context, instrumentKeys []string) ([]domain.StockUniverse, error) {
	if len(instrumentKeys) == 0 {
		return []domain.StockUniverse{}, nil
	}

	// Call the repository to get stocks by instrument keys
	stocks, err := s.repo.GetStocksByInstrumentKeys(ctx, instrumentKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to get stocks by instrument keys: %w", err)
	}

	return stocks, nil
}
