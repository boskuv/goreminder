package service

import (
	"context"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/logger"
)

// calendarTaskHistoryMap builds a compact task snapshot for task_history JSON columns.
func calendarTaskHistoryMap(task *models.Task) map[string]interface{} {
	if task == nil {
		return nil
	}
	result := map[string]interface{}{
		"id":                    task.ID,
		"title":                 task.Title,
		"description":           task.Description,
		"status":                task.Status,
		"requires_confirmation": task.RequiresConfirmation,
		"muted":                 task.Muted,
		models.TaskHistorySourceKey: models.TaskHistorySourceGoogleCalendar,
	}
	if !task.StartDate.IsZero() {
		result["start_date"] = task.StartDate.UTC()
	}
	if task.FinishDate != nil {
		result["finish_date"] = task.FinishDate.UTC()
	}
	if task.CronExpression != nil {
		result["cron_expression"] = *task.CronExpression
	}
	if task.RRule != nil {
		result["rrule"] = *task.RRule
	}
	if task.MessengerRelatedUserID != nil {
		result["messenger_related_user_id"] = *task.MessengerRelatedUserID
	}
	if task.ParentID != nil {
		result["parent_id"] = *task.ParentID
	}
	if task.GroupID != nil {
		result["group_id"] = *task.GroupID
	}
	return result
}

func calendarImportedFieldsChanged(before, after *models.Task) bool {
	if before == nil || after == nil {
		return before != after
	}
	if before.Title != after.Title || before.Description != after.Description {
		return true
	}
	if !before.StartDate.Equal(after.StartDate) {
		return true
	}
	if !recurrencePtrsEqual(before.RRule, after.RRule) {
		return true
	}
	if before.Status != after.Status {
		return true
	}
	if (before.FinishDate == nil) != (after.FinishDate == nil) {
		return true
	}
	if before.FinishDate != nil && after.FinishDate != nil && !before.FinishDate.Equal(*after.FinishDate) {
		return true
	}
	if (before.GroupID == nil) != (after.GroupID == nil) {
		return true
	}
	if before.GroupID != nil && after.GroupID != nil && *before.GroupID != *after.GroupID {
		return true
	}
	if (before.MessengerRelatedUserID == nil) != (after.MessengerRelatedUserID == nil) {
		return true
	}
	if before.MessengerRelatedUserID != nil && after.MessengerRelatedUserID != nil &&
		*before.MessengerRelatedUserID != *after.MessengerRelatedUserID {
		return true
	}
	if before.Muted != after.Muted {
		return true
	}
	return false
}

// recordImportedTaskHistory writes task_history for calendar import mutations.
// Best-effort: sync must not fail if history insert fails.
func (s *CalendarSyncService) recordImportedTaskHistory(ctx context.Context, action models.TaskHistoryAction, task *models.Task, oldValue, newValue map[string]interface{}) {
	if s.taskHistory == nil || task == nil {
		return
	}
	if newValue != nil {
		newValue[models.TaskHistorySourceKey] = models.TaskHistorySourceGoogleCalendar
	}
	if oldValue != nil {
		oldValue[models.TaskHistorySourceKey] = models.TaskHistorySourceGoogleCalendar
	}
	history := &models.TaskHistory{
		TaskID:   task.ID,
		UserID:   task.UserID,
		Action:   string(action),
		OldValue: oldValue,
		NewValue: newValue,
	}
	if err := s.taskHistory.CreateTaskHistory(ctx, history); err != nil {
		log := logger.WithTraceContext(ctx, s.logger)
		log.Warn().Err(err).Int64("task.id", task.ID).Str("action", string(action)).
			Msg("failed to record task history for calendar import")
	}
}
