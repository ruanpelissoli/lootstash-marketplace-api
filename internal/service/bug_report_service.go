package service

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

// BugReportService handles bug report business logic
type BugReportService struct {
	repo repository.BugReportRepository
}

// NewBugReportService creates a new bug report service
func NewBugReportService(repo repository.BugReportRepository) *BugReportService {
	return &BugReportService{repo: repo}
}

// Create creates a new bug report
func (s *BugReportService) Create(ctx context.Context, userID string, req *dto.CreateBugReportRequest) (*models.BugReport, error) {
	report := &models.BugReport{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Status:      "open",
	}

	if err := s.repo.Create(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}

// List lists bug reports with optional status filter
func (s *BugReportService) List(ctx context.Context, status string, offset, limit int) ([]*models.BugReport, int, error) {
	return s.repo.List(ctx, status, offset, limit)
}

// UpdateStatus updates a bug report's status
func (s *BugReportService) UpdateStatus(ctx context.Context, id string, req *dto.UpdateBugReportRequest) (*models.BugReport, error) {
	report, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	report.Status = req.Status
	if err := s.repo.Update(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}

// ToResponse converts a bug report model to a basic response DTO
func (s *BugReportService) ToResponse(report *models.BugReport) *dto.BugReportResponse {
	return &dto.BugReportResponse{
		ID:          report.ID,
		Title:       report.Title,
		Description: report.Description,
		Status:      report.Status,
		CreatedAt:   report.CreatedAt,
	}
}

// ToAdminResponse converts a bug report model to an admin response DTO with reporter info
func (s *BugReportService) ToAdminResponse(report *models.BugReport) *dto.BugReportAdminResponse {
	resp := &dto.BugReportAdminResponse{
		ID:          report.ID,
		Title:       report.Title,
		Description: report.Description,
		Status:      report.Status,
		ReporterID:  report.UserID,
		CreatedAt:   report.CreatedAt,
		UpdatedAt:   report.UpdatedAt,
	}

	if report.Reporter != nil {
		resp.ReporterUsername = report.Reporter.Username
		resp.ReporterAvatar = report.Reporter.GetAvatarURL()
	}

	return resp
}
