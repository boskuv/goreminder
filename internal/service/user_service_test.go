package service

import (
	"context"
	"errors"
	"testing"

	mock_repository "github.com/boskuv/goreminder/internal/mocks/repository"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/queue"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUserService_CreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_repository.NewMockUserRepository(ctrl)
	taskRepo := mock_repository.NewMockTaskRepository(ctrl)
	messengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	producer := &queue.Producer{}
	svc := NewUserService(userRepo, taskRepo, messengerRepo, producer)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		user := &models.User{Name: "John", Email: "john@example.com", PasswordHash: "hash"}
		userRepo.EXPECT().CreateUser(gomock.Any(), user).Return(int64(1), nil)
		id, err := svc.CreateUser(ctx, user)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	t.Run("validation error", func(t *testing.T) {
		user := &models.User{Name: "", Email: "john@example.com", PasswordHash: "hash"}
		id, err := svc.CreateUser(ctx, user)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})

	t.Run("repo error", func(t *testing.T) {
		user := &models.User{Name: "John", Email: "john@example.com", PasswordHash: "hash"}
		userRepo.EXPECT().CreateUser(gomock.Any(), user).Return(int64(0), errors.New("db error"))
		id, err := svc.CreateUser(ctx, user)
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})
}

func TestUserService_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_repository.NewMockUserRepository(ctrl)
	taskRepo := mock_repository.NewMockTaskRepository(ctrl)
	messengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	producer := &queue.Producer{}
	svc := NewUserService(userRepo, taskRepo, messengerRepo, producer)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		user := &models.User{ID: 1, Name: "John"}
		userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(user, nil)
		result, err := svc.GetUser(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, user, result)
	})

	t.Run("repo error", func(t *testing.T) {
		userRepo.EXPECT().GetUserByID(gomock.Any(), int64(2)).Return(nil, errors.New("not found"))
		result, err := svc.GetUser(ctx, 2)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_repository.NewMockUserRepository(ctrl)
	taskRepo := mock_repository.NewMockTaskRepository(ctrl)
	messengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	producer := &queue.Producer{}
	svc := NewUserService(userRepo, taskRepo, messengerRepo, producer)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		user := &models.User{ID: 1, Name: "John", Email: "john@example.com", PasswordHash: "hash"}
		updateReq := &models.UserUpdateRequest{
			Name:         ptrString("Jane"),
			Email:        ptrString("jane@example.com"),
			PasswordHash: ptrString("newhash"),
			Timezone:     ptrString("Europe/Moscow"),
			LanguageCode: ptrString("en"),
			Role:         ptrString("admin"),
		}
		userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(user, nil)
		userRepo.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Return(nil)
		updated, err := svc.UpdateUser(ctx, 1, updateReq)
		assert.NoError(t, err)
		assert.Equal(t, "Jane", updated.Name)
		assert.Equal(t, "jane@example.com", updated.Email)
		assert.Equal(t, "newhash", updated.PasswordHash)
		assert.Equal(t, "Europe/Moscow", *updated.Timezone)
		assert.Equal(t, "en", *updated.LanguageCode)
		assert.Equal(t, "admin", *updated.Role)
	})

	t.Run("get user error", func(t *testing.T) {
		updateReq := &models.UserUpdateRequest{Name: ptrString("Jane")}
		userRepo.EXPECT().GetUserByID(gomock.Any(), int64(2)).Return(nil, errors.New("not found"))
		updated, err := svc.UpdateUser(ctx, 2, updateReq)
		assert.Error(t, err)
		assert.Nil(t, updated)
	})

	t.Run("update user error", func(t *testing.T) {
		user := &models.User{ID: 3, Name: "John"}
		updateReq := &models.UserUpdateRequest{Name: ptrString("Jane")}
		userRepo.EXPECT().GetUserByID(gomock.Any(), int64(3)).Return(user, nil)
		userRepo.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Return(errors.New("update error"))
		updated, err := svc.UpdateUser(ctx, 3, updateReq)
		assert.Error(t, err)
		assert.Nil(t, updated)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_repository.NewMockUserRepository(ctrl)
	taskRepo := mock_repository.NewMockTaskRepository(ctrl)
	messengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	producer := &queue.Producer{}
	svc := NewUserService(userRepo, taskRepo, messengerRepo, producer)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		user := &models.User{ID: 1, Name: "John"}
		userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(user, nil)
		taskRepo.EXPECT().GetTasksByUserID(gomock.Any(), int64(1)).Return([]*models.Task{}, nil)
		messengerRepo.EXPECT().DeleteMessengerRelatedUserByUserID(gomock.Any(), int64(1)).Return(nil)
		userRepo.EXPECT().DeleteUser(gomock.Any(), int64(1)).Return(nil)
		err := svc.DeleteUser(ctx, 1)
		assert.NoError(t, err)
	})

	t.Run("get user error", func(t *testing.T) {
		userRepo.EXPECT().GetUserByID(gomock.Any(), int64(2)).Return(nil, errors.New("not found"))
		err := svc.DeleteUser(ctx, 2)
		assert.Error(t, err)
	})

	t.Run("delete user error", func(t *testing.T) {
		user := &models.User{ID: 3, Name: "John"}
		userRepo.EXPECT().GetUserByID(gomock.Any(), int64(3)).Return(user, nil)
		taskRepo.EXPECT().GetTasksByUserID(gomock.Any(), int64(3)).Return([]*models.Task{}, nil)
		messengerRepo.EXPECT().DeleteMessengerRelatedUserByUserID(gomock.Any(), int64(3)).Return(nil)
		userRepo.EXPECT().DeleteUser(gomock.Any(), int64(3)).Return(errors.New("delete error"))
		err := svc.DeleteUser(ctx, 3)
		assert.Error(t, err)
	})
}
