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

// CreateMessengerRelatedUser inserts a new messenger-related user into the database
func (r *MessengerRepository) CreateMessengerRelatedUser(messengerRelatedUser *models.MessengerRelatedUser) (int64, error) {
	query, args, err := r.sb.Insert("user_messengers").
		Columns("chat_id", "messenger_id", "user_id").
		Values(messengerRelatedUser.ChatID, messengerRelatedUser.MessengerID, messengerRelatedUser.UserID).
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
