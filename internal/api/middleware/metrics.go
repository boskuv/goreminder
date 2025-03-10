package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response time (seconds)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)

	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of requests received",
		},
		[]string{"method", "route", "status"},
	)
)

func InitMetrics() {
	prometheus.MustRegister(requestDuration, requestCount)
}

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		duration := time.Since(start).Seconds()
		statusCode := c.Writer.Status()

		requestDuration.WithLabelValues(c.Request.Method, c.FullPath(), strconv.Itoa(statusCode)).Observe(duration)
		requestCount.WithLabelValues(c.Request.Method, c.FullPath(), strconv.Itoa(statusCode)).Inc()
	}
}
