package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/boskuv/goreminder/internal/api/handlers"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(router *gin.Engine, taskHandler *handlers.TaskHandler, userHandler *handlers.UserHandler, messengerHandler *handlers.MessengerHandler, backlogHandler *handlers.BacklogHandler, targetHandler *handlers.TargetHandler, digestHandler *handlers.DigestHandler) {
	api := router.Group("/api/v1")
	{
		// Task routes
		api.GET("/tasks", taskHandler.GetAllTasks)
		api.POST("/tasks", taskHandler.CreateTask)
		api.POST("/tasks/queue", taskHandler.QueueTask)
		api.GET("/tasks/:id", taskHandler.GetTask)
		api.GET("/tasks/:id/history", taskHandler.GetTaskHistory)
		api.GET("/users/:user_id/tasks", taskHandler.GetUserTasks)
		api.GET("/users/:user_id/tasks/history", taskHandler.GetUserTaskHistory)
		api.PUT("/tasks/:id", taskHandler.UpdateTask)
		api.POST("/tasks/:id/done", taskHandler.MarkTaskAsDone)
		api.DELETE("/tasks/:id", taskHandler.DeleteTask)

		// User routes
		api.GET("/users", userHandler.GetAllUsers)
		api.POST("/users", userHandler.CreateUser)
		api.GET("/users/:user_id", userHandler.GetUser)
		api.PUT("/users/:user_id", userHandler.UpdateUser)
		api.DELETE("/users/:user_id", userHandler.DeleteUser)

		// Messenger routes
		api.GET("/messengers", messengerHandler.GetAllMessengers)
		api.POST("/messengers", messengerHandler.CreateMessenger)
		api.GET("/messengers/:messenger_id", messengerHandler.GetMessenger)
		api.GET("/messengers/by-name/:messenger_name", messengerHandler.GetMessengerIDByName)
		api.POST("/messengerRelatedUsers", messengerHandler.CreateMessengerRelatedUser)
		api.GET("/messengerRelatedUsers", messengerHandler.GetMessengerRelatedUser)
		api.GET("/messengerRelatedUsers/all", messengerHandler.GetAllMessengerRelatedUsers)
		api.GET("/messengerRelatedUsers/:messenger_user_id/user", messengerHandler.GetUserID)

		// Backlog routes
		api.GET("/backlogs", backlogHandler.GetAllBacklogs)
		api.POST("/backlogs", backlogHandler.CreateBacklog)
		api.POST("/backlogs/batch", backlogHandler.CreateBacklogsBatch)
		api.GET("/backlogs/:id", backlogHandler.GetBacklog)
		api.PUT("/backlogs/:id", backlogHandler.UpdateBacklog)
		api.DELETE("/backlogs/:id", backlogHandler.DeleteBacklog)

		// Target routes
		api.GET("/targets", targetHandler.GetAllTargets)
		api.POST("/targets", targetHandler.CreateTarget)
		api.GET("/targets/:id", targetHandler.GetTarget)
		api.PUT("/targets/:id", targetHandler.UpdateTarget)
		api.DELETE("/targets/:id", targetHandler.DeleteTarget)

		// Digest routes
		api.GET("/digests", digestHandler.GetDigest)
		api.POST("/digests/settings", digestHandler.CreateDigestSettings)
		api.GET("/digests/settings", digestHandler.GetDigestSettings)
		api.PUT("/digests/settings", digestHandler.UpdateDigestSettings)
		api.DELETE("/digests/settings", digestHandler.DeleteDigestSettings)
		api.GET("/digests/settings/all", digestHandler.GetAllDigestSettings)
	}
}
