package dto

import "time"

// CreateBugReportRequest represents a request to submit a bug report
type CreateBugReportRequest struct {
	Title       string `json:"title" validate:"required,min=5,max=200"`
	Description string `json:"description" validate:"required,min=10,max=5000"`
}

// UpdateBugReportRequest represents a request to update a bug report's status
type UpdateBugReportRequest struct {
	Status string `json:"status" validate:"required,oneof=open resolved closed"`
}

// BugReportResponse represents the response after submitting a bug report
type BugReportResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

// BugReportAdminResponse extends BugReportResponse with reporter info for admin views
type BugReportAdminResponse struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Status           string    `json:"status"`
	ReporterID       string    `json:"reporterId"`
	ReporterUsername string    `json:"reporterUsername"`
	ReporterAvatar   string    `json:"reporterAvatar,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// BugReportFilterRequest represents filter parameters for listing bug reports
type BugReportFilterRequest struct {
	Pagination
	Status string `query:"status"`
}
