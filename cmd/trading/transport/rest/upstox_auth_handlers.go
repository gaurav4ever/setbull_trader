// cmd/trading/transport/rest/upstox_auth_handlers.go

package rest

import (
	"net/http"
	"time"

	"setbull_trader/internal/core/adapters/client/upstox"
	"setbull_trader/pkg/apperrors"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// UpstoxAuthHandler handles authentication-related HTTP endpoints
type UpstoxAuthHandler struct {
	authService *upstox.AuthService
}

// NewUpstoxAuthHandler creates a new UpstoxAuthHandler
func NewUpstoxAuthHandler(authService *upstox.AuthService) *UpstoxAuthHandler {
	return &UpstoxAuthHandler{
		authService: authService,
	}
}

// RegisterRoutes registers the Upstox authentication routes
func (h *UpstoxAuthHandler) RegisterRoutes(router *gin.Engine) {
	upstoxGroup := router.Group("/api/v1/upstox")
	{
		upstoxGroup.GET("/login", h.InitiateLogin)
		upstoxGroup.GET("/callback", h.HandleCallback)
		// upstoxGroup.GET("/logout", h.Logout)

		// Historical candle data endpoints
		upstoxGroup.GET("/historical/:instrument/:interval/:to_date", h.GetHistoricalCandleData)
		upstoxGroup.GET("/historical/:instrument/:interval/:to_date/:from_date", h.GetHistoricalCandleDataWithRange)
		upstoxGroup.GET("/intraday/:instrument/:interval", h.GetIntraDayCandleData)
	}
}

// LoginRequest represents the request to initiate login
type LoginRequest struct {
	RedirectAfterLogin string `json:"redirectAfterLogin" form:"redirectAfterLogin"`
}

// LoginResponse represents the response with login URL
type LoginResponse struct {
	LoginURL  string `json:"loginUrl"`
	SessionID string `json:"sessionId"`
}

// InitiateLogin initiates the Upstox login flow
// @Summary Initiate Upstox login
// @Description Starts the OAuth2 flow for Upstox authentication
// @Tags upstox
// @Accept json
// @Produce json
// @Param redirectAfterLogin query string false "URL to redirect to after successful login"
// @Success 200 {object} LoginResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/upstox/login [get]
func (h *UpstoxAuthHandler) InitiateLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		RespondWithError(c, apperrors.NewBadRequestError("Invalid request parameters", err))
		return
	}

	// Initiate login flow
	loginURL, sessionID, err := h.authService.InitiateLogin(c.Request.Context())
	if err != nil {
		RespondWithError(c, apperrors.NewInternalServerError("Failed to initiate login", err))
		return
	}

	// Store the session ID in a cookie
	h.setSessionCookie(c, sessionID)

	// Store the redirect URL in a cookie if provided
	if req.RedirectAfterLogin != "" {
		c.SetCookie(
			"upstox_redirect",
			req.RedirectAfterLogin,
			int(15*time.Minute.Seconds()),
			"/",
			"",
			false,
			true,
		)
	}

	c.JSON(http.StatusOK, LoginResponse{
		LoginURL:  loginURL,
		SessionID: sessionID,
	})
}

// CallbackRequest represents the callback request
type CallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state" binding:"required"`
}

// HandleCallback handles the Upstox authorization callback
// @Summary Handle Upstox callback
// @Description Processes the callback from Upstox after user authentication
// @Tags upstox
// @Accept json
// @Produce json
// @Param code query string true "Authorization code"
// @Param state query string true "State parameter"
// @Success 302 {string} string "Redirects to frontend"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/upstox/callback [get]
func (h *UpstoxAuthHandler) HandleCallback(c *gin.Context) {
	var req CallbackRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		RespondWithError(c, apperrors.NewBadRequestError("Invalid callback parameters", err))
		return
	}

	// Process the callback
	_, err := h.authService.HandleCallback(c.Request.Context(), req.Code, req.State)
	if err != nil {
		RespondWithError(c, apperrors.NewInternalServerError("Failed to process callback", err))
		return
	}

	// Get redirect URL from cookie
	redirectURL, err := c.Cookie("upstox_redirect")
	if err != nil || redirectURL == "" {
		redirectURL = "/dashboard" // Default redirect
	}

	// Clear the redirect cookie
	c.SetCookie("upstox_redirect", "", -1, "/", "", false, true)

	// Redirect to the frontend
	c.Redirect(http.StatusFound, redirectURL)
}

// Logout logs out the user from Upstox
// @Summary Logout from Upstox
// @Description Logs out the user from Upstox by invalidating the token
// @Tags upstox
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/upstox/logout [get]
// func (h *UpstoxAuthHandler) Logout(c *gin.Context) {
// 	// Get session ID from cookie
// 	sessionID, err := c.Cookie("upstox_session")
// 	if err != nil {
// 		RespondWithError(c, apperrors.NewUnauthorizedError("Not authenticated", errors.New("missing session cookie")))
// 		return
// 	}

