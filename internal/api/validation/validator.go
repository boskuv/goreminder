package validation

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidateInt64Param validates a path parameter as int64
func ValidateInt64Param(c *gin.Context, paramName string) (int64, error) {
	paramValue := c.Param(paramName)
	if paramValue == "" {
		return 0, fmt.Errorf("parameter '%s' is required", paramName)
	}

	if !govalidator.IsInt(paramValue) {
		return 0, fmt.Errorf("parameter '%s' must be a valid integer", paramName)
	}

	value, err := strconv.ParseInt(paramValue, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parameter '%s' must be a valid int64: %w", paramName, err)
	}

	if value <= 0 {
		return 0, fmt.Errorf("parameter '%s' must be greater than 0", paramName)
	}

	return value, nil
}

// ValidateStringParam validates a path parameter as string
func ValidateStringParam(c *gin.Context, paramName string, required bool) (string, error) {
	paramValue := c.Param(paramName)
	if required && paramValue == "" {
		return "", fmt.Errorf("parameter '%s' is required", paramName)
	}

	if paramValue != "" && !govalidator.IsPrintableASCII(paramValue) {
		return "", fmt.Errorf("parameter '%s' contains invalid characters", paramName)
	}

	return paramValue, nil
}

// ValidateInt64Query validates a query parameter as int64
func ValidateInt64Query(c *gin.Context, paramName string, defaultValue int64, minValue int64) (int64, error) {
	paramValue := c.Query(paramName)
	if paramValue == "" {
		return defaultValue, nil
	}

	if !govalidator.IsInt(paramValue) {
		return 0, fmt.Errorf("query parameter '%s' must be a valid integer", paramName)
	}

	value, err := strconv.ParseInt(paramValue, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("query parameter '%s' must be a valid int64: %w", paramName, err)
	}

	if value < minValue {
		return 0, fmt.Errorf("query parameter '%s' must be greater than or equal to %d", paramName, minValue)
	}

	return value, nil
}

// ValidateOptionalInt64Query validates an optional query parameter as int64
func ValidateOptionalInt64Query(c *gin.Context, paramName string, minValue int64) (*int64, error) {
	paramValue := c.Query(paramName)
	if paramValue == "" {
		return nil, nil
	}

	if !govalidator.IsInt(paramValue) {
		return nil, fmt.Errorf("query parameter '%s' must be a valid integer", paramName)
	}

	value, err := strconv.ParseInt(paramValue, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("query parameter '%s' must be a valid int64: %w", paramName, err)
	}

	if value < minValue {
		return nil, fmt.Errorf("query parameter '%s' must be greater than or equal to %d", paramName, minValue)
	}

	return &value, nil
}

// ValidateStringQuery validates a query parameter as string
func ValidateStringQuery(c *gin.Context, paramName string, required bool) (string, error) {
	paramValue := c.Query(paramName)
	if required && paramValue == "" {
		return "", fmt.Errorf("query parameter '%s' is required", paramName)
	}

	if paramValue != "" && !govalidator.IsPrintableASCII(paramValue) {
		return "", fmt.Errorf("query parameter '%s' contains invalid characters", paramName)
	}

	return paramValue, nil
}

// ValidateEmail validates an email address
func ValidateEmail(email string) error {
	if !govalidator.IsEmail(email) {
		return fmt.Errorf("invalid email format: %s", email)
	}
	return nil
}

// ValidateURL validates a URL
func ValidateURL(url string) error {
	if !govalidator.IsURL(url) {
		return fmt.Errorf("invalid URL format: %s", url)
	}
	return nil
}

// ValidateUUID validates a UUID
func ValidateUUID(uuid string) error {
	if !govalidator.IsUUID(uuid) {
		return fmt.Errorf("invalid UUID format: %s", uuid)
	}
	return nil
}

// GetValidationErrors returns a formatted error message from validation errors
func GetValidationErrors(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var errors []string
		for _, e := range validationErrors {
			field := e.Field()
			tag := e.Tag()
			param := e.Param()

			var msg string
			switch tag {
			case "required":
				msg = fmt.Sprintf("field '%s' is required", field)
			case "cron":
				msg = fmt.Sprintf("field '%s' must be a valid cron expression", field)
			case "task_status":
				msg = fmt.Sprintf("field '%s' must be a valid task status (pending, scheduled, done, rescheduled, postponed, deleted)", field)
			case "email":
				msg = fmt.Sprintf("field '%s' must be a valid email address", field)
			case "min":
				msg = fmt.Sprintf("field '%s' must be at least %s", field, param)
			case "max":
				msg = fmt.Sprintf("field '%s' must be at most %s", field, param)
			default:
				msg = fmt.Sprintf("field '%s' failed validation for tag '%s'", field, tag)
			}
			errors = append(errors, msg)
		}
		return strings.Join(errors, "; ")
	}
	return err.Error()
}

// HandleValidationError returns a JSON error response for validation errors
func HandleValidationError(c *gin.Context, err error) {
	// Try to get formatted validation errors
	if validationErr := GetValidationErrors(err); validationErr != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": validationErr,
		})
		return
	}
	// Fallback to original error message
	c.JSON(http.StatusBadRequest, gin.H{
		"error": err.Error(),
	})
}

// ValidateOptionalStringQuery validates an optional query parameter as string
func ValidateOptionalStringQuery(c *gin.Context, paramName string) (string, error) {
	paramValue := c.Query(paramName)
	if paramValue != "" && !govalidator.IsPrintableASCII(paramValue) {
		return "", fmt.Errorf("query parameter '%s' contains invalid characters", paramName)
	}
	return paramValue, nil
}

// ValidateOptionalTimeQuery validates an optional query parameter as time.Time
func ValidateOptionalTimeQuery(c *gin.Context, paramName string) (*time.Time, error) {
	paramValue := c.Query(paramName)
	if paramValue == "" {
		return nil, nil
	}

	parsedTime, err := time.Parse(time.RFC3339, paramValue)
	if err != nil {
		return nil, fmt.Errorf("query parameter '%s' must be a valid RFC3339 time format: %w", paramName, err)
	}

	return &parsedTime, nil
}
