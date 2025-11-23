package validation

import (
	"fmt"
	"reflect"
	"time"

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

	// Register future date validator
	if err := v.RegisterValidation("future_date", validateFutureDate); err != nil {
		return fmt.Errorf("failed to register future_date validator: %w", err)
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

// validateFutureDate validates that a date is in the future (not in the past) in UTC
// It accepts both time.Time and *time.Time (pointer) types
// For zero values (not provided), validation is skipped (returns true)
// This allows the "required" validator to handle required fields
func validateFutureDate(fl validator.FieldLevel) bool {
	var date time.Time

	field := fl.Field()
	kind := field.Kind()

	switch kind {
	case reflect.Struct:
		// Check if it's time.Time
		if timeType, ok := field.Interface().(time.Time); ok {
			date = timeType
			// For non-pointer fields, zero value means field was not provided in JSON
			// Skip validation for zero values - let "required" validator handle required fields
			if date.IsZero() {
				return true
			}
		} else {
			return false
		}
	case reflect.Ptr:
		if field.IsNil() {
			// nil pointer is valid for optional fields
			return true
		}
		elem := field.Elem()
		if elem.Kind() == reflect.Struct {
			if timeType, ok := elem.Interface().(time.Time); ok {
				date = timeType
				// For pointer types, if not nil but zero, it's still considered provided
				// So we validate it (zero date will fail validation)
			} else {
				return false
			}
		} else {
			return false
		}
	default:
		return false
	}

	// Check if date is zero (uninitialized time.Time)
	// For pointer types that are not nil, zero means invalid
	if date.IsZero() {
		return false
	}

	// Check if date is in the future (after current UTC time)
	// Allow current time or future time (>= now)
	now := time.Now().UTC()
	return date.After(now) || date.Equal(now)
}
