package dto

// TaskHistoryResponse represents the response DTO for task history
type TaskHistoryResponse struct {
	ID        int64                  `json:"id"`
	TaskID    int64                  `json:"task_id"`
	UserID    int64                  `json:"user_id"`
	Action    string                 `json:"action"`
	OldValue  map[string]interface{} `json:"old_value,omitempty"`
	NewValue  map[string]interface{} `json:"new_value,omitempty"`
	CreatedAt string                 `json:"created_at"` // ISO 8601 format
}
