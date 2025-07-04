package domain

import (
	"time"
)

// IndicatorValue represents a single value of a technical indicator at a specific timestamp
type IndicatorValue struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// TechnicalIndicators encapsulates all technical indicators for a security
type TechnicalIndicators struct {
	InstrumentKey string           `json:"instrument_key"`
	Interval      string           `json:"interval"`
	StartTime     time.Time        `json:"start_time"`
	EndTime       time.Time        `json:"end_time"`
	EMA9          []IndicatorValue `json:"ema_9,omitempty"`
	EMA50         []IndicatorValue `json:"ema_50,omitempty"`
	RSI14         []IndicatorValue `json:"rsi_14,omitempty"`
	ATR14         []IndicatorValue `json:"atr_14,omitempty"`
	VolumeMA10    []IndicatorValue `json:"volume_ma_10,omitempty"`
	BBUpper       []IndicatorValue `json:"bb_upper,omitempty"`
	BBMiddle      []IndicatorValue `json:"bb_middle,omitempty"`
	BBLower       []IndicatorValue `json:"bb_lower,omitempty"`
	BBWidth       []IndicatorValue `json:"bb_width,omitempty"`
	LowestBBWidth []IndicatorValue `json:"lowest_bb_width,omitempty"`
	MorningRange  float64          `json:"morningRange,omitempty"`
}

// StockScreeningCriteria represents the criteria for stock screening
type StockScreeningCriteria struct {
	MinPrice        float64 `json:"minPrice" validate:"required,gt=0"`
	MaxPrice        float64 `json:"maxPrice" validate:"required,gt=0"`
	EMAPricePercent float64 `json:"emaPricePercent" validate:"required"`
	MinVolume       int64   `json:"minVolume" validate:"required,gt=0"`
	MinRSI          float64 `json:"minRsi" validate:"required,gte=0,lte=100"`
	IsBullish       bool    `json:"isBullish"`
}

// ScreeningResult represents the result of stock screening
type ScreeningResult struct {
	Stock          *Stock               `json:"stock"`
	Indicators     *TechnicalIndicators `json:"indicators"`
	ScreeningScore float64              `json:"screeningScore"`
	Rank           int                  `json:"rank"`
	IsMRValid      bool                 `json:"isMrValid"`
	MRValue        float64              `json:"mrValue"`
}

// IntraDayStrategy represents a trading strategy for intraday
type IntraDayStrategy struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Probability float64 `json:"probability"`
	RequiresMR  bool    `json:"requiresMr"`
	MinMRValue  float64 `json:"minMrValue"`
}

// StrategyRecommendation represents a recommendation for a strategy
type StrategyRecommendation struct {
	Stock           *Stock            `json:"stock"`
	Strategy        *IntraDayStrategy `json:"strategy"`
	Probability     float64           `json:"probability"`
	RecommendedMR   float64           `json:"recommendedMr"`
	ConfidenceScore float64           `json:"confidenceScore"`
}

// DailyScenario represents the market scenario for a trading day
type DailyScenario struct {
	Date                time.Time                 `json:"date"`
	OverallTrend        string                    `json:"overallTrend"` // "BULLISH", "BEARISH", "NEUTRAL"
	TopStocks           []*ScreeningResult        `json:"topStocks"`
	TopStrategies       []*StrategyRecommendation `json:"topStrategies"`
	MarketVolatility    string                    `json:"marketVolatility"` // "HIGH", "MEDIUM", "LOW"
	RecommendedStrategy *IntraDayStrategy         `json:"recommendedStrategy"`
}
