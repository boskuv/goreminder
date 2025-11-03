package service

import (
	"context"
	"errors"
	"io"
	"testing"

	mock_repository "github.com/boskuv/goreminder/internal/mocks/repository"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMessengerService_CreateMessenger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	svc := NewMessengerService(mockMessengerRepo, mockUserRepo, testLogger)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		messenger := &models.Messenger{Name: "Telegram"}
		mockMessengerRepo.EXPECT().CreateMessenger(gomock.Any(), messenger).Return(int64(1), nil)
		id, err := svc.CreateMessenger(ctx, messenger)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		messenger := &models.Messenger{Name: ""}
		id, err := svc.CreateMessenger(ctx, messenger)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
		assert.Contains(t, err.Error(), "messenger data is incomplete")
	})

	t.Run("repo error", func(t *testing.T) {
		messenger := &models.Messenger{Name: "Telegram"}
		mockMessengerRepo.EXPECT().CreateMessenger(gomock.Any(), messenger).Return(int64(0), errors.New("db error"))
		id, err := svc.CreateMessenger(ctx, messenger)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})
}

func TestMessengerService_GetMessenger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	svc := NewMessengerService(mockMessengerRepo, mockUserRepo, testLogger)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		messenger := &models.Messenger{ID: 1, Name: "Telegram"}
		mockMessengerRepo.EXPECT().GetMessengerByID(gomock.Any(), int64(1)).Return(messenger, nil)
		result, err := svc.GetMessenger(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, messenger, result)
	})

	t.Run("repo error", func(t *testing.T) {
		mockMessengerRepo.EXPECT().GetMessengerByID(gomock.Any(), int64(2)).Return(nil, errors.New("not found"))
		result, err := svc.GetMessenger(ctx, 2)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestMessengerService_GetMessengerIDByName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	svc := NewMessengerService(mockMessengerRepo, mockUserRepo, testLogger)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		messengerName := "Telegram"
		mockMessengerRepo.EXPECT().GetMessengerIDByName(gomock.Any(), messengerName).Return(int64(1), nil)
		id, err := svc.GetMessengerIDByName(ctx, messengerName)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	t.Run("repo error", func(t *testing.T) {
		messengerName := "NonExistent"
		mockMessengerRepo.EXPECT().GetMessengerIDByName(gomock.Any(), messengerName).Return(int64(0), errors.New("not found"))
		id, err := svc.GetMessengerIDByName(ctx, messengerName)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})
}

func TestMessengerService_CreateMessengerRelatedUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	svc := NewMessengerService(mockMessengerRepo, mockUserRepo, testLogger)
	ctx := context.Background()

	t.Run("success - with all fields", func(t *testing.T) {
		userID := int64(1)
		messengerID := int64(1)
		messengerRelatedUser := &models.MessengerRelatedUser{
			UserID:          &userID,
			MessengerID:     &messengerID,
			MessengerUserID: "user123",
			ChatID:          "chat456",
		}

		// Mock user existence check
		mockUserRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
		// Mock messenger existence check
		mockMessengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID}, nil)
		// Mock creation
		mockMessengerRepo.EXPECT().CreateMessengerRelatedUser(gomock.Any(), messengerRelatedUser).Return(int64(1), nil)

		id, err := svc.CreateMessengerRelatedUser(ctx, messengerRelatedUser)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	t.Run("validation error - missing messengerUserID", func(t *testing.T) {
		userID := int64(1)
		messengerID := int64(1)
		messengerRelatedUser := &models.MessengerRelatedUser{
			UserID:      &userID,
			MessengerID: &messengerID,
			ChatID:      "chat456",
			// Missing MessengerUserID
		}

		id, err := svc.CreateMessengerRelatedUser(ctx, messengerRelatedUser)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
		assert.Contains(t, err.Error(), "messenger_user data is incomplete")
	})

	t.Run("validation error - missing chatID", func(t *testing.T) {
		userID := int64(1)
		messengerID := int64(1)
		messengerRelatedUser := &models.MessengerRelatedUser{
			UserID:          &userID,
			MessengerID:     &messengerID,
			MessengerUserID: "user123",
			// Missing ChatID
		}

		id, err := svc.CreateMessengerRelatedUser(ctx, messengerRelatedUser)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
		assert.Contains(t, err.Error(), "messenger_user data is incomplete")
	})

	t.Run("validation error - missing userID", func(t *testing.T) {
		messengerID := int64(1)
		messengerRelatedUser := &models.MessengerRelatedUser{
			MessengerID:     &messengerID,
			MessengerUserID: "user123",
			ChatID:          "chat456",
			// Missing UserID
		}

		id, err := svc.CreateMessengerRelatedUser(ctx, messengerRelatedUser)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
		assert.Contains(t, err.Error(), "messenger_user data is incomplete")
	})

	t.Run("validation error - missing messengerID", func(t *testing.T) {
		userID := int64(1)
		messengerRelatedUser := &models.MessengerRelatedUser{
			UserID:          &userID,
			MessengerUserID: "user123",
			ChatID:          "chat456",
			// Missing MessengerID
		}

		id, err := svc.CreateMessengerRelatedUser(ctx, messengerRelatedUser)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
		assert.Contains(t, err.Error(), "messenger_user data is incomplete")
	})

	t.Run("user not found error", func(t *testing.T) {
		userID := int64(999)
		messengerID := int64(1)
		messengerRelatedUser := &models.MessengerRelatedUser{
			UserID:          &userID,
			MessengerID:     &messengerID,
			MessengerUserID: "user123",
			ChatID:          "chat456",
		}

		// Mock user not found
		mockUserRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(nil, errors.New("user not found"))

		id, err := svc.CreateMessengerRelatedUser(ctx, messengerRelatedUser)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})

	t.Run("messenger not found error", func(t *testing.T) {
		userID := int64(1)
		messengerID := int64(999)
		messengerRelatedUser := &models.MessengerRelatedUser{
			UserID:          &userID,
			MessengerID:     &messengerID,
			MessengerUserID: "user123",
			ChatID:          "chat456",
		}

		// Mock user exists
		mockUserRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
		// Mock messenger not found
		mockMessengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(nil, errors.New("messenger not found"))

		id, err := svc.CreateMessengerRelatedUser(ctx, messengerRelatedUser)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})

	t.Run("creation repo error", func(t *testing.T) {
		userID := int64(1)
		messengerID := int64(1)
		messengerRelatedUser := &models.MessengerRelatedUser{
			UserID:          &userID,
			MessengerID:     &messengerID,
			MessengerUserID: "user123",
			ChatID:          "chat456",
		}

		// Mock user exists
		mockUserRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
		// Mock messenger exists
		mockMessengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID}, nil)
		// Mock creation error
		mockMessengerRepo.EXPECT().CreateMessengerRelatedUser(gomock.Any(), messengerRelatedUser).Return(int64(0), errors.New("db error"))

		id, err := svc.CreateMessengerRelatedUser(ctx, messengerRelatedUser)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})
}

