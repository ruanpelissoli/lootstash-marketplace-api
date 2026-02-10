package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/middleware"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// SubscriptionHandler handles subscription-related requests
type SubscriptionHandler struct {
	service *service.SubscriptionService
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(service *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: service}
}

// GetMe handles GET /api/v1/subscriptions/me
func (h *SubscriptionHandler) GetMe(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	info, err := h.service.GetSubscriptionInfo(c.Context(), userID)
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to get subscription info",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get subscription info",
			Code:    500,
		})
	}

	return c.JSON(info)
}

// Checkout handles POST /api/v1/subscriptions/checkout
func (h *SubscriptionHandler) Checkout(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// Parse optional request body for priceId
	var req dto.CheckoutRequest
	if err := c.BodyParser(&req); err != nil && len(c.Body()) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	resp, err := h.service.CreateCheckoutSession(c.Context(), userID, req.PriceID)
	if err != nil {
		if err == service.ErrInvalidPriceID {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "invalid_price_id",
				Message: "The provided price ID is not valid",
				Code:    400,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to create checkout session",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create checkout session",
			Code:    500,
		})
	}

	return c.JSON(resp)
}

// Cancel handles POST /api/v1/subscriptions/cancel
func (h *SubscriptionHandler) Cancel(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	err := h.service.CancelSubscription(c.Context(), userID)
	if err != nil {
		if err == service.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "No active subscription found",
				Code:    404,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to cancel subscription",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to cancel subscription",
			Code:    500,
		})
	}

	return c.JSON(dto.SuccessResponse{Success: true, Message: "Subscription will be cancelled at end of billing period"})
}

// BillingHistory handles GET /api/v1/subscriptions/billing-history
func (h *SubscriptionHandler) BillingHistory(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	resp, err := h.service.GetBillingHistory(c.Context(), userID)
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to get billing history",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get billing history",
			Code:    500,
		})
	}

	return c.JSON(resp)
}
