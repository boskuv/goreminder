package repository

import (
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/boskuv/goreminder/internal/models"
)

type MessengerRepository struct {
	db *sqlx.DB
	sb squirrel.StatementBuilderType
}

func NewMessengerRepository(db *sqlx.DB) *MessengerRepository {
	return &MessengerRepository{
		db: db,
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// CreateMessenger inserts a new messenger into the database
func (r *MessengerRepository) CreateMessenger(messenger *models.Messenger) (int64, error) {
	query, args, err := r.sb.Insert("messengers").
		Columns("name").
		Values(messenger.Name).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to build query")
	}

	var id int64
	err = r.db.QueryRow(query, args...).Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert messenger")
	}

	return id, nil
}

// GetMessengerByID retrieves a messenger by its ID
func (r *MessengerRepository) GetMessengerByID(id int64) (*models.Messenger, error) {
	query, args, err := r.sb.Select("name", "created_at").
		From("messengers").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query")
	}

	var messenger models.Messenger
	err = r.db.Get(&messenger, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch messenger")
	}

	return &messenger, nil
}

// GetMessengerIDByName retrieves a messenger ID by its name
func (r *MessengerRepository) GetMessengerIDByName(messengerName string) (int64, error) {
	query, args, err := r.sb.Select("id").
		From("messengers").
		Where(squirrel.Eq{"name": messengerName}).
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to build query")
	}

	var messengerID int64
	err = r.db.Get(&messengerID, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch messenger_id")
	}

	return messengerID, nil
}

// CreateMessengerRelatedUser inserts a new messenger-related user into the database
func (r *MessengerRepository) CreateMessengerRelatedUser(messengerRelatedUser *models.MessengerRelatedUser) (int64, error) {
	query, args, err := r.sb.Insert("user_messengers").
		Columns("chat_id", "messenger_id", "messenger_user_id", "user_id").
		Values(messengerRelatedUser.ChatID, messengerRelatedUser.MessengerID, messengerRelatedUser.MessengerUserID, messengerRelatedUser.UserID).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to build query")
	}

	var id int64
	err = r.db.QueryRow(query, args...).Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert user_messenger")
	}

	return id, nil
}

// GetMessengerRelatedUser retrieves a messenger-related user by chatID, messengerUserID, userID and messengerID
func (r *MessengerRepository) GetMessengerRelatedUser(chatID string, messengerUserID string, userID *int64, messengerID *int64) (*models.MessengerRelatedUser, error) {
	query, args, err := r.sb.Select("user_id", "messenger_id", "messenger_user_id", "chat_id", "created_at", "updated_at").
		From("user_messengers").
		Where(squirrel.Eq{"chat_id": chatID}).
		Where(squirrel.Eq{"messenger_user_id": messengerUserID}).
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.Eq{"messenger_id": messengerID}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query")
	}

	var messengerRelatedUser models.MessengerRelatedUser
	err = r.db.Get(&messengerRelatedUser, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch messenger-related user")
	}

	return &messengerRelatedUser, nil
}

// GetUserID retrieves a userID user by messengerUserID
func (r *MessengerRepository) GetUserID(messengerUserID string) (int64, error) {
	query, args, err := r.sb.Select("user_id").
		From("user_messengers").
		Where(squirrel.Eq{"messenger_user_id": messengerUserID}).
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to build query")
	}

	var userID int64
	err = r.db.Get(&userID, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch user_id")
	}

	return userID, nil
}
