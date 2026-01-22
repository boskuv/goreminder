package handlers

import (
	"fmt"
	"net/http"

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

// MessengerHandler handles user-related HTTP requests
type MessengerHandler struct {
	logger           zerolog.Logger
	messengerService *service.MessengerService
}

// NewMessengerHandler creates a new MessengerHandler
func NewMessengerHandler(messengerService *service.MessengerService, logger zerolog.Logger) *MessengerHandler {
	return &MessengerHandler{
		logger:           logger,
		messengerService: messengerService,
	}
}

// @Summary Create a new messenger type
// @Description Creates a new messenger type
// @Tags Messengers
// @Accept json
// @Produce json
// @Param messenger body dto.CreateMessengerRequest true "Messenger to create"
// @Success 201 {object} dto.CreateResponse "Created messenger ID"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/messengers [post]
func (h *MessengerHandler) CreateMessenger(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var req dto.CreateMessengerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().
			Err(err).
			Msg("invalid request payload for messenger creation")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert DTO to model for service
	messengerModel := mapper.CreateMessengerRequestToModel(&req)

	messengerID, err := h.messengerService.CreateMessenger(c.Request.Context(), messengerModel)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while adding new messenger type")

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": messengerID})
}

// @Summary Get messenger by ID
// @Description Retrieves a messenger by its ID
// @Tags Messengers
// @Produce json
// @Param id path int true "Messenger ID"
// @Success 200 {object} dto.MessengerResponse "Messenger details"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 404 {object} dto.ErrorResponse "Messenger not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/messengers/{messenger_id} [get]
func (h *MessengerHandler) GetMessenger(c *gin.Context) {
	messengerID, err := validation.ValidateInt64Param(c, "messenger_id")
	if err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	messenger, err := h.messengerService.GetMessenger(c.Request.Context(), messengerID)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while getting messenger by its id")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("messenger with id `%d` not found", messengerID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert model to response DTO
	response := mapper.MessengerModelToResponse(messenger)
	c.JSON(http.StatusOK, response)
}

// @Summary Get messenger ID by name
// @Description Retrieves a messenger ID by its name
// @Tags Messengers
// @Produce json
// @Param messenger_name path string true "Messenger name"
// @Success 200 {object} dto.IDResponse "Messenger ID"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 404 {object} dto.ErrorResponse "Messenger not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/messengers/by-name/{messenger_name} [get]
func (h *MessengerHandler) GetMessengerIDByName(c *gin.Context) {
	messengerName, err := validation.ValidateStringParam(c, "messenger_name", true)
	if err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	messengerID, err := h.messengerService.GetMessengerIDByName(c.Request.Context(), messengerName)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while getting messenger by its name")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("messenger with name `%s` not found", messengerName),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": messengerID})
}

// @Summary Create a new messenger-related user
// @Description Creates a new messenger-related user
// @Tags Messengers
// @Accept json
// @Produce json
// @Param messenger body dto.CreateMessengerRelatedUserRequest true "MessengerRelatedUser to create"
// @Success 201 {object} dto.CreateResponse "Created messenger-related user ID"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 422 {object} dto.ErrorResponse "Unprocessable entity"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/messengerRelatedUsers [post]
func (h *MessengerHandler) CreateMessengerRelatedUser(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var req dto.CreateMessengerRelatedUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().
			Err(err).
			Msg("invalid request payload for messenger-related user creation")

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert DTO to model for service
	messengerRelatedUserModel := mapper.CreateMessengerRelatedUserRequestToModel(&req)

	messengerRelatedUserID, err := h.messengerService.CreateMessengerRelatedUser(c.Request.Context(), messengerRelatedUserModel)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while creating a messenger-related user")

		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": messengerRelatedUserID})
}

// @Summary Get messenger-related user by chatID, messengerUserID, userID and messengerID
// @Description Retrieves a messenger-related user by chatID, messengerUserID, userID and messengerID
// @Tags Messengers
// @Produce json
// @Param chat_id query string true "Chat ID"
// @Param messenger_user_id query string true "Messenger User ID"
// @Param user_id query int false "User ID"
// @Param messenger_id query int false "Messenger ID"
// @Success 200 {object} dto.MessengerRelatedUserResponse "Messenger-related user details"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 404 {object} dto.ErrorResponse "Messenger-related user not found"
// @Failure 422 {object} dto.ErrorResponse "Unprocessable entity"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/messengerRelatedUsers [get]
func (h *MessengerHandler) GetMessengerRelatedUser(c *gin.Context) {
	chatID, err := validation.ValidateStringQuery(c, "chat_id", true)
	if err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	messengerUserID, err := validation.ValidateStringQuery(c, "messenger_user_id", true)
	if err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	userID, err := validation.ValidateOptionalInt64Query(c, "user_id", 1)
	if err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	messengerID, err := validation.ValidateOptionalInt64Query(c, "messenger_id", 1)
	if err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	messengerRelatedUser, err := h.messengerService.GetMessengerRelatedUser(c.Request.Context(), chatID, messengerUserID, userID, messengerID)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while getting a messenger-related user")

		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert model to response DTO
	response := mapper.MessengerRelatedUserModelToResponse(messengerRelatedUser)
	c.JSON(http.StatusOK, response)
}

