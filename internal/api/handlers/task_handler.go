package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/service"
)

// TaskHandler handles task-related HTTP requests
type TaskHandler struct {
	Logger      zerolog.Logger
	TaskService *service.TaskService
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(logger zerolog.Logger, taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{
		Logger:      logger,
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
		h.Logger.Error().Stack().Err(errors.Wrap(err, "invalid input data")).Msg("Error while processing request with task struct parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid input data", http.StatusBadRequest))
		return
	}

	taskID, err := h.TaskService.CreateTask(&task)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while creating a task")
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
		h.Logger.Error().Stack().Err(errors.Wrap(err, "failed to parse userID")).Msg("Error while processing request with id parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid task ID", http.StatusBadRequest))
		return
	}

	task, err := h.TaskService.GetTask(taskID)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while getting a task by its id")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, task)
}

// @Summary Get all user's tasks by userID
// @Description Retrieves all tasks by userID
// @Tags Tasks
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} []models.Task
// @Failure 400 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/users/{user_id}/tasks [get]
func (h *TaskHandler) GetUserTasks(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "failed to parse userID")).Msg("Error while processing request with userID parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid user ID", http.StatusBadRequest))
		return
	}

	tasks, err := h.TaskService.GetUserTasks(userID)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while getting tasks by userID parameter")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// @Summary Update a task
// @Description Updates a task by its ID
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param task body models.TaskUpdateRequest true "Task update details"
// @Success 200 {object} models.Task
// @Failure 400 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/tasks/{id} [put]
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "failed to parse taskID")).Msg("Error while processing request with taskID parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid task ID", http.StatusBadRequest))
		return
	}

	var taskUpdateRequest models.TaskUpdateRequest
	if err := c.ShouldBindJSON(&taskUpdateRequest); err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "invalid input data")).Msg("Error while processing request with taskUpdateRequest struct parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid input data", http.StatusBadRequest))
		return
	}

	updatedTask, err := h.TaskService.UpdateTask(taskID, &taskUpdateRequest)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while updating a task")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, updatedTask)
}

// @Summary Soft delete a task
// @Description Marks a task as deleted by its ID (soft delete)
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Success 204 {object} models.APIError
// @Failure 400 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/tasks/{id} [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "failed to parse taskID")).Msg("Error while processing request with taskID parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid task ID", http.StatusBadRequest))
		return
	}

	err = h.TaskService.DeleteTask(taskID)
	if err != nil {

		h.Logger.Error().Stack().Err(err).Msg("Error while deleting a task")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.Status(http.StatusNoContent) // 204 No Content status for successful deletion
}
