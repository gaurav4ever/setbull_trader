package response

type Ohlc struct {
	Open    float64 `json:"open,omitempty"`
	High    float64 `json:"high,omitempty"`
	Low     float64 `json:"low,omitempty"`
	Close   float64 `json:"close,omitempty"`
	BBWidth float64 `json:"bb_width,omitempty"`
}

type MarketQuotesResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Data      map[string]Ohlc   `json:"data"`
	Errors    map[string]string `json:"errors,omitempty"`
}

// Unit test skeleton (to be implemented in *_test.go file)
// func TestMarketQuotesResponse_Serialization(t *testing.T) {
// 	// TODO: Implement serialization tests for MarketQuotesResponse
// }
