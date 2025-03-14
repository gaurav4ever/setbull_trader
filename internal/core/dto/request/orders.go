package request

// PlaceOrderRequest represents a simplified request to place an order
type PlaceOrderRequest struct {
	TransactionType string  `json:"transactionType" validate:"required,oneof=BUY SELL"`
	ExchangeSegment string  `json:"exchangeSegment" validate:"required,oneof=NSE_EQ NSE_FNO BSE_EQ BSE_FNO MCX_COMM"`
	ProductType     string  `json:"productType" validate:"required,oneof=CNC INTRADAY MARGIN MTF CO BO"`
	OrderType       string  `json:"orderType" validate:"required,oneof=LIMIT MARKET STOP_LOSS STOP_LOSS_MARKET"`
	SecurityID      string  `json:"securityId" validate:"required"`
	Quantity        int     `json:"quantity" validate:"required,gt=0"`
	DisclosedQty    int     `json:"disclosedQty,omitempty"`
	Price           float64 `json:"price"`
	TriggerPrice    float64 `json:"triggerPrice,omitempty"`
	Validity        string  `json:"validity" validate:"required,oneof=DAY IOC"`
	IsAMO           bool    `json:"isAMO,omitempty"`
	TargetPrice     float64 `json:"targetPrice,omitempty"`   // For Bracket Order - used as BOProfitValue
	StopLossPrice   float64 `json:"stopLossPrice,omitempty"` // For Bracket/Cover Order - used as BOStopLossValue
}

// ModifyOrderRequest represents a simplified request to modify an order
type ModifyOrderRequest struct {
	OrderType    string  `json:"orderType,omitempty" validate:"omitempty,oneof=LIMIT MARKET STOP_LOSS STOP_LOSS_MARKET"`
	Quantity     int     `json:"quantity,omitempty" validate:"omitempty,gt=0"`
	Price        float64 `json:"price,omitempty"`
	DisclosedQty int     `json:"disclosedQty,omitempty"`
	TriggerPrice float64 `json:"triggerPrice,omitempty"`
	Validity     string  `json:"validity,omitempty" validate:"omitempty,oneof=DAY IOC"`
}

// TradeHistoryRequest represents a request to get trade history
type TradeHistoryRequest struct {
	FromDate   string `json:"fromDate" validate:"required,datetime=2006-01-02"`
	ToDate     string `json:"toDate" validate:"required,datetime=2006-01-02"`
	PageNumber int    `json:"pageNumber" validate:"gte=0"`
}
