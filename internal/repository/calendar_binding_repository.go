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

type CalendarBindingRepository interface {
	Create(ctx context.Context, binding *models.CalendarBinding) (int64, error)
	GetByID(ctx context.Context, id int64) (*models.CalendarBinding, error)
	ListByUserID(ctx context.Context, userID int64) ([]*models.CalendarBinding, error)
	Update(ctx context.Context, binding *models.CalendarBinding) error
	SoftDelete(ctx context.Context, id int64) error
	ListActiveForSync(ctx context.Context) ([]*models.CalendarBinding, error)
	ClearSyncToken(ctx context.Context, id int64) error
}

type calendarBindingRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

func NewCalendarBindingRepository(db *sqlx.DB, logger zerolog.Logger) CalendarBindingRepository {
	return &calendarBindingRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("calendar-binding-repository"),
		logger: logger,
	}
}

const calendarBindingColumns = "id, user_id, google_account_id, google_calendar_id, calendar_summary, direction, group_id, messenger_related_user_id, sync_token, last_synced_at, last_error, status, delete_policy, created_at, updated_at"

func (r *calendarBindingRepository) Create(ctx context.Context, binding *models.CalendarBinding) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "calendar_binding_repository.Create",
		trace.WithAttributes(attribute.Int64("user.id", binding.UserID)))
	defer span.End()

	if binding.Direction == "" {
		binding.Direction = models.CalendarBindingDirectionImport
	}
	if binding.Status == "" {
		binding.Status = models.CalendarBindingStatusActive
	}
	if binding.DeletePolicy == "" {
		binding.DeletePolicy = models.CalendarDeletePolicySoftDeleteImported
	}

	query, args, err := r.sb.Insert("calendar_bindings").
		Columns("user_id", "google_account_id", "google_calendar_id", "calendar_summary", "direction", "group_id", "messenger_related_user_id", "status", "delete_policy").
		Values(binding.UserID, binding.GoogleAccountID, binding.GoogleCalendarID, binding.CalendarSummary, binding.Direction, binding.GroupID, binding.MessengerRelatedUserID, binding.Status, binding.DeletePolicy).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build create calendar binding query")
	}

	var id int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to create calendar binding")
	}
	span.SetAttributes(attribute.Int64("calendar_binding.id", id))
	span.SetStatus(codes.Ok, "created")
	return id, nil
}

func (r *calendarBindingRepository) GetByID(ctx context.Context, id int64) (*models.CalendarBinding, error) {
	ctx, span := r.tracer.Start(ctx, "calendar_binding_repository.GetByID",
		trace.WithAttributes(attribute.Int64("calendar_binding.id", id)))
	defer span.End()

	query, args, err := r.sb.Select(calendarBindingColumns).
		From("calendar_bindings").
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{"deleted_at": nil}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build get calendar binding query")
	}

	var binding models.CalendarBinding
	err = r.db.GetContext(ctx, &binding, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no calendar binding found for id")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get calendar binding")
	}
	span.SetStatus(codes.Ok, "found")
	return &binding, nil
}

func (r *calendarBindingRepository) ListByUserID(ctx context.Context, userID int64) ([]*models.CalendarBinding, error) {
	ctx, span := r.tracer.Start(ctx, "calendar_binding_repository.ListByUserID",
		trace.WithAttributes(attribute.Int64("user.id", userID)))
	defer span.End()

	query, args, err := r.sb.Select(calendarBindingColumns).
		From("calendar_bindings").
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build list calendar bindings query")
	}

	var bindings []*models.CalendarBinding
	if err := r.db.SelectContext(ctx, &bindings, query, args...); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to list calendar bindings")
	}
	span.SetStatus(codes.Ok, "listed")
	return bindings, nil
}

func (r *calendarBindingRepository) Update(ctx context.Context, binding *models.CalendarBinding) error {
	ctx, span := r.tracer.Start(ctx, "calendar_binding_repository.Update",
		trace.WithAttributes(attribute.Int64("calendar_binding.id", binding.ID)))
	defer span.End()

	query, args, err := r.sb.Update("calendar_bindings").
		Set("calendar_summary", binding.CalendarSummary).
		Set("direction", binding.Direction).
		Set("group_id", binding.GroupID).
		Set("messenger_related_user_id", binding.MessengerRelatedUserID).
		Set("sync_token", binding.SyncToken).
		Set("last_synced_at", binding.LastSyncedAt).
		Set("last_error", binding.LastError).
		Set("status", binding.Status).
		Set("delete_policy", binding.DeletePolicy).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": binding.ID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build update calendar binding query")
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to update calendar binding")
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no calendar binding found for id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "updated")
	return nil
}

func (r *calendarBindingRepository) SoftDelete(ctx context.Context, id int64) error {
	ctx, span := r.tracer.Start(ctx, "calendar_binding_repository.SoftDelete",
		trace.WithAttributes(attribute.Int64("calendar_binding.id", id)))
	defer span.End()

	now := time.Now().UTC()
	query, args, err := r.sb.Update("calendar_bindings").
		Set("deleted_at", now).
		Set("status", models.CalendarBindingStatusDisconnected).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{"deleted_at": nil}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build soft delete calendar binding query")
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to soft delete calendar binding")
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no calendar binding found for id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "deleted")
	return nil
}

func (r *calendarBindingRepository) ListActiveForSync(ctx context.Context) ([]*models.CalendarBinding, error) {
	ctx, span := r.tracer.Start(ctx, "calendar_binding_repository.ListActiveForSync")
	defer span.End()

	query, args, err := r.sb.Select(calendarBindingColumns).
		From("calendar_bindings").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"status": models.CalendarBindingStatusActive}).
		Where(squirrel.Eq{"direction": []string{
			string(models.CalendarBindingDirectionImport),
			string(models.CalendarBindingDirectionBoth),
		}}).
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build list active bindings query")
	}

	var bindings []*models.CalendarBinding
	if err := r.db.SelectContext(ctx, &bindings, query, args...); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to list active bindings")
	}
	span.SetStatus(codes.Ok, "listed")
	return bindings, nil
}

func (r *calendarBindingRepository) ClearSyncToken(ctx context.Context, id int64) error {
	ctx, span := r.tracer.Start(ctx, "calendar_binding_repository.ClearSyncToken",
		trace.WithAttributes(attribute.Int64("calendar_binding.id", id)))
	defer span.End()

	query, args, err := r.sb.Update("calendar_bindings").
		Set("sync_token", nil).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{"deleted_at": nil}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build clear sync token query")
	}
	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to clear sync token")
	}
	span.SetStatus(codes.Ok, "cleared")
	return nil
}
