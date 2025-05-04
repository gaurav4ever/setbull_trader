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

// GetQuotes fetches OHLC data for the given instrumentKeys and interval, using the provided userID for authentication.
// Returns a MarketQuotesResponse DTO.
func (s *MarketQuoteService) GetQuotes(ctx context.Context, userID string, instrumentKeys []string, interval string) *response.MarketQuotesResponse {
	data, errorsMap, err := s.upstoxAuth.GetMarketQuote(ctx, userID, instrumentKeys, interval)
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
	for k, v := range data {
		resp.Data[k] = response.Ohlc{
			Open:  v.Open,
			High:  v.High,
			Low:   v.Low,
			Close: v.Close,
		}
	}
	return resp
}
