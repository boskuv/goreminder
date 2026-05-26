package dto

import "time"

type InitAttachmentRequest struct {
	OriginalName   string `json:"original_name" binding:"required" example:"report.pdf"`
	ContentType    string `json:"content_type" binding:"required" example:"application/pdf"`
	SizeBytes      int64  `json:"size_bytes" binding:"required,gt=0" example:"1048576"`
	IdempotencyKey string `json:"idempotency_key,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type InitAttachmentResponse struct {
	ID              string            `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TaskID          int64             `json:"task_id" example:"1"`
	Status          string            `json:"status" example:"pending" enums:"pending,ready,failed"`
	UploadURL       string            `json:"upload_url" example:"http://localhost:9000/goreminder-attachments/attachments/1/42/550e8400-e29b-41d4-a716-446655440000?X-Amz-Algorithm=AWS4-HMAC-SHA256"`
	ExpiresAt       time.Time         `json:"expires_at" example:"2026-05-21T12:15:00Z"`
	RequiredHeaders map[string]string `json:"required_headers"`
}

type AttachmentResponse struct {
	ID           string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TaskID       int64     `json:"task_id" example:"1"`
	OriginalName string    `json:"original_name" example:"report.pdf"`
	ContentType  string    `json:"content_type" example:"application/pdf"`
	SizeBytes    int64     `json:"size_bytes" example:"1048576"`
	Status       string    `json:"status" example:"ready" enums:"pending,ready,failed"`
	CreatedAt    time.Time `json:"created_at" example:"2026-05-21T12:00:00Z"`
}

type ListAttachmentsResponse struct {
	Attachments []AttachmentResponse `json:"attachments"`
}

type DownloadAttachmentResponse struct {
	DownloadURL string    `json:"download_url" example:"http://localhost:9000/goreminder-attachments/attachments/1/42/550e8400-e29b-41d4-a716-446655440000?X-Amz-Algorithm=AWS4-HMAC-SHA256"`
	ExpiresAt   time.Time `json:"expires_at" example:"2026-05-21T12:30:00Z"`
}

type AttachmentsDisabledResponse struct {
	Error   string `json:"error" example:"attachments_disabled"`
	Message string `json:"message" example:"Attachment storage is disabled in this deployment"`
}

type ProxyDownloadDisabledResponse struct {
	Error   string `json:"error" example:"proxy_download_disabled"`
	Message string `json:"message" example:"Proxy download is disabled; use GET .../download for a presigned URL"`
}