// 	// Delete the token
// 	err = h.authService.DeleteToken(c.Request.Context(), sessionID)
// 	if err != nil {
// 		log.Error("Failed to delete token: %v", err)
// 	}

// 	// Clear the session cookie
// 	c.SetCookie("upstox_session", "", -1, "/", "", false, true)

// 	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "Successfully logged out"})
// }

// GetHistoricalCandleData gets historical candle data
// @Summary Get historical candle data
// @Description Gets historical OHLC candle data for a specific instrument
// @Tags upstox
// @Accept json
// @Produce json
// @Param instrument path string true "Instrument key"
// @Param interval path string true "Candle interval"
// @Param to_date path string true "End date (YYYY-MM-DD)"
// @Success 200 {object} swagger.GetHistoricalCandleResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/upstox/historical/{instrument}/{interval}/{to_date} [get]
func (h *UpstoxAuthHandler) GetHistoricalCandleData(c *gin.Context) {
	// Get session ID from cookie
	sessionID, err := c.Cookie("upstox_session")
	if err != nil {
		RespondWithError(c, apperrors.NewUnauthorizedError("Not authenticated", errors.New("missing session cookie")))
		return
	}

	// Get parameters from path
	instrument := c.Param("instrument")
	interval := c.Param("interval")
	toDate := c.Param("to_date")

	// Get historical candle data
	response, err := h.authService.GetHistoricalCandleData(c.Request.Context(), sessionID, instrument, interval, toDate)
	if err != nil {
		RespondWithError(c, apperrors.NewInternalServerError("Failed to get historical candle data", err))
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetHistoricalCandleDataWithRange gets historical candle data with date range
// @Summary Get historical candle data with date range
// @Description Gets historical OHLC candle data for a specific instrument with a date range
// @Tags upstox
// @Accept json
// @Produce json
// @Param instrument path string true "Instrument key"
// @Param interval path string true "Candle interval"
// @Param to_date path string true "End date (YYYY-MM-DD)"
// @Param from_date path string true "Start date (YYYY-MM-DD)"
// @Success 200 {object} swagger.GetHistoricalCandleResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/upstox/historical/{instrument}/{interval}/{to_date}/{from_date} [get]
func (h *UpstoxAuthHandler) GetHistoricalCandleDataWithRange(c *gin.Context) {
	// Get session ID from cookie
	sessionID, err := c.Cookie("upstox_session")
	if err != nil {
		RespondWithError(c, apperrors.NewUnauthorizedError("Not authenticated", errors.New("missing session cookie")))
		return
	}

	// Get parameters from path
	instrument := c.Param("instrument")
	interval := c.Param("interval")
	toDate := c.Param("to_date")
	fromDate := c.Param("from_date")

	// Get historical candle data with range
	response, err := h.authService.GetHistoricalCandleDataWithDateRange(c.Request.Context(), sessionID, instrument, interval, toDate, fromDate)
	if err != nil {
		RespondWithError(c, apperrors.NewInternalServerError("Failed to get historical candle data with range", err))
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetIntraDayCandleData gets intra-day candle data
// @Summary Get intra-day candle data
// @Description Gets intra-day OHLC candle data for a specific instrument
// @Tags upstox
// @Accept json
// @Produce json
// @Param instrument path string true "Instrument key"
// @Param interval path string true "Candle interval"
// @Success 200 {object} swagger.GetIntraDayCandleResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/upstox/intraday/{instrument}/{interval} [get]
func (h *UpstoxAuthHandler) GetIntraDayCandleData(c *gin.Context) {
	// Get session ID from cookie
	sessionID, err := c.Cookie("upstox_session")
	if err != nil {
		RespondWithError(c, apperrors.NewUnauthorizedError("Not authenticated", errors.New("missing session cookie")))
		return
	}

	// Get parameters from path
	instrument := c.Param("instrument")
	interval := c.Param("interval")

	// Get intra-day candle data
	response, err := h.authService.GetIntraDayCandleData(c.Request.Context(), sessionID, instrument, interval)
	if err != nil {
		RespondWithError(c, apperrors.NewInternalServerError("Failed to get intra-day candle data", err))
		return
	}

	c.JSON(http.StatusOK, response)
}

// setSessionCookie sets the session cookie
func (h *UpstoxAuthHandler) setSessionCookie(c *gin.Context, sessionID string) {
	c.SetCookie(
		"upstox_session",
		sessionID,
		int(24*7*time.Hour.Seconds()), // 7 days
		"/",
		"",
		false, // In production, set to true for HTTPS
		true,  // HttpOnly
	)
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// ErrorResponse represents a generic error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// RespondWithError sends an error response
func RespondWithError(c *gin.Context, err error) {
	var statusCode int
	var message string

	appErr, ok := err.(*apperrors.AppError)
	if ok {
		statusCode = appErr.Code
		message = appErr.Message
	} else {
		statusCode = http.StatusInternalServerError
		message = "Internal server error"
	}

	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Message: message,
		Error:   err.Error(),
	})
}
