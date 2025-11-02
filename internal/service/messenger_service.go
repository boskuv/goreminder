package service

import (
	"context"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pkg/errors"
)

// MessengerService defines methods for messenger-related business logic
type MessengerService struct {
	messengerRepo repository.MessengerRepository
	userRepo      repository.UserRepository
	tracer        trace.Tracer
}

// NewMessengerService creates a new instance of MessengerService
func NewMessengerService(messengerRepo repository.MessengerRepository, userRepo repository.UserRepository) *MessengerService {
	return &MessengerService{
		messengerRepo: messengerRepo,
		userRepo:      userRepo,
		tracer:        otel.Tracer("messenger-service"),
	}
}

// CreateMessenger implements BL of adding new messenger
func (s *MessengerService) CreateMessenger(ctx context.Context, messenger *models.Messenger) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "messenger_service.CreateMessenger",
		trace.WithAttributes(
			attribute.String("messenger.name", messenger.Name),
		))
	defer span.End()

	// perform some validation before creating the messenger
	if messenger.Name == "" {
		err := errors.New("messenger data is incomplete")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	messengerID, err := s.messengerRepo.CreateMessenger(ctx, messenger)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int64("messenger.id", messengerID))
	span.SetStatus(codes.Ok, "messenger created successfully")
	return messengerID, nil
}

// GetMessenger implements BL of retrieving existing messenger by its id
func (s *MessengerService) GetMessenger(ctx context.Context, messengerID int64) (*models.Messenger, error) {
	ctx, span := s.tracer.Start(ctx, "messenger_service.GetMessenger",
		trace.WithAttributes(
			attribute.Int64("messenger.id", messengerID),
		))
	defer span.End()

	messenger, err := s.messengerRepo.GetMessengerByID(ctx, messengerID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	span.SetStatus(codes.Ok, "messenger retrieved successfully")
	return messenger, nil
}

// GetMessengerIDByName implements BL of retrieving existing messenger by its name
func (s *MessengerService) GetMessengerIDByName(ctx context.Context, messengerName string) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "messenger_service.GetMessengerIDByName",
		trace.WithAttributes(
			attribute.String("messenger.name", messengerName),
		))
	defer span.End()

	messengerID, err := s.messengerRepo.GetMessengerIDByName(ctx, messengerName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int64("messenger.id", messengerID))
	span.SetStatus(codes.Ok, "messenger ID retrieved successfully")
	return messengerID, nil
}

// CreateMessengerRelatedUser implements BL of adding new messenger-related user
func (s *MessengerService) CreateMessengerRelatedUser(ctx context.Context, messengerRelatedUser *models.MessengerRelatedUser) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "messenger_service.CreateMessengerRelatedUser")
	defer span.End()

	// perform some validation before creating the messenger-related user
	if messengerRelatedUser.MessengerUserID == "" || messengerRelatedUser.ChatID == "" || messengerRelatedUser.UserID == nil || messengerRelatedUser.MessengerID == nil {
		err := errors.New("messenger_user data is incomplete")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	if messengerRelatedUser.UserID != nil {
		span.SetAttributes(attribute.Int64("user.id", *messengerRelatedUser.UserID))
	}
	if messengerRelatedUser.MessengerID != nil {
		span.SetAttributes(attribute.Int64("messenger.id", *messengerRelatedUser.MessengerID))
	}
	span.SetAttributes(
		attribute.String("messenger_user.id", messengerRelatedUser.MessengerUserID),
		attribute.String("chat.id", messengerRelatedUser.ChatID),
	)

	// check if user and messenger exist
	_, err := s.userRepo.GetUserByID(ctx, *messengerRelatedUser.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	_, err = s.messengerRepo.GetMessengerByID(ctx, *messengerRelatedUser.MessengerID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	messengerRelatedUserID, err := s.messengerRepo.CreateMessengerRelatedUser(ctx, messengerRelatedUser)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int64("messenger_related_user.id", messengerRelatedUserID))
	span.SetStatus(codes.Ok, "messenger related user created successfully")
	return messengerRelatedUserID, nil
}

// GetMessengerRelatedUser implements BL of retrieving existing messenger-related user by chatID, messengerUserID, userID and messengerIDs
func (s *MessengerService) GetMessengerRelatedUser(ctx context.Context, chatID string, messengerUserID string, userID *int64, messengerID *int64) (*models.MessengerRelatedUser, error) {
	ctx, span := s.tracer.Start(ctx, "messenger_service.GetMessengerRelatedUser",
		trace.WithAttributes(
			attribute.String("chat.id", chatID),
			attribute.String("messenger_user.id", messengerUserID),
		))
	defer span.End()

	if userID != nil {
		span.SetAttributes(attribute.Int64("user.id", *userID))
	}
	if messengerID != nil {
		span.SetAttributes(attribute.Int64("messenger.id", *messengerID))
	}

	// check if user exists (only if userID is provided)
	if userID != nil {
		_, err := s.userRepo.GetUserByID(ctx, *userID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
	}

	// check if messenger exists (only if messengerID is provided)
	if messengerID != nil {
		_, err := s.messengerRepo.GetMessengerByID(ctx, *messengerID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
	}

	messengerRelatedUser, err := s.messengerRepo.GetMessengerRelatedUser(ctx, chatID, messengerUserID, userID, messengerID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	if messengerRelatedUser != nil {
		span.SetAttributes(attribute.Int64("messenger_related_user.id", messengerRelatedUser.ID))
	}
	span.SetStatus(codes.Ok, "messenger related user retrieved successfully")
	return messengerRelatedUser, nil
}

// GetUserID implements BL of retrieving existing user by messengerUserID
// TODO: add messengerUD + messengerUserID 422
func (s *MessengerService) GetUserID(ctx context.Context, messengerUserID string) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "messenger_service.GetUserID",
		trace.WithAttributes(
			attribute.String("messenger_user.id", messengerUserID),
		))
	defer span.End()

	userID, err := s.messengerRepo.GetUserID(ctx, messengerUserID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int64("user.id", userID))
	span.SetStatus(codes.Ok, "user ID retrieved successfully")
	return userID, nil
}
