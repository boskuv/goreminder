package validation

import (
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/robfig/cron/v3"

	"github.com/boskuv/goreminder/internal/models"
)

// RegisterCustomValidators registers custom validators for go-playground/validator
func RegisterCustomValidators(v *validator.Validate) error {
	// Register cron expression validator
	if err := v.RegisterValidation("cron", validateCronExpression); err != nil {
		return fmt.Errorf("failed to register cron validator: %w", err)
	}

	// Register task status validator
	if err := v.RegisterValidation("task_status", validateTaskStatus); err != nil {
		return fmt.Errorf("failed to register task_status validator: %w", err)
	}

	return nil
}

// validateCronExpression validates a cron expression
// It accepts both string and *string (pointer) types
func validateCronExpression(fl validator.FieldLevel) bool {
	var cronExpr string

	field := fl.Field()
	kind := field.Kind()

	switch kind {
	case reflect.String:
		cronExpr = field.String()
	case reflect.Ptr:
		if field.IsNil() {
			// nil pointer is valid for optional fields
			return true
		}
		elem := field.Elem()
		if elem.Kind() == reflect.String {
			cronExpr = elem.String()
		} else {
			return false
		}
	default:
		return false
	}

	// Empty string is valid for optional fields
	if cronExpr == "" {
		return true
	}

	// Validate cron expression using robfig/cron
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(cronExpr)
	return err == nil
}

// validateTaskStatus validates a task status against valid statuses
// It accepts both string and *string (pointer) types
func validateTaskStatus(fl validator.FieldLevel) bool {
	var status string

	field := fl.Field()
	kind := field.Kind()

	switch kind {
	case reflect.String:
		status = field.String()
	case reflect.Ptr:
		if field.IsNil() {
			// nil pointer is valid for optional fields
			return true
		}
		elem := field.Elem()
		if elem.Kind() == reflect.String {
			status = elem.String()
		} else {
			return false
		}
	default:
		return false
	}

	// Empty string is valid for optional fields
	if status == "" {
		return true
	}

	// Validate status using models.ValidateTaskStatus
	return models.ValidateTaskStatus(status) == nil
}
