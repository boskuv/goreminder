package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/boskuv/goreminder/internal/api/handlers"
)

// RegisterSystemRoutes registers all API system routes
func RegisterSystemRoutes(router *gin.Engine, version string) {
	// Initialize handlers
	versionHandler := handlers.NewVersionHandler(version)
	healthCheckHandler := handlers.NewHealthCheckHandler()

	// Register routes
	router.GET("/version", versionHandler.Version)
	router.GET("/healthcheck", healthCheckHandler.HealthCheck)
	handlers.RegisterDocsRoute(router)
}
