package transport

import (
	"bytes"
	"encoding/json"
	"io"
	"setbull_trader/pkg/log"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequestLoggerMiddleware logs details about each incoming request
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log the request path and method
		path := c.Request.URL.Path
		method := c.Request.Method

		// Log query parameters
		queryParams := c.Request.URL.Query()
		if len(queryParams) > 0 {
			queryJSON, _ := json.Marshal(queryParams)
			log.Info("Request: %s %s | Query params: %s", method, path, string(queryJSON))
		}

		// For POST, PUT requests, log the request body
		if method == "POST" || method == "PUT" {
			// Read the request body
			var bodyBytes []byte
			if c.Request.Body != nil {
				bodyBytes, _ = io.ReadAll(c.Request.Body)

				// Restore the request body for subsequent handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// Only log JSON bodies
				contentType := c.GetHeader("Content-Type")
				if strings.Contains(contentType, "application/json") {
					var prettyJSON bytes.Buffer
					if err := json.Indent(&prettyJSON, bodyBytes, "", "  "); err == nil {
						log.Info("Request: %s %s | Body: %s", method, path, prettyJSON.String())
					} else {
						// If not valid JSON, log as-is
						log.Info("Request: %s %s | Body: %s", method, path, string(bodyBytes))
					}
				}
			}
		}

		// Process request
		c.Next()
	}
}
