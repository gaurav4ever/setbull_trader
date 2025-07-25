package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// StockUniverse represents a stock in our universe of tradable instruments
// This is the main entity that stores information about stocks from NSE via Upstox
type StockUniverse struct {
	ID             int64   `gorm:"primaryKey;column:id" json:"id"`
	Symbol         string  `gorm:"uniqueIndex;column:symbol;size:30" json:"symbol"`      // Trading symbol (e.g., RELIANCE, INFY)
	Name           string  `gorm:"column:name;size:100" json:"name"`                     // Full company name
	Exchange       string  `gorm:"column:exchange;size:10" json:"exchange"`              // Exchange code (e.g., NSE, BSE)
	InstrumentType string  `gorm:"column:instrument_type;size:20" json:"instrumentType"` // Type of instrument (e.g., EQ, FUT, OPT)
	ISIN           string  `gorm:"column:isin;size:20" json:"isin"`                      // International Securities Identification Number
	InstrumentKey  string  `gorm:"column:instrument_key;size:50" json:"instrumentKey"`   // Unique key used by Upstox to identify the instrument
	TradingSymbol  string  `gorm:"column:trading_symbol;size:50" json:"tradingSymbol"`   // Symbol used for trading on the exchange
	ExchangeToken  string  `gorm:"column:exchange_token;size:20" json:"exchangeToken"`   // Token assigned by the exchange
	LastPrice      float64 `gorm:"column:last_price" json:"lastPrice"`                   // Last traded price
	TickSize       float64 `gorm:"column:tick_size" json:"tickSize"`                     // Minimum price movement
	LotSize        int     `gorm:"column:lot_size" json:"lotSize"`                       // Standard lot size for trading
	SecurityID     string  `gorm:"column:security_id;size:20" json:"securityId"`         // Numeric Security ID for Dhan API
	IsSelected     bool    `gorm:"column:is_selected;default:false" json:"isSelected"`   // Whether this stock is selected for trading
	//Metadata       JSON      `gorm:"column:metadata;type:json" json:"metadata"`            // Additional metadata in JSON format
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"` // When the record was created
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"` // When the record was last updated
}

// TableName specifies the database table name for the StockUniverse model
func (StockUniverse) TableName() string {
	return "stock_universe"
}

// JSON is a custom type to handle JSON data in the database
// This allows storing flexible metadata about stocks
type JSON json.RawMessage

// Value converts the JSON to a value that can be stored in the database
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

// Scan reads the JSON value from the database
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = JSON("null")
		return nil
	}

	// Convert the database value to JSON
	switch v := value.(type) {
	case []byte:
		*j = JSON(v)
		return nil
	case string:
		*j = JSON(v)
		return nil
	default:
		return errors.New("unsupported type for JSON scanning")
	}
}

// MarshalJSON returns the JSON encoding
func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON sets the JSON value
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSON: UnmarshalJSON on nil pointer")
	}
	*j = JSON(data)
	return nil
}
