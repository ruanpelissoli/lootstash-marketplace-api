package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/middleware"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// DeviceTokenHandler handles device token-related requests
type DeviceTokenHandler struct {
	service   *service.DeviceTokenService
	validator *validator.Validate
}

// NewDeviceTokenHandler creates a new device token handler
func NewDeviceTokenHandler(service *service.DeviceTokenService) *DeviceTokenHandler {
	return &DeviceTokenHandler{
		service:   service,
		validator: validator.New(),
	}
}

// Register handles POST /api/v1/device-tokens
func (h *DeviceTokenHandler) Register(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.RegisterDeviceTokenRequest
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

	if err := h.service.Register(c.Context(), userID, &req); err != nil {
		logger.FromContext(c.UserContext()).Error("failed to register device token",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to register device token",
			Code:    500,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse{Success: true})
}

// Unregister handles DELETE /api/v1/device-tokens
func (h *DeviceTokenHandler) Unregister(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.UnregisterDeviceTokenRequest
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

	if err := h.service.Unregister(c.Context(), userID, &req); err != nil {
		logger.FromContext(c.UserContext()).Error("failed to unregister device token",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to unregister device token",
			Code:    500,
		})
	}

	return c.JSON(dto.SuccessResponse{Success: true})
}
