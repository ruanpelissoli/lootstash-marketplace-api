package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// WebhookHandler handles webhook requests
type WebhookHandler struct {
	subscriptionService *service.SubscriptionService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(subscriptionService *service.SubscriptionService) *WebhookHandler {
	return &WebhookHandler{subscriptionService: subscriptionService}
}

// StripeWebhook handles POST /api/v1/webhooks/stripe
func (h *WebhookHandler) StripeWebhook(c *fiber.Ctx) error {
	payload := c.Body()
	sigHeader := c.Get("Stripe-Signature")

	if err := h.subscriptionService.HandleWebhook(c.Context(), payload, sigHeader); err != nil {
		logger.Log.Error("stripe webhook error",
			"error", err.Error(),
		)
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "webhook_error",
			Message: err.Error(),
			Code:    400,
		})
	}

	return c.JSON(fiber.Map{"received": true})
}
