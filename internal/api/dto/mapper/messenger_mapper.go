package mapper

import (
	"time"

	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/models"
)

// CreateMessengerRequestToModel converts CreateMessengerRequest DTO to models.Messenger
func CreateMessengerRequestToModel(req *dto.CreateMessengerRequest) *models.Messenger {
	return &models.Messenger{
		Name: req.Name,
	}
}

// MessengerModelToResponse converts models.Messenger to MessengerResponse DTO
func MessengerModelToResponse(messenger *models.Messenger) *dto.MessengerResponse {
	return &dto.MessengerResponse{
		Name:      messenger.Name,
		CreatedAt: messenger.CreatedAt.Format(time.RFC3339),
	}
}

// CreateMessengerRelatedUserRequestToModel converts CreateMessengerRelatedUserRequest DTO to models.MessengerRelatedUser
func CreateMessengerRelatedUserRequestToModel(req *dto.CreateMessengerRelatedUserRequest) *models.MessengerRelatedUser {
	return &models.MessengerRelatedUser{
		UserID:          req.UserID,
		MessengerID:     req.MessengerID,
		MessengerUserID: req.MessengerUserID,
		ChatID:          req.ChatID,
	}
}

// MessengerRelatedUserModelToResponse converts models.MessengerRelatedUser to MessengerRelatedUserResponse DTO
func MessengerRelatedUserModelToResponse(mru *models.MessengerRelatedUser) *dto.MessengerRelatedUserResponse {
	response := &dto.MessengerRelatedUserResponse{
		ID:              mru.ID,
		UserID:          mru.UserID,
		MessengerID:     mru.MessengerID,
		MessengerUserID: mru.MessengerUserID,
		ChatID:          mru.ChatID,
		CreatedAt:       mru.CreatedAt.Format(time.RFC3339),
	}
	if mru.UpdatedAt != nil {
		updatedAt := mru.UpdatedAt.Format(time.RFC3339)
		response.UpdatedAt = &updatedAt
	}
	return response
}
