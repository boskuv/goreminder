package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/config"
	tokencrypto "github.com/boskuv/goreminder/pkg/crypto"
	"github.com/boskuv/goreminder/pkg/googlecalendar"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/boskuv/goreminder/pkg/observability"
)

// CalendarExportHook is called after local task mutations that may need calendar export.
type CalendarExportHook interface {
	OnTaskChanged(ctx context.Context, taskID int64, action string) // created|updated|deleted
}

// NoopCalendarExportHook is used when Google Calendar integration is disabled.
type NoopCalendarExportHook struct{}

func (NoopCalendarExportHook) OnTaskChanged(context.Context, int64, string) {}

// ImportedTaskScheduler publishes messenger worker events for calendar-imported tasks.
type ImportedTaskScheduler interface {
	PublishImportedTaskSchedule(ctx context.Context, task *models.Task) error
	PublishImportedTaskDelete(ctx context.Context, task *models.Task) error
}

// NoopImportedTaskScheduler is used when the queue/worker path is unavailable.
type NoopImportedTaskScheduler struct{}

func (NoopImportedTaskScheduler) PublishImportedTaskSchedule(context.Context, *models.Task) error {
	return nil
}
func (NoopImportedTaskScheduler) PublishImportedTaskDelete(context.Context, *models.Task) error {
	return nil
}

// CreateBindingRequest holds parameters for creating a calendar binding.
type CreateBindingRequest struct {
	UserID                   int64
	GoogleCalendarID         string
	CalendarSummary          *string
	Direction                models.CalendarBindingDirection
	GroupID                  *int64
	MessengerRelatedUserID   *int
	DeletePolicy             models.CalendarDeletePolicy
}

type googleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// CalendarSyncService orchestrates OAuth, bindings, import sync, and export outbox.
type CalendarSyncService struct {
	cfg                 config.GoogleCalendarConfiguration
	oauthCfg            *oauth2.Config
	cipher              *tokencrypto.TokenCipher
	clientFactory       googlecalendar.ClientFactory
	googleAccounts      repository.GoogleAccountRepository
	bindings            repository.CalendarBindingRepository
	syncLinks           repository.TaskSyncLinkRepository
	outbox              repository.SyncOutboxRepository
	tasks               repository.TaskRepository
	users               repository.UserRepository
	messengers          repository.MessengerRepository
	taskHistory         repository.TaskHistoryRepository
	importedScheduler   ImportedTaskScheduler
	httpClient          *http.Client
	tracer              trace.Tracer
	logger              zerolog.Logger
}

// NewCalendarSyncService constructs a CalendarSyncService.
func NewCalendarSyncService(
	cfg config.GoogleCalendarConfiguration,
	oauthCfg *oauth2.Config,
	cipher *tokencrypto.TokenCipher,
	clientFactory googlecalendar.ClientFactory,
	googleAccounts repository.GoogleAccountRepository,
	bindings repository.CalendarBindingRepository,
	syncLinks repository.TaskSyncLinkRepository,
	outbox repository.SyncOutboxRepository,
	tasks repository.TaskRepository,
	users repository.UserRepository,
	messengers repository.MessengerRepository,
	taskHistory repository.TaskHistoryRepository,
	logger zerolog.Logger,
) *CalendarSyncService {
	return &CalendarSyncService{
		cfg:               cfg,
		oauthCfg:          oauthCfg,
		cipher:            cipher,
		clientFactory:     clientFactory,
		googleAccounts:    googleAccounts,
		bindings:          bindings,
		syncLinks:         syncLinks,
		outbox:            outbox,
		tasks:             tasks,
		users:             users,
		messengers:        messengers,
		taskHistory:       taskHistory,
		importedScheduler: NoopImportedTaskScheduler{},
		httpClient:        http.DefaultClient,
		tracer:            otel.Tracer("calendar-sync-service"),
		logger:            logger,
	}
}

// SetImportedTaskScheduler wires messenger worker publishing for imported tasks (call from main after TaskService is ready).
func (s *CalendarSyncService) SetImportedTaskScheduler(scheduler ImportedTaskScheduler) {
	if scheduler == nil {
		s.importedScheduler = NoopImportedTaskScheduler{}
		return
	}
	s.importedScheduler = scheduler
}

// StartOAuth returns the Google consent URL with state=userID.
func (s *CalendarSyncService) StartOAuth(ctx context.Context, userID int64) (string, error) {
	ctx, span := s.tracer.Start(ctx, "calendar_sync_service.StartOAuth",
		trace.WithAttributes(attribute.Int64("user.id", userID)))
	defer span.End()

	if _, err := s.users.GetUserByID(ctx, userID); err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", errors.WithStack(err)
	}

	url := s.oauthCfg.AuthCodeURL(strconv.FormatInt(userID, 10), oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	span.SetStatus(codes.Ok, "ok")
	return url, nil
}

