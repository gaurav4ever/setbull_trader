package dhan

// PlaceOrderRequest represents a request to place an order
type PlaceOrderRequest struct {
	DhanClientID      string  `json:"dhanClientId"`
	CorrelationID     string  `json:"correlationId,omitempty"`
	TransactionType   string  `json:"transactionType"` // BUY, SELL
	ExchangeSegment   string  `json:"exchangeSegment"` // NSE_EQ, NSE_FNO, BSE_EQ, BSE_FNO, MCX_COMM
	ProductType       string  `json:"productType"`     // CNC, INTRADAY, MARGIN, MTF, CO, BO
	OrderType         string  `json:"orderType"`       // LIMIT, MARKET, STOP_LOSS, STOP_LOSS_MARKET
	Validity          string  `json:"validity"`        // DAY, IOC
	SecurityID        string  `json:"securityId"`
	Quantity          int     `json:"quantity"`
	DisclosedQuantity int     `json:"disclosedQuantity,omitempty"`
	Price             float64 `json:"price"`
	TriggerPrice      float64 `json:"triggerPrice,omitempty"`
	AfterMarketOrder  bool    `json:"afterMarketOrder,omitempty"`
	AMOTime           string  `json:"amoTime,omitempty"` // OPEN, OPEN_30, OPEN_60, PRE_OPEN
	BOProfitValue     float64 `json:"boProfitValue,omitempty"`
	BOStopLossValue   float64 `json:"boStopLossValue,omitempty"`
}

// ModifyOrderRequest represents a request to modify an order
type ModifyOrderRequest struct {
	DhanClientID      string  `json:"dhanClientId"`
	OrderID           string  `json:"orderId"`
	OrderType         string  `json:"orderType"`         // LIMIT, MARKET, STOP_LOSS, STOP_LOSS_MARKET
	LegName           string  `json:"legName,omitempty"` // ENTRY_LEG, STOP_LOSS_LEG, TARGET_LEG, NA
	Quantity          int     `json:"quantity"`
	Price             float64 `json:"price"`
	DisclosedQuantity int     `json:"disclosedQuantity,omitempty"`
	TriggerPrice      float64 `json:"triggerPrice,omitempty"`
	Validity          string  `json:"validity"` // DAY, IOC
}

// OrderResponse represents a response after placing/modifying an order
type OrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderStatus string `json:"orderStatus"`
}

// TradeResponse represents a trade in the day's trades list
type TradeResponse struct {
	DhanClientID    string  `json:"dhanClientId"`
	OrderID         string  `json:"orderId"`
	ExchangeOrderID string  `json:"exchangeOrderId"`
	ExchangeTradeID string  `json:"exchangeTradeId"`
	TransactionType string  `json:"transactionType"`
	ExchangeSegment string  `json:"exchangeSegment"`
	ProductType     string  `json:"productType"`
	OrderType       string  `json:"orderType"`
	TradingSymbol   string  `json:"tradingSymbol"`
	CustomSymbol    string  `json:"customSymbol"`
	SecurityID      string  `json:"securityId"`
	TradedQuantity  int     `json:"tradedQuantity"`
	TradedPrice     float64 `json:"tradedPrice"`
	CreateTime      string  `json:"createTime"`
	UpdateTime      string  `json:"updateTime"`
	ExchangeTime    string  `json:"exchangeTime"`
	DrvExpiryDate   string  `json:"drvExpiryDate,omitempty"`
	DrvOptionType   string  `json:"drvOptionType,omitempty"`
	DrvStrikePrice  float64 `json:"drvStrikePrice,omitempty"`
}

// TradeHistoryResponse represents a trade in the trade history response
type TradeHistoryResponse struct {
	DhanClientID               string  `json:"dhanClientId"`
	OrderID                    string  `json:"orderId"`
	ExchangeOrderID            string  `json:"exchangeOrderId"`
	ExchangeTradeID            string  `json:"exchangeTradeId"`
	TransactionType            string  `json:"transactionType"`
	ExchangeSegment            string  `json:"exchangeSegment"`
	ProductType                string  `json:"productType"`
	OrderType                  string  `json:"orderType"`
	CustomSymbol               string  `json:"customSymbol"`
	SecurityID                 string  `json:"securityId"`
	TradedQuantity             int     `json:"tradedQuantity"`
	TradedPrice                float64 `json:"tradedPrice"`
	ISIN                       string  `json:"isin"`
	Instrument                 string  `json:"instrument"`
	SEBITax                    float64 `json:"sebiTax"`
	STT                        float64 `json:"stt"`
	BrokerageCharges           float64 `json:"brokerageCharges"`
	ServiceTax                 float64 `json:"serviceTax"`
	ExchangeTransactionCharges float64 `json:"exchangeTransactionCharges"`
	StampDuty                  float64 `json:"stampDuty"`
	CreateTime                 string  `json:"createTime"`
	UpdateTime                 string  `json:"updateTime"`
	ExchangeTime               string  `json:"exchangeTime"`
	DrvExpiryDate              string  `json:"drvExpiryDate,omitempty"`
	DrvOptionType              string  `json:"drvOptionType,omitempty"`
	DrvStrikePrice             float64 `json:"drvStrikePrice,omitempty"`
}

// Constants for TransactionType
const (
	TransactionTypeBuy  = "BUY"
	TransactionTypeSell = "SELL"
)

// Constants for ExchangeSegment
const (
	ExchangeSegmentNSEEQ   = "NSE_EQ"
	ExchangeSegmentNSEFNO  = "NSE_FNO"
	ExchangeSegmentBSEEQ   = "BSE_EQ"
	ExchangeSegmentBSEFNO  = "BSE_FNO"
	ExchangeSegmentMCXCOMM = "MCX_COMM"
)

// Constants for ProductType
const (
	ProductTypeCNC      = "CNC"
	ProductTypeIntraday = "INTRADAY"
	ProductTypeMargin   = "MARGIN"
	ProductTypeMTF      = "MTF"
	ProductTypeCO       = "CO"
	ProductTypeBO       = "BO"
)

// Constants for OrderType
const (
	OrderTypeLimit          = "LIMIT"
	OrderTypeMarket         = "MARKET"
	OrderTypeStopLoss       = "STOP_LOSS"
	OrderTypeStopLossMarket = "STOP_LOSS_MARKET"
)

// Constants for Validity
const (
	ValidityDay = "DAY"
	ValidityIOC = "IOC"
)

// Constants for AMOTime
const (
	AMOTimeOpen    = "OPEN"
	AMOTimeOpen30  = "OPEN_30"
	AMOTimeOpen60  = "OPEN_60"
	AMOTimePreOpen = "PRE_OPEN"
)

// Constants for LegName
const (
	LegNameEntryLeg    = "ENTRY_LEG"
	LegNameStopLossLeg = "STOP_LOSS_LEG"
	LegNameTargetLeg   = "TARGET_LEG"
	LegNameNA          = "NA"
)

// Constants for DrvOptionType
const (
	OptionTypeCall = "CALL"
	OptionTypePut  = "PUT"
	OptionTypeNA   = "NA"
)

// Constants for OrderStatus
const (
	OrderStatusTransit    = "TRANSIT"
	OrderStatusPending    = "PENDING"
	OrderStatusRejected   = "REJECTED"
	OrderStatusCancelled  = "CANCELLED"
	OrderStatusPartTraded = "PART_TRADED"
	OrderStatusTraded     = "TRADED"
	OrderStatusExpired    = "EXPIRED"
	OrderStatusModified   = "MODIFIED"
	OrderStatusTriggered  = "TRIGGERED"
)
