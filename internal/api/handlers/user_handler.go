package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	logger      zerolog.Logger
	userService *service.UserService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService *service.UserService, logger zerolog.Logger) *UserHandler {
	return &UserHandler{
		logger:      logger,
		userService: userService,
	}
}

// @Summary Create a new user
// @Description Creates a new user in the system
// @Tags Users
// @Accept json
// @Produce json
// @Param user body models.User true "User to create"
// @Success 201 {object} map[string]int64
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		log.Error().
			Err(err).
			Msg("invalid request payload for user creation")

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := h.userService.CreateUser(c.Request.Context(), &user)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while adding new user")

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, userID)
}

// @Summary Get user by userID
// @Description Retrieves user by userID
// @Tags Users
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{user_id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		// h.logger.Error().Stack().Err(errors.Wrap(err, "failed to parse userID")).Msg("Error while processing request with userID parameter")
		// c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid user ID", http.StatusBadRequest))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while getting user by its id")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("user with id `%d` not found", userID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// @Summary Update a user
// @Description Updates a user by ID
// @Tags Users
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Param user body models.UserUpdateRequest true "User update details"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{user_id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		// h.logger.Error().Stack().Err(errors.Wrap(err, "failed to parse userID")).Msg("Error while processing request with userID parameter")
		// c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid user ID", http.StatusBadRequest))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userUpdateRequest models.UserUpdateRequest
	if err := c.ShouldBindJSON(&userUpdateRequest); err != nil {
		// h.logger.Error().Stack().Err(errors.Wrap(err, "invalid input data")).Msg("Error while processing request with userUpdateRequest struct parameter")
		// c.JSON(http.StatusBadRequest, models.NewAPIError("Invalid input data", http.StatusBadRequest))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedUser, err := h.userService.UpdateUser(c.Request.Context(), userID, &userUpdateRequest)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while updating user")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("user with id `%d` not found", userID),
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
	}

	c.JSON(http.StatusOK, updatedUser)
}

// @Summary Soft delete a user
// @Description Marks a user as deleted by its ID (soft delete)
// @Tags Users
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Success 204 {object} nil
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{user_id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		// TODO: wrap?
		h.logger.Error().Stack().Err(errors.Wrap(err, "failed to parse userID")).Msg("error while processing request with userID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.userService.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while soft deleting user")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("user with id `%d` not found", userID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
