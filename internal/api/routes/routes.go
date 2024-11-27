// internal/api/routes/routes.go
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/boskuv/goreminder/internal/api/handlers"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(router *gin.Engine, taskHandler *handlers.TaskHandler, userHandler *handlers.UserHandler) {
	api := router.Group("/api/v1")
	{
		// Task routes
		api.POST("/tasks", taskHandler.CreateTask)
		api.GET("/tasks/:id", taskHandler.GetTask)

		// User routes
		api.POST("/users", userHandler.CreateUser)
	}
}
