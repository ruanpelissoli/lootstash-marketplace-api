package dto

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Pagination contains pagination parameters
type Pagination struct {
	Page    int `json:"page" query:"page"`
	PerPage int `json:"perPage" query:"perPage"`
}

// GetOffset returns the SQL offset for pagination
func (p *Pagination) GetOffset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 {
		p.PerPage = 20
	}
	if p.PerPage > 100 {
		p.PerPage = 100
	}
	return (p.Page - 1) * p.PerPage
}

// GetLimit returns the SQL limit for pagination
func (p *Pagination) GetLimit() int {
	if p.PerPage < 1 {
		return 20
	}
	if p.PerPage > 100 {
		return 100
	}
	return p.PerPage
}

// PaginatedResponse wraps paginated results
type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
	TotalCount int `json:"totalCount"`
	TotalPages int `json:"totalPages"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse[T any](data []T, page, perPage, totalCount int) PaginatedResponse[T] {
	totalPages := totalCount / perPage
	if totalCount%perPage > 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	return PaginatedResponse[T]{
		Data:       data,
		Page:       page,
		PerPage:    perPage,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// IDResponse represents a response with just an ID
type IDResponse struct {
	ID string `json:"id"`
}
