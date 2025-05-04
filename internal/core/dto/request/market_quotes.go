package request

type MarketQuotesRequest struct {
	InstrumentKeys []string `json:"instrumentKeys" validate:"required,min=1,dive,required"`
	Interval       string   `json:"interval,omitempty"`
}

// Unit test skeleton (to be implemented in *_test.go file)
// func TestMarketQuotesRequest_Validation(t *testing.T) {
// 	// TODO: Implement validation tests for MarketQuotesRequest
// }
