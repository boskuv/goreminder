package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/api/dto/mapper"
	"github.com/boskuv/goreminder/internal/api/validation"
	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/logger"
)

// CalendarHandler handles Google Calendar integration HTTP requests.
type CalendarHandler struct {
	logger  zerolog.Logger
	service CalendarService
}

// NewCalendarHandler creates a CalendarHandler.
func NewCalendarHandler(svc CalendarService, logger zerolog.Logger) *CalendarHandler {
	return &CalendarHandler{logger: logger, service: svc}
}

func (h *CalendarHandler) writeErr(c *gin.Context, err error) {
	if errors.Is(err, errs.ErrValidation) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, errs.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, errs.ErrUnprocessableEntity) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

// @Summary Start Google OAuth
// @Tags Calendar
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} dto.OAuthStartResponse
// @Router /api/v1/users/{user_id}/calendar/oauth/start [get]
func (h *CalendarHandler) StartOAuth(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	url, err := h.service.StartOAuth(ctx, userID)
	if err != nil {
		log.Error().Err(err).Int64("user.id", userID).Msg("failed to start google oauth")
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.OAuthStartResponse{URL: url})
}

// @Summary Google OAuth callback
// @Tags Calendar
// @Produce json
// @Param code query string true "Authorization code"
// @Param state query string true "OAuth state (user id)"
// @Success 200 {object} dto.GoogleAccountResponse
// @Router /api/v1/calendar/oauth/callback [get]
func (h *CalendarHandler) OAuthCallback(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and state are required"})
		return
	}

	account, err := h.service.HandleOAuthCallback(ctx, code, state)
	if err != nil {
		log.Error().Err(err).Msg("google oauth callback failed")
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, mapper.GoogleAccountToResponse(account))
}

// @Summary List Google calendars for connected account
// @Tags Calendar
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {array} dto.GoogleCalendarListItem
// @Router /api/v1/users/{user_id}/calendar/calendars [get]
func (h *CalendarHandler) ListCalendars(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	calendars, err := h.service.ListGoogleCalendars(ctx, userID)
	if err != nil {
		log.Error().Err(err).Int64("user.id", userID).Msg("failed to list google calendars")
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, mapper.GoogleCalendarsToResponse(calendars))
}

// @Summary Create calendar binding
// @Tags Calendar
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Param body body dto.CreateCalendarBindingRequest true "Binding"
// @Success 201 {object} dto.CalendarBindingResponse
// @Router /api/v1/users/{user_id}/calendar/bindings [post]
func (h *CalendarHandler) CreateBinding(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var req dto.CreateCalendarBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	direction := models.CalendarBindingDirection(req.Direction)
	if req.Direction == "" {
		direction = models.CalendarBindingDirectionImport
	}
	policy := models.CalendarDeletePolicy(req.DeletePolicy)
	if req.DeletePolicy == "" {
		policy = models.CalendarDeletePolicySoftDeleteImported
	}

	binding, err := h.service.CreateBinding(ctx, service.CreateBindingRequest{
		UserID:                 userID,
		GoogleCalendarID:       req.GoogleCalendarID,
		CalendarSummary:        req.CalendarSummary,
		Direction:              direction,
		GroupID:                req.GroupID,
		MessengerRelatedUserID: req.MessengerRelatedUserID,
		DeletePolicy:           policy,
	})
	if err != nil {
		log.Error().Err(err).Int64("user.id", userID).Msg("failed to create calendar binding")
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, mapper.CalendarBindingToResponse(binding))
}

// @Summary List calendar bindings
// @Tags Calendar
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {array} dto.CalendarBindingResponse
// @Router /api/v1/users/{user_id}/calendar/bindings [get]
func (h *CalendarHandler) ListBindings(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	bindings, err := h.service.ListBindings(ctx, userID)
	if err != nil {
		log.Error().Err(err).Int64("user.id", userID).Msg("failed to list calendar bindings")
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, mapper.CalendarBindingsToResponse(bindings))
}

// @Summary Delete calendar binding
// @Tags Calendar
// @Param user_id path int true "User ID"
// @Param binding_id path int true "Binding ID"
// @Success 204
// @Router /api/v1/users/{user_id}/calendar/bindings/{binding_id} [delete]
func (h *CalendarHandler) DeleteBinding(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	bindingID, err := strconv.ParseInt(c.Param("binding_id"), 10, 64)
	if err != nil || bindingID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid binding_id"})
		return
	}

	bindings, err := h.service.ListBindings(ctx, userID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	owned := false
	for _, b := range bindings {
		if b.ID == bindingID {
			owned = true
			break
		}
	}
	if !owned {
		c.JSON(http.StatusNotFound, gin.H{"error": "binding not found for user"})
		return
	}

	if err := h.service.DeleteBinding(ctx, bindingID); err != nil {
		log.Error().Err(err).Int64("calendar_binding.id", bindingID).Msg("failed to delete calendar binding")
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary Force sync a calendar binding
// @Tags Calendar
// @Param user_id path int true "User ID"
// @Param binding_id path int true "Binding ID"
// @Success 202 {object} map[string]string
// @Router /api/v1/users/{user_id}/calendar/bindings/{binding_id}/sync [post]
func (h *CalendarHandler) ForceSync(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	bindingID, err := strconv.ParseInt(c.Param("binding_id"), 10, 64)
	if err != nil || bindingID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid binding_id"})
		return
	}

	bindings, err := h.service.ListBindings(ctx, userID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	owned := false
	for _, b := range bindings {
		if b.ID == bindingID {
			owned = true
			break
		}
	}
	if !owned {
		c.JSON(http.StatusNotFound, gin.H{"error": "binding not found for user"})
		return
	}

	if err := h.service.SyncBinding(ctx, bindingID); err != nil {
		log.Error().Err(err).Int64("calendar_binding.id", bindingID).Msg("force sync failed")
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "synced"})
}

// @Summary Disconnect Google account
// @Tags Calendar
// @Param user_id path int true "User ID"
// @Success 204
// @Router /api/v1/users/{user_id}/calendar/disconnect [delete]
func (h *CalendarHandler) Disconnect(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if err := h.service.DisconnectGoogle(ctx, userID); err != nil {
		log.Error().Err(err).Int64("user.id", userID).Msg("failed to disconnect google")
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary Enable export for a single task
// @Tags Calendar
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param body body dto.EnableTaskExportRequest true "Export config"
// @Success 200 {object} dto.TaskExternalResponse
// @Router /api/v1/tasks/{id}/calendar/export [post]
func (h *CalendarHandler) EnableTaskExport(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req dto.EnableTaskExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	link, err := h.service.EnableTaskExport(ctx, taskID, req.CalendarBindingID)
	if err != nil {
		log.Error().Err(err).Int64("task.id", taskID).Msg("failed to enable task export")
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, mapper.TaskSyncLinkToExternalResponse(link))
}

// @Summary Get external sync metadata for a task
// @Tags Calendar
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} dto.TaskExternalResponse
// @Router /api/v1/tasks/{id}/calendar/external [get]
func (h *CalendarHandler) GetTaskExternal(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	link, err := h.service.GetTaskExternal(ctx, taskID)
	if err != nil {
		log.Error().Err(err).Int64("task.id", taskID).Msg("failed to get task external")
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, mapper.TaskSyncLinkToExternalResponse(link))
}