// HandleOAuthCallback exchanges the code, stores encrypted tokens, and upserts google_accounts.
func (s *CalendarSyncService) HandleOAuthCallback(ctx context.Context, code, state string) (*models.GoogleAccount, error) {
	ctx, span := s.tracer.Start(ctx, "calendar_sync_service.HandleOAuthCallback")
	defer span.End()

	userID, err := strconv.ParseInt(state, 10, 64)
	if err != nil || userID <= 0 {
		err = errors.Wrap(errs.ErrValidation, "invalid oauth state")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if _, err := s.users.GetUserByID(ctx, userID); err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	token, err := s.oauthCfg.Exchange(ctx, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "oauth token exchange failed")
	}

	info, err := s.fetchUserInfo(ctx, token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	accessEnc, err := s.cipher.Encrypt([]byte(token.AccessToken))
	if err != nil {
		return nil, errors.Wrap(err, "encrypt access token")
	}
	refreshEnc, err := s.cipher.Encrypt([]byte(token.RefreshToken))
	if err != nil {
		return nil, errors.Wrap(err, "encrypt refresh token")
	}

	account := &models.GoogleAccount{
		UserID:          userID,
		GoogleSub:       info.ID,
		Email:           info.Email,
		AccessTokenEnc:  accessEnc,
		RefreshTokenEnc: refreshEnc,
		TokenExpiry:     token.Expiry,
		Scopes:          strings.Join(s.oauthCfg.Scopes, " "),
	}
	id, err := s.googleAccounts.Upsert(ctx, account)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}
	account.ID = id
	span.SetAttributes(attribute.Int64("google_account.id", id), attribute.Int64("user.id", userID))
	span.SetStatus(codes.Ok, "connected")
	return account, nil
}

func (s *CalendarSyncService) fetchUserInfo(ctx context.Context, token *oauth2.Token) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, errors.Wrap(err, "build userinfo request")
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "fetch userinfo")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read userinfo body")
	}
	if resp.StatusCode >= 300 {
		return nil, errors.Errorf("userinfo status %d: %s", resp.StatusCode, string(body))
	}
	var info googleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, errors.Wrap(err, "decode userinfo")
	}
	if info.ID == "" {
		return nil, errors.New("userinfo missing id")
	}
	return &info, nil
}

