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
	userID, err := validation.ValidateInt64Param(c, "user_id")
	if err != nil {
		validation.HandleValidationError(c, err)
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
	userID, err := validation.ValidateInt64Param(c, "user_id")
	if err != nil {
		validation.HandleValidationError(c, err)
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
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/users/{user_id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID, err := validation.ValidateInt64Param(c, "user_id")
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while processing request with userID parameter")
		validation.HandleValidationError(c, err)
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
		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Get all users
// @Description Retrieves all users with pagination and ordering
// @Tags Users
// @Produce json
// @Param page query int false "Page number (default: 1)" default(1)
// @Param page_size query int false "Page size (default: 50)" default(50)
// @Param order_by query string false "Order by field (default: created_at DESC)" default(created_at DESC)
// @Success 200 {object} dto.PaginatedUsersResponse "Paginated list of users"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/users [get]
func (h *UserHandler) GetAllUsers(c *gin.Context) {
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
		Msg("getting all users")

	users, totalCount, err := h.userService.GetAllUsers(ctx, int(page), int(pageSize), orderBy)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error while getting all users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate total pages
	totalPages := (totalCount + int(pageSize) - 1) / int(pageSize)
	if totalPages == 0 {
		totalPages = 1
	}

	// Convert models to response DTOs
	responses := make([]dto.UserResponse, len(users))
	for i, user := range users {
		responses[i] = *mapper.UserModelToResponse(user)
	}

	response := dto.PaginatedUsersResponse{
		Data: responses,
		Pagination: dto.PaginationResponse{
			Page:       int(page),
			PageSize:   int(pageSize),
			TotalPages: totalPages,
			TotalCount: totalCount,
		},
	}

	log.Info().
		Int("users.count", len(users)).
		Int("total_count", totalCount).
		Msg("users retrieved successfully")

	c.JSON(http.StatusOK, response)
}
