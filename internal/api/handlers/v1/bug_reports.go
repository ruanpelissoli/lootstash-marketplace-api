package v1

import (
	"database/sql"
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/middleware"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// BugReportHandler handles bug report-related requests
type BugReportHandler struct {
	service   *service.BugReportService
	validator *validator.Validate
}

// NewBugReportHandler creates a new bug report handler
func NewBugReportHandler(service *service.BugReportService) *BugReportHandler {
	return &BugReportHandler{
		service:   service,
		validator: validator.New(),
	}
}

// Create handles POST /api/v1/bug-reports
func (h *BugReportHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.CreateBugReportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	if err := h.validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
			Code:    400,
		})
	}

	report, err := h.service.Create(c.Context(), userID, &req)
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to create bug report",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create bug report",
			Code:    500,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(h.service.ToResponse(report))
}

// List handles GET /api/v1/bug-reports (admin only)
func (h *BugReportHandler) List(c *fiber.Ctx) error {
	var filter dto.BugReportFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	reports, count, err := h.service.List(c.Context(), filter.Status, filter.GetOffset(), filter.GetLimit())
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to list bug reports",
			"error", err.Error(),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list bug reports",
			Code:    500,
		})
	}

	responses := make([]dto.BugReportAdminResponse, 0, len(reports))
	for _, report := range reports {
		responses = append(responses, *h.service.ToAdminResponse(report))
	}

	return c.JSON(dto.NewPaginatedResponse(responses, filter.Page, filter.GetLimit(), count))
}

// UpdateStatus handles PATCH /api/v1/bug-reports/:id (admin only)
func (h *BugReportHandler) UpdateStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	var req dto.UpdateBugReportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	if err := h.validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
			Code:    400,
		})
	}

	report, err := h.service.UpdateStatus(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Bug report not found",
				Code:    404,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to update bug report",
			"error", err.Error(),
			"bug_report_id", id,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update bug report",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToAdminResponse(report))
}
