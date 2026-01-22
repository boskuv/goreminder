package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

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

// DigestHandler handles digest-related HTTP requests
type DigestHandler struct {
	logger        zerolog.Logger
	digestService *service.DigestService
}

// NewDigestHandler creates a new DigestHandler
func NewDigestHandler(digestService *service.DigestService, logger zerolog.Logger) *DigestHandler {
	return &DigestHandler{
		logger:        logger,
		digestService: digestService,
	}
}

// @Summary Create digest settings
// @Description Creates new digest settings for a user
// @Tags Digests
// @Accept json
// @Produce json
// @Param settings body dto.CreateDigestSettingsRequest true "Digest settings to create"
// @Success 201 {object} dto.CreateResponse "Created digest settings ID"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 422 {object} dto.ErrorResponse "Unprocessable entity"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/digests/settings [post]
func (h *DigestHandler) CreateDigestSettings(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var req dto.CreateDigestSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Msg("invalid request payload for digest settings creation")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("user.id", req.UserID).
		Msg("creating digest settings")

	// Convert DTO to model for service
	settingsModel := mapper.CreateDigestSettingsRequestToModel(&req)

	settingsID, err := h.digestService.CreateDigestSettings(ctx, settingsModel)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", req.UserID).
			Msg("error while creating digest settings")

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
		Int64("digest_settings.id", settingsID).
		Int64("user.id", req.UserID).
		Msg("digest settings created successfully")

	c.JSON(http.StatusCreated, gin.H{"id": settingsID})
}

