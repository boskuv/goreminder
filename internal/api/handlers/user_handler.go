package handlers

import (
	"net/http"
	"strconv"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type UserHandler struct {
	Logger      zerolog.Logger
	userService *service.UserService
}

func NewUserHandler(logger zerolog.Logger, userService *service.UserService) *UserHandler {
	return &UserHandler{
		Logger:      logger,
		userService: userService,
	}
}

// CreateUser handles creating a new user
// @Summary Create a new user
// @Description Creates a new user in the system
// @Tags Users
// @Accept json
// @Produce json
// @Param user body models.User true "User data"
// @Success 201 {object} map[string]int64
// @Failure 400 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Example { "id": 1, "name": "John Doe", "email": "john.doe@example.com", "password": "password123", "created_at": "2024-11-27T10:00:00Z" }
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "invalid request payload")).Msg("Error while processing request with user struct parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	createdUser, err := h.userService.CreateUser(&user)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while creating user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, createdUser)
}

// @Summary Get user by userID
// @Description Retrieves user by userID
// @Tags Users
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} models.User
// @Failure 400 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/users/{user_id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "failed to parse userID")).Msg("Error while processing request with userID parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid user ID", http.StatusBadRequest))
		return
	}

	user, err := h.userService.GetUser(userID)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while getting user by userID parameter")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, user)
}

// @Summary Update a user
// @Description Updates a user by its ID
// @Tags Users
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Param user body models.UserUpdateRequest true "User update details"
// @Success 200 {object} models.User
// @Failure 400 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/users/{user_id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "failed to parse userID")).Msg("Error while processing request with userID parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid user ID", http.StatusBadRequest))
		return
	}

	var userUpdateRequest models.UserUpdateRequest
	if err := c.ShouldBindJSON(&userUpdateRequest); err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "invalid input data")).Msg("Error while processing request with userUpdateRequest struct parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid input data", http.StatusBadRequest))
		return
	}

	updatedUser, err := h.userService.UpdateUser(userID, &userUpdateRequest)
	if err != nil {
		h.Logger.Error().Stack().Err(err).Msg("Error while updating a user")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

// @Summary Soft delete a user
// @Description Marks a user as deleted by its ID (soft delete)
// @Tags Users
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Success 204 {object} models.APIError
// @Failure 400 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/users/{user_id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		h.Logger.Error().Stack().Err(errors.Wrap(err, "failed to parse userID")).Msg("Error while processing request with userID parameter")
		c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid user ID", http.StatusBadRequest))
		return
	}

	err = h.userService.DeleteUser(userID)
	if err != nil {

		h.Logger.Error().Stack().Err(err).Msg("Error while deleting a user")
		c.JSON(http.StatusInternalServerError, models.HTTPError(err, http.StatusInternalServerError))
		return
	}

	c.Status(http.StatusNoContent) // 204 No Content status for successful deletion
}
