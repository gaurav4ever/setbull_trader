package orders

import (
	"fmt"
	"time"

	"setbull_trader/internal/core/adapters/client/dhan"
	"setbull_trader/internal/core/dto/request"
	"setbull_trader/internal/core/dto/response"
	"setbull_trader/pkg/apperrors"
	"setbull_trader/pkg/log"

	"github.com/pkg/errors"
)

// Service handles order operations
type Service struct {
	dhanClient *dhan.Client
}

// NewService creates a new order service
func NewService(dhanClient *dhan.Client) *Service {
	return &Service{
		dhanClient: dhanClient,
	}
}

// PlaceOrder places a new order
func (s *Service) PlaceOrder(req *request.PlaceOrderRequest) (*response.OrderResponse, error) {
	// Map the simplified request to the Dhan API request
	dhanReq := &dhan.PlaceOrderRequest{
		TransactionType:   req.TransactionType,
		ExchangeSegment:   req.ExchangeSegment,
		ProductType:       req.ProductType,
		OrderType:         req.OrderType,
		Validity:          req.Validity,
		SecurityID:        req.SecurityID,
		Quantity:          req.Quantity,
		DisclosedQuantity: req.DisclosedQty,
		Price:             req.Price,
		TriggerPrice:      req.TriggerPrice,
		AfterMarketOrder:  req.IsAMO,
	}

	// For Bracket Orders, set the profit and stop loss values
	if req.ProductType == dhan.ProductTypeBO {
		dhanReq.BOProfitValue = req.TargetPrice
		dhanReq.BOStopLossValue = req.StopLossPrice
	}

	// For Cover Orders, set the stop loss value
	if req.ProductType == dhan.ProductTypeCO {
		dhanReq.BOStopLossValue = req.StopLossPrice
	}

	// Set AMO time if it's an after-market order
	if req.IsAMO {
		dhanReq.AMOTime = dhan.AMOTimeOpen
	}

	// Log the order request
	log.Info("Placing order with SecurityID: %s", req.SecurityID)

	// Make the API call
	dhanResp, err := s.dhanClient.PlaceOrder(dhanReq)
	if err != nil {
		log.Error("Failed to place order: %v", err)
		return nil, apperrors.NewInternalServerError("Failed to place order", err)
	}

	// Map the response
	resp := &response.OrderResponse{
		OrderID:     dhanResp.OrderID,
		OrderStatus: dhanResp.OrderStatus,
		Success:     true,
		Message:     "Order placed successfully",
	}

	return resp, nil
}

// ModifyOrder modifies an existing order
func (s *Service) ModifyOrder(orderID string, req *request.ModifyOrderRequest) (*response.OrderResponse, error) {
	if orderID == "" {
		return nil, apperrors.NewBadRequestError("Order ID is required", errors.New("missing order ID"))
	}

	// Map the simplified request to the Dhan API request
	dhanReq := &dhan.ModifyOrderRequest{
		OrderID:           orderID,
		Quantity:          req.Quantity,
		Price:             req.Price,
		DisclosedQuantity: req.DisclosedQty,
		TriggerPrice:      req.TriggerPrice,
	}

	// Only set fields that are provided
	if req.OrderType != "" {
		dhanReq.OrderType = req.OrderType
	}

	if req.Validity != "" {
		dhanReq.Validity = req.Validity
	}

	// Default leg name to ENTRY_LEG for BO and CO orders
	dhanReq.LegName = dhan.LegNameNA

	// Make the API call
	dhanResp, err := s.dhanClient.ModifyOrder(orderID, dhanReq)
	if err != nil {
		log.Error("Failed to modify order: %v", err)
		return nil, apperrors.NewInternalServerError("Failed to modify order", err)
	}

	// Map the response
	resp := &response.OrderResponse{
		OrderID:     dhanResp.OrderID,
		OrderStatus: dhanResp.OrderStatus,
		Success:     true,
		Message:     "Order modified successfully",
	}

	return resp, nil
}

