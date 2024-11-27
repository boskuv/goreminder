// internal/models/error.go
package models

// APIError represents an error returned to the API client
type APIError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// NewAPIError creates a new instance of APIError
func NewAPIError(message string, code int) *APIError {
	return &APIError{
		Message: message,
		Code:    code,
	}
}

// Error satisfies the error interface
func (e *APIError) Error() string {
	return e.Message
}

// HTTPError converts a generic error to an APIError
func HTTPError(err error, code int) *APIError {
	return &APIError{
		Message: err.Error(),
		Code:    code,
	}
}
