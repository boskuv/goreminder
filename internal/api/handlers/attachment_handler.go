package handlers

import (
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/api/dto/mapper"
	"github.com/boskuv/goreminder/internal/api/validation"
	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/attachments"
	"github.com/boskuv/goreminder/pkg/logger"
)

type AttachmentHandler struct {
	log                   zerolog.Logger
	taskService           *service.TaskService
	attClient             attachments.Client
	attachmentsEnabled    bool
	directUploadMaxBytes  int64
	proxyDownloadEnabled  bool
	proxyDownloadMaxBytes int64
}

func NewAttachmentHandler(
	taskService *service.TaskService,
	attClient attachments.Client,
	attachmentsEnabled bool,
	directUploadMaxBytes int64,
	proxyDownloadEnabled bool,
	proxyDownloadMaxBytes int64,
	logger zerolog.Logger,
) *AttachmentHandler {
	return &AttachmentHandler{
		log:                   logger,
		taskService:           taskService,
		attClient:             attClient,
		attachmentsEnabled:    attachmentsEnabled,
		directUploadMaxBytes:  directUploadMaxBytes,
		proxyDownloadEnabled:  proxyDownloadEnabled,
		proxyDownloadMaxBytes: proxyDownloadMaxBytes,
	}
}

// @Summary Create or initialize task attachment
// @Description JSON body: presigned upload (pending + upload_url). multipart/form-data with field file: direct upload (max 2MB, status ready, no complete step).
// @Tags Task Attachments
// @Accept json
// @Accept mpfd
// @Produce json
// @Param id path int true "Task ID" example(42)
// @Param file formData file false "File for direct upload (multipart only)"
// @Param idempotency_key formData string false "Idempotency key (multipart only)" example(550e8400-e29b-41d4-a716-446655440000)
// @Param body body dto.InitAttachmentRequest false "Presigned init (JSON only)"
// @Success 201 {object} dto.InitAttachmentResponse "Presigned init (JSON)"
// @Success 201 {object} dto.AttachmentResponse "Direct upload (multipart)"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 413 {object} dto.ErrorResponse
// @Failure 503 {object} dto.AttachmentsDisabledResponse
// @Router /api/v1/tasks/{id}/attachments [post]
func (h *AttachmentHandler) CreateAttachment(c *gin.Context) {
	ct := c.GetHeader("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		h.uploadDirect(c)
		return
	}
	h.initPresigned(c)
}

func (h *AttachmentHandler) initPresigned(c *gin.Context) {
	h.withTask(c, func(ctx *gin.Context, taskID int64) {
		var req dto.InitAttachmentRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			validation.HandleValidationError(ctx, err)
			return
		}
		task, err := h.taskService.GetTask(ctx.Request.Context(), taskID)
		if err != nil {
			h.mapTaskError(ctx, err)
			return
		}
		res, err := h.attClient.InitUpload(ctx.Request.Context(), attachments.InitUploadInput{
			TaskID:         taskID,
			UserID:         task.UserID,
			OriginalName:   req.OriginalName,
			ContentType:    req.ContentType,
			SizeBytes:      req.SizeBytes,
			IdempotencyKey: req.IdempotencyKey,
		})
		if err != nil {
			h.mapAttachmentError(ctx, err)
			return
		}
		ctx.JSON(http.StatusCreated, dto.InitAttachmentResponse{
			ID:              res.AttachmentID,
			TaskID:          res.TaskID,
			Status:          res.Status,
			UploadURL:       res.UploadURL,
			ExpiresAt:       res.ExpiresAt,
			RequiredHeaders: res.RequiredHeaders,
		})
	})
}

