package service

import (
	"fmt"

	"github.com/boskuv/goreminder/internal/models"
)

// maxPreRemindBeforeSeconds caps preliminary reminder offset (30 days).
const maxPreRemindBeforeSeconds int64 = 30 * 24 * 60 * 60

func validatePreRemindBeforeSeconds(v *int64) error {
	if v == nil {
		return nil
	}
	if *v < 0 {
		return fmt.Errorf("pre_remind_before_seconds must be >= 0")
	}
	if *v > maxPreRemindBeforeSeconds {
		return fmt.Errorf("pre_remind_before_seconds must be <= %d (30 days)", maxPreRemindBeforeSeconds)
	}
	return nil
}

// applyPreRemindBeforeSecondsUpdate applies a partial update:
// omit (nil) = no change; 0 = clear; >0 = set (after validation).
func applyPreRemindBeforeSecondsUpdate(task *models.Task, incoming *int64) error {
	if incoming == nil {
		return nil
	}
	if err := validatePreRemindBeforeSeconds(incoming); err != nil {
		return err
	}
	if *incoming == 0 {
		task.PreRemindBeforeSeconds = nil
		return nil
	}
	sec := *incoming
	task.PreRemindBeforeSeconds = &sec
	return nil
}

func preRemindBeforeSecondsEqual(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// clonePreRemindBeforeSeconds copies a non-nil positive offset for child tasks.
func clonePreRemindBeforeSeconds(v *int64) *int64 {
	if v == nil || *v <= 0 {
		return nil
	}
	sec := *v
	return &sec
}
