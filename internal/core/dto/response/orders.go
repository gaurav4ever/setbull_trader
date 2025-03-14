package response

// OrderResponse represents a simplified response after placing/modifying an order
type OrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderStatus string `json:"orderStatus"`
	Success     bool   `json:"success"`
	Message     string `json:"message,omitempty"`
}

// TradeResponse represents a simplified trade response
type TradeResponse struct {
	OrderID         string  `json:"orderId"`
	ExchangeOrderID string  `json:"exchangeOrderId"`
	ExchangeTradeID string  `json:"exchangeTradeId"`
	TransactionType string  `json:"transactionType"`
	ExchangeSegment string  `json:"exchangeSegment"`
	ProductType     string  `json:"productType"`
	OrderType       string  `json:"orderType"`
	Symbol          string  `json:"symbol"`
	SecurityID      string  `json:"securityId"`
	Quantity        int     `json:"quantity"`
	Price           float64 `json:"price"`
	Timestamp       string  `json:"timestamp"`
	ExpiryDate      string  `json:"expiryDate,omitempty"`
	OptionType      string  `json:"optionType,omitempty"`
	StrikePrice     float64 `json:"strikePrice,omitempty"`
}

// TradeHistoryResponse represents a more detailed trade response
type TradeHistoryResponse struct {
	OrderID         string  `json:"orderId"`
	ExchangeOrderID string  `json:"exchangeOrderId"`
	ExchangeTradeID string  `json:"exchangeTradeId"`
	TransactionType string  `json:"transactionType"`
	ExchangeSegment string  `json:"exchangeSegment"`
	ProductType     string  `json:"productType"`
	OrderType       string  `json:"orderType"`
	Symbol          string  `json:"symbol"`
	SecurityID      string  `json:"securityId"`
	Quantity        int     `json:"quantity"`
	Price           float64 `json:"price"`
	ISIN            string  `json:"isin"`
	Instrument      string  `json:"instrument"`
	BrokerageFees   float64 `json:"brokerageFees"`
	TaxesFees       float64 `json:"taxesFees"`
	Timestamp       string  `json:"timestamp"`
	ExpiryDate      string  `json:"expiryDate,omitempty"`
	OptionType      string  `json:"optionType,omitempty"`
	StrikePrice     float64 `json:"strikePrice,omitempty"`
}

// TradesListResponse represents a list of trades response
type TradesListResponse struct {
	Trades []TradeResponse `json:"trades"`
	Count  int             `json:"count"`
}

// TradeHistoryListResponse represents a list of trade history response
type TradeHistoryListResponse struct {
	Trades []TradeHistoryResponse `json:"trades"`
	Count  int                    `json:"count"`
}

// GenericResponse represents a generic API response
type GenericResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}
