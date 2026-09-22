package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/api/dto/mapper"
	"github.com/boskuv/goreminder/internal/api/validation"
	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/pkg/logger"
)

// TaskGroupHandler handles task-group-related HTTP requests
type TaskGroupHandler struct {
	logger           zerolog.Logger
	taskGroupService TaskGroupService
}

// NewTaskGroupHandler creates a new TaskGroupHandler
func NewTaskGroupHandler(taskGroupService TaskGroupService, logger zerolog.Logger) *TaskGroupHandler {
	return &TaskGroupHandler{
		logger:           logger,
		taskGroupService: taskGroupService,
	}
}

// @Summary Create a new task group
// @Description Creates a new task group and associates it with a user
// @Tags TaskGroups
// @Accept json
// @Produce json
// @Param task_group body dto.CreateTaskGroupRequest true "Task group to create"
// @Success 201 {object} dto.CreateResponse "Created task group ID"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 422 {object} dto.ErrorResponse "Unprocessable entity"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/task-groups [post]
func (h *TaskGroupHandler) CreateTaskGroup(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var req dto.CreateTaskGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Msg("invalid request payload for task group creation")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("user.id", req.UserID).
		Str("task_group.name", req.Name).
		Msg("creating task group")

	groupModel := mapper.CreateTaskGroupRequestToModel(&req)

	groupID, err := h.taskGroupService.CreateTaskGroup(ctx, groupModel)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", req.UserID).
			Msg("error while adding new task group")

		if errors.Is(err, errs.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{
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

	log.Info().
		Int64("task_group.id", groupID).
		Int64("user.id", req.UserID).
		Msg("task group created successfully")

	c.JSON(http.StatusCreated, gin.H{"id": groupID})
}

// @Summary Get task group by ID
// @Description Retrieves a task group by its ID
// @Tags TaskGroups
// @Produce json
// @Param id path int true "Task group ID"
// @Success 200 {object} dto.TaskGroupResponse "Task group"
// @Failure 404 {object} dto.ErrorResponse "Not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/task-groups/{id} [get]
func (h *TaskGroupHandler) GetTaskGroup(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("id", idStr).
			Msg("invalid task group id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid task group id: %s", idStr),
		})
		return
	}

	log.Info().
		Int64("task_group.id", id).
		Msg("getting task group")

	group, err := h.taskGroupService.GetTaskGroupByID(ctx, id)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task_group.id", id).
			Msg("error while getting task group by id")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task group with id `%d` not found", id),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := mapper.TaskGroupModelToResponse(group)
	log.Info().
		Int64("task_group.id", id).
		Msg("task group retrieved successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Get all task groups
// @Description Retrieves all task groups with pagination, ordering, and filtering
// @Tags TaskGroups
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(50)
// @Param order_by query string false "Order by" default(created_at DESC)
// @Param user_id query int false "Filter by user ID"
// @Success 200 {object} dto.PaginatedTaskGroupsResponse "Paginated task groups"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/task-groups [get]
func (h *TaskGroupHandler) GetAllTaskGroups(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	page, err := validation.ValidateInt64Query(c, "page", 1, 1)
	if err != nil {
		log.Info().Err(err).Msg("invalid page query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	pageSize, err := validation.ValidateInt64Query(c, "page_size", 50, 1)
	if err != nil {
		log.Info().Err(err).Msg("invalid page_size query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	orderBy, err := validation.ValidateOptionalStringQuery(c, "order_by")
	if err != nil {
		log.Info().Err(err).Msg("invalid order_by query parameter")
		validation.HandleValidationError(c, err)
		return
	}
	if orderBy == "" {
		orderBy = "created_at DESC"
	}

	var userID *int64
	userIDStr := c.Query("user_id")
	if userIDStr != "" {
		uid, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			log.Info().Err(err).Msg("invalid user_id query parameter")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid user_id parameter",
			})
			return
		}
		userID = &uid
	}

	log.Info().
		Int64("page", page).
		Int64("page_size", pageSize).
		Str("order_by", orderBy).
		Msg("getting all task groups")

	groups, totalCount, err := h.taskGroupService.GetAllTaskGroups(ctx, int(page), int(pageSize), orderBy, userID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Msg("error while getting all task groups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := (totalCount + int(pageSize) - 1) / int(pageSize)
	if totalPages == 0 {
		totalPages = 1
	}

	responsesPtr := mapper.TaskGroupsModelToResponse(groups)
	responses := make([]dto.TaskGroupResponse, len(responsesPtr))
	for i, resp := range responsesPtr {
		responses[i] = *resp
	}

	response := dto.PaginatedTaskGroupsResponse{
		Data: responses,
		Pagination: dto.PaginationResponse{
			Page:       int(page),
			PageSize:   int(pageSize),
			TotalPages: totalPages,
			TotalCount: totalCount,
		},
	}

	log.Info().
		Int("task_groups.count", len(groups)).
		Int("total_count", totalCount).
		Msg("task groups retrieved successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Update task group
// @Description Updates a task group by its ID
// @Tags TaskGroups
// @Accept json
// @Produce json
// @Param id path int true "Task group ID"
// @Param task_group body dto.UpdateTaskGroupRequest true "Task group update data"
// @Success 200 {object} dto.TaskGroupResponse "Updated task group"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 404 {object} dto.ErrorResponse "Not found"
// @Failure 422 {object} dto.ErrorResponse "Unprocessable entity"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/task-groups/{id} [put]
func (h *TaskGroupHandler) UpdateTaskGroup(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("id", idStr).
			Msg("invalid task group id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid task group id: %s", idStr),
		})
		return
	}

	var req dto.UpdateTaskGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Int64("task_group.id", id).
			Msg("invalid request payload for task group update")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("task_group.id", id).
		Msg("updating task group")

	updateRequest := mapper.UpdateTaskGroupRequestToModel(&req)

	group, err := h.taskGroupService.UpdateTaskGroup(ctx, id, updateRequest)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task_group.id", id).
			Msg("error while updating task group")

		if errors.Is(err, errs.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task group with id `%d` not found", id),
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

	response := mapper.TaskGroupModelToResponse(group)
	log.Info().
		Int64("task_group.id", id).
		Msg("task group updated successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Delete task group
// @Description Deletes a task group by its ID (soft delete); tasks.group_id is set to NULL
// @Tags TaskGroups
// @Produce json
// @Param id path int true "Task group ID"
// @Success 204 "No Content"
// @Failure 404 {object} dto.ErrorResponse "Not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/task-groups/{id} [delete]
func (h *TaskGroupHandler) DeleteTaskGroup(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("id", idStr).
			Msg("invalid task group id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid task group id: %s", idStr),
		})
		return
	}

	log.Info().
		Int64("task_group.id", id).
		Msg("deleting task group")

	err = h.taskGroupService.DeleteTaskGroup(ctx, id)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task_group.id", id).
			Msg("error while deleting task group")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task group with id `%d` not found", id),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task_group.id", id).
		Msg("task group deleted successfully")

	c.Status(http.StatusNoContent)
}
