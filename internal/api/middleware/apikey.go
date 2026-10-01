package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIKeyConfig holds configuration for X-API-Key gatekeeping.
type APIKeyConfig struct {
	Enabled bool
	APIKey  string
}

// APIKeyMiddleware requires header X-API-Key to match config when enabled.
func APIKeyMiddleware(cfg APIKeyConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	expected := cfg.APIKey
	return func(c *gin.Context) {
		got := c.GetHeader("X-API-Key")
		if got == "" || got != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing API key"})
			return
		}
		c.Next()
	}
}
