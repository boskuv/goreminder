package service

import (
	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"

	"github.com/pkg/errors"
)

// MessengerService defines methods for messenger-related business logic
type MessengerService struct {
	messengerRepo repository.MessengerRepository
	userRepo      repository.UserRepository
}

// NewMessengerService creates a new instance of MessengerService
func NewMessengerService(messengerRepo repository.MessengerRepository, userRepo repository.UserRepository) *MessengerService {
	return &MessengerService{messengerRepo: messengerRepo, userRepo: userRepo}
}

// CreateMessenger implements BL of adding new messenger
func (s *MessengerService) CreateMessenger(messenger *models.Messenger) (int64, error) {
	// perform some validation before creating the messenger
	if messenger.Name == "" {
		return 0, errors.WithStack(errors.New("messenger data is incomplete"))
	}

	messengerID, err := s.messengerRepo.CreateMessenger(messenger)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return messengerID, nil
}

// GetMessenger implements BL of retrieving existing messenger by its id
func (s *MessengerService) GetMessenger(messengerID int64) (*models.Messenger, error) {
	messenger, err := s.messengerRepo.GetMessengerByID(messengerID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return messenger, nil
}

// GetMessengerIDByName implements BL of retrieving existing messenger by its name
func (s *MessengerService) GetMessengerIDByName(messengerName string) (int64, error) {
	messengerID, err := s.messengerRepo.GetMessengerIDByName(messengerName)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return messengerID, nil
}

// CreateMessengerRelatedUser implements BL of adding new messenger-related user
func (s *MessengerService) CreateMessengerRelatedUser(messengerRelatedUser *models.MessengerRelatedUser) (int64, error) {
	// perform some validation before creating the messenger-related user
	if messengerRelatedUser.MessengerUserID == "" || messengerRelatedUser.ChatID == "" || messengerRelatedUser.UserID == nil || messengerRelatedUser.MessengerID == nil {
		return 0, errors.WithStack(errors.New("messenger_user data is incomplete"))
	}

	// check if user and messenger exist
	_, err := s.userRepo.GetUserByID(*messengerRelatedUser.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}

		return 0, errors.WithStack(err)
	}

	_, err = s.messengerRepo.GetMessengerByID(*messengerRelatedUser.MessengerID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}

		return 0, errors.WithStack(err)
	}

	messengerRelatedUserID, err := s.messengerRepo.CreateMessengerRelatedUser(messengerRelatedUser)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return messengerRelatedUserID, nil
}

// GetMessengerRelatedUser implements BL of retrieving existing messenger-related user by chatID, messengerUserID, userID and messengerIDs
func (s *MessengerService) GetMessengerRelatedUser(chatID string, messengerUserID string, userID *int64, messengerID *int64) (*models.MessengerRelatedUser, error) {
	// check if user exists (only if userID is provided)
	if userID != nil {
		_, err := s.userRepo.GetUserByID(*userID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}

			return nil, errors.WithStack(err)
		}
	}

	// check if messenger exists (only if messengerID is provided)
	if messengerID != nil {
		_, err := s.messengerRepo.GetMessengerByID(*messengerID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}

			return nil, errors.WithStack(err)
		}
	}

	messengerRelatedUser, err := s.messengerRepo.GetMessengerRelatedUser(chatID, messengerUserID, userID, messengerID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return messengerRelatedUser, nil
}

// GetUserID implements BL of retrieving existing user by messengerUserID
// TODO: add messengerUD + messengerUserID 422
func (s *MessengerService) GetUserID(messengerUserID string) (int64, error) {
	userID, err := s.messengerRepo.GetUserID(messengerUserID)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return userID, nil
}
