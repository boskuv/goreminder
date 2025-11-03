package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// LoggerMiddleware creates a request logging middleware using zerolog
func LoggerMiddleware(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get request ID from context
		requestID := GetRequestID(c)

		// Build log event
		logEvent := log.Info().
			Int("status", c.Writer.Status()).
			Str("method", c.Request.Method).
			Str("path", path).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent())

		if query != "" {
			logEvent = logEvent.Str("query", query)
		}

		if requestID != "" {
			logEvent = logEvent.Str("request_id", requestID)
		}

		// Log errors
		if len(c.Errors) > 0 {
			errs := make([]error, len(c.Errors))
			for i, e := range c.Errors {
				errs[i] = e.Err
			}
			logEvent = logEvent.Errs("errors", errs)
		}

		// Log based on status code
		if c.Writer.Status() >= 500 {
			logEvent.Msg("Request failed")
		} else if c.Writer.Status() >= 400 {
			logEvent.Msg("Request error")
		} else {
			logEvent.Msg("Request completed")
		}
	}
}
