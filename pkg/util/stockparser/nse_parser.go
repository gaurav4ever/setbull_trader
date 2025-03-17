// New file: pkg/utils/stockparser/nse_parser.go

package stockparser

import (
	"bufio"
	"io"
	"strings"

	"setbull_trader/internal/domain"
)

// NSEStockEntry represents a single entry in the NSE stocks file
type NSEStockEntry struct {
	Symbol     string
	SecurityID string
}

// ParseNSEStockFile parses the NSE stocks file and returns a slice of NSEStockEntry
func ParseNSEStockFile(reader io.Reader) ([]NSEStockEntry, error) {
	scanner := bufio.NewScanner(reader)
	entries := make([]NSEStockEntry, 0)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.TrimSpace(line) == "" {
			continue // Skip empty lines
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			// Format: SYMBOL,SECURITY_ID
			entry := NSEStockEntry{
				Symbol:     strings.TrimSpace(parts[0]),
				SecurityID: strings.TrimSpace(parts[1]),
			}
			entries = append(entries, entry)
		} else {
			// Fallback for old format (just symbol)
			entry := NSEStockEntry{
				Symbol:     strings.TrimSpace(line),
				SecurityID: strings.TrimSpace(line), // Use symbol as security ID
			}
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// ConvertToStocks converts NSEStockEntry slice to domain.Stock slice
func ConvertToStocks(entries []NSEStockEntry) []*domain.Stock {
	stocks := make([]*domain.Stock, 0, len(entries))

	for _, entry := range entries {
		stock := &domain.Stock{
			Symbol:     entry.Symbol,
			Name:       entry.Symbol, // Use symbol as name
			SecurityID: entry.SecurityID,
		}
		stocks = append(stocks, stock)
	}

	return stocks
}

// This function would typically be called from a service that loads the stocks file
func LoadStocksFromReader(reader io.Reader) ([]*domain.Stock, error) {
	entries, err := ParseNSEStockFile(reader)
	if err != nil {
		return nil, err
	}

	return ConvertToStocks(entries), nil
}
