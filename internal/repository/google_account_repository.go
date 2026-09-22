package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
)

type GoogleAccountRepository interface {
	Upsert(ctx context.Context, account *models.GoogleAccount) (int64, error)
	GetByUserID(ctx context.Context, userID int64) (*models.GoogleAccount, error)
	GetByID(ctx context.Context, id int64) (*models.GoogleAccount, error)
	UpdateTokens(ctx context.Context, id int64, accessEnc, refreshEnc []byte, expiry time.Time, scopes string) error
	Revoke(ctx context.Context, id int64) error
}

type googleAccountRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

func NewGoogleAccountRepository(db *sqlx.DB, logger zerolog.Logger) GoogleAccountRepository {
	return &googleAccountRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("google-account-repository"),
		logger: logger,
	}
}

func (r *googleAccountRepository) Upsert(ctx context.Context, account *models.GoogleAccount) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "google_account_repository.Upsert",
		trace.WithAttributes(attribute.Int64("user.id", account.UserID)))
	defer span.End()

	now := time.Now().UTC()
	query := `
INSERT INTO google_accounts (
  user_id, google_sub, email, access_token_enc, refresh_token_enc, token_expiry, scopes, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
ON CONFLICT (user_id, google_sub) DO UPDATE SET
  email = EXCLUDED.email,
  access_token_enc = EXCLUDED.access_token_enc,
  refresh_token_enc = COALESCE(NULLIF(EXCLUDED.refresh_token_enc, ''::bytea), google_accounts.refresh_token_enc),
  token_expiry = EXCLUDED.token_expiry,
  scopes = EXCLUDED.scopes,
  revoked_at = NULL,
  updated_at = EXCLUDED.updated_at
RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		account.UserID, account.GoogleSub, account.Email,
		account.AccessTokenEnc, account.RefreshTokenEnc, account.TokenExpiry, account.Scopes, now,
	).Scan(&id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to upsert google account")
	}
	span.SetAttributes(attribute.Int64("google_account.id", id))
	span.SetStatus(codes.Ok, "upserted")
	return id, nil
}

func (r *googleAccountRepository) GetByUserID(ctx context.Context, userID int64) (*models.GoogleAccount, error) {
	ctx, span := r.tracer.Start(ctx, "google_account_repository.GetByUserID",
		trace.WithAttributes(attribute.Int64("user.id", userID)))
	defer span.End()

	query, args, err := r.sb.Select(
		"id", "user_id", "google_sub", "email", "access_token_enc", "refresh_token_enc",
		"token_expiry", "scopes", "revoked_at", "created_at", "updated_at",
	).From("google_accounts").
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.Eq{"revoked_at": nil}).
		OrderBy("id DESC").
		Limit(1).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build get google account by user id query")
	}

	var account models.GoogleAccount
	err = r.db.GetContext(ctx, &account, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no google account found for user")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get google account by user id")
	}
	span.SetStatus(codes.Ok, "found")
	return &account, nil
}

func (r *googleAccountRepository) GetByID(ctx context.Context, id int64) (*models.GoogleAccount, error) {
	ctx, span := r.tracer.Start(ctx, "google_account_repository.GetByID",
		trace.WithAttributes(attribute.Int64("google_account.id", id)))
	defer span.End()

	query, args, err := r.sb.Select(
		"id", "user_id", "google_sub", "email", "access_token_enc", "refresh_token_enc",
		"token_expiry", "scopes", "revoked_at", "created_at", "updated_at",
	).From("google_accounts").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build get google account by id query")
	}

	var account models.GoogleAccount
	err = r.db.GetContext(ctx, &account, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no google account found for id")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get google account by id")
	}
	span.SetStatus(codes.Ok, "found")
	return &account, nil
}

func (r *googleAccountRepository) UpdateTokens(ctx context.Context, id int64, accessEnc, refreshEnc []byte, expiry time.Time, scopes string) error {
	ctx, span := r.tracer.Start(ctx, "google_account_repository.UpdateTokens",
		trace.WithAttributes(attribute.Int64("google_account.id", id)))
	defer span.End()

	builder := r.sb.Update("google_accounts").
		Set("access_token_enc", accessEnc).
		Set("token_expiry", expiry).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": id})
	if len(refreshEnc) > 0 {
		builder = builder.Set("refresh_token_enc", refreshEnc)
	}
	if scopes != "" {
		builder = builder.Set("scopes", scopes)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build update tokens query")
	}
	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to update google account tokens")
	}
	span.SetStatus(codes.Ok, "updated")
	return nil
}

func (r *googleAccountRepository) Revoke(ctx context.Context, id int64) error {
	ctx, span := r.tracer.Start(ctx, "google_account_repository.Revoke",
		trace.WithAttributes(attribute.Int64("google_account.id", id)))
	defer span.End()

	now := time.Now().UTC()
	query, args, err := r.sb.Update("google_accounts").
		Set("revoked_at", now).
		Set("access_token_enc", []byte{}).
		Set("refresh_token_enc", []byte{}).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build revoke google account query")
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to revoke google account")
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no google account found for id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "revoked")
	return nil
}