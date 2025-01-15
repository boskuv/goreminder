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

// GetMessenger retrieves a messenger by its ID
func (s *MessengerService) GetMessenger(messengerID int64) (*models.Messenger, error) {
	messenger, err := s.messengerRepo.GetMessengerByID(messengerID)
	if err != nil {
		return nil, err
	}

	return messenger, nil
}

// GetMessengerIDByName retrieves a messenger ID by its name
func (s *MessengerService) GetMessengerIDByName(messengerName string) (int64, error) {
	messengerID, err := s.messengerRepo.GetMessengerIDByName(messengerName)
	if err != nil {
		return 0, err
	}

	return messengerID, nil
}

// CreateMessengerRelatedUser creates a new messenger-related user in the system
func (s *MessengerService) CreateMessengerRelatedUser(messengerRelatedUser *models.MessengerRelatedUser) (int64, error) {
	// Perform some validation before creating the messenger
	// TODO: check if UserID and MessengerID exist
	if messengerRelatedUser.ChatID == "" && messengerRelatedUser.UserID == nil && messengerRelatedUser.MessengerID == nil {
		return 0, errors.WithStack(errors.New("messenger_user data is incomplete"))
	}

	// Call the repository to insert the messenger-related user into the database
	messengerRelatedUserID, err := s.messengerRepo.CreateMessengerRelatedUser(messengerRelatedUser)
	if err != nil {
		return 0, err
	}

	return messengerRelatedUserID, nil
}

// GetMessengerRelatedUser retrieves a messenger-related user by chatID, userID and messengerIDs
func (s *MessengerService) GetMessengerRelatedUser(chatID string, userID *int64, messengerID *int64) (*models.MessengerRelatedUser, error) {
	messengerRelatedUser, err := s.messengerRepo.GetMessengerRelatedUser(chatID, userID, messengerID)
	if err != nil {
		return nil, err
	}

	return messengerRelatedUser, nil
}
