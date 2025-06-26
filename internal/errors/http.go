package errors

import "errors"

var (
	ErrNotFound            = errors.New("no data found matching criteria") // 404
	ErrValidation          = errors.New("invalid input data")              // 400
	ErrUnprocessableEntity = errors.New("unprocessable entity")            // 422
// Forbidden 403
// Unauthorized 401
// Method Not Allowed 405
// Conflict 409
// Unprocessable Entity 422
)
