package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/service"
)

// TaskHandler handles task-related HTTP requests
type TaskHandler struct {
	logger      zerolog.Logger
	taskService *service.TaskService
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(logger zerolog.Logger, taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{
		logger:      logger,
		taskService: taskService,
	}
}

// @Summary Create a new task
// @Description Creates a new task and associates it with a user
// @Tags Tasks
// @Accept json
// @Produce json
// @Param task body models.Task true "Task to create"
// @Success 201 {object} map[string]int64
// @Failure 400 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var task models.Task // TODO: separate struct
	if err := c.ShouldBindJSON(&task); err != nil {
		// h.logger.Error().Stack().Err(errors.Wrap(err, "invalid input data")).Msg("error while processing request with task struct parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskID, err := h.taskService.CreateTask(c.Request.Context(), &task)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while adding new task")

		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("task with id `%d` has been successfully added", taskID)})
}

// @Summary Get task by ID
// @Description Retrieves a task by its ID
// @Tags Tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} models.Task
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks/{id} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		// h.logger.Error().Stack().Err(errors.Wrap(err, "failed to parse userID")).Msg("error while processing request with id parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.taskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while getting task by its id")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task with id `%d` not found", taskID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Failure 400 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{user_id}/tasks [get]
func (h *TaskHandler) GetUserTasks(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		// h.logger.Error().Stack().Err(errors.Wrap(err, "failed to parse userID")).Msg("error while processing request with userID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tasks, err := h.taskService.GetUserTasks(c.Request.Context(), userID)
	if err != nil {
		// h.logger.Error().Stack().Err(err).Msg("error while getting tasks by userID parameter")
		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": fmt.Sprintf("user with id `%d` not found", userID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks/{id} [put]
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		// h.logger.Error().Stack().Err(errors.Wrap(err, "failed to parse taskID")).Msg("error while processing request with taskID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var taskUpdateRequest models.TaskUpdateRequest
	if err := c.ShouldBindJSON(&taskUpdateRequest); err != nil {
		// h.logger.Error().Stack().Err(errors.Wrap(err, "invalid input data")).Msg("error while processing request with taskUpdateRequest struct parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedTask, err := h.taskService.UpdateTask(c.Request.Context(), taskID, &taskUpdateRequest)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while updating task")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task with id `%d` not found", taskID),
			})
			return
		}
		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Success 204 {object} nil
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks/{id} [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		// TODO: wrap?
		h.logger.Error().Stack().Err(errors.Wrap(err, "failed to parse taskID")).Msg("error while processing request with taskID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.taskService.DeleteTask(c.Request.Context(), taskID)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while soft deleting task")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task with id `%d` not found", taskID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Send task to queue
// @Description Send task to queue with predefined action
// @Tags Tasks
// @Accept json
// @Produce json
// @Param task body models.ScheduledTask true "Task to enqueue"
// @Success 201 {object} models.Task
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks/queue [post]
func (h *TaskHandler) QueueTask(c *gin.Context) {
	var scheduledTask models.ScheduledTask
	if err := c.ShouldBindJSON(&scheduledTask); err != nil {
		//h.logger.Error().Stack().Err(errors.Wrap(err, "invalid input data")).Msg("error while processing request with task struct parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.taskService.QueueTask(c.Request.Context(), &scheduledTask)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while enqueuing task")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}
		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("task with id `%d` has been successfully enqueued", scheduledTask.TaskID)})
}

// @Summary Get task history by task ID
// @Description Retrieves history entries for a specific task
// @Tags Tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} []models.TaskHistory
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks/{id}/history [get]
func (h *TaskHandler) GetTaskHistory(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	histories, err := h.taskService.GetTaskHistory(c.Request.Context(), taskID)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while getting task history")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task with id `%d` not found", taskID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, histories)
}

// @Summary Get task history by user ID
// @Description Retrieves task history entries for a user with pagination
// @Tags Tasks
// @Produce json
// @Param user_id path int true "User ID"
// @Param limit query int false "Limit (default: 50)" default(50)
// @Param offset query int false "Offset (default: 0)" default(0)
// @Success 200 {object} []models.TaskHistory
// @Failure 400 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{user_id}/tasks/history [get]
func (h *TaskHandler) GetUserTaskHistory(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := 50
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	histories, err := h.taskService.GetUserTaskHistory(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while getting user task history")

		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": fmt.Sprintf("user with id `%d` not found", userID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, histories)
}
