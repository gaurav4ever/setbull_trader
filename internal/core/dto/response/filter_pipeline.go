package response

import (
	"setbull_trader/internal/domain"
	"time"
)

type FilterPipelineResponse struct {
	Status string             `json:"status"`
	Data   FilterPipelineData `json:"data"`
}

type FilterPipelineData struct {
	BullishStocks []domain.FilteredStock `json:"bullish_stocks"`
	BearishStocks []domain.FilteredStock `json:"bearish_stocks"`
	Metrics       PipelineMetrics        `json:"metrics"`
}

type PipelineMetrics struct {
	TotalStocks     int           `json:"total_stocks"`
	BasicFilterPass int           `json:"basic_filter_pass"`
	EMAFilterPass   int           `json:"ema_filter_pass"`
	RSIFilterPass   int           `json:"rsi_filter_pass"`
	BullishStocks   int           `json:"bullish_stocks"`
	BearishStocks   int           `json:"bearish_stocks"`
	ProcessingTime  time.Duration `json:"processing_time_ms"`
}
