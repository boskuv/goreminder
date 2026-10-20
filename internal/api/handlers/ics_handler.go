package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/boskuv/goreminder/internal/api/dto"
	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/logger"
)

// ICSHandler handles one-shot iCalendar (.ics) import.
type ICSHandler struct {
	logger  zerolog.Logger
	service ICSImporter
}

// NewICSHandler creates an ICSHandler.
func NewICSHandler(svc ICSImporter, log zerolog.Logger) *ICSHandler {
	return &ICSHandler{logger: log, service: svc}
}

// @Summary Import tasks from an ICS file
// @Description One-shot VEVENT import (Google Calendar field parity). Product flags (requires_confirmation, muted, pre_remind, etc.) stay local defaults. VTODO is skipped. Partial per-event errors do not fail the whole file.
// @Tags ICS
// @Accept mpfd
// @Produce json
// @Param user_id path int true "User ID"
// @Param file formData file true "iCalendar (.ics) file"
// @Param group_id formData int false "Optional task group id applied to all created tasks"
// @Param messenger_related_user_id formData int false "Optional messenger related user id (schedules reminders when set)"
// @Success 200 {object} dto.ICSImportResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 422 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users/{user_id}/ics/import [post]
func (h *ICSHandler) Import(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	opts := service.ICSImportOptions{UserID: userID}

	if raw := c.PostForm("group_id"); raw != "" {
		gid, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || gid <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
			return
		}
		opts.GroupID = &gid
	}

	if raw := c.PostForm("messenger_related_user_id"); raw != "" {
		mru, parseErr := strconv.Atoi(raw)
		if parseErr != nil || mru <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid messenger_related_user_id"})
			return
		}
		opts.MessengerRelatedUserID = &mru
	}

	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to open uploaded file"})
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Warn().Err(cerr).Msg("failed to close ics upload")
		}
	}()

	result, err := h.service.Import(ctx, f, opts)
	if err != nil {
		errEvent(log, err).Int64("user.id", userID).Msg("ics import failed")
		if errors.Is(err, errs.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := dto.ICSImportResponse{
		CreatedTaskIDs: result.CreatedTaskIDs,
		SkippedTODO:    result.SkippedTODO,
		Count:          len(result.CreatedTaskIDs),
	}
	for _, e := range result.Errors {
		resp.Errors = append(resp.Errors, dto.ICSImportItemError{
			Index:   e.Index,
			UID:     e.UID,
			Message: e.Message,
		})
	}

	log.Info().
		Int64("user.id", userID).
		Int("created", resp.Count).
		Int("errors", len(resp.Errors)).
		Msg("ics import completed")
	c.JSON(http.StatusOK, resp)
}