func (h *AttachmentHandler) uploadDirect(c *gin.Context) {
	h.withTask(c, func(ctx *gin.Context, taskID int64) {
		fileHeader, err := ctx.FormFile("file")
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
			return
		}
		if fileHeader.Size > h.directUploadMaxBytes {
			ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "file_too_large",
				"message": "use JSON init + presigned upload for files larger than direct upload limit",
			})
			return
		}

		f, err := fileHeader.Open()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		defer f.Close()

		limited := io.LimitReader(f, h.directUploadMaxBytes+1)
		data, err := io.ReadAll(limited)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if int64(len(data)) > h.directUploadMaxBytes {
			ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "file_too_large",
				"message": "use JSON init + presigned upload for files larger than direct upload limit",
			})
			return
		}

		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		task, err := h.taskService.GetTask(ctx.Request.Context(), taskID)
		if err != nil {
			h.mapTaskError(ctx, err)
			return
		}

		att, err := h.attClient.UploadDirect(ctx.Request.Context(), attachments.UploadDirectInput{
			TaskID:         taskID,
			UserID:         task.UserID,
			OriginalName:   fileHeader.Filename,
			ContentType:    contentType,
			Data:           data,
			IdempotencyKey: ctx.PostForm("idempotency_key"),
		})
		if err != nil {
			h.mapAttachmentError(ctx, err)
			return
		}
		ctx.JSON(http.StatusCreated, attachmentResponse(att))
	})
}

// @Summary Complete attachment upload
// @Tags Task Attachments
// @Param id path int true "Task ID" example(42)
// @Param attachment_id path string true "Attachment ID" example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {object} dto.AttachmentResponse
// @Router /api/v1/tasks/{id}/attachments/{attachment_id}/complete [post]
func (h *AttachmentHandler) CompleteUpload(c *gin.Context) {
	h.withTask(c, func(ctx *gin.Context, taskID int64) {
		attID := ctx.Param("attachment_id")
		att, err := h.attClient.CompleteUpload(ctx.Request.Context(), taskID, attID)
		if err != nil {
			h.mapAttachmentError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, attachmentResponse(att))
	})
}

// @Summary List task attachments
// @Tags Task Attachments
// @Param id path int true "Task ID" example(42)
// @Success 200 {object} dto.ListAttachmentsResponse
// @Router /api/v1/tasks/{id}/attachments [get]
func (h *AttachmentHandler) ListAttachments(c *gin.Context) {
	h.withTask(c, func(ctx *gin.Context, taskID int64) {
		if _, err := h.taskService.GetTask(ctx.Request.Context(), taskID); err != nil {
			h.mapTaskError(ctx, err)
			return
		}
		list, err := h.attClient.ListAttachments(ctx.Request.Context(), taskID)
		if err != nil {
			h.mapAttachmentError(ctx, err)
			return
		}
		out := mapper.AttachmentsToResponse(list)
		if out == nil {
			out = []dto.AttachmentResponse{}
		}
		ctx.JSON(http.StatusOK, dto.ListAttachmentsResponse{Attachments: out})
	})
}

// @Summary Get attachment download URL
// @Tags Task Attachments
// @Param id path int true "Task ID" example(42)
// @Param attachment_id path string true "Attachment ID" example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {object} dto.DownloadAttachmentResponse
// @Router /api/v1/tasks/{id}/attachments/{attachment_id}/download [get]
func (h *AttachmentHandler) GetDownloadURL(c *gin.Context) {
	h.withTask(c, func(ctx *gin.Context, taskID int64) {
		attID := ctx.Param("attachment_id")
		url, err := h.attClient.GetDownloadURL(ctx.Request.Context(), taskID, attID)
		if err != nil {
			h.mapAttachmentError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, dto.DownloadAttachmentResponse{
			DownloadURL: url.DownloadURL,
			ExpiresAt:   url.ExpiresAt,
		})
	})
}

