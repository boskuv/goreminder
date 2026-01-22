package dto

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string `json:"message" example:"error message"`
}

// CreateResponse represents a successful creation response
type CreateResponse struct {
	ID int64 `json:"id" example:"1"`
}

// IDResponse represents a response with an ID
type IDResponse struct {
	ID int64 `json:"id" example:"1"`
}

// UserIDResponse represents a response with a user ID
type UserIDResponse struct {
	UserID int64 `json:"user_id" example:"1"`
}

// BatchCreateResponse represents a successful batch creation response
type BatchCreateResponse struct {
	IDs   []int64 `json:"ids"`
	Count int     `json:"count" example:"3"`
}

// TaskCreateResponse represents a successful task creation response
type TaskCreateResponse struct {
	ID      int64  `json:"id" example:"1"`
	ChildID *int64 `json:"child_id,omitempty" example:"2"`
}
