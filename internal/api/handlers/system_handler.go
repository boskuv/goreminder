package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// VersionHandler contains application version info
type VersionHandler struct {
	AppVersion string
}

// HealthCheckHandler handles health checks
type HealthCheckHandler struct{}

// NewVersionHandler creates a new VersionHandler
func NewVersionHandler(version string) *VersionHandler {
	return &VersionHandler{AppVersion: version}
}

// NewHealthCheckHandler creates a new HealthCheckHandler
func NewHealthCheckHandler() *HealthCheckHandler {
	return &HealthCheckHandler{}
}

// Version provides the current version of the application
// @Summary Get application version
// @Description Returns the current version of the application
// @Tags System
// @Produce json
// @Success 200 {object} map[string]string
// @Router /version [get]
func (h *VersionHandler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": h.AppVersion})
}

// HealthCheck verifies the health of the application
// @Summary Check application health
// @Description Checks if the application is running and healthy
// @Tags System
// @Produce json
// @Success 200 {object} map[string]string
// @Router /healthcheck [get]
func (h *HealthCheckHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
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
