package normalizer

import (
	"regexp"
	"setbull_trader/internal/domain"
	"strings"
)

// StockNormalizer handles normalization of stock data
// This ensures consistency in the data before it's stored in the database
type StockNormalizer struct {
	// Configuration options for normalization
	excludePatterns []*regexp.Regexp
}

// NewStockNormalizer creates a new instance of StockNormalizer
// It sets up default exclusion patterns for stocks we don't want to include
func NewStockNormalizer() *StockNormalizer {
	// Create default patterns for excluding certain stocks
	// For example, we might want to exclude indices, bonds, or certain types of derivatives
	defaultPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^NIFTY`),     // Exclude NIFTY indices
		regexp.MustCompile(`^BANKNIFTY`), // Exclude BANKNIFTY indices
		regexp.MustCompile(`^FINNIFTY`),  // Exclude FINNIFTY indices
		regexp.MustCompile(`-BE$`),       // Exclude book entry instruments
	}

	return &StockNormalizer{
		excludePatterns: defaultPatterns,
	}
}

// AddExcludePattern adds a new pattern to exclude stocks
// pattern: Regular expression pattern as a string
func (n *StockNormalizer) AddExcludePattern(pattern string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	n.excludePatterns = append(n.excludePatterns, regex)
	return nil
}

// NormalizeStocks processes a slice of StockUniverse objects
// It filters out unwanted stocks and normalizes the data of the remaining ones
// Returns:
// - A slice of normalized StockUniverse objects
func (n *StockNormalizer) NormalizeStocks(stocks []domain.StockUniverse) []domain.StockUniverse {
	normalized := make([]domain.StockUniverse, 0, len(stocks))

	for _, stock := range stocks {
		// Skip if the stock should be excluded
		if n.shouldExclude(stock) {
			continue
		}

		// Normalize the stock data
		normalizedStock := n.normalizeStock(stock)
		normalized = append(normalized, normalizedStock)
	}

	return normalized
}

// shouldExclude checks if a stock should be excluded based on the patterns
// Returns true if the stock should be excluded
func (n *StockNormalizer) shouldExclude(stock domain.StockUniverse) bool {
	// Check against exclude patterns
	for _, pattern := range n.excludePatterns {
		if pattern.MatchString(stock.Symbol) || pattern.MatchString(stock.TradingSymbol) {
			return true
		}
	}

	// Additional exclusion logic
	// Exclude stocks with zero or negative prices
	if stock.LastPrice <= 0 {
		return true
	}

	// Exclude stocks with zero lot size
	if stock.LotSize <= 0 {
		return true
	}

	// Only include equity instruments from NSE
	if stock.InstrumentType != "EQ" || stock.Exchange != "NSE" {
		return true
	}

	return false
}

// normalizeStock normalizes a single StockUniverse object
// This ensures data consistency across all stocks
func (n *StockNormalizer) normalizeStock(stock domain.StockUniverse) domain.StockUniverse {
	// Create a copy to avoid modifying the original
	normalized := stock

	// Normalize the symbol (trim spaces, convert to uppercase)
	normalized.Symbol = strings.TrimSpace(strings.ToUpper(normalized.Symbol))

	// Normalize the name (trim spaces)
	normalized.Name = strings.TrimSpace(normalized.Name)

	// Normalize the exchange (uppercase)
	normalized.Exchange = strings.ToUpper(normalized.Exchange)

	// Normalize the instrument type (uppercase)
	normalized.InstrumentType = strings.ToUpper(normalized.InstrumentType)

	// Ensure ISIN is uppercase
	if normalized.ISIN != "" {
		normalized.ISIN = strings.ToUpper(normalized.ISIN)
	}

	return normalized
}
