package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"
const RequestIDKey = "request_id"

// RequestIDMiddleware generates a unique request ID for each request
// and adds it to the context and response headers
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in header (from client)
		requestID := c.GetHeader(RequestIDHeader)

		// If not provided by client, generate a new one
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in context for use in handlers
		c.Set(RequestIDKey, requestID)

		// Add request ID to response header
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
