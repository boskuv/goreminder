package service

import (
	"context"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// MessengerService defines methods for messenger-related business logic
type MessengerService struct {
	messengerRepo repository.MessengerRepository
	userRepo      repository.UserRepository
	tracer        trace.Tracer
	logger        zerolog.Logger
}

// NewMessengerService creates a new instance of MessengerService
func NewMessengerService(messengerRepo repository.MessengerRepository, userRepo repository.UserRepository, logger zerolog.Logger) *MessengerService {
	return &MessengerService{
		messengerRepo: messengerRepo,
		userRepo:      userRepo,
		tracer:        otel.Tracer("messenger-service"),
		logger:        logger,
	}
}

// CreateMessenger implements BL of adding new messenger
func (s *MessengerService) CreateMessenger(ctx context.Context, messenger *models.Messenger) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "messenger_service.CreateMessenger",
		trace.WithAttributes(
			attribute.String("messenger.name", messenger.Name),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Str("messenger.name", messenger.Name).
		Msg("creating messenger")

	// perform some validation before creating the messenger
	if messenger.Name == "" {
		err := errors.New("messenger data is incomplete")
		log.Debug().
			Err(err).
			Msg("messenger validation failed: name is empty")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	messengerID, err := s.messengerRepo.CreateMessenger(ctx, messenger)
	if err != nil {
		log.Debug().
			Err(err).
			Str("messenger.name", messenger.Name).
			Msg("failed to create messenger")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	log.Debug().
		Int64("messenger.id", messengerID).
		Str("messenger.name", messenger.Name).
		Msg("messenger created successfully")
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

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("messenger.id", messengerID).
		Msg("getting messenger")

	messenger, err := s.messengerRepo.GetMessengerByID(ctx, messengerID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("messenger.id", messengerID).
			Msg("failed to get messenger")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	log.Debug().
		Int64("messenger.id", messengerID).
		Msg("messenger retrieved successfully")
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

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Str("messenger.name", messengerName).
		Msg("getting messenger id by name")

	messengerID, err := s.messengerRepo.GetMessengerIDByName(ctx, messengerName)
	if err != nil {
		log.Debug().
			Err(err).
			Str("messenger.name", messengerName).
			Msg("failed to get messenger id by name")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	log.Debug().
		Int64("messenger.id", messengerID).
		Str("messenger.name", messengerName).
		Msg("messenger id retrieved successfully")
	span.SetAttributes(attribute.Int64("messenger.id", messengerID))
	span.SetStatus(codes.Ok, "messenger ID retrieved successfully")
	return messengerID, nil
}

// CreateMessengerRelatedUser implements BL of adding new messenger-related user
func (s *MessengerService) CreateMessengerRelatedUser(ctx context.Context, messengerRelatedUser *models.MessengerRelatedUser) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "messenger_service.CreateMessengerRelatedUser")
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Str("messenger_user.id", messengerRelatedUser.MessengerUserID).
		Str("chat.id", messengerRelatedUser.ChatID).
		Msg("creating messenger-related user")

	// perform some validation before creating the messenger-related user
	if messengerRelatedUser.MessengerUserID == "" || messengerRelatedUser.ChatID == "" || messengerRelatedUser.UserID == nil || messengerRelatedUser.MessengerID == nil {
		err := errors.New("messenger_user data is incomplete")
		log.Debug().
			Err(err).
			Msg("messenger-related user validation failed")
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
		log.Debug().
			Err(err).
			Str("messenger_user.id", messengerRelatedUser.MessengerUserID).
			Msg("failed to create messenger-related user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	log.Debug().
		Int64("messenger_related_user.id", messengerRelatedUserID).
		Str("messenger_user.id", messengerRelatedUser.MessengerUserID).
		Msg("messenger-related user created successfully")
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

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Str("chat.id", chatID).
		Str("messenger_user.id", messengerUserID).
		Msg("getting messenger-related user")

	if userID != nil {
		span.SetAttributes(attribute.Int64("user.id", *userID))
		log = log.With().Int64("user.id", *userID).Logger()
	}
	if messengerID != nil {
		span.SetAttributes(attribute.Int64("messenger.id", *messengerID))
		log = log.With().Int64("messenger.id", *messengerID).Logger()
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
		log.Debug().
			Err(err).
			Str("chat.id", chatID).
			Str("messenger_user.id", messengerUserID).
			Msg("failed to get messenger-related user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	if messengerRelatedUser != nil {
		span.SetAttributes(attribute.Int64("messenger_related_user.id", messengerRelatedUser.ID))
		log = log.With().Int64("messenger_related_user.id", messengerRelatedUser.ID).Logger()
	}
	log.Debug().
		Str("chat.id", chatID).
		Str("messenger_user.id", messengerUserID).
		Msg("messenger-related user retrieved successfully")
	span.SetStatus(codes.Ok, "messenger related user retrieved successfully")
	return messengerRelatedUser, nil
}

// GetUserID implements BL of retrieving existing user by messengerUserID
func (s *MessengerService) GetUserID(ctx context.Context, messengerUserID string) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "messenger_service.GetUserID",
		trace.WithAttributes(
			attribute.String("messenger_user.id", messengerUserID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Str("messenger_user.id", messengerUserID).
		Msg("getting user id by messenger user id")

	userID, err := s.messengerRepo.GetUserID(ctx, messengerUserID)
	if err != nil {
		log.Debug().
			Err(err).
			Str("messenger_user.id", messengerUserID).
			Msg("failed to get user id by messenger user id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	log.Debug().
		Int64("user.id", userID).
		Str("messenger_user.id", messengerUserID).
		Msg("user id retrieved successfully")
	span.SetAttributes(attribute.Int64("user.id", userID))
	span.SetStatus(codes.Ok, "user ID retrieved successfully")
	return userID, nil
}

// GetAllMessengers implements BL of retrieving all messengers with pagination and ordering
func (s *MessengerService) GetAllMessengers(ctx context.Context, page, pageSize int, orderBy string) ([]*models.Messenger, int, error) {
	ctx, span := s.tracer.Start(ctx, "messenger_service.GetAllMessengers",
		trace.WithAttributes(
			attribute.Int("page", page),
			attribute.Int("page_size", pageSize),
			attribute.String("order_by", orderBy),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int("page", page).
		Int("page_size", pageSize).
		Str("order_by", orderBy).
		Msg("getting all messengers")

	messengers, totalCount, err := s.messengerRepo.GetAllMessengers(ctx, page, pageSize, orderBy)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all messengers")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.WithStack(err)
	}

	log.Debug().
		Int("messengers.count", len(messengers)).
		Int("total_count", totalCount).
		Msg("messengers retrieved successfully")
	span.SetAttributes(
		attribute.Int("messengers.count", len(messengers)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "messengers retrieved successfully")
	return messengers, totalCount, nil
}

// GetAllMessengerRelatedUsers implements BL of retrieving all messenger-related users with pagination and ordering
func (s *MessengerService) GetAllMessengerRelatedUsers(ctx context.Context, page, pageSize int, orderBy string) ([]*models.MessengerRelatedUser, int, error) {
	ctx, span := s.tracer.Start(ctx, "messenger_service.GetAllMessengerRelatedUsers",
		trace.WithAttributes(
			attribute.Int("page", page),
			attribute.Int("page_size", pageSize),
			attribute.String("order_by", orderBy),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int("page", page).
		Int("page_size", pageSize).
		Str("order_by", orderBy).
		Msg("getting all messenger-related users")

	messengerRelatedUsers, totalCount, err := s.messengerRepo.GetAllMessengerRelatedUsers(ctx, page, pageSize, orderBy)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all messenger-related users")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.WithStack(err)
	}

	log.Debug().
		Int("messenger_related_users.count", len(messengerRelatedUsers)).
		Int("total_count", totalCount).
		Msg("messenger-related users retrieved successfully")
	span.SetAttributes(
		attribute.Int("messenger_related_users.count", len(messengerRelatedUsers)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "messenger-related users retrieved successfully")
	return messengerRelatedUsers, totalCount, nil
}
