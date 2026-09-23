package mapper

import (
	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/googlecalendar"
)

// GoogleAccountToResponse maps a GoogleAccount model to a safe API response.
func GoogleAccountToResponse(account *models.GoogleAccount) *dto.GoogleAccountResponse {
	if account == nil {
		return nil
	}
	return &dto.GoogleAccountResponse{
		ID:        account.ID,
		UserID:    account.UserID,
		GoogleSub: account.GoogleSub,
		Email:     account.Email,
		Scopes:    account.Scopes,
		RevokedAt: account.RevokedAt,
		CreatedAt: account.CreatedAt,
	}
}

// CalendarBindingToResponse maps a CalendarBinding to API DTO.
func CalendarBindingToResponse(b *models.CalendarBinding) *dto.CalendarBindingResponse {
	if b == nil {
		return nil
	}
	return &dto.CalendarBindingResponse{
		ID:                     b.ID,
		UserID:                 b.UserID,
		GoogleAccountID:        b.GoogleAccountID,
		GoogleCalendarID:       b.GoogleCalendarID,
		CalendarSummary:        b.CalendarSummary,
		Direction:              string(b.Direction),
		GroupID:                b.GroupID,
		MessengerRelatedUserID: b.MessengerRelatedUserID,
		LastSyncedAt:           b.LastSyncedAt,
		LastError:              b.LastError,
		Status:                 string(b.Status),
		DeletePolicy:           string(b.DeletePolicy),
		CreatedAt:              b.CreatedAt,
		UpdatedAt:              b.UpdatedAt,
	}
}

// CalendarBindingsToResponse maps a slice of bindings.
func CalendarBindingsToResponse(bindings []*models.CalendarBinding) []*dto.CalendarBindingResponse {
	out := make([]*dto.CalendarBindingResponse, len(bindings))
	for i, b := range bindings {
		out[i] = CalendarBindingToResponse(b)
	}
	return out
}

// GoogleCalendarsToResponse maps Google calendar list items.
func GoogleCalendarsToResponse(calendars []googlecalendar.Calendar) []dto.GoogleCalendarListItem {
	out := make([]dto.GoogleCalendarListItem, len(calendars))
	for i, c := range calendars {
		out[i] = dto.GoogleCalendarListItem{
			ID:      c.ID,
			Summary: c.Summary,
			Primary: c.Primary,
		}
	}
	return out
}

// TaskSyncLinkToExternalResponse maps a sync link to the external badge DTO.
func TaskSyncLinkToExternalResponse(link *models.TaskSyncLink) *dto.TaskExternalResponse {
	if link == nil {
		return nil
	}
	return &dto.TaskExternalResponse{
		Provider:          link.Provider,
		CalendarID:        link.GoogleCalendarID,
		EventID:           link.GoogleEventID,
		Origin:            string(link.Origin),
		SyncEnabled:       link.SyncEnabled,
		LastSyncedAt:      link.LastSyncedAt,
		LastError:         link.LastError,
		CalendarBindingID: link.CalendarBindingID,
	}
}
