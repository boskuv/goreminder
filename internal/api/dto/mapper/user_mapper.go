package mapper

import (
	"time"

	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/models"
)

// CreateUserRequestToModel converts CreateUserRequest DTO to models.User
func CreateUserRequestToModel(req *dto.CreateUserRequest) *models.User {
	return &models.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: req.PasswordHash,
		Timezone:     req.Timezone,
		LanguageCode: req.LanguageCode,
		Role:         req.Role,
	}
}

// UpdateUserRequestToModel converts UpdateUserRequest DTO to models.UserUpdateRequest
func UpdateUserRequestToModel(req *dto.UpdateUserRequest) *models.UserUpdateRequest {
	return &models.UserUpdateRequest{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: req.PasswordHash,
		Timezone:     req.Timezone,
		LanguageCode: req.LanguageCode,
		Role:         req.Role,
	}
}

// UserModelToResponse converts models.User to UserResponse DTO
func UserModelToResponse(user *models.User) *dto.UserResponse {
	return &dto.UserResponse{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		Timezone:     user.Timezone,
		LanguageCode: user.LanguageCode,
		Role:         user.Role,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339),
	}
}
