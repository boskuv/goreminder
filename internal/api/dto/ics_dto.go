package dto

// ICSImportItemError is a per-VEVENT failure during import.
type ICSImportItemError struct {
	Index   int    `json:"index" example:"0"`
	UID     string `json:"uid,omitempty" example:"event-1@example.com"`
	Message string `json:"message" example:"DTSTART: event has no start"`
}

// ICSImportResponse is returned by POST /users/{user_id}/ics/import.
type ICSImportResponse struct {
	CreatedTaskIDs []int64              `json:"created_task_ids"`
	SkippedTODO    int                  `json:"skipped_todo" example:"0"`
	Errors         []ICSImportItemError `json:"errors,omitempty"`
	Count          int                  `json:"count" example:"3"`
}