// GetUserID retrieves a userID user by messengerUserID
// @Summary Get a userID by messengerUserID
// @Description Retrieves a userID by messengerUserID
// @Tags Messengers
// @Produce json
// @Param messenger_user_id path string true "Messenger UserID"
// @Success 200 {object} dto.UserIDResponse "User ID"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 404 {object} dto.ErrorResponse "User not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/messengerRelatedUsers/{messenger_user_id}/user [get]
func (h *MessengerHandler) GetUserID(c *gin.Context) {
	messengerUserID, err := validation.ValidateStringParam(c, "messenger_user_id", true)
	if err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	userID, err := h.messengerService.GetUserID(c.Request.Context(), messengerUserID)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while getting a userID")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_id": userID})
}

// @Summary Get all messengers
// @Description Retrieves all messengers with pagination and ordering
// @Tags Messengers
// @Produce json
// @Param page query int false "Page number (default: 1)" default(1)
// @Param page_size query int false "Page size (default: 50)" default(50)
// @Param order_by query string false "Order by field (default: created_at DESC)" default(created_at DESC)
// @Success 200 {object} dto.PaginatedMessengersResponse "Paginated list of messengers"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/messengers [get]
func (h *MessengerHandler) GetAllMessengers(c *gin.Context) {
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

	log.Info().
		Int64("page", page).
		Int64("page_size", pageSize).
		Str("order_by", orderBy).
		Msg("getting all messengers")

	messengers, totalCount, err := h.messengerService.GetAllMessengers(ctx, int(page), int(pageSize), orderBy)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while getting all messengers")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate total pages
	totalPages := (totalCount + int(pageSize) - 1) / int(pageSize)
	if totalPages == 0 {
		totalPages = 1
	}

	// Convert models to response DTOs
	responses := make([]dto.MessengerResponse, len(messengers))
	for i, messenger := range messengers {
		responses[i] = *mapper.MessengerModelToResponse(messenger)
	}

	response := dto.PaginatedMessengersResponse{
		Data: responses,
		Pagination: dto.PaginationResponse{
			Page:       int(page),
			PageSize:   int(pageSize),
			TotalPages: totalPages,
			TotalCount: totalCount,
		},
	}

	log.Info().
		Int("messengers.count", len(messengers)).
		Int("total_count", totalCount).
		Msg("messengers retrieved successfully")

	c.JSON(http.StatusOK, response)
}

// @Summary Get all messenger-related users
// @Description Retrieves all messenger-related users with pagination and ordering
// @Tags Messengers
// @Produce json
// @Param page query int false "Page number (default: 1)" default(1)
// @Param page_size query int false "Page size (default: 50)" default(50)
// @Param order_by query string false "Order by field (default: created_at DESC)" default(created_at DESC)
// @Success 200 {object} dto.PaginatedMessengerRelatedUsersResponse "Paginated list of messenger-related users"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/messengerRelatedUsers/all [get]
func (h *MessengerHandler) GetAllMessengerRelatedUsers(c *gin.Context) {
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

	log.Info().
		Int64("page", page).
		Int64("page_size", pageSize).
		Str("order_by", orderBy).
		Msg("getting all messenger-related users")

	messengerRelatedUsers, totalCount, err := h.messengerService.GetAllMessengerRelatedUsers(ctx, int(page), int(pageSize), orderBy)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while getting all messenger-related users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate total pages
	totalPages := (totalCount + int(pageSize) - 1) / int(pageSize)
	if totalPages == 0 {
		totalPages = 1
	}

	// Convert models to response DTOs
	responses := make([]dto.MessengerRelatedUserResponse, len(messengerRelatedUsers))
	for i, mru := range messengerRelatedUsers {
		responses[i] = *mapper.MessengerRelatedUserModelToResponse(mru)
	}

	response := dto.PaginatedMessengerRelatedUsersResponse{
		Data: responses,
		Pagination: dto.PaginationResponse{
			Page:       int(page),
			PageSize:   int(pageSize),
			TotalPages: totalPages,
			TotalCount: totalCount,
		},
	}

	log.Info().
		Int("messenger_related_users.count", len(messengerRelatedUsers)).
		Int("total_count", totalCount).
		Msg("messenger-related users retrieved successfully")

	c.JSON(http.StatusOK, response)
}
