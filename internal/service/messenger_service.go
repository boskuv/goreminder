package service

import (
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"

	"github.com/pkg/errors"
)

// MessengerService handles messenger-related business logic
type MessengerService struct {
	messengerRepo repository.MessengerRepository
}

// NewMessengerService creates a new instance of MessengerService
func NewMessengerService(messengerRepo repository.MessengerRepository) *MessengerService {
	return &MessengerService{messengerRepo: messengerRepo}
}

// CreateMessenger creates a new messenger in the system
func (s *MessengerService) CreateMessenger(messenger *models.Messenger) (int64, error) {
	// Perform some validation before creating the messenger
	if messenger.Name == "" {
		return 0, errors.WithStack(errors.New("messenger data is incomplete"))
	}

	// Call the repository to insert the user into the database
	messengerID, err := s.messengerRepo.CreateMessenger(messenger)
	if err != nil {
		return 0, err
	}

	return messengerID, nil
}
