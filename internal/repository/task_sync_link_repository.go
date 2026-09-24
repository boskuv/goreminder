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

type TaskSyncLinkRepository interface {
	Create(ctx context.Context, link *models.TaskSyncLink) (int64, error)
	GetByTaskID(ctx context.Context, taskID int64) (*models.TaskSyncLink, error)
	GetByEvent(ctx context.Context, provider, calendarID, eventID string) (*models.TaskSyncLink, error)
	ListByBindingID(ctx context.Context, bindingID int64) ([]*models.TaskSyncLink, error)
	ListByCalendarIDAndOrigin(ctx context.Context, calendarID string, origin models.TaskSyncLinkOrigin) ([]*models.TaskSyncLink, error)
	ListTaskIDsByProvider(ctx context.Context, userID int64, provider string) ([]int64, error)
	Update(ctx context.Context, link *models.TaskSyncLink) error
	Delete(ctx context.Context, id int64) error
	SetSyncEnabled(ctx context.Context, id int64, enabled bool) error
}

type taskSyncLinkRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

func NewTaskSyncLinkRepository(db *sqlx.DB, logger zerolog.Logger) TaskSyncLinkRepository {
	return &taskSyncLinkRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("task-sync-link-repository"),
		logger: logger,
	}
}

const taskSyncLinkColumns = "id, task_id, provider, google_calendar_id, google_event_id, etag, google_updated_at, origin, sync_enabled, export_opt_in, calendar_binding_id, duration_seconds, last_synced_at, last_error, created_at, updated_at"

func (r *taskSyncLinkRepository) Create(ctx context.Context, link *models.TaskSyncLink) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "task_sync_link_repository.Create",
		trace.WithAttributes(attribute.Int64("task.id", link.TaskID)))
	defer span.End()

	if link.Provider == "" {
		link.Provider = models.TaskSyncProviderGoogleCalendar
	}

	query, args, err := r.sb.Insert("task_sync_links").
		Columns("task_id", "provider", "google_calendar_id", "google_event_id", "etag", "google_updated_at", "origin", "sync_enabled", "export_opt_in", "calendar_binding_id", "duration_seconds", "last_synced_at", "last_error").
		Values(link.TaskID, link.Provider, link.GoogleCalendarID, link.GoogleEventID, link.ETag, link.GoogleUpdatedAt, link.Origin, link.SyncEnabled, link.ExportOptIn, link.CalendarBindingID, link.DurationSeconds, link.LastSyncedAt, link.LastError).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build create task sync link query")
	}

	var id int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to create task sync link")
	}
	span.SetAttributes(attribute.Int64("task_sync_link.id", id))
	span.SetStatus(codes.Ok, "created")
	return id, nil
}

func (r *taskSyncLinkRepository) GetByTaskID(ctx context.Context, taskID int64) (*models.TaskSyncLink, error) {
	ctx, span := r.tracer.Start(ctx, "task_sync_link_repository.GetByTaskID",
		trace.WithAttributes(attribute.Int64("task.id", taskID)))
	defer span.End()

	query, args, err := r.sb.Select(taskSyncLinkColumns).
		From("task_sync_links").
		Where(squirrel.Eq{"task_id": taskID}).
		OrderBy("id DESC").
		Limit(1).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build get sync link by task query")
	}

	var link models.TaskSyncLink
	err = r.db.GetContext(ctx, &link, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no sync link found for task")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get sync link by task")
	}
	span.SetStatus(codes.Ok, "found")
	return &link, nil
}

func (r *taskSyncLinkRepository) GetByEvent(ctx context.Context, provider, calendarID, eventID string) (*models.TaskSyncLink, error) {
	ctx, span := r.tracer.Start(ctx, "task_sync_link_repository.GetByEvent")
	defer span.End()

	query, args, err := r.sb.Select(taskSyncLinkColumns).
		From("task_sync_links").
		Where(squirrel.Eq{"provider": provider}).
		Where(squirrel.Eq{"google_calendar_id": calendarID}).
		Where(squirrel.Eq{"google_event_id": eventID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build get sync link by event query")
	}

	var link models.TaskSyncLink
	err = r.db.GetContext(ctx, &link, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no sync link found for event")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get sync link by event")
	}
	span.SetStatus(codes.Ok, "found")
	return &link, nil
}

