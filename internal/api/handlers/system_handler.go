package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/boskuv/goreminder/pkg/version"
)

// VersionHandler contains application version info
type VersionHandler struct {
	AppVersion string
}

// ReadinessCheck reports whether the process can accept traffic.
// Return nil when ready; a non-nil error means not ready (HTTP 503).
type ReadinessCheck func(ctx context.Context) error

// HealthCheckHandler handles health and readiness probes
type HealthCheckHandler struct {
	readyCheck ReadinessCheck
}

// NewVersionHandler creates a new VersionHandler
func NewVersionHandler(version string) *VersionHandler {
	return &VersionHandler{AppVersion: version}
}

// NewHealthCheckHandler creates a new HealthCheckHandler.
// readyCheck is used by GET /ready; if nil, /ready always reports ready.
func NewHealthCheckHandler(readyCheck ReadinessCheck) *HealthCheckHandler {
	return &HealthCheckHandler{readyCheck: readyCheck}
}

// Version provides the current version of the application
// @Summary Get application version
// @Description Returns the current version and build information of the application
// @Tags System
// @Produce json
// @Success 200 {object} dto.VersionResponse "Version information"
// @Router /version [get]
func (h *VersionHandler) Version(c *gin.Context) {
	// Return full version information including build metadata
	versionInfo := version.GetFullVersion()
	// Override version with the one passed to handler (for consistency)
	versionInfo["version"] = h.AppVersion
	c.JSON(http.StatusOK, versionInfo)
}

// HealthCheck verifies the health of the application
// @Summary Check application health
// @Description Checks if the application is running and healthy (liveness; no dependency checks)
// @Tags System
// @Produce json
// @Success 200 {object} dto.HealthCheckResponse "Health status"
// @Router /healthcheck [get]
func (h *HealthCheckHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// Ready verifies whether the application can accept traffic (readiness for rolling updates)
// @Summary Check application readiness
// @Description Checks whether dependencies required to serve traffic are available (database ping; RabbitMQ when producer.enabled). Use for Kubernetes/Docker readiness probes during rolling updates.
// @Tags System
// @Produce json
// @Success 200 {object} dto.ReadyCheckResponse "Ready to accept traffic"
// @Failure 503 {object} dto.ReadyCheckResponse "Not ready"
// @Router /ready [get]
func (h *HealthCheckHandler) Ready(c *gin.Context) {
	if h.readyCheck == nil {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.readyCheck(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"reason": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// RegisterDocsRoute registers the Swagger documentation route
// @Summary Swagger API documentation
// @Description Access the API documentation
// @Tags System
// @Produce html
// @Router /docs [get]
func RegisterDocsRoute(router *gin.Engine) {
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
