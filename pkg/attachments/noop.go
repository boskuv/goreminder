package attachments

import "context"

type noopClient struct{}

func NewNoopClient() Client {
	return &noopClient{}
}

func (n *noopClient) InitUpload(context.Context, InitUploadInput) (*InitUploadResult, error) {
	return nil, ErrDisabled
}

func (n *noopClient) UploadDirect(context.Context, UploadDirectInput) (*Attachment, error) {
	return nil, ErrDisabled
}

func (n *noopClient) CompleteUpload(context.Context, int64, string) (*Attachment, error) {
	return nil, ErrDisabled
}

func (n *noopClient) ListAttachments(context.Context, int64) ([]Attachment, error) {
	return nil, ErrDisabled
}

func (n *noopClient) GetDownloadURL(context.Context, int64, string) (*DownloadURL, error) {
	return nil, ErrDisabled
}

func (n *noopClient) DownloadDirect(context.Context, int64, string) (*DownloadDirectResult, error) {
	return nil, ErrDisabled
}

func (n *noopClient) DeleteAttachment(context.Context, int64, string) error {
	return ErrDisabled
}

func (n *noopClient) PurgeByTask(context.Context, int64, []int64) error {
	return nil
}

func (n *noopClient) PurgeByUser(context.Context, int64) error {
	return nil
}
