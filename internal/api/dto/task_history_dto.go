package dto

// TaskHistoryResponse represents the response DTO for task history
type TaskHistoryResponse struct {
	ID        int64                  `json:"id" example:"1"`
	TaskID    int64                  `json:"task_id" example:"1"`
	UserID    int64                  `json:"user_id" example:"1"`
	Action    string                 `json:"action" example:"created" enums:"created,updated,deleted,status_changed"`
	OldValue  map[string]interface{} `json:"old_value,omitempty"`
	NewValue  map[string]interface{} `json:"new_value,omitempty"`
	CreatedAt string                 `json:"created_at" example:"2024-01-10T08:00:00Z"` // ISO 8601 format
}