// @Summary Download attachment file via API (proxy)
// @Description Returns file bytes for ready attachments within proxyDownloadMaxBytes when proxyDownloadEnabled is true. Use GET .../download for presigned URL (large files).
// @Tags Task Attachments
// @Produce application/octet-stream
// @Param id path int true "Task ID" example(42)
// @Param attachment_id path string true "Attachment ID" example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {file} binary "File content"
// @Failure 404 {object} dto.ErrorResponse
// @Failure 413 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ProxyDownloadDisabledResponse
// @Failure 503 {object} dto.AttachmentsDisabledResponse
// @Router /api/v1/tasks/{id}/attachments/{attachment_id}/content [get]
func (h *AttachmentHandler) GetAttachmentContent(c *gin.Context) {
	if !h.attachmentsEnabled {
		writeAttachmentsDisabled(c)
		return
	}
	if !h.proxyDownloadEnabled {
		writeProxyDownloadDisabled(c)
		return
	}
	h.withTask(c, func(ctx *gin.Context, taskID int64) {
		if _, err := h.taskService.GetTask(ctx.Request.Context(), taskID); err != nil {
			h.mapTaskError(ctx, err)
			return
		}
		attID := ctx.Param("attachment_id")
		res, err := h.attClient.DownloadDirect(ctx.Request.Context(), taskID, attID)
		if err != nil {
			h.mapAttachmentError(ctx, err)
			return
		}
		if res.SizeBytes > h.proxyDownloadMaxBytes {
			ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "file_too_large",
				"message": "use GET .../download for presigned download",
			})
			return
		}
		filename := res.OriginalName
		if filename == "" {
			filename = "attachment"
		}
		contentType := res.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
		ctx.Header("Content-Disposition", disposition)
		ctx.Header("Content-Type", contentType)
		ctx.Header("Content-Length", strconv.FormatInt(res.SizeBytes, 10))
		ctx.Data(http.StatusOK, contentType, res.Data)
	})
}

// @Summary Delete attachment
// @Tags Task Attachments
// @Param id path int true "Task ID" example(42)
// @Param attachment_id path string true "Attachment ID" example(550e8400-e29b-41d4-a716-446655440000)
// @Success 204
// @Router /api/v1/tasks/{id}/attachments/{attachment_id} [delete]
func (h *AttachmentHandler) DeleteAttachment(c *gin.Context) {
	h.withTask(c, func(ctx *gin.Context, taskID int64) {
		attID := ctx.Param("attachment_id")
		if err := h.attClient.DeleteAttachment(ctx.Request.Context(), taskID, attID); err != nil {
			h.mapAttachmentError(ctx, err)
			return
		}
		ctx.Status(http.StatusNoContent)
	})
}

func (h *AttachmentHandler) withTask(c *gin.Context, fn func(*gin.Context, int64)) {
	if !h.attachmentsEnabled {
		writeAttachmentsDisabled(c)
		return
	}
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}
	fn(c, taskID)
}

func writeAttachmentsDisabled(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, dto.AttachmentsDisabledResponse{
		Error:   "attachments_disabled",
		Message: "Attachment storage is disabled in this deployment",
	})
}

func writeProxyDownloadDisabled(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, dto.ProxyDownloadDisabledResponse{
		Error:   "proxy_download_disabled",
		Message: "Proxy download is disabled; use GET .../attachments/{attachment_id}/download for a presigned URL",
	})
}

func writeAttachmentsUnavailable(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error":   "attachments_unavailable",
		"message": "Attachment service is temporarily unavailable",
	})
}

func (h *AttachmentHandler) mapTaskError(c *gin.Context, err error) {
	if errors.Is(err, errs.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func (h *AttachmentHandler) mapAttachmentError(c *gin.Context, err error) {
	if errors.Is(err, attachments.ErrDisabled) {
		writeAttachmentsDisabled(c)
		return
	}
	if errors.Is(err, attachments.ErrUnavailable) {
		writeAttachmentsUnavailable(c)
		return
	}
	if errors.Is(err, attachments.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, attachments.ErrPayloadTooLarge) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error":   "file_too_large",
			"message": "use GET .../download for presigned download",
		})
		return
	}
	if errors.Is(err, attachments.ErrValidation) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, attachments.ErrConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	log := logger.WithTraceContext(c.Request.Context(), h.log)
	log.Error().Err(err).Msg("attachment operation failed")
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func attachmentResponse(a *attachments.Attachment) dto.AttachmentResponse {
	list := mapper.AttachmentsToResponse([]attachments.Attachment{*a})
	return list[0]
}
