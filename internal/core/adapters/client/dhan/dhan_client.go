package dhan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"setbull_trader/internal/trading/config"

	"setbull_trader/pkg/log"

	"github.com/pkg/errors"
)

// Client represents a Dhan API client
type Client struct {
	httpClient  *http.Client
	baseURL     string
	accessToken string
	clientID    string
}

// NewClient creates a new Dhan API client
func NewClient(cfg *config.DhanConfig) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:     cfg.BaseURL,
		accessToken: cfg.AccessToken,
		clientID:    cfg.ClientID,
	}
}

// do executes an HTTP request and returns the response
func (c *Client) do(req *http.Request, v interface{}) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("access-token", c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(bodyBytes))
	}

	if v != nil {
		if err := json.Unmarshal(bodyBytes, v); err != nil {
			log.Error("Failed to unmarshal response: %v, Body: %s", err, string(bodyBytes))
			return errors.Wrap(err, "failed to unmarshal response")
		}
	}

	return nil
}

// PlaceOrder places a new order
func (c *Client) PlaceOrder(request *PlaceOrderRequest) (*OrderResponse, error) {
	url := fmt.Sprintf("%s/v2/orders", c.baseURL)

	// Set the client ID if not provided
	if request.DhanClientID == "" {
		request.DhanClientID = c.clientID
	}

	// Log the security ID being used
	log.Info("Placing order with Dhan API using SecurityID: %s", request.SecurityID)

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	var response OrderResponse
	if err := c.do(req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// ModifyOrder modifies an existing order
func (c *Client) ModifyOrder(orderID string, request *ModifyOrderRequest) (*OrderResponse, error) {
	url := fmt.Sprintf("%s/v2/orders/%s", c.baseURL, orderID)

	// Set the client ID if not provided
	if request.DhanClientID == "" {
		request.DhanClientID = c.clientID
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	var response OrderResponse
	if err := c.do(req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetAllTrades retrieves all trades for the day
func (c *Client) GetAllTrades() ([]TradeResponse, error) {
	url := fmt.Sprintf("%s/v2/trades", c.baseURL)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	var response []TradeResponse
	if err := c.do(req, &response); err != nil {
		return nil, err
	}

	return response, nil
}

// GetTradeHistory retrieves trade history for a date range
func (c *Client) GetTradeHistory(fromDate, toDate string, pageNumber int) ([]TradeHistoryResponse, error) {
	url := fmt.Sprintf("%s/v2/trades/%s/%s/%d", c.baseURL, fromDate, toDate, pageNumber)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	var response []TradeHistoryResponse
	if err := c.do(req, &response); err != nil {
		return nil, err
	}

	return response, nil
}
