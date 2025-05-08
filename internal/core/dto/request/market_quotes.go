package request

type MarketQuotesRequest struct {
	InstrumentKeys []string `json:"instrumentKeys,omitempty" validate:"omitempty,min=1,dive,required"`
	Symbols        []string `json:"symbols,omitempty" validate:"omitempty,min=1,dive,required"`
	KeyType        string   `json:"keyType,omitempty" validate:"omitempty,oneof=symbol instrument_key"`
	Interval       string   `json:"interval,omitempty"`
}

// Unit test skeleton (to be implemented in *_test.go file)
// func TestMarketQuotesRequest_Validation(t *testing.T) {
// 	// TODO: Implement validation tests for MarketQuotesRequest
// }
