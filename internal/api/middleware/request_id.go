package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/boskuv/goreminder/pkg/logger"
)

const RequestIDHeader = "X-Request-ID"

// RequestIDKey is stored in gin.Context and request context.Context.
const RequestIDKey = logger.RequestIDContextKey

// RequestIDMiddleware generates a unique request ID for each request
// and adds it to gin context, request context, and response headers.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(RequestIDKey, requestID)
		c.Request = c.Request.WithContext(
			context.WithValue(c.Request.Context(), RequestIDKey, requestID),
		)
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID retrieves the request ID from the gin context
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(RequestIDKey); exists {
		if requestID, ok := id.(string); ok {
			return requestID
		}
	}
	return ""
}
