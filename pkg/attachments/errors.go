package attachments

import "errors"

var (
	ErrDisabled     = errors.New("attachments disabled")
	ErrUnavailable  = errors.New("attachments service unavailable")
	ErrNotFound     = errors.New("attachment not found")
	ErrValidation   = errors.New("validation failed")
	ErrConflict       = errors.New("conflict")
	ErrPayloadTooLarge = errors.New("payload too large")
)
