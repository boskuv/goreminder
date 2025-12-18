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

// TargetHandler handles target-related HTTP requests
type TargetHandler struct {
	logger        zerolog.Logger
	targetService *service.TargetService
}

// NewTargetHandler creates a new TargetHandler
func NewTargetHandler(targetService *service.TargetService, logger zerolog.Logger) *TargetHandler {
	return &TargetHandler{
		logger:        logger,
		targetService: targetService,
	}
}

// @Summary Create a new target item
// @Description Creates a new target item and associates it with a user
// @Tags Targets
// @Accept json
// @Produce json
// @Param target body dto.CreateTargetRequest true "Target to create"
// @Success 201 {object} map[string]int64 "Created target ID"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/targets [post]
func (h *TargetHandler) CreateTarget(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var req dto.CreateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Msg("invalid request payload for target creation")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("user.id", req.UserID).
		Str("target.title", req.Title).
		Msg("creating target")

	// Convert DTO to model for service
	targetModel := mapper.CreateTargetRequestToModel(&req)

	targetID, err := h.targetService.CreateTarget(ctx, targetModel)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", req.UserID).
			Msg("error while adding new target")

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
		Int64("target.id", targetID).
		Int64("user.id", req.UserID).
		Msg("target created successfully")

	c.JSON(http.StatusCreated, gin.H{"id": targetID})
}

// @Summary Get target by ID
// @Description Retrieves a target item by its ID
// @Tags Targets
// @Produce json
// @Param id path int true "Target ID"
// @Success 200 {object} dto.TargetResponse "Target item"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/targets/{id} [get]
func (h *TargetHandler) GetTarget(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("id", idStr).
			Msg("invalid target id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid target id: %s", idStr),
		})
		return
	}

	log.Info().
		Int64("target.id", id).
		Msg("getting target")

	target, err := h.targetService.GetTargetByID(ctx, id)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("target.id", id).
			Msg("error while getting target by id")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("target with id `%d` not found", id),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := mapper.TargetModelToResponse(target)
	log.Info().
		Int64("target.id", id).
		Msg("target retrieved successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Get all target items
// @Description Retrieves all target items with pagination, ordering, and filtering
// @Tags Targets
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(50)
// @Param order_by query string false "Order by" default(created_at DESC)
// @Param user_id query int false "Filter by user ID"
// @Success 200 {object} dto.PaginatedTargetsResponse "Paginated targets"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/targets [get]
func (h *TargetHandler) GetAllTargets(c *gin.Context) {
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
		Msg("getting all targets")

	targets, totalCount, err := h.targetService.GetAllTargets(ctx, int(page), int(pageSize), orderBy, userID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Msg("error while getting all targets")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate total pages
	totalPages := (totalCount + int(pageSize) - 1) / int(pageSize)
	if totalPages == 0 {
		totalPages = 1
	}

	// Convert models to response DTOs
	responsesPtr := mapper.TargetsModelToResponse(targets)
	responses := make([]dto.TargetResponse, len(responsesPtr))
	for i, resp := range responsesPtr {
		responses[i] = *resp
	}

	response := dto.PaginatedTargetsResponse{
		Data: responses,
		Pagination: dto.PaginationResponse{
			Page:       int(page),
			PageSize:   int(pageSize),
			TotalPages: totalPages,
			TotalCount: totalCount,
		},
	}

	log.Info().
		Int("targets.count", len(targets)).
		Int("total_count", totalCount).
		Msg("targets retrieved successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Update target item
// @Description Updates a target item by its ID
// @Tags Targets
// @Accept json
// @Produce json
// @Param id path int true "Target ID"
// @Param target body dto.UpdateTargetRequest true "Target update data"
// @Success 200 {object} dto.TargetResponse "Updated target item"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/targets/{id} [put]
func (h *TargetHandler) UpdateTarget(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("id", idStr).
			Msg("invalid target id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid target id: %s", idStr),
		})
		return
	}

	var req dto.UpdateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Int64("target.id", id).
			Msg("invalid request payload for target update")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("target.id", id).
		Msg("updating target")

	// Convert DTO to model for service
	updateRequest := mapper.UpdateTargetRequestToModel(&req)

	target, err := h.targetService.UpdateTarget(ctx, id, updateRequest)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("target.id", id).
			Msg("error while updating target")

		if errors.Is(err, errs.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("target with id `%d` not found", id),
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

	response := mapper.TargetModelToResponse(target)
	log.Info().
		Int64("target.id", id).
		Msg("target updated successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Delete target item
// @Description Deletes a target item by its ID (soft delete)
// @Tags Targets
// @Produce json
// @Param id path int true "Target ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/targets/{id} [delete]
func (h *TargetHandler) DeleteTarget(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("id", idStr).
			Msg("invalid target id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid target id: %s", idStr),
		})
		return
	}

	log.Info().
		Int64("target.id", id).
		Msg("deleting target")

	err = h.targetService.DeleteTarget(ctx, id)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("target.id", id).
			Msg("error while deleting target")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("target with id `%d` not found", id),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("target.id", id).
		Msg("target deleted successfully")

	c.Status(http.StatusNoContent)
}
