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
// @Success 201 {object} map[string]int64 "Created messenger ID"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
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
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Messenger not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/messengers/{messenger_id} [get]
func (h *MessengerHandler) GetMessenger(c *gin.Context) {
	messengerID, err := strconv.ParseInt(c.Param("messenger_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
// @Success 200 {object} map[string]int64 "Messenger ID"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Messenger not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/messengers/by-name/{messenger_name} [get]
func (h *MessengerHandler) GetMessengerIDByName(c *gin.Context) {
	messengerName := c.Param("messenger_name")

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
// @Success 201 {object} map[string]int64 "Created messenger-related user ID"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
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
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Messenger-related user not found"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/messengerRelatedUsers [get]
func (h *MessengerHandler) GetMessengerRelatedUser(c *gin.Context) {
	chatID := c.Query("chat_id")
	messengerUserID := c.Query("messenger_user_id")

	var userID *int64
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		userIDQuery, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id parameter"})
			return
		}
		userID = &userIDQuery
	}

	var messengerID *int64
	if messengerIDStr := c.Query("messenger_id"); messengerIDStr != "" {
		messengerIDQuery, err := strconv.ParseInt(messengerIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid messenger_id parameter"})
			return
		}
		messengerID = &messengerIDQuery
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
// @Success 200 {object} map[string]int64 "User ID"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/messengerRelatedUsers/{messenger_user_id}/user [get]
func (h *MessengerHandler) GetUserID(c *gin.Context) {
	messengerUserID := c.Param("messenger_user_id")

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