// @Summary Get digest settings
// @Description Retrieves digest settings for a user
// @Tags Digests
// @Produce json
// @Param user_id query int true "User ID"
// @Param messenger_related_user_id query int false "Messenger Related User ID"
// @Success 200 {object} dto.DigestSettingsResponse "Digest settings"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 404 {object} dto.ErrorResponse "Not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/digests/settings [get]
func (h *DigestHandler) GetDigestSettings(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := validation.ValidateInt64Query(c, "user_id", 0, 1)
	if err != nil {
		log.Info().Err(err).Msg("invalid user_id query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	var messengerRelatedUserID *int
	mruIDStr := c.Query("messenger_related_user_id")
	if mruIDStr != "" {
		mruID, err := strconv.Atoi(mruIDStr)
		if err != nil {
			log.Info().Err(err).Msg("invalid messenger_related_user_id query parameter")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid messenger_related_user_id parameter",
			})
			return
		}
		messengerRelatedUserID = &mruID
	}

	log.Info().
		Int64("user.id", userID).
		Msg("getting digest settings")

	settings, err := h.digestService.GetDigestSettings(ctx, userID, messengerRelatedUserID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", userID).
			Msg("error while getting digest settings")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("digest settings for user id `%d` not found", userID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := mapper.DigestSettingsModelToResponse(settings)
	log.Info().
		Int64("user.id", userID).
		Msg("digest settings retrieved successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Update digest settings
// @Description Updates digest settings for a user
// @Tags Digests
// @Accept json
// @Produce json
// @Param user_id query int true "User ID"
// @Param messenger_related_user_id query int false "Messenger Related User ID"
// @Param settings body dto.UpdateDigestSettingsRequest true "Digest settings update data"
// @Success 200 {object} dto.DigestSettingsResponse "Updated digest settings"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 404 {object} dto.ErrorResponse "Not found"
// @Failure 422 {object} dto.ErrorResponse "Unprocessable entity"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/digests/settings [put]
func (h *DigestHandler) UpdateDigestSettings(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := validation.ValidateInt64Query(c, "user_id", 0, 1)
	if err != nil {
		log.Info().Err(err).Msg("invalid user_id query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	var messengerRelatedUserID *int
	mruIDStr := c.Query("messenger_related_user_id")
	if mruIDStr != "" {
		mruID, err := strconv.Atoi(mruIDStr)
		if err != nil {
			log.Info().Err(err).Msg("invalid messenger_related_user_id query parameter")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid messenger_related_user_id parameter",
			})
			return
		}
		messengerRelatedUserID = &mruID
	}

	var req dto.UpdateDigestSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Int64("user.id", userID).
			Msg("invalid request payload for digest settings update")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("user.id", userID).
		Msg("updating digest settings")

	// Convert DTO to model for service
	updateRequest := mapper.UpdateDigestSettingsRequestToModel(&req)

	settings, err := h.digestService.UpdateDigestSettings(ctx, userID, messengerRelatedUserID, updateRequest)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", userID).
			Msg("error while updating digest settings")

		if errors.Is(err, errs.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("digest settings for user id `%d` not found", userID),
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

	response := mapper.DigestSettingsModelToResponse(settings)
	log.Info().
		Int64("user.id", userID).
		Msg("digest settings updated successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Get digest
// @Description Generates a digest for a user with statistics and tasks
// @Tags Digests
// @Produce json
// @Param user_id query int true "User ID"
// @Param messenger_related_user_id query int false "Messenger Related User ID"
// @Param start_date_from query string false "Filter by start_date from (RFC3339 format, inclusive)"
// @Param start_date_to query string false "Filter by start_date to (RFC3339 format, inclusive)"
// @Success 200 {object} dto.DigestResponse "Digest data"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 422 {object} dto.ErrorResponse "Unprocessable entity"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/digests [get]
func (h *DigestHandler) GetDigest(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := validation.ValidateInt64Query(c, "user_id", 0, 1)
	if err != nil {
		log.Info().Err(err).Msg("invalid user_id query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	var messengerRelatedUserID *int
	mruIDStr := c.Query("messenger_related_user_id")
	if mruIDStr != "" {
		mruID, err := strconv.Atoi(mruIDStr)
		if err != nil {
			log.Info().Err(err).Msg("invalid messenger_related_user_id query parameter")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid messenger_related_user_id parameter",
			})
			return
		}
		messengerRelatedUserID = &mruID
	}

	// Optional date filters
	startDateFrom, err := validation.ValidateOptionalTimeQuery(c, "start_date_from")
	if err != nil {
		log.Info().Err(err).Msg("invalid start_date_from query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	startDateTo, err := validation.ValidateOptionalTimeQuery(c, "start_date_to")
	if err != nil {
		log.Info().Err(err).Msg("invalid start_date_to query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	// Validate date range if both provided
	if startDateFrom != nil && startDateTo != nil {
		if startDateFrom.After(*startDateTo) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "start_date_from must be before or equal to start_date_to",
			})
			return
		}

		// Validate date range is not too large (e.g., max 1 year)
		maxRange := 365 * 24 * time.Hour
		if startDateTo.Sub(*startDateFrom) > maxRange {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "date range cannot exceed 365 days",
			})
			return
		}
	}

	log.Info().
		Int64("user.id", userID).
		Msg("generating digest")

	digest, err := h.digestService.GetDigest(ctx, userID, messengerRelatedUserID, startDateFrom, startDateTo)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", userID).
			Msg("error while generating digest")

		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := mapper.DigestServiceResponseToDTO(digest)
	log.Info().
		Int64("user.id", userID).
		Int("completed_backlogs_count", digest.CompletedBacklogsCount).
		Int("tasks.count", len(digest.Tasks)).
		Msg("digest generated successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Get all digest settings
// @Description Retrieves all digest settings with pagination, ordering, and filtering
// @Tags Digests
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(50)
// @Param order_by query string false "Order by" default(created_at DESC)
// @Param user_id query int false "Filter by user ID"
// @Success 200 {object} dto.PaginatedDigestSettingsResponse "Paginated digest settings"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/digests/settings/all [get]
func (h *DigestHandler) GetAllDigestSettings(c *gin.Context) {
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
		Msg("getting all digest settings")

	settings, totalCount, err := h.digestService.GetAllDigestSettings(ctx, int(page), int(pageSize), orderBy, userID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Msg("error while getting all digest settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate total pages
	totalPages := (totalCount + int(pageSize) - 1) / int(pageSize)
	if totalPages == 0 {
		totalPages = 1
	}

	// Convert models to response DTOs
	responses := mapper.DigestSettingsModelsToResponse(settings)

	response := dto.PaginatedDigestSettingsResponse{
		Data: responses,
		Pagination: dto.PaginationResponse{
			Page:       int(page),
			PageSize:   int(pageSize),
			TotalPages: totalPages,
			TotalCount: totalCount,
		},
	}

	log.Info().
		Int("settings.count", len(settings)).
		Int("total_count", totalCount).
		Msg("digest settings retrieved successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Delete digest settings
// @Description Deletes digest settings for a user
// @Tags Digests
// @Produce json
// @Param user_id query int true "User ID"
// @Param messenger_related_user_id query int false "Messenger Related User ID"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 404 {object} dto.ErrorResponse "Not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/digests/settings [delete]
func (h *DigestHandler) DeleteDigestSettings(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := validation.ValidateInt64Query(c, "user_id", 0, 1)
	if err != nil {
		log.Info().Err(err).Msg("invalid user_id query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	var messengerRelatedUserID *int
	mruIDStr := c.Query("messenger_related_user_id")
	if mruIDStr != "" {
		mruID, err := strconv.Atoi(mruIDStr)
		if err != nil {
			log.Info().Err(err).Msg("invalid messenger_related_user_id query parameter")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid messenger_related_user_id parameter",
			})
			return
		}
		messengerRelatedUserID = &mruID
	}

	log.Info().
		Int64("user.id", userID).
		Msg("deleting digest settings")

	err = h.digestService.DeleteDigestSettings(ctx, userID, messengerRelatedUserID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", userID).
			Msg("error while deleting digest settings")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("digest settings for user id `%d` not found", userID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("user.id", userID).
		Msg("digest settings deleted successfully")

	c.Status(http.StatusNoContent)
}