// ListGoogleCalendars lists calendars for the user's connected Google account.
func (s *CalendarSyncService) ListGoogleCalendars(ctx context.Context, userID int64) ([]googlecalendar.Calendar, error) {
	ctx, span := s.tracer.Start(ctx, "calendar_sync_service.ListGoogleCalendars",
		trace.WithAttributes(attribute.Int64("user.id", userID)))
	defer span.End()

	client, _, err := s.clientForUser(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	calendars, err := client.ListCalendars(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}
	span.SetStatus(codes.Ok, "ok")
	return calendars, nil
}

// CreateBinding creates a calendar binding for the user.
func (s *CalendarSyncService) CreateBinding(ctx context.Context, req CreateBindingRequest) (*models.CalendarBinding, error) {
	ctx, span := s.tracer.Start(ctx, "calendar_sync_service.CreateBinding",
		trace.WithAttributes(attribute.Int64("user.id", req.UserID)))
	defer span.End()

	account, err := s.googleAccounts.GetByUserID(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, "google account not connected")
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if strings.TrimSpace(req.GoogleCalendarID) == "" {
		err = errors.Wrap(errs.ErrValidation, "google_calendar_id is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	direction := req.Direction
	if direction == "" {
		direction = models.CalendarBindingDirectionImport
	}
	policy := req.DeletePolicy
	if policy == "" {
		policy = models.CalendarDeletePolicySoftDeleteImported
	}

	if req.MessengerRelatedUserID != nil {
		mru, err := s.messengers.GetMessengerRelatedUserByID(ctx, *req.MessengerRelatedUserID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, "messenger_related_user_id not found")
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
		if mru.UserID == nil || *mru.UserID != req.UserID {
			err = errors.Wrap(errs.ErrValidation, "messenger_related_user_id does not belong to user")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	binding := &models.CalendarBinding{
		UserID:                 req.UserID,
		GoogleAccountID:        account.ID,
		GoogleCalendarID:       req.GoogleCalendarID,
		CalendarSummary:        req.CalendarSummary,
		Direction:              direction,
		GroupID:                req.GroupID,
		MessengerRelatedUserID: req.MessengerRelatedUserID,
		Status:                 models.CalendarBindingStatusActive,
		DeletePolicy:           policy,
	}
	id, err := s.bindings.Create(ctx, binding)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}
	binding.ID = id
	span.SetAttributes(attribute.Int64("calendar_binding.id", id))
	span.SetStatus(codes.Ok, "created")
	return binding, nil
}

// ListBindings returns calendar bindings for a user.
func (s *CalendarSyncService) ListBindings(ctx context.Context, userID int64) ([]*models.CalendarBinding, error) {
	return s.bindings.ListByUserID(ctx, userID)
}

// DeleteBinding soft-deletes a binding and applies delete_policy to imported tasks.
func (s *CalendarSyncService) DeleteBinding(ctx context.Context, bindingID int64) error {
	ctx, span := s.tracer.Start(ctx, "calendar_sync_service.DeleteBinding",
		trace.WithAttributes(attribute.Int64("calendar_binding.id", bindingID)))
	defer span.End()

	binding, err := s.bindings.GetByID(ctx, bindingID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	links, err := s.syncLinks.ListByBindingID(ctx, bindingID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	for _, link := range links {
		if link.Origin != models.TaskSyncLinkOriginImported {
			continue
		}
		task, err := s.tasks.GetTaskByIDWithoutStatusFilter(ctx, link.TaskID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				continue
			}
			return errors.WithStack(err)
		}
		if err := s.applyImportedDeletePolicy(ctx, binding.DeletePolicy, task); err != nil {
			return err
		}
	}

	if err := s.bindings.SoftDelete(ctx, bindingID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}
	span.SetStatus(codes.Ok, "deleted")
	return nil
}

// DisconnectGoogle revokes tokens and disconnects all bindings for the user.
func (s *CalendarSyncService) DisconnectGoogle(ctx context.Context, userID int64) error {
	ctx, span := s.tracer.Start(ctx, "calendar_sync_service.DisconnectGoogle",
		trace.WithAttributes(attribute.Int64("user.id", userID)))
	defer span.End()

	account, err := s.googleAccounts.GetByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	bindings, err := s.bindings.ListByUserID(ctx, userID)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, b := range bindings {
		if err := s.DeleteBinding(ctx, b.ID); err != nil {
			return err
		}
	}

	// Best-effort remote revoke
	if token, err := s.decryptToken(account); err == nil && token.AccessToken != "" {
		_ = s.revokeRemoteToken(ctx, token.AccessToken)
	}

	if err := s.googleAccounts.Revoke(ctx, account.ID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}
	span.SetStatus(codes.Ok, "disconnected")
	return nil
}

func (s *CalendarSyncService) revokeRemoteToken(ctx context.Context, accessToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://oauth2.googleapis.com/revoke?token="+accessToken, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SyncBinding imports events for a binding (and is used by ForceSync).
func (s *CalendarSyncService) SyncBinding(ctx context.Context, bindingID int64) error {
	ctx, span := s.tracer.Start(ctx, "calendar_sync_service.SyncBinding",
		trace.WithAttributes(attribute.Int64("calendar_binding.id", bindingID)))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	binding, err := s.bindings.GetByID(ctx, bindingID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}
	if binding.Direction != models.CalendarBindingDirectionImport && binding.Direction != models.CalendarBindingDirectionBoth {
		span.SetStatus(codes.Ok, "skip non-import")
		return nil
	}

	client, _, err := s.clientForAccountID(ctx, binding.GoogleAccountID)
	if err != nil {
		s.markBindingError(ctx, binding, err)
		observability.CalendarSyncErrors.Inc()
		return err
	}

	err = s.importEvents(ctx, client, binding)
	if err != nil {
		s.markBindingError(ctx, binding, err)
		observability.CalendarSyncErrors.Inc()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error().Err(err).Int64("calendar_binding.id", bindingID).Msg("calendar sync failed")
		return err
	}

	now := time.Now().UTC()
	binding.LastSyncedAt = &now
	binding.LastError = nil
	binding.Status = models.CalendarBindingStatusActive
	_ = s.bindings.Update(ctx, binding)
	observability.CalendarSyncSuccess.Inc()
	span.SetStatus(codes.Ok, "synced")
	return nil
}

func (s *CalendarSyncService) importEvents(ctx context.Context, client googlecalendar.CalendarClient, binding *models.CalendarBinding) error {
	windowDays := s.cfg.InitialSyncWindowDays
	if windowDays <= 0 {
		windowDays = 90
	}

	var pageToken string
	for {
		opts := googlecalendar.ListEventsOpts{PageToken: pageToken, MaxResults: 250}
		if binding.SyncToken != nil && *binding.SyncToken != "" {
			opts.SyncToken = *binding.SyncToken
		} else {
			now := time.Now().UTC()
			min := now.AddDate(0, 0, -windowDays)
			max := now.AddDate(0, 0, windowDays)
			opts.TimeMin = &min
			opts.TimeMax = &max
		}

		result, err := client.ListEvents(ctx, binding.GoogleCalendarID, opts)
		if err != nil {
			if googlecalendar.IsSyncTokenInvalid(err) {
				_ = s.bindings.ClearSyncToken(ctx, binding.ID)
				binding.SyncToken = nil
				return s.importEvents(ctx, client, binding)
			}
			return errors.WithStack(err)
		}

		for _, ev := range result.Events {
			if err := s.upsertImportedEvent(ctx, binding, ev); err != nil {
				return err
			}
		}

		if result.NextPageToken != "" {
			pageToken = result.NextPageToken
			continue
		}
		if result.NextSyncToken != "" {
			tok := result.NextSyncToken
			binding.SyncToken = &tok
		}
		break
	}
	return nil
}

func (s *CalendarSyncService) upsertImportedEvent(ctx context.Context, binding *models.CalendarBinding, ev googlecalendar.Event) error {
	link, err := s.syncLinks.GetByEvent(ctx, models.TaskSyncProviderGoogleCalendar, binding.GoogleCalendarID, ev.ID)
	if err != nil && !errors.Is(err, errs.ErrNotFound) {
		return errors.WithStack(err)
	}

	cancelled := strings.EqualFold(ev.Status, "cancelled")
	if cancelled {
		if link == nil {
			return nil
		}
		return s.applyCancelledEvent(ctx, binding, link)
	}

	// Anti-loop: event was created/exported by us — ensure link exists, do not create a second task.
	if link == nil {
		if taskID, ok := TaskIDFromExtendedProperties(ev.ExtendedProperties); ok && taskID > 0 {
			if existing, err := s.syncLinks.GetByTaskID(ctx, taskID); err == nil {
				link = existing
			} else if errors.Is(err, errs.ErrNotFound) {
				now := time.Now().UTC()
				etag := ev.ETag
				dur := EventDurationSeconds(ev)
				updated := ev.Updated
				bindingID := binding.ID
				link = &models.TaskSyncLink{
					TaskID:            taskID,
					Provider:          models.TaskSyncProviderGoogleCalendar,
					GoogleCalendarID:  binding.GoogleCalendarID,
					GoogleEventID:     ev.ID,
					ETag:              &etag,
					GoogleUpdatedAt:   &updated,
					Origin:            models.TaskSyncLinkOriginExported,
					SyncEnabled:       true,
					CalendarBindingID: &bindingID,
					DurationSeconds:   dur,
					LastSyncedAt:      &now,
				}
				if _, err := s.syncLinks.Create(ctx, link); err != nil {
					return errors.WithStack(err)
				}
				return nil
			} else {
				return errors.WithStack(err)
			}
		}
	}

	fields, err := MapEventToTaskFields(ev, binding.UserID, binding.GroupID, binding.MessengerRelatedUserID)
	if err != nil {
		return errors.Wrap(err, "map event to task")
	}

	now := time.Now().UTC()
	etag := ev.ETag
	dur := EventDurationSeconds(ev)
	updated := ev.Updated

	if link == nil {
		taskID, err := s.tasks.CreateTask(ctx, fields)
		if err != nil {
			return errors.WithStack(err)
		}
		fields.ID = taskID
		s.recordImportedTaskHistory(ctx, models.TaskHistoryActionCreated, fields, nil, calendarTaskHistoryMap(fields))
		bindingID := binding.ID
		link = &models.TaskSyncLink{
			TaskID:            taskID,
			Provider:          models.TaskSyncProviderGoogleCalendar,
			GoogleCalendarID:  binding.GoogleCalendarID,
			GoogleEventID:     ev.ID,
			ETag:              &etag,
			GoogleUpdatedAt:   &updated,
			Origin:            models.TaskSyncLinkOriginImported,
			SyncEnabled:       true,
			CalendarBindingID: &bindingID,
			DurationSeconds:   dur,
			LastSyncedAt:      &now,
		}
		if _, err = s.syncLinks.Create(ctx, link); err != nil {
			return errors.WithStack(err)
		}
		if err := s.importedScheduler.PublishImportedTaskSchedule(ctx, fields); err != nil {
			log := logger.WithTraceContext(ctx, s.logger)
			log.Warn().Err(err).Int64("task.id", taskID).Msg("failed to publish schedule for imported calendar task")
		}
		return nil
	}

	task, err := s.tasks.GetTaskByIDWithoutStatusFilter(ctx, link.TaskID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			taskID, err := s.tasks.CreateTask(ctx, fields)
			if err != nil {
				return errors.WithStack(err)
			}
			fields.ID = taskID
			link.TaskID = taskID
			s.recordImportedTaskHistory(ctx, models.TaskHistoryActionCreated, fields, nil, calendarTaskHistoryMap(fields))
			if err := s.importedScheduler.PublishImportedTaskSchedule(ctx, fields); err != nil {
				log := logger.WithTraceContext(ctx, s.logger)
				log.Warn().Err(err).Int64("task.id", taskID).Msg("failed to publish schedule for recreated imported calendar task")
			}
		} else {
			return errors.WithStack(err)
		}
	} else {
		oldSnapshot := *task
		oldHistoryMap := calendarTaskHistoryMap(&oldSnapshot)
		oldStatus := task.Status

		scheduleChanged := !task.StartDate.Equal(fields.StartDate) ||
			!recurrencePtrsEqual(task.RRule, fields.RRule) ||
			task.Title != fields.Title ||
			task.Description != fields.Description
		wasScheduled := task.Status == string(models.TaskStatusScheduled) ||
			task.Status == string(models.TaskStatusRescheduled) ||
			task.Status == string(models.TaskStatusPostponed)

		task.Title = fields.Title
		task.Description = fields.Description
		task.StartDate = fields.StartDate
		task.RRule = fields.RRule
		task.GroupID = fields.GroupID
		if fields.MessengerRelatedUserID != nil {
			task.MessengerRelatedUserID = fields.MessengerRelatedUserID
		}
		task.Status = fields.Status
		if fields.Status == string(models.TaskStatusDone) {
			task.FinishDate = fields.FinishDate
		} else if fields.Status == string(models.TaskStatusScheduled) {
			task.FinishDate = nil
		}

		if calendarImportedFieldsChanged(&oldSnapshot, task) {
			if err := s.tasks.UpdateTask(ctx, task); err != nil {
				return errors.WithStack(err)
			}
			newHistoryMap := calendarTaskHistoryMap(task)
			if oldStatus != task.Status {
				s.recordImportedTaskHistory(ctx, models.TaskHistoryActionStatusChanged, task,
					map[string]interface{}{"status": oldStatus},
					map[string]interface{}{"status": task.Status},
				)
			}
			// Full snapshot when non-status fields changed, or always as updated companion when only status?
			// Record updated when title/start/rrule/group/mru/finish/muted-relevant fields changed.
			nonStatusChanged := oldSnapshot.Title != task.Title ||
				oldSnapshot.Description != task.Description ||
				!oldSnapshot.StartDate.Equal(task.StartDate) ||
				!recurrencePtrsEqual(oldSnapshot.RRule, task.RRule) ||
				(oldSnapshot.GroupID == nil) != (task.GroupID == nil) ||
				(oldSnapshot.GroupID != nil && task.GroupID != nil && *oldSnapshot.GroupID != *task.GroupID) ||
				(oldSnapshot.MessengerRelatedUserID == nil) != (task.MessengerRelatedUserID == nil) ||
				(oldSnapshot.MessengerRelatedUserID != nil && task.MessengerRelatedUserID != nil &&
					*oldSnapshot.MessengerRelatedUserID != *task.MessengerRelatedUserID) ||
				(oldSnapshot.FinishDate == nil) != (task.FinishDate == nil) ||
				(oldSnapshot.FinishDate != nil && task.FinishDate != nil && !oldSnapshot.FinishDate.Equal(*task.FinishDate))
			if nonStatusChanged || oldStatus == task.Status {
				s.recordImportedTaskHistory(ctx, models.TaskHistoryActionUpdated, task, oldHistoryMap, newHistoryMap)
			}
		}

		if task.MessengerRelatedUserID != nil {
			if fields.Status == string(models.TaskStatusScheduled) && (scheduleChanged || !wasScheduled) {
				if err := s.importedScheduler.PublishImportedTaskSchedule(ctx, task); err != nil {
					log := logger.WithTraceContext(ctx, s.logger)
					log.Warn().Err(err).Int64("task.id", task.ID).Msg("failed to republish schedule for imported calendar task")
				}
			} else if wasScheduled && fields.Status != string(models.TaskStatusScheduled) {
				if err := s.importedScheduler.PublishImportedTaskDelete(ctx, task); err != nil {
					log := logger.WithTraceContext(ctx, s.logger)
					log.Warn().Err(err).Int64("task.id", task.ID).Msg("failed to delete worker job for past imported calendar task")
				}
			}
		}
	}

	link.ETag = &etag
	link.GoogleUpdatedAt = &updated
	link.DurationSeconds = dur
	link.LastSyncedAt = &now
	link.LastError = nil
	bindingID := binding.ID
	link.CalendarBindingID = &bindingID
	return errors.WithStack(s.syncLinks.Update(ctx, link))
}

func recurrencePtrsEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func (s *CalendarSyncService) applyCancelledEvent(ctx context.Context, binding *models.CalendarBinding, link *models.TaskSyncLink) error {
	task, err := s.tasks.GetTaskByIDWithoutStatusFilter(ctx, link.TaskID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return s.syncLinks.Delete(ctx, link.ID)
		}
		return errors.WithStack(err)
	}
	return s.applyImportedDeletePolicy(ctx, binding.DeletePolicy, task)
}

// applyImportedDeletePolicy applies binding delete_policy to an imported task and records history.
func (s *CalendarSyncService) applyImportedDeletePolicy(ctx context.Context, policy models.CalendarDeletePolicy, task *models.Task) error {
	softDelete, mute := ApplyDeletePolicy(policy, task)
	if softDelete {
		_ = s.importedScheduler.PublishImportedTaskDelete(ctx, task)
		oldMap := calendarTaskHistoryMap(task)
		if err := s.tasks.DeleteTask(ctx, task.ID); err != nil {
			return errors.WithStack(err)
		}
		s.recordImportedTaskHistory(ctx, models.TaskHistoryActionDeleted, task, oldMap, map[string]interface{}{
			"status": string(models.TaskStatusDeleted),
		})
		return nil
	}
	if mute && !task.Muted {
		_ = s.importedScheduler.PublishImportedTaskDelete(ctx, task)
		oldMap := calendarTaskHistoryMap(task)
		task.Muted = true
		if err := s.tasks.UpdateTask(ctx, task); err != nil {
			return errors.WithStack(err)
		}
		s.recordImportedTaskHistory(ctx, models.TaskHistoryActionUpdated, task, oldMap, calendarTaskHistoryMap(task))
	}
	return nil
}

func (s *CalendarSyncService) markBindingError(ctx context.Context, binding *models.CalendarBinding, err error) {
	msg := err.Error()
	binding.LastError = &msg
	binding.Status = models.CalendarBindingStatusError
	_ = s.bindings.Update(ctx, binding)
}

// OnTaskChanged implements CalendarExportHook.
func (s *CalendarSyncService) OnTaskChanged(ctx context.Context, taskID int64, action string) {
	log := logger.WithTraceContext(ctx, s.logger)
	if err := s.EnqueueExport(ctx, taskID, action); err != nil {
		log.Debug().Err(err).Int64("task.id", taskID).Str("action", action).Msg("calendar export enqueue skipped/failed")
	}
}

// EnqueueExport queues an export upsert/delete when an export/both binding exists
// or the task has an explicit sync_enabled link to an export/both binding.
// Import-only bindings never receive local→Google pushes (even if the task was imported via that link).
func (s *CalendarSyncService) EnqueueExport(ctx context.Context, taskID int64, action string) error {
	task, err := s.tasks.GetTaskByIDWithoutStatusFilter(ctx, taskID)
	if err != nil {
		// DeleteTask soft-deletes first, then notifies export. GetTask* filters deleted_at,
		// so fall back to sync-link delete — same outcomes as the live-task path for Google.
		if action == "deleted" && errors.Is(err, errs.ErrNotFound) {
			return s.enqueueExportDeleteForGoneTask(ctx, taskID)
		}
		return errors.WithStack(err)
	}
	if !ShouldExportTask(task) {
		return nil
	}

	kind := OutboxKindExportUpsert
	if action == "deleted" {
		kind = OutboxKindExportDelete
	}

	enqueued := false
	bindingIDs := map[int64]struct{}{}

	if link, err := s.syncLinks.GetByTaskID(ctx, taskID); err == nil &&
		link.SyncEnabled && link.CalendarBindingID != nil {
		if b, bErr := s.bindings.GetByID(ctx, *link.CalendarBindingID); bErr == nil && bindingAllowsTaskExport(b.Direction) {
			bindingIDs[b.ID] = struct{}{}
		}
	}

	bindings, err := s.bindings.ListByUserID(ctx, task.UserID)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, b := range bindings {
		if !bindingAllowsTaskExport(b.Direction) {
			continue
		}
		if b.GroupID != nil {
			if task.GroupID == nil || *task.GroupID != *b.GroupID {
				continue
			}
		}
		bindingIDs[b.ID] = struct{}{}
	}

	for bindingID := range bindingIDs {
		payload, _ := json.Marshal(map[string]any{
			"task_id":    taskID,
			"binding_id": bindingID,
			"action":     action,
		})
		if _, err := s.outbox.Enqueue(ctx, kind, payload); err != nil {
			return errors.WithStack(err)
		}
		enqueued = true
	}

	// Also enqueue delete if an existing exported link exists without matching binding filter
	if !enqueued && action == "deleted" {
		return s.enqueueExportDeleteByExportedLink(ctx, taskID)
	}
	return nil
}

// enqueueExportDeleteForGoneTask handles action=deleted after the task row is soft-deleted
// (GetTaskByIDWithoutStatusFilter returns NotFound). Uses sync link only — no Google event
// exists without a link, and import-only bindings still do not get a local→Google delete.
func (s *CalendarSyncService) enqueueExportDeleteForGoneTask(ctx context.Context, taskID int64) error {
	link, err := s.syncLinks.GetByTaskID(ctx, taskID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil
		}
		return errors.WithStack(err)
	}

	if link.SyncEnabled && link.CalendarBindingID != nil {
		if b, bErr := s.bindings.GetByID(ctx, *link.CalendarBindingID); bErr == nil && bindingAllowsTaskExport(b.Direction) {
			payload, _ := json.Marshal(map[string]any{
				"task_id":    taskID,
				"binding_id": b.ID,
				"action":     "deleted",
			})
			_, err := s.outbox.Enqueue(ctx, OutboxKindExportDelete, payload)
			return errors.WithStack(err)
		}
	}

	if link.Origin != models.TaskSyncLinkOriginExported {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{"task_id": taskID, "link_id": link.ID})
	_, err = s.outbox.Enqueue(ctx, OutboxKindExportDelete, payload)
	return errors.WithStack(err)
}

func (s *CalendarSyncService) enqueueExportDeleteByExportedLink(ctx context.Context, taskID int64) error {
	link, err := s.syncLinks.GetByTaskID(ctx, taskID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil
		}
		return errors.WithStack(err)
	}
	if link.Origin != models.TaskSyncLinkOriginExported {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{"task_id": taskID, "link_id": link.ID})
	_, err = s.outbox.Enqueue(ctx, OutboxKindExportDelete, payload)
	return errors.WithStack(err)
}

// EnableTaskExport opts a single task into export for the given binding (even without a group).
func (s *CalendarSyncService) EnableTaskExport(ctx context.Context, taskID, bindingID int64) (*models.TaskSyncLink, error) {
	ctx, span := s.tracer.Start(ctx, "calendar_sync_service.EnableTaskExport",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
			attribute.Int64("calendar_binding.id", bindingID),
		))
	defer span.End()

	task, err := s.tasks.GetTaskByIDWithoutStatusFilter(ctx, taskID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}
	if !ShouldExportTask(task) {
		err = errors.Wrap(errs.ErrValidation, "confirmation child tasks cannot be exported individually")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	binding, err := s.bindings.GetByID(ctx, bindingID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}
	if binding.UserID != task.UserID {
		err = errors.Wrap(errs.ErrUnprocessableEntity, "binding does not belong to task owner")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if !bindingAllowsTaskExport(binding.Direction) {
		err = errors.Wrap(errs.ErrValidation, "binding direction must be export or both")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	link, err := s.syncLinks.GetByTaskID(ctx, taskID)
	if err != nil && !errors.Is(err, errs.ErrNotFound) {
		return nil, errors.WithStack(err)
	}
	if link == nil {
		link = &models.TaskSyncLink{
			TaskID:            taskID,
			Provider:          models.TaskSyncProviderGoogleCalendar,
			GoogleCalendarID:  binding.GoogleCalendarID,
			GoogleEventID:     fmt.Sprintf("pending-%d", taskID),
			Origin:            models.TaskSyncLinkOriginExported,
			SyncEnabled:       true,
			CalendarBindingID: &bindingID,
		}
		id, err := s.syncLinks.Create(ctx, link)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
		link.ID = id
	} else {
		link.SyncEnabled = true
		link.CalendarBindingID = &bindingID
		link.GoogleCalendarID = binding.GoogleCalendarID
		if link.Origin == "" {
			link.Origin = models.TaskSyncLinkOriginExported
		}
		if err := s.syncLinks.Update(ctx, link); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
	}

	if err := s.EnqueueExport(ctx, taskID, "updated"); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "enabled")
	return link, nil
}

// ListTaskIDsByProvider returns task IDs for a user that have a sync link with the given provider.
func (s *CalendarSyncService) ListTaskIDsByProvider(ctx context.Context, userID int64, provider string) ([]int64, error) {
	return s.syncLinks.ListTaskIDsByProvider(ctx, userID, provider)
}

type outboxPayload struct {
	TaskID    int64 `json:"task_id"`
	BindingID int64 `json:"binding_id"`
	LinkID    int64 `json:"link_id"`
	Action    string `json:"action"`
}

// ProcessOutbox claims due outbox items and processes export upserts/deletes.
func (s *CalendarSyncService) ProcessOutbox(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "calendar_sync_service.ProcessOutbox")
	defer span.End()

	items, err := s.outbox.ClaimDue(ctx, 50)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	if pending, err := s.outbox.CountPending(ctx); err == nil {
		observability.CalendarOutboxDepth.Set(float64(pending))
	}

	for _, item := range items {
		if err := s.processOutboxItem(ctx, item); err != nil {
			attempts := item.Attempts + 1
			if attempts >= maxOutboxAttempts {
				_ = s.outbox.MarkFailed(ctx, item.ID, err.Error())
				observability.CalendarOutboxProcessed.WithLabelValues("failed").Inc()
				continue
			}
			next := time.Now().UTC().Add(OutboxBackoff(attempts))
			_ = s.outbox.MarkRetry(ctx, item.ID, attempts, next, err.Error())
			observability.CalendarOutboxProcessed.WithLabelValues("retry").Inc()
			continue
		}
		_ = s.outbox.MarkDone(ctx, item.ID)
		observability.CalendarOutboxProcessed.WithLabelValues("done").Inc()
	}
	span.SetStatus(codes.Ok, "processed")
	return nil
}

func (s *CalendarSyncService) processOutboxItem(ctx context.Context, item *models.SyncOutbox) error {
	var payload outboxPayload
	if err := json.Unmarshal(item.Payload, &payload); err != nil {
		return errors.Wrap(err, "decode outbox payload")
	}

	switch item.Kind {
	case OutboxKindExportUpsert:
		return s.exportUpsert(ctx, payload)
	case OutboxKindExportDelete:
		return s.exportDelete(ctx, payload)
	default:
		return fmt.Errorf("unknown outbox kind %q", item.Kind)
	}
}

func (s *CalendarSyncService) exportUpsert(ctx context.Context, payload outboxPayload) error {
	task, err := s.tasks.GetTaskByIDWithoutStatusFilter(ctx, payload.TaskID)
	if err != nil {
		return errors.WithStack(err)
	}
	if !ShouldExportTask(task) {
		return nil
	}

	binding, err := s.bindings.GetByID(ctx, payload.BindingID)
	if err != nil {
		return errors.WithStack(err)
	}
	if !bindingAllowsTaskExport(binding.Direction) {
		return nil
	}
	client, _, err := s.clientForAccountID(ctx, binding.GoogleAccountID)
	if err != nil {
		return err
	}

	link, err := s.syncLinks.GetByTaskID(ctx, task.ID)
	if err != nil && !errors.Is(err, errs.ErrNotFound) {
		return errors.WithStack(err)
	}

	var existingDur *int
	if link != nil {
		existingDur = link.DurationSeconds
	}
	ev := BuildExportEvent(task, s.cfg.DefaultEventDurationMinutes, existingDur)
	now := time.Now().UTC()

	if link != nil && link.GoogleEventID != "" &&
		!strings.HasPrefix(link.GoogleEventID, "pending-") &&
		link.GoogleCalendarID == binding.GoogleCalendarID {
		updated, err := client.PatchEvent(ctx, binding.GoogleCalendarID, link.GoogleEventID, ev)
		if err != nil {
			return errors.WithStack(err)
		}
		etag := updated.ETag
		u := updated.Updated
		link.ETag = &etag
		link.GoogleUpdatedAt = &u
		link.LastSyncedAt = &now
		link.LastError = nil
		dur := EventDurationSeconds(*updated)
		if dur != nil {
			link.DurationSeconds = dur
		}
		return errors.WithStack(s.syncLinks.Update(ctx, link))
	}

	created, err := client.CreateEvent(ctx, binding.GoogleCalendarID, ev)
	if err != nil {
		return errors.WithStack(err)
	}
	etag := created.ETag
	u := created.Updated
	bindingID := binding.ID
	dur := EventDurationSeconds(*created)
	if link != nil {
		link.GoogleEventID = created.ID
		link.GoogleCalendarID = binding.GoogleCalendarID
		link.ETag = &etag
		link.GoogleUpdatedAt = &u
		link.Origin = models.TaskSyncLinkOriginExported
		link.SyncEnabled = true
		link.CalendarBindingID = &bindingID
		link.DurationSeconds = dur
		link.LastSyncedAt = &now
		link.LastError = nil
		return errors.WithStack(s.syncLinks.Update(ctx, link))
	}
	newLink := &models.TaskSyncLink{
		TaskID:            task.ID,
		Provider:          models.TaskSyncProviderGoogleCalendar,
		GoogleCalendarID:  binding.GoogleCalendarID,
		GoogleEventID:     created.ID,
		ETag:              &etag,
		GoogleUpdatedAt:   &u,
		Origin:            models.TaskSyncLinkOriginExported,
		SyncEnabled:       true,
		CalendarBindingID: &bindingID,
		DurationSeconds:   dur,
		LastSyncedAt:      &now,
	}
	_, err = s.syncLinks.Create(ctx, newLink)
	return errors.WithStack(err)
}

func (s *CalendarSyncService) exportDelete(ctx context.Context, payload outboxPayload) error {
	var link *models.TaskSyncLink
	var err error
	if payload.TaskID > 0 {
		link, err = s.syncLinks.GetByTaskID(ctx, payload.TaskID)
	}
	if (link == nil || err != nil) && payload.LinkID > 0 {
		// fallback unused; links are keyed by task
		_ = payload.LinkID
	}
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil
		}
		return errors.WithStack(err)
	}
	if link == nil {
		return nil
	}

	var accountID int64
	if link.CalendarBindingID != nil {
		binding, err := s.bindings.GetByID(ctx, *link.CalendarBindingID)
		if err == nil {
			accountID = binding.GoogleAccountID
		}
	}
	if accountID == 0 && payload.BindingID > 0 {
		binding, err := s.bindings.GetByID(ctx, payload.BindingID)
		if err == nil {
			accountID = binding.GoogleAccountID
		}
	}
	if accountID == 0 {
		_ = s.syncLinks.Delete(ctx, link.ID)
		return nil
	}

	client, _, err := s.clientForAccountID(ctx, accountID)
	if err != nil {
		return err
	}
	if err := client.DeleteEvent(ctx, link.GoogleCalendarID, link.GoogleEventID); err != nil {
		// treat already-deleted as success
		if !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "Not Found") {
			return errors.WithStack(err)
		}
	}
	return errors.WithStack(s.syncLinks.Delete(ctx, link.ID))
}

