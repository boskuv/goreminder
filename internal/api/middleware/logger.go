package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/boskuv/goreminder/pkg/logger"
)

// LoggerMiddleware creates a request logging middleware using zerolog.
// It should run after RequestID and Tracing so access logs include
// request_id / trace_id / span_id after c.Next().
func LoggerMiddleware(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		reqLog := logger.WithTraceContext(c.Request.Context(), log)

		var logEvent *zerolog.Event
		switch {
		case status >= 500:
			logEvent = reqLog.Error()
		case status >= 400:
			logEvent = reqLog.Warn()
		default:
			logEvent = reqLog.Info()
		}

		logEvent = logEvent.
			Int("status", status).
			Str("method", c.Request.Method).
			Str("path", path).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent())

		if query != "" {
			logEvent = logEvent.Str("query", query)
		}

		if len(c.Errors) > 0 {
			errs := make([]error, len(c.Errors))
			for i, e := range c.Errors {
				errs[i] = e.Err
			}
			logEvent = logEvent.Errs("errors", errs)
		}

		switch {
		case status >= 500:
			logEvent.Msg("Request failed")
		case status >= 400:
			logEvent.Msg("Request error")
		default:
			logEvent.Msg("Request completed")
		}
	}
}
