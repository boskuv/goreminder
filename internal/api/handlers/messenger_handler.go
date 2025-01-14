package handlers

import (
	"net/http"

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
