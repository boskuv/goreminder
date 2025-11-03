package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// CorsConfig holds configuration for CORS
type CorsConfig struct {
	Enabled          bool
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// CorsMiddleware creates a CORS middleware
func CorsMiddleware(config CorsConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// If CORS is disabled, just continue
		if !config.Enabled {
			c.Next()
			return
		}

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range config.AllowOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				c.Header("Access-Control-Allow-Origin", allowedOrigin)
				break
			}
		}

		if !allowed && len(config.AllowOrigins) > 0 {
			c.Header("Access-Control-Allow-Origin", config.AllowOrigins[0])
		}

		// Set default methods if not provided
		methods := config.AllowMethods
		if len(methods) == 0 {
			methods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
		}
		c.Header("Access-Control-Allow-Methods", joinStrings(methods, ", "))

		// Set default headers if not provided
		headers := config.AllowHeaders
		if len(headers) == 0 {
			headers = []string{"Content-Type", "Authorization", "X-Request-ID"}
		}
		c.Header("Access-Control-Allow-Headers", joinStrings(headers, ", "))

		// Expose headers
		if len(config.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", joinStrings(config.ExposeHeaders, ", "))
		}

		// Allow credentials
		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Max age
		if config.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
