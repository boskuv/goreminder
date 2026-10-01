package models

// TaskGroupUpdateRequest represents a request to update a task group.
// All fields are optional (pointers) to support partial updates.
type TaskGroupUpdateRequest struct {
	Name *string `json:"name,omitempty"`
}
