package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/boskuv/goreminder/internal/api/handlers"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(router *gin.Engine, taskHandler *handlers.TaskHandler, userHandler *handlers.UserHandler, messengerHandler *handlers.MessengerHandler) {
	api := router.Group("/api/v1")
	{
		// Task routes
		api.POST("/tasks", taskHandler.CreateTask)
		api.GET("/tasks/:id", taskHandler.GetTask) // TODO: task_id
		api.GET("/users/:user_id/tasks", taskHandler.GetUserTasks)
		api.PUT("/tasks/:id", taskHandler.UpdateTask)
		api.DELETE("/tasks/:id", taskHandler.DeleteTask)

		// User routes
		api.POST("/users", userHandler.CreateUser)
		api.GET("/users/:user_id", userHandler.GetUser)
		api.PUT("/users/:user_id", userHandler.UpdateUser)
		api.DELETE("/users/:user_id", userHandler.DeleteUser)

		// Messenger routes
		api.POST("/messengers", messengerHandler.CreateMessenger)
		api.GET("/messengers/:messenger_id", messengerHandler.GetMessenger)
		api.GET("/messengers/by-name/:messenger_name", messengerHandler.GetMessengerIDByName)
		api.POST("/messengerRelatedUsers", messengerHandler.CreateMessengerRelatedUser)
		api.GET("/messengerRelatedUsers", messengerHandler.GetMessengerRelatedUser)
		api.GET("/messengerRelatedUsers/:messenger_user_id/user", messengerHandler.GetUserID)
	}
}
