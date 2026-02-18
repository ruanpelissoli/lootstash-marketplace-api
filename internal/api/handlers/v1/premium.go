package v1

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/middleware"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// PremiumHandler handles premium feature requests
type PremiumHandler struct {
	subscriptionService *service.SubscriptionService
	listingService      *service.ListingService
	validator           *validator.Validate
}

// NewPremiumHandler creates a new premium handler
func NewPremiumHandler(subscriptionService *service.SubscriptionService, listingService *service.ListingService) *PremiumHandler {
	return &PremiumHandler{
		subscriptionService: subscriptionService,
		listingService:      listingService,
		validator:           validator.New(),
	}
}

// UpdateFlair handles PATCH /api/v1/me/flair
func (h *PremiumHandler) UpdateFlair(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.UpdateFlairRequest
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
			Message: "Invalid flair value. Must be one of: none, gold, flame, ice, necro, royal",
			Code:    400,
		})
	}

	err := h.subscriptionService.UpdateFlair(c.Context(), userID, req.Flair)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "Premium subscription required to set profile flair",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to update flair",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update flair",
			Code:    500,
		})
	}

	return c.JSON(dto.SuccessResponse{Success: true, Message: "Flair updated"})
}

// UpdateUsernameColor handles PATCH /api/v1/me/username-color
func (h *PremiumHandler) UpdateUsernameColor(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.UpdateUsernameColorRequest
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
			Message: "Color is required",
			Code:    400,
		})
	}

	if req.Color != "none" && !hexColorRegex.MatchString(req.Color) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid color value. Must be 'none' or a valid hex color (e.g. #FF5733)",
			Code:    400,
		})
	}

	err := h.subscriptionService.UpdateUsernameColor(c.Context(), userID, req.Color)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "Premium subscription required to set username color",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to update username color",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update username color",
			Code:    500,
		})
	}

	return c.JSON(dto.SuccessResponse{Success: true, Message: "Username color updated"})
}

// PriceHistory handles GET /api/v1/marketplace/price-history
func (h *PremiumHandler) PriceHistory(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	itemName := c.Query("item")
	if itemName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "item query parameter is required",
			Code:    400,
		})
	}

	days := 30
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

	resp, err := h.subscriptionService.GetPriceHistory(c.Context(), userID, itemName, days)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "Premium subscription required to view price history",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get price history",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get price history",
			Code:    500,
		})
	}

	return c.JSON(resp)
}

// ListingCount handles GET /api/v1/my/listings/count
func (h *PremiumHandler) ListingCount(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	listings, _, err := h.listingService.ListBySellerID(c.Context(), userID, "active", 0, 0)
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to count listings",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to count listings",
			Code:    500,
		})
	}

	return c.JSON(dto.ListingCountResponse{Count: len(listings)})
}
