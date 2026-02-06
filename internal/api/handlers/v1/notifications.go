package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/middleware"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// NotificationHandler handles notification-related requests
type NotificationHandler struct {
	service   *service.NotificationService
	validator *validator.Validate
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(service *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		service:   service,
		validator: validator.New(),
	}
}

// List handles GET /api/v1/notifications
func (h *NotificationHandler) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var filter dto.NotificationsFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	unreadOnly := false
	if filter.Unread != nil && *filter.Unread {
		unreadOnly = true
	}

	notifications, count, err := h.service.GetByUserID(c.Context(), userID, unreadOnly, filter.Type, filter.GetOffset(), filter.GetLimit())
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to list notifications",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list notifications",
			Code:    500,
		})
	}

	// Convert to response
	items := make([]dto.NotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		items = append(items, *h.service.ToResponse(notification))
	}

	return c.JSON(dto.NewPaginatedResponse(items, filter.Page, filter.GetLimit(), count))
}

// Count handles GET /api/v1/notifications/count
func (h *NotificationHandler) Count(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	count, err := h.service.CountUnread(c.Context(), userID)
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to count notifications",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to count notifications",
			Code:    500,
		})
	}

	return c.JSON(dto.NotificationCountResponse{Count: count})
}

// MarkRead handles POST /api/v1/notifications/read
func (h *NotificationHandler) MarkRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.MarkNotificationsReadRequest
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

	if err := h.service.MarkAsRead(c.Context(), userID, req.NotificationIDs); err != nil {
		logger.FromContext(c.UserContext()).Error("failed to mark notifications as read",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to mark notifications as read",
			Code:    500,
		})
	}

	return c.JSON(dto.SuccessResponse{Success: true})
}
