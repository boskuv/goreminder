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

// MessengerHandler handles messenger-related HTTP requests
type MessengerHandler struct {
	Logger           zerolog.Logger
	MessengerService *service.MessengerService // TODO: capital letter?
}

// NewMessengerHandler creates a new MessengerHandler
func NewMessengerHandler(logger zerolog.Logger, messengerService *service.MessengerService) *MessengerHandler {
	return &MessengerHandler{
		Logger:           logger,
		MessengerService: messengerService,
	}
}

// @Summary Create a new messenger type
// @Description Creates a new messenger type
// @Tags Messengers
// @Accept json
// @Produce json
// @Param messenger body models.Messenger true "Messenger to create"
// @Success 201 {object} map[string]int64
// @Failure 400 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/messengers [post]
func (h *MessengerHandler) CreateMessenger(c *gin.Context) {
	var messenger models.Messenger
	if err := c.ShouldBindJSON(&messenger); err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "invalid input data")).Msg("Error while processing request with messenger struct parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid input data", http.StatusBadRequest))
		return
	}

	messengerID, err := h.MessengerService.CreateMessenger(&messenger)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while creating a messenger type")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"messenger_id": messengerID})
}

// @Summary Get messenger by ID
// @Description Retrieves a messenger by its ID
// @Tags Messengers
// @Produce json
// @Param id path int true "Messenger ID"
// @Success 200 {object} models.Messenger
// @Failure 400 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/messengers/{messenger_id} [get]
func (h *MessengerHandler) GetMessenger(c *gin.Context) {
	messengerID, err := strconv.ParseInt(c.Param("messenger_id"), 10, 64)
	if err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "failed to parse messengerID")).Msg("Error while processing request with id parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid messenger ID", http.StatusBadRequest))
		return
	}

	messenger, err := h.MessengerService.GetMessenger(messengerID)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while getting a messenger by its id")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, messenger)
}

// @Summary Get messenger ID by name
// @Description Retrieves a messenger ID by its name
// @Tags Messengers
// @Produce json
// @Param messenger_name path string true "Messenger name"
// @Success 200 {object} map[string]int64
// @Failure 400 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/messengers/by-name/{messenger_name} [get]
func (h *MessengerHandler) GetMessengerIDByName(c *gin.Context) {
	messengerName := c.Param("messenger_name")

	messengerID, err := h.MessengerService.GetMessengerIDByName(messengerName)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while getting a messenger ID by its name")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, messengerID)
}

// @Summary Сreate a new messenger-related user
// @Description Сreates a new messenger-related user
// @Tags Messengers
// @Accept json
// @Produce json
// @Param messenger body models.MessengerRelatedUser true "MessengerRelatedUser to create"
// @Success 201 {object} map[string]int64
// @Failure 400 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/messengerRelatedUsers [post]
func (h *MessengerHandler) CreateMessengerRelatedUser(c *gin.Context) {
	var messengerRelatedUser models.MessengerRelatedUser
	if err := c.ShouldBindJSON(&messengerRelatedUser); err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "invalid input data")).Msg("Error while processing request with messenger-related user struct parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid input data", http.StatusBadRequest))
		return
	}

	messengerRelatedUserID, err := h.MessengerService.CreateMessengerRelatedUser(&messengerRelatedUser)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while creating a messenger-related user")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"messenger_related_user_id": messengerRelatedUserID})
}

// @Summary Get messenger-related user by chatID, userID and messengerID
// @Description Retrieves a messenger-related user by chatID, userID and messengerID
// @Tags Messengers
// @Produce json
// @Param chat_id query string true "Chat ID"
// @Param user_id query int false "User ID"
// @Param messenger_id query int false "Messenger ID"
// @Success 200 {object} models.MessengerRelatedUser
// @Failure 400 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/messengerRelatedUsers [get]
func (h *MessengerHandler) GetMessengerRelatedUser(c *gin.Context) {
	chatID := c.Query("chat_id")

	userIDQuery, err := strconv.ParseInt(c.Query("user_id"), 10, 64)
	if err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "failed to parse userID")).Msg("Error while processing request with id parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid user ID", http.StatusBadRequest))
		return
	}
	userID := &userIDQuery

	messengerIDQuery, err := strconv.ParseInt(c.Query("messenger_id"), 10, 64)
	if err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "failed to parse messengerID")).Msg("Error while processing request with id parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid messenger ID", http.StatusBadRequest))
		return
	}
	messengerID := &messengerIDQuery

	messengerRelatedUser, err := h.MessengerService.GetMessengerRelatedUser(chatID, userID, messengerID)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while getting a messenger-related user")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, messengerRelatedUser)
}
