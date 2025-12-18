package dto

// PaginationResponse represents a paginated response
type PaginationResponse struct {
	Page       int `json:"page" example:"1"`
	PageSize   int `json:"page_size" example:"50"`
	TotalPages int `json:"total_pages" example:"10"`
	TotalCount int `json:"total_count" example:"500"`
}

// PaginatedMessengersResponse represents a paginated response for messengers
type PaginatedMessengersResponse struct {
	Data       []MessengerResponse `json:"data"`
	Pagination PaginationResponse  `json:"pagination"`
}

// PaginatedMessengerRelatedUsersResponse represents a paginated response for messenger-related users
type PaginatedMessengerRelatedUsersResponse struct {
	Data       []MessengerRelatedUserResponse `json:"data"`
	Pagination PaginationResponse             `json:"pagination"`
}

// PaginatedTasksResponse represents a paginated response for tasks
type PaginatedTasksResponse struct {
	Data       []TaskResponse     `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginatedUsersResponse represents a paginated response for users
type PaginatedUsersResponse struct {
	Data       []UserResponse     `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginatedBacklogsResponse represents a paginated response for backlogs
type PaginatedBacklogsResponse struct {
	Data       []BacklogResponse  `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginatedTargetsResponse represents a paginated response for targets
type PaginatedTargetsResponse struct {
	Data       []TargetResponse   `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}
