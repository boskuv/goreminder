package attachments

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	attachmentsv1 "github.com/boskuv/goreminder/api/gen/attachments/v1"
)

type GRPCConfig struct {
	Addr               string
	Timeout            time.Duration
	MaxSendMessageSize int
}

type grpcClient struct {
	api     attachmentsv1.AttachmentServiceClient
	timeout time.Duration
}

func NewGRPCClient(cfg GRPCConfig) (Client, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("attachments grpc addr is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if cfg.MaxSendMessageSize > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(cfg.MaxSendMessageSize),
			grpc.MaxCallRecvMsgSize(cfg.MaxSendMessageSize),
		))
	}
	conn, err := grpc.NewClient(cfg.Addr, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "dial attachments grpc")
	}
	return &grpcClient{
		api:     attachmentsv1.NewAttachmentServiceClient(conn),
		timeout: cfg.Timeout,
	}, nil
}

func (c *grpcClient) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.timeout)
}

func (c *grpcClient) InitUpload(ctx context.Context, req InitUploadInput) (*InitUploadResult, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	resp, err := c.api.InitUpload(ctx, &attachmentsv1.InitUploadRequest{
		TaskId:         req.TaskID,
		UserId:         req.UserID,
		OriginalName:   req.OriginalName,
		ContentType:    req.ContentType,
		SizeBytes:      req.SizeBytes,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	var exp time.Time
	if resp.GetExpiresAt() != nil {
		exp = resp.GetExpiresAt().AsTime()
	}
	return &InitUploadResult{
		AttachmentID:    resp.GetAttachmentId(),
		TaskID:          resp.GetTaskId(),
		Status:          resp.GetStatus(),
		UploadURL:       resp.GetUploadUrl(),
		ExpiresAt:       exp,
		RequiredHeaders: resp.GetRequiredHeaders(),
	}, nil
}

func (c *grpcClient) UploadDirect(ctx context.Context, req UploadDirectInput) (*Attachment, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	resp, err := c.api.UploadDirect(ctx, &attachmentsv1.UploadDirectRequest{
		TaskId:         req.TaskID,
		UserId:         req.UserID,
		OriginalName:   req.OriginalName,
		ContentType:    req.ContentType,
		Data:           req.Data,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return protoToAttachment(resp), nil
}

func (c *grpcClient) CompleteUpload(ctx context.Context, taskID int64, attachmentID string) (*Attachment, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	resp, err := c.api.CompleteUpload(ctx, &attachmentsv1.CompleteUploadRequest{
		TaskId: taskID, AttachmentId: attachmentID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return protoToAttachment(resp), nil
}

func (c *grpcClient) ListAttachments(ctx context.Context, taskID int64) ([]Attachment, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	resp, err := c.api.ListAttachments(ctx, &attachmentsv1.ListAttachmentsRequest{TaskId: taskID})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	out := make([]Attachment, 0, len(resp.GetAttachments()))
	for _, a := range resp.GetAttachments() {
		if x := protoToAttachment(a); x != nil {
			out = append(out, *x)
		}
	}
	return out, nil
}

func (c *grpcClient) GetDownloadURL(ctx context.Context, taskID int64, attachmentID string) (*DownloadURL, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	resp, err := c.api.GetDownloadURL(ctx, &attachmentsv1.GetDownloadURLRequest{
		TaskId: taskID, AttachmentId: attachmentID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	var exp time.Time
	if resp.GetExpiresAt() != nil {
		exp = resp.GetExpiresAt().AsTime()
	}
	return &DownloadURL{DownloadURL: resp.GetDownloadUrl(), ExpiresAt: exp}, nil
}

func (c *grpcClient) DownloadDirect(ctx context.Context, taskID int64, attachmentID string) (*DownloadDirectResult, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	resp, err := c.api.DownloadDirect(ctx, &attachmentsv1.DownloadDirectRequest{
		TaskId: taskID, AttachmentId: attachmentID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &DownloadDirectResult{
		Data:         resp.GetData(),
		ContentType:  resp.GetContentType(),
		OriginalName: resp.GetOriginalName(),
		SizeBytes:    resp.GetSizeBytes(),
	}, nil
}

func (c *grpcClient) DeleteAttachment(ctx context.Context, taskID int64, attachmentID string) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	_, err := c.api.DeleteAttachment(ctx, &attachmentsv1.DeleteAttachmentRequest{
		TaskId: taskID, AttachmentId: attachmentID,
	})
	return mapGRPCError(err)
}

func (c *grpcClient) PurgeByTask(ctx context.Context, taskID int64, childTaskIDs []int64) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	_, err := c.api.PurgeByTask(ctx, &attachmentsv1.PurgeByTaskRequest{
		TaskId: taskID, ChildTaskIds: childTaskIDs,
	})
	return mapGRPCError(err)
}

func (c *grpcClient) PurgeByUser(ctx context.Context, userID int64) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	_, err := c.api.PurgeByUser(ctx, &attachmentsv1.PurgeByUserRequest{UserId: userID})
	return mapGRPCError(err)
}

func protoToAttachment(a *attachmentsv1.Attachment) *Attachment {
	if a == nil {
		return nil
	}
	var created time.Time
	if a.GetCreatedAt() != nil {
		created = a.GetCreatedAt().AsTime()
	}
	return &Attachment{
		ID:           a.GetId(),
		TaskID:       a.GetTaskId(),
		UserID:       a.GetUserId(),
		OriginalName: a.GetOriginalName(),
		ContentType:  a.GetContentType(),
		SizeBytes:    a.GetSizeBytes(),
		Status:       a.GetStatus(),
		CreatedAt:    created,
	}
}

func mapGRPCError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return errors.Wrap(ErrUnavailable, err.Error())
	}
	switch st.Code() {
	case codes.NotFound:
		return ErrNotFound
	case codes.InvalidArgument:
		return errors.Wrap(ErrValidation, st.Message())
	case codes.FailedPrecondition:
		return errors.Wrap(ErrConflict, st.Message())
	case codes.ResourceExhausted:
		return errors.Wrap(ErrPayloadTooLarge, st.Message())
	case codes.Unavailable, codes.DeadlineExceeded:
		return errors.Wrap(ErrUnavailable, st.Message())
	default:
		return errors.Wrap(ErrUnavailable, st.Message())
	}
}
