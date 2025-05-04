package upstox

import (
	"testing"
)

func TestAuthService_GetMarketQuote(t *testing.T) {
	t.Run("all-success", func(t *testing.T) {
		// TODO: Mock Upstox API to return valid OHLC for all instrumentKeys
		// TODO: Call GetMarketQuote and check all keys in data, errors map is empty
	})

	t.Run("partial-failure", func(t *testing.T) {
		// TODO: Mock Upstox API to return valid OHLC for some keys, error for others
		// TODO: Call GetMarketQuote and check data and errors maps
	})

	t.Run("all-failure", func(t *testing.T) {
		// TODO: Mock Upstox API to return error for all instrumentKeys
		// TODO: Call GetMarketQuote and check data is empty, errors map has all keys
	})
}