func TestMessengerService_GetMessengerRelatedUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	svc := NewMessengerService(mockMessengerRepo, mockUserRepo, testLogger)
	ctx := context.Background()

	t.Run("success with all parameters", func(t *testing.T) {
		userID := int64(1)
		messengerID := int64(1)
		expectedUser := &models.MessengerRelatedUser{
			ID:              1,
			UserID:          &userID,
			MessengerID:     &messengerID,
			MessengerUserID: "user123",
			ChatID:          "chat456",
		}

		// Mock user existence check
		mockUserRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
		// Mock messenger existence check
		mockMessengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID}, nil)
		// Mock retrieval
		mockMessengerRepo.EXPECT().GetMessengerRelatedUser(gomock.Any(), "chat456", "user123", &userID, &messengerID).Return(expectedUser, nil)

		result, err := svc.GetMessengerRelatedUser(ctx, "chat456", "user123", &userID, &messengerID)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, result)
	})

	t.Run("success with nil pointers", func(t *testing.T) {
		expectedUser := &models.MessengerRelatedUser{
			ID:              1,
			UserID:          nil,
			MessengerID:     nil,
			MessengerUserID: "user123",
			ChatID:          "chat456",
		}

		// When userID and messengerID are nil, no validation checks are performed
		mockMessengerRepo.EXPECT().GetMessengerRelatedUser(gomock.Any(), "chat456", "user123", nil, nil).Return(expectedUser, nil)

		result, err := svc.GetMessengerRelatedUser(ctx, "chat456", "user123", nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, result)
	})

	t.Run("user not found error", func(t *testing.T) {
		userID := int64(999)
		messengerID := int64(1)

		// Mock user not found
		mockUserRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(nil, errors.New("user not found"))

		result, err := svc.GetMessengerRelatedUser(ctx, "chat456", "user123", &userID, &messengerID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("messenger not found error", func(t *testing.T) {
		userID := int64(1)
		messengerID := int64(999)

		// Mock user exists
		mockUserRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
		// Mock messenger not found
		mockMessengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(nil, errors.New("messenger not found"))

		result, err := svc.GetMessengerRelatedUser(ctx, "chat456", "user123", &userID, &messengerID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("repo error", func(t *testing.T) {
		userID := int64(1)
		messengerID := int64(1)

		// Mock user exists
		mockUserRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
		// Mock messenger exists
		mockMessengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID}, nil)
		// Mock retrieval error
		mockMessengerRepo.EXPECT().GetMessengerRelatedUser(gomock.Any(), "chat456", "user123", &userID, &messengerID).Return(nil, errors.New("not found"))

		result, err := svc.GetMessengerRelatedUser(ctx, "chat456", "user123", &userID, &messengerID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestMessengerService_GetUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	svc := NewMessengerService(mockMessengerRepo, mockUserRepo, testLogger)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		messengerUserID := "user123"
		mockMessengerRepo.EXPECT().GetUserID(gomock.Any(), messengerUserID).Return(int64(1), nil)
		userID, err := svc.GetUserID(ctx, messengerUserID)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), userID)
	})

	t.Run("repo error", func(t *testing.T) {
		messengerUserID := "nonexistent"
		mockMessengerRepo.EXPECT().GetUserID(gomock.Any(), messengerUserID).Return(int64(0), errors.New("not found"))
		userID, err := svc.GetUserID(ctx, messengerUserID)
		assert.Error(t, err)
		assert.Equal(t, int64(0), userID)
	})
}

func TestMessengerService_NewMessengerService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	svc := NewMessengerService(mockMessengerRepo, mockUserRepo, testLogger)

	assert.NotNil(t, svc)
	assert.Equal(t, mockMessengerRepo, svc.messengerRepo)
	assert.Equal(t, mockUserRepo, svc.userRepo)
}
