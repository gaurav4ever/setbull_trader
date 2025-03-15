package transport

import (
	"net/http"
	"strconv"
	"time"

	"setbull_trader/internal/core/dto/request"
	"setbull_trader/internal/core/dto/response"
	"setbull_trader/internal/core/service/orders"
	"setbull_trader/internal/service"
	"setbull_trader/pkg/apperrors"
	"setbull_trader/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

// HTTPHandler handles HTTP requests
type HTTPHandler struct {
	orderService          *orders.Service
	stockService          *service.StockService
	tradeParamsService    *service.TradeParametersService
	executionPlanService  *service.ExecutionPlanService
	orderExecutionService *service.OrderExecutionService
	utilityService        *service.UtilityService
	validator             *validator.Validate
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(
	orderService *orders.Service,
	stockService *service.StockService,
	tradeParamsService *service.TradeParametersService,
	executionPlanService *service.ExecutionPlanService,
	orderExecutionService *service.OrderExecutionService,
	utilityService *service.UtilityService,
) *HTTPHandler {
	return &HTTPHandler{
		orderService:          orderService,
		stockService:          stockService,
		tradeParamsService:    tradeParamsService,
		executionPlanService:  executionPlanService,
		orderExecutionService: orderExecutionService,
		utilityService:        utilityService,
		validator:             validator.New(),
	}
}

func (h *HTTPHandler) RegisterRoutes(router *gin.Engine) {

	router.Use(CORSMiddleware())
	router.Use(RequestLoggerMiddleware())

	// API group with version
	api := router.Group("/api/v1")

	// Orders endpoints
	orders := api.Group("/orders")
	{
		orders.POST("", h.placeOrder)
		orders.PUT("/:orderID", h.modifyOrder)
		orders.DELETE("/:orderID", h.cancelOrder)
	}

	// Trades endpoints
	trades := api.Group("/trades")
	{
		trades.GET("", h.getAllTrades)
		trades.GET("/history", h.getTradeHistory)
	}

	// Health check
	api.GET("/health", h.healthCheck)
}

func (h *HTTPHandler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// placeOrder handles order placement
func (h *HTTPHandler) placeOrder(c *gin.Context) {
	var req request.PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apperrors.NewErrorResponse("Invalid request format", err))
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, apperrors.NewErrorResponse("Validation error", err))
		return
	}

	// Call service
	resp, err := h.orderService.PlaceOrder(&req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// modifyOrder handles order modification
func (h *HTTPHandler) modifyOrder(c *gin.Context) {
	orderID := c.Param("orderID")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, response.GenericResponse{
			Success: false,
			Error:   "Order ID is required",
		})
		return
	}

	var req request.ModifyOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apperrors.NewErrorResponse("Invalid request format", err))
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, apperrors.NewErrorResponse("Validation error", err))
		return
	}

	// Call service
	resp, err := h.orderService.ModifyOrder(orderID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// cancelOrder handles order cancellation
func (h *HTTPHandler) cancelOrder(c *gin.Context) {
	orderID := c.Param("orderID")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, response.GenericResponse{
			Success: false,
			Error:   "Order ID is required",
		})
		return
	}

	// Call service
	resp, err := h.orderService.CancelOrder(orderID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// getAllTrades handles getting all trades
func (h *HTTPHandler) getAllTrades(c *gin.Context) {
	// Call service
	resp, err := h.orderService.GetAllTrades()
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// getTradeHistory handles getting trade history
func (h *HTTPHandler) getTradeHistory(c *gin.Context) {
	var req request.TradeHistoryRequest

	// Set default values
	req.FromDate = c.DefaultQuery("fromDate", time.Now().AddDate(0, 0, -7).Format("2006-01-02"))
	req.ToDate = c.DefaultQuery("toDate", time.Now().Format("2006-01-02"))

	// Parse page number with default 0
	pageNumber := 0
	if c.Query("page") != "" {
		if page, err := strconv.Atoi(c.Query("page")); err == nil {
			pageNumber = page
		}
	}
	req.PageNumber = pageNumber

	// Log the extracted parameters
	log.Info("Processing trade history request | FromDate: %s, ToDate: %s, Page: %d",
		req.FromDate, req.ToDate, req.PageNumber)

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		log.Error("Trade history validation error: %v", err)
		c.JSON(http.StatusBadRequest, apperrors.NewErrorResponse("Validation error", err))
		return
	}

	// Call service
	resp, err := h.orderService.GetTradeHistory(&req)
	if err != nil {
		handleError(c, err)
		return
	}

	// Log response summary
	log.Info("Trade history response | Count: %d", resp.Count)
	c.JSON(http.StatusOK, resp)
}

// handleError handles error responses
func handleError(c *gin.Context, err error) {
	log.Error("HTTP handler error: %v", err)

	// Check if it's an AppError
	if appErr, ok := err.(*apperrors.AppError); ok {
		c.JSON(appErr.Code, response.GenericResponse{
			Success: false,
			Error:   appErr.Message,
		})
		return
	}

	// Default to internal server error
	c.JSON(http.StatusInternalServerError, response.GenericResponse{
		Success: false,
		Error:   "Internal server error",
	})
}

// Helper to parse integers
func scanf(in string, format string, a ...interface{}) (int, error) {
	// Implement your parsing logic here
	return 0, errors.New("Not implemented") // Return 0 as a placeholder
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // Allow any origin
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
