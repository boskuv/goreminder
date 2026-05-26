package attachments

import "time"

type Attachment struct {
	ID           string    `json:"id"`
	TaskID       int64     `json:"task_id"`
	UserID       int64     `json:"user_id"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type UploadDirectInput struct {
	TaskID         int64
	UserID         int64
	OriginalName   string
	ContentType    string
	Data           []byte
	IdempotencyKey string
}

type InitUploadInput struct {
	TaskID         int64
	UserID         int64
	OriginalName   string
	ContentType    string
	SizeBytes      int64
	IdempotencyKey string
}

type InitUploadResult struct {
	AttachmentID    string            `json:"id"`
	TaskID          int64             `json:"task_id"`
	Status          string            `json:"status"`
	UploadURL       string            `json:"upload_url"`
	ExpiresAt       time.Time         `json:"expires_at"`
	RequiredHeaders map[string]string `json:"required_headers"`
}

type DownloadURL struct {
	DownloadURL string    `json:"download_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type DownloadDirectResult struct {
	Data         []byte
	ContentType  string
	OriginalName string
	SizeBytes    int64
}
