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
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/logger"
)

// BacklogHandler handles backlog-related HTTP requests
type BacklogHandler struct {
	logger         zerolog.Logger
	backlogService *service.BacklogService
}

// NewBacklogHandler creates a new BacklogHandler
func NewBacklogHandler(backlogService *service.BacklogService, logger zerolog.Logger) *BacklogHandler {
	return &BacklogHandler{
		logger:         logger,
		backlogService: backlogService,
	}
}

// @Summary Create a new backlog item
// @Description Creates a new backlog item and associates it with a user
// @Tags Backlogs
// @Accept json
// @Produce json
// @Param backlog body dto.CreateBacklogRequest true "Backlog to create"
// @Success 201 {object} map[string]int64 "Created backlog ID"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/backlogs [post]
func (h *BacklogHandler) CreateBacklog(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var req dto.CreateBacklogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Msg("invalid request payload for backlog creation")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("user.id", req.UserID).
		Str("backlog.title", req.Title).
		Msg("creating backlog")

	// Convert DTO to model for service
	backlogModel := mapper.CreateBacklogRequestToModel(&req)

	backlogID, err := h.backlogService.CreateBacklog(ctx, backlogModel)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", req.UserID).
			Msg("error while adding new backlog")

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
		Int64("backlog.id", backlogID).
		Int64("user.id", req.UserID).
		Msg("backlog created successfully")

	c.JSON(http.StatusCreated, gin.H{"id": backlogID})
}

// @Summary Create multiple backlog items from batch
// @Description Creates multiple backlog items from a batch string separated by separator
// @Tags Backlogs
// @Accept json
// @Produce json
// @Param backlog body dto.CreateBacklogsBatchRequest true "Batch backlog items to create"
// @Success 201 {object} map[string]interface{} "Created backlog IDs"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/backlogs/batch [post]
func (h *BacklogHandler) CreateBacklogsBatch(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var req dto.CreateBacklogsBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Msg("invalid request payload for batch backlog creation")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("user.id", req.UserID).
		Str("separator", req.Separator).
		Msg("creating backlogs in batch")

	ids, err := h.backlogService.CreateBacklogsBatch(ctx, req.Items, req.Separator, req.UserID, req.MessengerRelatedUserID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", req.UserID).
			Msg("error while creating batch backlogs")

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
		Int64("user.id", req.UserID).
		Int("created.count", len(ids)).
		Msg("batch backlogs created successfully")

	c.JSON(http.StatusCreated, gin.H{"ids": ids, "count": len(ids)})
}

// @Summary Get backlog by ID
// @Description Retrieves a backlog item by its ID
// @Tags Backlogs
// @Produce json
// @Param id path int true "Backlog ID"
// @Success 200 {object} dto.BacklogResponse "Backlog item"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/backlogs/{id} [get]
func (h *BacklogHandler) GetBacklog(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("id", idStr).
			Msg("invalid backlog id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid backlog id: %s", idStr),
		})
		return
	}

	log.Info().
		Int64("backlog.id", id).
		Msg("getting backlog")

	backlog, err := h.backlogService.GetBacklogByID(ctx, id)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("backlog.id", id).
			Msg("error while getting backlog by id")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("backlog with id `%d` not found", id),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := mapper.BacklogModelToResponse(backlog)
	log.Info().
		Int64("backlog.id", id).
		Msg("backlog retrieved successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Get all backlog items
// @Description Retrieves all backlog items with pagination, ordering, and filtering
// @Tags Backlogs
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(50)
// @Param order_by query string false "Order by" default(created_at DESC)
// @Param user_id query int false "Filter by user ID"
// @Success 200 {object} dto.PaginatedBacklogsResponse "Paginated backlogs"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/backlogs [get]
func (h *BacklogHandler) GetAllBacklogs(c *gin.Context) {
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
		Msg("getting all backlogs")

	backlogs, totalCount, err := h.backlogService.GetAllBacklogs(ctx, int(page), int(pageSize), orderBy, userID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Msg("error while getting all backlogs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate total pages
	totalPages := (totalCount + int(pageSize) - 1) / int(pageSize)
	if totalPages == 0 {
		totalPages = 1
	}

	// Convert models to response DTOs
	responsesPtr := mapper.BacklogsModelToResponse(backlogs)
	responses := make([]dto.BacklogResponse, len(responsesPtr))
	for i, resp := range responsesPtr {
		responses[i] = *resp
	}

	response := dto.PaginatedBacklogsResponse{
		Data: responses,
		Pagination: dto.PaginationResponse{
			Page:       int(page),
			PageSize:   int(pageSize),
			TotalPages: totalPages,
			TotalCount: totalCount,
		},
	}

	log.Info().
		Int("backlogs.count", len(backlogs)).
		Int("total_count", totalCount).
		Msg("backlogs retrieved successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Update backlog item
// @Description Updates a backlog item by its ID
// @Tags Backlogs
// @Accept json
// @Produce json
// @Param id path int true "Backlog ID"
// @Param backlog body dto.UpdateBacklogRequest true "Backlog update data"
// @Success 200 {object} dto.BacklogResponse "Updated backlog item"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/backlogs/{id} [put]
func (h *BacklogHandler) UpdateBacklog(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("id", idStr).
			Msg("invalid backlog id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid backlog id: %s", idStr),
		})
		return
	}

	var req dto.UpdateBacklogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Int64("backlog.id", id).
			Msg("invalid request payload for backlog update")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("backlog.id", id).
		Msg("updating backlog")

	// Convert DTO to model for service
	updateRequest := mapper.UpdateBacklogRequestToModel(&req)

	backlog, err := h.backlogService.UpdateBacklog(ctx, id, updateRequest)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("backlog.id", id).
			Msg("error while updating backlog")

		if errors.Is(err, errs.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("backlog with id `%d` not found", id),
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

	response := mapper.BacklogModelToResponse(backlog)
	log.Info().
		Int64("backlog.id", id).
		Msg("backlog updated successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Delete backlog item
// @Description Deletes a backlog item by its ID (soft delete)
// @Tags Backlogs
// @Produce json
// @Param id path int true "Backlog ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/backlogs/{id} [delete]
func (h *BacklogHandler) DeleteBacklog(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("id", idStr).
			Msg("invalid backlog id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid backlog id: %s", idStr),
		})
		return
	}

	log.Info().
		Int64("backlog.id", id).
		Msg("deleting backlog")

	err = h.backlogService.DeleteBacklog(ctx, id)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("backlog.id", id).
			Msg("error while deleting backlog")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("backlog with id `%d` not found", id),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("backlog.id", id).
		Msg("backlog deleted successfully")

	c.Status(http.StatusNoContent)
}
