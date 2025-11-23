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
// @Param user body dto.CreateUserRequest true "User to create"
// @Success 201 {object} map[string]int64 "Created user ID"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().
			Err(err).
			Msg("invalid request payload for user creation")

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert DTO to model for service
	userModel := mapper.CreateUserRequestToModel(&req)

	userID, err := h.userService.CreateUser(c.Request.Context(), userModel)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while adding new user")

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": userID})
}

// @Summary Get user by userID
// @Description Retrieves user by userID
// @Tags Users
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} dto.UserResponse "User details"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/users/{user_id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
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

	// Convert model to response DTO
	response := mapper.UserModelToResponse(user)
	c.JSON(http.StatusOK, response)
}

// @Summary Update a user
// @Description Updates a user by ID
// @Tags Users
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Param user body dto.UpdateUserRequest true "User update details"
// @Success 200 {object} dto.UserResponse "Updated user"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/users/{user_id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert DTO to model update request
	updateRequest := mapper.UpdateUserRequestToModel(&req)

	updatedUser, err := h.userService.UpdateUser(c.Request.Context(), userID, updateRequest)
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
		return
	}

	// Convert model to response DTO
	response := mapper.UserModelToResponse(updatedUser)
	c.JSON(http.StatusOK, response)
}

// @Summary Soft delete a user
// @Description Marks a user as deleted by its ID (soft delete)
// @Tags Users
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/users/{user_id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
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
