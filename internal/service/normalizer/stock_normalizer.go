package normalizer

import (
	"regexp"
	"setbull_trader/internal/domain"
	"setbull_trader/pkg/log"
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
	defaultPatterns := []*regexp.Regexp{ // Exclude NIFTY indices
		regexp.MustCompile(`^BANKNIFTY`), // Exclude BANKNIFTY indices
		regexp.MustCompile(`^FINNIFTY`),  // Exclude FINNIFTY indices
		regexp.MustCompile(`-BE`),        // Exclude book entry instruments
	}

	log.Info("Stock normalizer initialized with %d default exclusion patterns", len(defaultPatterns))
	for i, pattern := range defaultPatterns {
		log.Info("Exclusion pattern %d: %s", i+1, pattern.String())
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
		log.Error("Failed to add exclusion pattern: %s - Error: %v", pattern, err)
		return err
	}
	n.excludePatterns = append(n.excludePatterns, regex)
	log.Info("Added new exclusion pattern: %s", pattern)
	return nil
}

// NormalizeStocks processes a slice of StockUniverse objects
// It filters out unwanted stocks and normalizes the data of the remaining ones
// Returns:
// - A slice of normalized StockUniverse objects
func (n *StockNormalizer) NormalizeStocks(stocks []domain.StockUniverse) []domain.StockUniverse {
	log.Info("Starting normalization of %d stocks", len(stocks))
	normalized := make([]domain.StockUniverse, 0, len(stocks))
	excludedCount := 0
	includedCount := 0

	// Create counters for different exclusion reasons
	excludeReasons := map[string]int{
		"pattern_match": 0,
		"zero_lot_size": 0,
		"not_eq_or_nse": 0,
	}

	for _, stock := range stocks {
		// Skip if the stock should be excluded
		excluded, reason := n.shouldExcludeWithReason(stock)
		if excluded {
			excludedCount++
			excludeReasons[reason]++

			//log.Info("Excluded stock %d: %s (%s) - Reason: %s", i, stock.Symbol, stock.Name, reason)
			continue
		}

		// Normalize the stock data
		normalizedStock := n.normalizeStock(stock)
		normalized = append(normalized, normalizedStock)
		includedCount++

		//log.Info("Included stock %s: %s (%s) - Exchange: %s, Type: %s", normalizedStock.ExchangeToken, normalizedStock.Symbol, normalizedStock.Name,
		//	normalizedStock.Exchange, normalizedStock.InstrumentType)
	}

	// Log summary statistics
	log.Info("Normalization complete - Total: %d, Included: %d, Excluded: %d",
		len(stocks), includedCount, excludedCount)
	log.Info("Exclusion reasons - Pattern matches: %d, Zero price: %d, Zero lot size: %d, Not EQ/NSE: %d",
		excludeReasons["pattern_match"], excludeReasons["zero_price"],
		excludeReasons["zero_lot_size"], excludeReasons["not_eq_or_nse"])

	return normalized
}

// shouldExcludeWithReason checks if a stock should be excluded and returns the reason
// Returns a boolean indicating exclusion and a string with the reason
func (n *StockNormalizer) shouldExcludeWithReason(stock domain.StockUniverse) (bool, string) {
	// Check against exclude patterns
	for _, pattern := range n.excludePatterns {
		if pattern.MatchString(stock.Symbol) || pattern.MatchString(stock.TradingSymbol) {
			return true, "pattern_match"
		}
	}

	// Exclude stocks with zero lot size
	if stock.LotSize <= 0 {
		return true, "zero_lot_size"
	}

	// Only include equity instruments from NSE
	// if stock.InstrumentType != "EQ" || stock.Exchange != "NSE" {
	// 	return true, "not_eq_or_nse"
	// }

	return false, ""
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
