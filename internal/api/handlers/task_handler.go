package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/service"
)

// TaskHandler handles task-related HTTP requests
type TaskHandler struct {
	TaskService *service.TaskService
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{
		TaskService: taskService,
	}
}

// @Summary Create a new task
// @Description Creates a new task and associates it with a user
// @Tags Tasks
// @Accept json
// @Produce json
// @Param task body models.Task true "Task to create"
// @Success 201 {object} map[string]int64
// @Failure 400 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var task models.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid input data", http.StatusBadRequest))
		return
	}

	taskID, err := h.TaskService.CreateTask(&task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"task_id": taskID})
}

// @Summary Get task by ID
// @Description Retrieves a task by its ID
// @Tags Tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} models.Task
// @Failure 400 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/tasks/{id} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid task ID", http.StatusBadRequest))
		return
	}

	task, err := h.TaskService.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, task)
}

// @Summary Get all user's tasks by userID
// @Description Retrieves all tasks by userID
// @Tags Tasks
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} []models.Task
// @Failure 400 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/users/{userId}/tasks [get]
func (h *TaskHandler) GetUserTasks(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid user ID", http.StatusBadRequest))
		return
	}

	tasks, err := h.TaskService.TaskRepo.GetTasksByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, tasks)
}
