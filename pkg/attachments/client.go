package attachments

import "context"

// Client talks to the attachments gRPC service.
type Client interface {
	InitUpload(ctx context.Context, req InitUploadInput) (*InitUploadResult, error)
	UploadDirect(ctx context.Context, req UploadDirectInput) (*Attachment, error)
	CompleteUpload(ctx context.Context, taskID int64, attachmentID string) (*Attachment, error)
	ListAttachments(ctx context.Context, taskID int64) ([]Attachment, error)
	GetDownloadURL(ctx context.Context, taskID int64, attachmentID string) (*DownloadURL, error)
	DownloadDirect(ctx context.Context, taskID int64, attachmentID string) (*DownloadDirectResult, error)
	DeleteAttachment(ctx context.Context, taskID int64, attachmentID string) error
	PurgeByTask(ctx context.Context, taskID int64, childTaskIDs []int64) error
	PurgeByUser(ctx context.Context, userID int64) error
}