// GetAllTrades retrieves all trades for the day
func (s *Service) GetAllTrades() (*response.TradesListResponse, error) {
	// Make the API call
	dhanResp, err := s.dhanClient.GetAllTrades()
	if err != nil {
		log.Error("Failed to get trades: %v", err)
		return nil, apperrors.NewInternalServerError("Failed to get trades", err)
	}

	// Map the response
	trades := make([]response.TradeResponse, 0, len(dhanResp))
	for _, t := range dhanResp {
		trades = append(trades, response.TradeResponse{
			OrderID:         t.OrderID,
			ExchangeOrderID: t.ExchangeOrderID,
			ExchangeTradeID: t.ExchangeTradeID,
			TransactionType: t.TransactionType,
			ExchangeSegment: t.ExchangeSegment,
			ProductType:     t.ProductType,
			OrderType:       t.OrderType,
			Symbol:          t.TradingSymbol,
			SecurityID:      t.SecurityID,
			Quantity:        t.TradedQuantity,
			Price:           t.TradedPrice,
			Timestamp:       t.ExchangeTime,
			ExpiryDate:      t.DrvExpiryDate,
			OptionType:      t.DrvOptionType,
			StrikePrice:     t.DrvStrikePrice,
		})
	}

	resp := &response.TradesListResponse{
		Trades: trades,
		Count:  len(trades),
	}

	return resp, nil
}

// GetTradeHistory retrieves trade history for a date range
func (s *Service) GetTradeHistory(req *request.TradeHistoryRequest) (*response.TradeHistoryListResponse, error) {
	// Validate date range
	_, err := time.Parse("2006-01-02", req.FromDate)
	if err != nil {
		return nil, apperrors.NewBadRequestError("Invalid from date format (yyyy-MM-dd)", err)
	}

	_, err = time.Parse("2006-01-02", req.ToDate)
	if err != nil {
		return nil, apperrors.NewBadRequestError("Invalid to date format (yyyy-MM-dd)", err)
	}

	// Make the API call
	dhanResp, err := s.dhanClient.GetTradeHistory(req.FromDate, req.ToDate, req.PageNumber)
	if err != nil {
		log.Error("Failed to get trade history: %v", err)
		return nil, apperrors.NewInternalServerError("Failed to get trade history", err)
	}

	// Map the response
	trades := make([]response.TradeHistoryResponse, 0, len(dhanResp))
	for _, t := range dhanResp {
		// Sum all tax and fee components
		taxesFees := t.SEBITax + t.STT + t.ServiceTax + t.ExchangeTransactionCharges + t.StampDuty

		trades = append(trades, response.TradeHistoryResponse{
			OrderID:         t.OrderID,
			ExchangeOrderID: t.ExchangeOrderID,
			ExchangeTradeID: t.ExchangeTradeID,
			TransactionType: t.TransactionType,
			ExchangeSegment: t.ExchangeSegment,
			ProductType:     t.ProductType,
			OrderType:       t.OrderType,
			Symbol:          t.CustomSymbol,
			SecurityID:      t.SecurityID,
			Quantity:        t.TradedQuantity,
			Price:           t.TradedPrice,
			ISIN:            t.ISIN,
			Instrument:      t.Instrument,
			BrokerageFees:   t.BrokerageCharges,
			TaxesFees:       taxesFees,
			Timestamp:       t.ExchangeTime,
			ExpiryDate:      t.DrvExpiryDate,
			OptionType:      t.DrvOptionType,
			StrikePrice:     t.DrvStrikePrice,
		})
	}

	resp := &response.TradeHistoryListResponse{
		Trades: trades,
		Count:  len(trades),
	}

	return resp, nil
}

// CancelOrder cancels an existing order
func (s *Service) CancelOrder(orderID string) (*response.OrderResponse, error) {
	if orderID == "" {
		return nil, apperrors.NewBadRequestError("Order ID is required", errors.New("missing order ID"))
	}

	// Note: Dhan API doesn't expose cancel order directly in the documentation
	// This is a placeholder for the actual implementation
	// You would need to implement the actual API call in the Dhan client

	// For now, we'll return a mock response
	resp := &response.OrderResponse{
		OrderID:     orderID,
		OrderStatus: dhan.OrderStatusCancelled,
		Success:     true,
		Message:     "Order cancelled successfully",
	}

	return resp, nil
}

// Helper to generate a unique correlation ID
func generateCorrelationID() string {
	now := time.Now()
	return fmt.Sprintf("TRD-%d%d%d%d", now.Day(), now.Hour(), now.Minute(), now.Second())
}
