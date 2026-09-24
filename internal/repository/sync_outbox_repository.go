package repository

import (
	"context"
	"encoding/json"
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

type SyncOutboxRepository interface {
	Enqueue(ctx context.Context, kind string, payload json.RawMessage) (int64, error)
	ClaimDue(ctx context.Context, limit int) ([]*models.SyncOutbox, error)
	MarkDone(ctx context.Context, id int64) error
	MarkRetry(ctx context.Context, id int64, attempts int, nextRetryAt time.Time, lastError string) error
	MarkFailed(ctx context.Context, id int64, lastError string) error
	CountPending(ctx context.Context) (int, error)
	CountByUserID(ctx context.Context, userID int64) (pending, processing, failed int, err error)
}

type syncOutboxRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

func NewSyncOutboxRepository(db *sqlx.DB, logger zerolog.Logger) SyncOutboxRepository {
	return &syncOutboxRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("sync-outbox-repository"),
		logger: logger,
	}
}

func (r *syncOutboxRepository) Enqueue(ctx context.Context, kind string, payload json.RawMessage) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "sync_outbox_repository.Enqueue",
		trace.WithAttributes(attribute.String("kind", kind)))
	defer span.End()

	if payload == nil {
		payload = json.RawMessage(`{}`)
	}

	query, args, err := r.sb.Insert("sync_outbox").
		Columns("kind", "payload", "status", "next_retry_at").
		Values(kind, payload, models.SyncOutboxStatusPending, time.Now().UTC()).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build enqueue outbox query")
	}

	var id int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to enqueue outbox item")
	}
	span.SetAttributes(attribute.Int64("sync_outbox.id", id))
	span.SetStatus(codes.Ok, "enqueued")
	return id, nil
}

func (r *syncOutboxRepository) ClaimDue(ctx context.Context, limit int) ([]*models.SyncOutbox, error) {
	ctx, span := r.tracer.Start(ctx, "sync_outbox_repository.ClaimDue",
		trace.WithAttributes(attribute.Int("limit", limit)))
	defer span.End()

	if limit <= 0 {
		limit = 50
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to begin claim due transaction")
	}
	defer func() { _ = tx.Rollback() }()

	query := `
SELECT id, kind, payload, attempts, next_retry_at, last_error, status, created_at, updated_at
FROM sync_outbox
WHERE status = $1 AND next_retry_at <= $2
ORDER BY id ASC
LIMIT $3
FOR UPDATE SKIP LOCKED`

	var items []*models.SyncOutbox
	if err := tx.SelectContext(ctx, &items, query, models.SyncOutboxStatusPending, time.Now().UTC(), limit); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to select due outbox items")
	}
	if len(items) == 0 {
		span.SetStatus(codes.Ok, "none")
		return items, nil
	}

	ids := make([]int64, len(items))
	for i, item := range items {
		ids[i] = item.ID
		item.Status = models.SyncOutboxStatusProcessing
	}

	updateQuery, updateArgs, err := r.sb.Update("sync_outbox").
		Set("status", models.SyncOutboxStatusProcessing).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": ids}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build claim update query")
	}
	if _, err := tx.ExecContext(ctx, updateQuery, updateArgs...); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to mark outbox items processing")
	}
	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to commit claim due")
	}
	span.SetAttributes(attribute.Int("claimed", len(items)))
	span.SetStatus(codes.Ok, "claimed")
	return items, nil
}

func (r *syncOutboxRepository) MarkDone(ctx context.Context, id int64) error {
	return r.updateStatus(ctx, id, models.SyncOutboxStatusDone, nil, nil, "")
}

func (r *syncOutboxRepository) MarkRetry(ctx context.Context, id int64, attempts int, nextRetryAt time.Time, lastError string) error {
	ctx, span := r.tracer.Start(ctx, "sync_outbox_repository.MarkRetry",
		trace.WithAttributes(attribute.Int64("sync_outbox.id", id)))
	defer span.End()

	query, args, err := r.sb.Update("sync_outbox").
		Set("status", models.SyncOutboxStatusPending).
		Set("attempts", attempts).
		Set("next_retry_at", nextRetryAt).
		Set("last_error", lastError).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build mark retry query")
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to mark outbox retry")
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no outbox item found for id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "retry")
	return nil
}

func (r *syncOutboxRepository) MarkFailed(ctx context.Context, id int64, lastError string) error {
	return r.updateStatus(ctx, id, models.SyncOutboxStatusFailed, nil, nil, lastError)
}

func (r *syncOutboxRepository) updateStatus(ctx context.Context, id int64, status models.SyncOutboxStatus, attempts *int, nextRetryAt *time.Time, lastError string) error {
	ctx, span := r.tracer.Start(ctx, "sync_outbox_repository.updateStatus",
		trace.WithAttributes(attribute.Int64("sync_outbox.id", id)))
	defer span.End()

	builder := r.sb.Update("sync_outbox").
		Set("status", status).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": id})
	if attempts != nil {
		builder = builder.Set("attempts", *attempts)
	}
	if nextRetryAt != nil {
		builder = builder.Set("next_retry_at", *nextRetryAt)
	}
	if lastError != "" {
		builder = builder.Set("last_error", lastError)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build update outbox status query")
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to update outbox status")
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no outbox item found for id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "updated")
	return nil
}

func (r *syncOutboxRepository) CountPending(ctx context.Context) (int, error) {
	ctx, span := r.tracer.Start(ctx, "sync_outbox_repository.CountPending")
	defer span.End()

	query, args, err := r.sb.Select("COUNT(*)").
		From("sync_outbox").
		Where(squirrel.Eq{"status": []string{
			string(models.SyncOutboxStatusPending),
			string(models.SyncOutboxStatusProcessing),
		}}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build count pending query")
	}
	var count int
	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to count pending outbox")
	}
	span.SetStatus(codes.Ok, "counted")
	return count, nil
}

// CountByUserID returns open/failed export outbox rows whose payload.task_id belongs to userID.
func (r *syncOutboxRepository) CountByUserID(ctx context.Context, userID int64) (pending, processing, failed int, err error) {
	ctx, span := r.tracer.Start(ctx, "sync_outbox_repository.CountByUserID",
		trace.WithAttributes(attribute.Int64("user.id", userID)))
	defer span.End()

	const q = `
SELECT so.status, COUNT(*)::int
FROM sync_outbox so
INNER JOIN tasks t ON t.id = (so.payload->>'task_id')::bigint
WHERE t.user_id = $1
  AND so.status IN ('pending', 'processing', 'failed')
GROUP BY so.status`

	rows, err := r.db.QueryxContext(ctx, q, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, 0, 0, errors.Wrap(err, "failed to count outbox by user")
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if scanErr := rows.Scan(&status, &count); scanErr != nil {
			span.RecordError(scanErr)
			span.SetStatus(codes.Error, scanErr.Error())
			return 0, 0, 0, errors.Wrap(scanErr, "failed to scan outbox counts")
		}
		switch models.SyncOutboxStatus(status) {
		case models.SyncOutboxStatusPending:
			pending = count
		case models.SyncOutboxStatusProcessing:
			processing = count
		case models.SyncOutboxStatusFailed:
			failed = count
		}
	}
	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, 0, 0, errors.Wrap(err, "failed iterating outbox counts")
	}
	span.SetStatus(codes.Ok, "counted")
	return pending, processing, failed, nil
}
