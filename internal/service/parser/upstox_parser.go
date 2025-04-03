package parser

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"setbull_trader/internal/domain"
	"strings"
)

// UpstoxStockData represents the structure of a stock entry in the nse_upstox.json file
type UpstoxStockData struct {
	InstrumentKey  string  `json:"instrument_key"`
	ExchangeToken  string  `json:"exchange_token"`
	TradingSymbol  string  `json:"trading_symbol"`
	Name           string  `json:"name"`
	LastPrice      float64 `json:"last_price"`
	TickSize       float64 `json:"tick_size"`
	LotSize        int     `json:"lot_size"`
	InstrumentType string  `json:"instrument_type"`
	Exchange       string  `json:"exchange"`
}

// UpstoxParser handles parsing of the nse_upstox.json file
type UpstoxParser struct {
	filePath string
}

// NewUpstoxParser creates a new instance of UpstoxParser
// filePath: Path to the nse_upstox.json file
func NewUpstoxParser(filePath string) *UpstoxParser {
	return &UpstoxParser{
		filePath: filePath,
	}
}

// ParseFile reads and parses the nse_upstox.json file into a slice of StockUniverse objects
// Returns:
// - A slice of StockUniverse objects
// - Error if any occurred during parsing
func (p *UpstoxParser) ParseFile() ([]domain.StockUniverse, error) {
	// Open the file
	file, err := os.Open(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", p.filePath, err)
	}
	defer file.Close()

	// Read the file content
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", p.filePath, err)
	}

	// Parse the JSON content
	var upstoxStocks []UpstoxStockData
	if err := json.Unmarshal(content, &upstoxStocks); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to StockUniverse objects
	stocks := make([]domain.StockUniverse, 0, len(upstoxStocks))
	for _, upstoxStock := range upstoxStocks {
		// Extract the ISIN from the instrument key if available
		// The format is typically: NSE_EQ_INE123456789
		var isin string
		parts := strings.Split(upstoxStock.InstrumentKey, "_")
		if len(parts) >= 3 && strings.HasPrefix(parts[2], "IN") {
			isin = parts[2]
		}

		// Create a StockUniverse object
		stock := domain.StockUniverse{
			Symbol:         extractSymbol(upstoxStock.TradingSymbol),
			Name:           upstoxStock.Name,
			Exchange:       upstoxStock.Exchange,
			InstrumentType: upstoxStock.InstrumentType,
			ISIN:           isin,
			InstrumentKey:  upstoxStock.InstrumentKey,
			TradingSymbol:  upstoxStock.TradingSymbol,
			ExchangeToken:  upstoxStock.ExchangeToken,
			LastPrice:      upstoxStock.LastPrice,
			TickSize:       upstoxStock.TickSize,
			LotSize:        upstoxStock.LotSize,
			IsSelected:     false, // Default to not selected
			Metadata:       createMetadata(upstoxStock),
		}

		stocks = append(stocks, stock)
	}

	return stocks, nil
}

// extractSymbol extracts the clean symbol from the trading symbol
// For example: "RELIANCE-EQ" becomes "RELIANCE"
func extractSymbol(tradingSymbol string) string {
	// Split by hyphen and take the first part
	parts := strings.Split(tradingSymbol, "-")
	return parts[0]
}

// createMetadata creates a JSON metadata field with additional information
// This allows storing extra data that might be useful but doesn't fit in the main columns
func createMetadata(stock UpstoxStockData) domain.JSON {
	// Create a map with any additional data we want to store
	metadata := map[string]interface{}{
		"original_data": stock,
		// Add any other metadata fields as needed
	}

	// Convert to JSON
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		// If there's an error, return empty JSON
		return domain.JSON("{}")
	}

	return domain.JSON(jsonData)
}