// GetTaskExternal returns sync link info for a task.
func (s *CalendarSyncService) GetTaskExternal(ctx context.Context, taskID int64) (*models.TaskSyncLink, error) {
	return s.syncLinks.GetByTaskID(ctx, taskID)
}

func (s *CalendarSyncService) clientForUser(ctx context.Context, userID int64) (googlecalendar.CalendarClient, *models.GoogleAccount, error) {
	account, err := s.googleAccounts.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, nil, errors.Wrap(errs.ErrUnprocessableEntity, "google account not connected")
		}
		return nil, nil, errors.WithStack(err)
	}
	return s.clientForAccount(ctx, account)
}

func (s *CalendarSyncService) clientForAccountID(ctx context.Context, accountID int64) (googlecalendar.CalendarClient, *models.GoogleAccount, error) {
	account, err := s.googleAccounts.GetByID(ctx, accountID)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	if account.RevokedAt != nil {
		return nil, nil, errors.Wrap(errs.ErrUnprocessableEntity, "google account revoked")
	}
	return s.clientForAccount(ctx, account)
}

func (s *CalendarSyncService) clientForAccount(ctx context.Context, account *models.GoogleAccount) (googlecalendar.CalendarClient, *models.GoogleAccount, error) {
	token, err := s.decryptToken(account)
	if err != nil {
		return nil, nil, err
	}
	client, err := s.clientFactory(ctx, token)
	if err != nil {
		return nil, nil, errors.Wrap(err, "create calendar client")
	}
	return client, account, nil
}

func (s *CalendarSyncService) decryptToken(account *models.GoogleAccount) (*oauth2.Token, error) {
	access, err := s.cipher.Decrypt(account.AccessTokenEnc)
	if err != nil {
		return nil, errors.Wrap(err, "decrypt access token")
	}
	refresh, err := s.cipher.Decrypt(account.RefreshTokenEnc)
	if err != nil {
		return nil, errors.Wrap(err, "decrypt refresh token")
	}
	return &oauth2.Token{
		AccessToken:  string(access),
		RefreshToken: string(refresh),
		Expiry:       account.TokenExpiry,
		TokenType:    "Bearer",
	}, nil
}
