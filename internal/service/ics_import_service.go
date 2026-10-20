package service

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/googlecalendar"
	"github.com/boskuv/goreminder/pkg/ics"
	"github.com/boskuv/goreminder/pkg/logger"
)

// ICSImportService creates tasks from a one-shot .ics upload (not a live sync).
type ICSImportService struct {
	tasks     *TaskService
	users     repository.UserRepository
	groups    repository.TaskGroupRepository
	messenger repository.MessengerRepository
	logger    zerolog.Logger
	tracer    trace.Tracer
}

// NewICSImportService constructs an ICSImportService.
func NewICSImportService(
	tasks *TaskService,
	users repository.UserRepository,
	groups repository.TaskGroupRepository,
	messenger repository.MessengerRepository,
	log zerolog.Logger,
) *ICSImportService {
	return &ICSImportService{
		tasks:     tasks,
		users:     users,
		groups:    groups,
		messenger: messenger,
		logger:    log,
		tracer:    otel.Tracer("goreminder/service"),
	}
}

// ICSImportOptions are request-scoped fields applied to every created task.
type ICSImportOptions struct {
	UserID                 int64
	GroupID                *int64
	MessengerRelatedUserID *int
}

// ICSImportItemError is a per-event failure that did not abort the whole import.
type ICSImportItemError struct {
	Index   int    `json:"index"`
	UID     string `json:"uid,omitempty"`
	Message string `json:"message"`
}

// ICSImportResult summarizes a completed import attempt.
type ICSImportResult struct {
	CreatedTaskIDs []int64             `json:"created_task_ids"`
	SkippedTODO    int                 `json:"skipped_todo"`
	Errors         []ICSImportItemError `json:"errors,omitempty"`
}

// Import parses the .ics body and creates tasks via TaskService.CreateTask.
// Product fields (requires_confirmation, muted, etc.) stay at defaults (confirmation=false).
// Partial VEVENT failures are returned in Errors; calendar-level parse failure is a hard error.
func (s *ICSImportService) Import(ctx context.Context, r io.Reader, opts ICSImportOptions) (*ICSImportResult, error) {
	ctx, span := s.tracer.Start(ctx, "ics_import_service.Import",
		trace.WithAttributes(attribute.Int64("user.id", opts.UserID)))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)

	if opts.UserID <= 0 {
		err := errors.Wrap(errs.ErrValidation, "user_id is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if _, err := s.users.GetUserByID(ctx, opts.UserID); err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	if opts.GroupID != nil {
		g, err := s.groups.GetTaskGroupByID(ctx, *opts.GroupID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, "group_id not found")
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
		if g.UserID != opts.UserID {
			err := errors.Wrap(errs.ErrValidation, "group_id does not belong to user")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	if opts.MessengerRelatedUserID != nil {
		mru, err := s.messenger.GetMessengerRelatedUserByID(ctx, *opts.MessengerRelatedUserID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
		if mru.UserID == nil || *mru.UserID != opts.UserID {
			err := errors.Wrap(errs.ErrValidation, "messenger_related_user_id does not belong to user")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	parsed, err := ics.Parse(r)
	if err != nil {
		err = errors.Wrap(errs.ErrValidation, fmt.Sprintf("invalid ics: %v", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	result := &ICSImportResult{
		CreatedTaskIDs: make([]int64, 0, len(parsed.Events)),
		SkippedTODO:    parsed.SkippedTODO,
	}
	for _, pe := range parsed.Errors {
		result.Errors = append(result.Errors, ICSImportItemError{
			Index:   pe.Index,
			UID:     pe.UID,
			Message: pe.Message,
		})
	}

	for i, ev := range parsed.Events {
		gcEv := icsEventToGoogle(ev)
		task, mapErr := MapEventToTaskFields(gcEv, opts.UserID, opts.GroupID, opts.MessengerRelatedUserID)
		if mapErr != nil {
			result.Errors = append(result.Errors, ICSImportItemError{
				Index:   i,
				UID:     ev.UID,
				Message: mapErr.Error(),
			})
			continue
		}
		// Google parity: never set confirmation; CreateTask must not spawn children.
		task.RequiresConfirmation = false
		task.HistorySource = models.TaskHistorySourceICSImport

		taskID, _, createErr := s.tasks.CreateTask(ctx, task)
		if createErr != nil {
			result.Errors = append(result.Errors, ICSImportItemError{
				Index:   i,
				UID:     ev.UID,
				Message: createErr.Error(),
			})
			log.Warn().Err(createErr).Str("uid", ev.UID).Msg("ics import: failed to create task")
			continue
		}
		task.ID = taskID
		result.CreatedTaskIDs = append(result.CreatedTaskIDs, taskID)

		if schedErr := s.tasks.PublishImportedTaskSchedule(ctx, task); schedErr != nil {
			log.Warn().Err(schedErr).Int64("task.id", taskID).Msg("ics import: failed to publish schedule")
		}
	}

	span.SetAttributes(
		attribute.Int("ics.created", len(result.CreatedTaskIDs)),
		attribute.Int("ics.errors", len(result.Errors)),
		attribute.Int("ics.skipped_todo", result.SkippedTODO),
	)
	span.SetStatus(codes.Ok, "ics import completed")
	return result, nil
}

func icsEventToGoogle(ev ics.Event) googlecalendar.Event {
	out := googlecalendar.Event{
		ID:          ev.UID,
		Summary:     ev.Summary,
		Description: ev.Description,
		Recurrence:  append([]string(nil), ev.Recurrence...),
	}
	start := ev.Start.UTC()
	if ev.AllDay {
		out.Start = googlecalendar.EventDateTime{
			Date: start.Format("2006-01-02"),
		}
	} else {
		out.Start = googlecalendar.EventDateTime{
			DateTime: &start,
			TimeZone: "UTC",
		}
	}
	if ev.End != nil {
		end := ev.End.UTC()
		if ev.AllDay {
			out.End = googlecalendar.EventDateTime{
				Date: end.Format("2006-01-02"),
			}
		} else {
			out.End = googlecalendar.EventDateTime{
				DateTime: &end,
				TimeZone: "UTC",
			}
		}
	}
	return out
}
