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

// ServiceRunHandler handles service run-related requests
type ServiceRunHandler struct {
	service   *service.ServiceRunService
	validator *validator.Validate
}

// NewServiceRunHandler creates a new service run handler
func NewServiceRunHandler(service *service.ServiceRunService) *ServiceRunHandler {
	return &ServiceRunHandler{
		service:   service,
		validator: validator.New(),
	}
}

// List handles GET /api/v1/service-runs
func (h *ServiceRunHandler) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var filter dto.ServiceRunsFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	runs, count, err := h.service.List(c.Context(), userID, filter.Role, filter.Status, filter.GetOffset(), filter.GetLimit())
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to list service runs",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list service runs",
			Code:    500,
		})
	}

	items := make([]dto.ServiceRunResponse, 0, len(runs))
	for _, run := range runs {
		items = append(items, *h.service.ToResponseWithUser(c.Context(), run, userID))
	}

	return c.JSON(dto.NewPaginatedResponse(items, filter.Page, filter.GetLimit(), count))
}

// GetByID handles GET /api/v1/service-runs/:id
func (h *ServiceRunHandler) GetByID(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	run, err := h.service.GetByID(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Service run not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have access to this service run",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get service run",
			"error", err.Error(),
			"service_run_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get service run",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToDetailResponse(c.Context(), run, userID))
}

// Complete handles POST /api/v1/service-runs/:id/complete
func (h *ServiceRunHandler) Complete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	run, _, err := h.service.Complete(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Service run not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "Only participants can complete a service run",
				Code:    403,
			})
		}
		if errors.Is(err, service.ErrInvalidState) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "Service run is not active",
				Code:    400,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to complete service run",
			"error", err.Error(),
			"service_run_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to complete service run",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToDetailResponse(c.Context(), run, userID))
}

// Cancel handles POST /api/v1/service-runs/:id/cancel
func (h *ServiceRunHandler) Cancel(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	var req dto.CancelServiceRunRequest
	_ = c.BodyParser(&req)

	run, err := h.service.Cancel(c.Context(), id, userID, req.Reason)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Service run not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "Only participants can cancel a service run",
				Code:    403,
			})
		}
		if errors.Is(err, service.ErrInvalidState) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "Service run is not active",
				Code:    400,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to cancel service run",
			"error", err.Error(),
			"service_run_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to cancel service run",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToDetailResponse(c.Context(), run, userID))
}
