package errors

import "errors"

var (
	ErrNotFound            = errors.New("no data found matching criteria") // 404
	ErrValidation          = errors.New("invalid input data")              // 400
	ErrUnprocessableEntity = errors.New("unprocessable entity")            // 422
	ErrConflict            = errors.New("conflict")                        // 409
)

// IsClientError reports whether err is an expected API client error (4xx-class).
func IsClientError(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrValidation) ||
		errors.Is(err, ErrUnprocessableEntity) ||
		errors.Is(err, ErrConflict)
}
