package googlecalendar

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// PrivateExtendedPropertyTaskID is stored on Google events to link back to goreminder tasks.
	PrivateExtendedPropertyTaskID = "goreminder_task_id"

	ScopeCalendarEvents   = "https://www.googleapis.com/auth/calendar.events"
	ScopeCalendarReadonly = "https://www.googleapis.com/auth/calendar.readonly"
	ScopeUserInfoEmail    = "https://www.googleapis.com/auth/userinfo.email"
	ScopeUserInfoProfile  = "https://www.googleapis.com/auth/userinfo.profile"
)

// NewOAuthConfig builds an OAuth2 config for Google Calendar + userinfo.
func NewOAuthConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			ScopeCalendarEvents,
			ScopeCalendarReadonly,
			ScopeUserInfoEmail,
			ScopeUserInfoProfile,
		},
		Endpoint: google.Endpoint,
	}
}
