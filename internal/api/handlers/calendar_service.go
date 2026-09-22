package handlers

import (
	"context"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/googlecalendar"
)

// CalendarService is the calendar use-case surface used by CalendarHandler.
type CalendarService interface {
	StartOAuth(ctx context.Context, userID int64) (string, error)
	HandleOAuthCallback(ctx context.Context, code, state string) (*models.GoogleAccount, error)
	ListGoogleCalendars(ctx context.Context, userID int64) ([]googlecalendar.Calendar, error)
	CreateBinding(ctx context.Context, req service.CreateBindingRequest) (*models.CalendarBinding, error)
	ListBindings(ctx context.Context, userID int64) ([]*models.CalendarBinding, error)
	DeleteBinding(ctx context.Context, bindingID int64) error
	DisconnectGoogle(ctx context.Context, userID int64) error
	SyncBinding(ctx context.Context, bindingID int64) error
	EnableTaskExport(ctx context.Context, taskID, bindingID int64) (*models.TaskSyncLink, error)
	GetTaskExternal(ctx context.Context, taskID int64) (*models.TaskSyncLink, error)
	ListTaskIDsByProvider(ctx context.Context, userID int64, provider string) ([]int64, error)
}
