package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
)

type MessengerRepository interface {
	CreateMessenger(ctx context.Context, messenger *models.Messenger) (int64, error)
	GetMessengerByID(ctx context.Context, id int64) (*models.Messenger, error)
	GetMessengerIDByName(ctx context.Context, messengerName string) (int64, error)
	CreateMessengerRelatedUser(ctx context.Context, messengerRelatedUser *models.MessengerRelatedUser) (int64, error)
	GetMessengerRelatedUser(ctx context.Context, chatID string, messengerUserID string, userID *int64, messengerID *int64) (*models.MessengerRelatedUser, error)
	GetUserID(ctx context.Context, messengerUserID string) (int64, error)
	GetMessengerRelatedUserByID(ctx context.Context, messengerUserID int) (*models.MessengerRelatedUser, error)
	DeleteMessengerRelatedUserByUserID(ctx context.Context, userID int64) error
}

type messengerRepository struct {
	db *sqlx.DB
	sb squirrel.StatementBuilderType
}

func NewMessengerRepository(db *sqlx.DB) MessengerRepository {
	return &messengerRepository{
		db: db,
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// CreateMessenger inserts a new messenger into the database
// default values are preset for: id, created_at (database-level)
func (r *messengerRepository) CreateMessenger(ctx context.Context, messenger *models.Messenger) (int64, error) {
	query, args, err := r.sb.Insert("messengers").
		Columns("name").
		Values(messenger.Name).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to build query while creating new messenger")
	}

	var id int64
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert messenger")
	}

	return id, nil
}

// GetMessengerByID retrieves a messenger by its ID
// Returns messenger entity and an error if occurred
func (r *messengerRepository) GetMessengerByID(ctx context.Context, id int64) (*models.Messenger, error) {
	query, args, err := r.sb.Select("name", "created_at").
		From("messengers").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query while getting messenger by id")
	}

	var messenger models.Messenger
	err = r.db.GetContext(ctx, &messenger, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Wrap(errs.ErrNotFound, "no messenger found for passed id")
		}

		return nil, errors.Wrap(err, "failed to get messenger by id")
	}

	return &messenger, nil
}

// GetMessengerIDByName retrieves a messenger ID by its name
// Returns messenger ID and an error if occurred
func (r *messengerRepository) GetMessengerIDByName(ctx context.Context, messengerName string) (int64, error) {
	query, args, err := r.sb.Select("id").
		From("messengers").
		Where(squirrel.Eq{"name": messengerName}).
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to build query while getting messenger id by name")
	}

	var messengerID int64
	err = r.db.GetContext(ctx, &messengerID, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.Wrap(errs.ErrNotFound, "no messenger found for passed name")
		}

		return 0, errors.Wrap(err, "failed to get messenger id by name")
	}

	return messengerID, nil
}

// CreateMessengerRelatedUser inserts a new messenger-related user into the database
// default values are preset for: id, created_at (database-level)
func (r *messengerRepository) CreateMessengerRelatedUser(ctx context.Context, messengerRelatedUser *models.MessengerRelatedUser) (int64, error) {
	query, args, err := r.sb.Insert("user_messengers").
		Columns("chat_id", "messenger_id", "messenger_user_id", "user_id").
		Values(messengerRelatedUser.ChatID, messengerRelatedUser.MessengerID, messengerRelatedUser.MessengerUserID, messengerRelatedUser.UserID).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to build query while creating new messenger-related user")
	}

	var id int64
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert messenger-related user")
	}

	return id, nil
}

// GetMessengerRelatedUser retrieves a messenger-related user by chatID, messengerUserID, userID and messengerID
// Returns messenger-related user entity and an error if occurred
func (r *messengerRepository) GetMessengerRelatedUser(ctx context.Context, chatID string, messengerUserID string, userID *int64, messengerID *int64) (*models.MessengerRelatedUser, error) {
	query, args, err := r.sb.Select("id", "user_id", "messenger_id", "messenger_user_id", "chat_id", "created_at", "updated_at").
		From("user_messengers").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"chat_id": chatID}).
		Where(squirrel.Eq{"messenger_user_id": messengerUserID}).
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.Eq{"messenger_id": messengerID}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query while getting messenger-related user")
	}

	var messengerRelatedUser models.MessengerRelatedUser
	err = r.db.GetContext(ctx, &messengerRelatedUser, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Wrap(errs.ErrNotFound, "no messenger-related user found for passed chatID, messengerUserID, userID and messengerID")
		}

		return nil, errors.Wrap(err, "failed to get messenger-related user")
	}

	return &messengerRelatedUser, nil
}

// GetUserID retrieves a userID user by messengerUserID
// Returns userID and an error if occurred
func (r *messengerRepository) GetUserID(ctx context.Context, messengerUserID string) (int64, error) {
	query, args, err := r.sb.Select("user_id").
		From("user_messengers").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"messenger_user_id": messengerUserID}).
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to build query while getting user id by messenger user id")
	}

	var userID int64
	err = r.db.GetContext(ctx, &userID, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.Wrap(errs.ErrNotFound, "no user found for passed messenger user id")
		}

		return 0, errors.Wrap(err, "failed to get user id by messenger user id")
	}

	return userID, nil
}

// GetMessengerRelatedUserByID retrieves a messenger-related user by its ID
// Returns messenger-related user entity and an error if occurred
func (r *messengerRepository) GetMessengerRelatedUserByID(ctx context.Context, messengerUserID int) (*models.MessengerRelatedUser, error) {
	query, args, err := r.sb.Select("id", "user_id", "messenger_id", "messenger_user_id", "chat_id", "created_at", "updated_at").
		From("user_messengers").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": messengerUserID}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query while getting messenger-related user by id")
	}

	var messengerRelatedUser models.MessengerRelatedUser
	err = r.db.GetContext(ctx, &messengerRelatedUser, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Wrap(errs.ErrNotFound, "no messenger-related user found for passed id")
		}

		return nil, errors.Wrap(err, "failed to get messenger-related user by id")
	}

	return &messengerRelatedUser, nil
}

// DeleteMessengerRelatedUserByUserID soft deletes messenger-related user by user id
// It sets the deleted_at timestamp to the current time
func (r *messengerRepository) DeleteMessengerRelatedUserByUserID(ctx context.Context, userID int64) error {
	query, args, err := r.sb.Update("user_messengers").
		Set("deleted_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build query while soft deleting messenger-related user")
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute soft delete query for messenger-related user")
	}

	return nil
}
