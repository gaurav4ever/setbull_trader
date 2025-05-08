package service

import (
	"context"
	"time"

	"setbull_trader/internal/core/adapters/client/upstox"
	"setbull_trader/internal/core/dto/response"
)

// MarketQuoteService handles business logic for fetching market quotes
// and formatting the response for the /market/quotes API.
type MarketQuoteService struct {
	upstoxAuth *upstox.AuthService
}

// NewMarketQuoteService creates a new MarketQuoteService
func NewMarketQuoteService(upstoxAuth *upstox.AuthService) *MarketQuoteService {
	return &MarketQuoteService{upstoxAuth: upstoxAuth}
}

// GetQuotes fetches OHLC data for the given keys and interval, using the provided userID for authentication.
// keyType specifies the type of keys provided: "symbol" or "instrument_key".
// Returns a MarketQuotesResponse DTO.
func (s *MarketQuoteService) GetQuotes(ctx context.Context, userID string, keys []string, interval string, keyType string, stockUniverseSvc *StockUniverseService) *response.MarketQuotesResponse {
	oldKeys := make([]string, 0, len(keys))
	resolvedKeys := make([]string, 0, len(keys))
	keyMap := make(map[string]string) // input key -> instrument_key
	errorsMap := make(map[string]string)

	// Resolve keys to instrument_key
	switch keyType {
	case "instrument_key":
		for _, k := range keys {
			oldKeys = append(oldKeys, k)
			stock, err := stockUniverseSvc.GetStocksByInstrumentKeys(ctx, []string{k})
			new_key := stock[0].Exchange + "_" + stock[0].InstrumentType + ":" + stock[0].TradingSymbol
			if err != nil || len(stock) == 0 {
				errorsMap[k] = "Instrument key not found in stock universe"
				continue
			}
			resolvedKeys = append(resolvedKeys, new_key)
			keyMap[k] = new_key
		}
	case "symbol":
		for _, symbol := range keys {
			oldKeys = append(oldKeys, symbol)
			stock, err := stockUniverseSvc.GetStockBySymbol(ctx, symbol)
			new_key := stock.Exchange + "_" + stock.InstrumentType + ":" + stock.TradingSymbol
			if err != nil || stock == nil {
				errorsMap[symbol] = "Symbol not found in stock universe"
				continue
			}
			resolvedKeys = append(resolvedKeys, new_key)
			keyMap[symbol] = new_key
		}
	default:
		for _, k := range keys {
			errorsMap[k] = "Unknown keyType; must be 'symbol' or 'instrument_key'"
		}
	}

	// If no valid instrument_keys, return error response
	if len(resolvedKeys) == 0 {
		return &response.MarketQuotesResponse{
			Status:    "error",
			Timestamp: time.Now().In(time.FixedZone("IST", 5*3600+1800)).Format(time.RFC3339),
			Data:      make(map[string]response.Ohlc),
			Errors:    errorsMap,
		}
	}

	// Call Upstox API with resolved instrument_keys
	data, upstoxErrors, err := s.upstoxAuth.GetMarketQuote(ctx, userID, oldKeys, resolvedKeys, interval)
	resp := &response.MarketQuotesResponse{
		Status:    "success",
		Timestamp: time.Now().In(time.FixedZone("IST", 5*3600+1800)).Format(time.RFC3339),
		Data:      make(map[string]response.Ohlc),
		Errors:    errorsMap,
	}
	if err != nil {
		resp.Status = "error"
		if resp.Errors == nil {
			resp.Errors = make(map[string]string)
		}
		resp.Errors["_fatal"] = err.Error()
	}

	// Map Upstox data back to input keys
	for inputKey, instKey := range keyMap {
		if ohlc, ok := data[instKey]; ok {
			resp.Data[inputKey] = response.Ohlc{
				Open:  ohlc.Open,
				High:  ohlc.High,
				Low:   ohlc.Low,
				Close: ohlc.Close,
			}
		} else if upstoxErrors != nil {
			if errMsg, exists := upstoxErrors[instKey]; exists {
				resp.Errors[inputKey] = errMsg
			}
		}
	}

	return resp
}