func (r *taskSyncLinkRepository) ListByBindingID(ctx context.Context, bindingID int64) ([]*models.TaskSyncLink, error) {
	ctx, span := r.tracer.Start(ctx, "task_sync_link_repository.ListByBindingID",
		trace.WithAttributes(attribute.Int64("calendar_binding.id", bindingID)))
	defer span.End()

	query, args, err := r.sb.Select(taskSyncLinkColumns).
		From("task_sync_links").
		Where(squirrel.Eq{"calendar_binding_id": bindingID}).
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build list sync links by binding query")
	}

	var links []*models.TaskSyncLink
	if err := r.db.SelectContext(ctx, &links, query, args...); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to list sync links by binding")
	}
	span.SetStatus(codes.Ok, "listed")
	return links, nil
}

func (r *taskSyncLinkRepository) ListByCalendarIDAndOrigin(ctx context.Context, calendarID string, origin models.TaskSyncLinkOrigin) ([]*models.TaskSyncLink, error) {
	ctx, span := r.tracer.Start(ctx, "task_sync_link_repository.ListByCalendarIDAndOrigin")
	defer span.End()

	query, args, err := r.sb.Select(taskSyncLinkColumns).
		From("task_sync_links").
		Where(squirrel.Eq{"google_calendar_id": calendarID}).
		Where(squirrel.Eq{"origin": origin}).
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build list sync links by calendar query")
	}

	var links []*models.TaskSyncLink
	if err := r.db.SelectContext(ctx, &links, query, args...); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to list sync links by calendar")
	}
	span.SetStatus(codes.Ok, "listed")
	return links, nil
}

func (r *taskSyncLinkRepository) ListTaskIDsByProvider(ctx context.Context, userID int64, provider string) ([]int64, error) {
	ctx, span := r.tracer.Start(ctx, "task_sync_link_repository.ListTaskIDsByProvider",
		trace.WithAttributes(attribute.Int64("user.id", userID), attribute.String("provider", provider)))
	defer span.End()

	query := `
SELECT tsl.task_id
FROM task_sync_links tsl
INNER JOIN tasks t ON t.id = tsl.task_id
WHERE t.user_id = $1 AND tsl.provider = $2 AND t.deleted_at IS NULL`

	var ids []int64
	if err := r.db.SelectContext(ctx, &ids, query, userID, provider); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to list task ids by provider")
	}
	span.SetStatus(codes.Ok, "listed")
	return ids, nil
}

func (r *taskSyncLinkRepository) Update(ctx context.Context, link *models.TaskSyncLink) error {
	ctx, span := r.tracer.Start(ctx, "task_sync_link_repository.Update",
		trace.WithAttributes(attribute.Int64("task_sync_link.id", link.ID)))
	defer span.End()

	query, args, err := r.sb.Update("task_sync_links").
		Set("google_calendar_id", link.GoogleCalendarID).
		Set("google_event_id", link.GoogleEventID).
		Set("etag", link.ETag).
		Set("google_updated_at", link.GoogleUpdatedAt).
		Set("origin", link.Origin).
		Set("sync_enabled", link.SyncEnabled).
		Set("export_opt_in", link.ExportOptIn).
		Set("calendar_binding_id", link.CalendarBindingID).
		Set("duration_seconds", link.DurationSeconds).
		Set("last_synced_at", link.LastSyncedAt).
		Set("last_error", link.LastError).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": link.ID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build update sync link query")
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to update sync link")
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no sync link found for id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "updated")
	return nil
}

func (r *taskSyncLinkRepository) Delete(ctx context.Context, id int64) error {
	ctx, span := r.tracer.Start(ctx, "task_sync_link_repository.Delete",
		trace.WithAttributes(attribute.Int64("task_sync_link.id", id)))
	defer span.End()

	query, args, err := r.sb.Delete("task_sync_links").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build delete sync link query")
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to delete sync link")
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no sync link found for id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "deleted")
	return nil
}

func (r *taskSyncLinkRepository) SetSyncEnabled(ctx context.Context, id int64, enabled bool) error {
	ctx, span := r.tracer.Start(ctx, "task_sync_link_repository.SetSyncEnabled",
		trace.WithAttributes(attribute.Int64("task_sync_link.id", id)))
	defer span.End()

	query, args, err := r.sb.Update("task_sync_links").
		Set("sync_enabled", enabled).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build set sync enabled query")
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to set sync enabled")
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no sync link found for id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "updated")
	return nil
}
