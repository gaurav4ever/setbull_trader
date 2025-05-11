package response

type StockGroupStockDTO struct {
	StockID       string `json:"stockId"`
	Symbol        string `json:"symbol"`
	InstrumentKey string `json:"instrument_key"`
	ExchangeToken string `json:"exchange_token,omitempty"`
}

type StockGroupResponse struct {
	ID        string               `json:"id"`
	EntryType string               `json:"entryType"`
	Status    string               `json:"status"`
	CreatedAt string               `json:"createdAt"`
	UpdatedAt string               `json:"updatedAt"`
	Stocks    []StockGroupStockDTO `json:"stocks"`
}
