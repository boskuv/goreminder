package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	ratelimit "github.com/ljahier/gin-ratelimit"
)

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	Enabled  bool
	Requests int
	Window   time.Duration
}

// RateLimitMiddleware creates a rate limiting middleware using gin-ratelimit
func RateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	if !config.Enabled {
		// Return a no-op middleware if rate limiting is disabled
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Default values if not provided
	requests := config.Requests
	if requests <= 0 {
		requests = 100 // Default: 100 requests
	}

	window := config.Window
	if window <= 0 {
		window = 1 * time.Minute // Default: 1 minute window
	}

	// Create token bucket for rate limiting
	tb := ratelimit.NewTokenBucket(requests, window)

	// Create rate limiter that limits by IP address
	return ratelimit.RateLimitByIP(tb)
}
