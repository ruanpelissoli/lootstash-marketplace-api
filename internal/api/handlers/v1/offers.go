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

// OfferHandler handles offer-related requests
type OfferHandler struct {
	service   *service.OfferService
	validator *validator.Validate
}

// NewOfferHandler creates a new offer handler
func NewOfferHandler(service *service.OfferService) *OfferHandler {
	return &OfferHandler{
		service:   service,
		validator: validator.New(),
	}
}

// List handles GET /api/v1/offers
func (h *OfferHandler) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var filter dto.OffersFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	offers, count, err := h.service.List(c.Context(), userID, filter.Role, filter.Status, filter.Type, filter.ListingID, filter.ServiceID, filter.GetOffset(), filter.GetLimit())
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to list offers",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list offers",
			Code:    500,
		})
	}

	// Convert to response
	items := make([]dto.OfferResponse, 0, len(offers))
	for _, offer := range offers {
		items = append(items, *h.service.ToResponse(offer))
	}

	return c.JSON(dto.NewPaginatedResponse(items, filter.Page, filter.GetLimit(), count))
}

// GetByID handles GET /api/v1/offers/:id
func (h *OfferHandler) GetByID(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	offer, err := h.service.GetByID(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Offer not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have access to this offer",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get offer",
			"error", err.Error(),
			"offer_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get offer",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToDetailResponse(offer, userID))
}

// Create handles POST /api/v1/offers
func (h *OfferHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.CreateOfferRequest
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

	offer, err := h.service.Create(c.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Listing or service not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrSelfAction) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "You cannot make an offer on your own listing or service",
				Code:    400,
			})
		}
		if errors.Is(err, service.ErrInvalidState) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "Not available for offers",
				Code:    400,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to create offer",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create offer",
			Code:    500,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(h.service.ToResponse(offer))
}

// Accept handles POST /api/v1/offers/:id/accept
func (h *OfferHandler) Accept(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	offer, trade, serviceRun, chat, err := h.service.Accept(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Offer not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "Only the owner can accept offers",
				Code:    403,
			})
		}
		if errors.Is(err, service.ErrInvalidState) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "Offer is not pending or listing already has an active trade",
				Code:    400,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to accept offer",
			"error", err.Error(),
			"offer_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to accept offer",
			Code:    500,
		})
	}

	resp := dto.AcceptOfferResponse{
		Offer:  h.service.ToResponse(offer),
		ChatID: chat.ID,
	}

	if trade != nil {
		resp.TradeID = trade.ID
	}
	if serviceRun != nil {
		resp.ServiceRunID = serviceRun.ID
	}

	return c.JSON(resp)
}

// Reject handles POST /api/v1/offers/:id/reject
func (h *OfferHandler) Reject(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	var req dto.RejectOfferRequest
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

	offer, err := h.service.Reject(c.Context(), id, userID, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Offer or decline reason not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "Only the owner can reject offers",
				Code:    403,
			})
		}
		if errors.Is(err, service.ErrInvalidState) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "Offer is not pending",
				Code:    400,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to reject offer",
			"error", err.Error(),
			"offer_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to reject offer",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToResponse(offer))
}

// Cancel handles POST /api/v1/offers/:id/cancel
func (h *OfferHandler) Cancel(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	offer, err := h.service.Cancel(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Offer not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "Only the requester can cancel their offer",
				Code:    403,
			})
		}
		if errors.Is(err, service.ErrInvalidState) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "Only pending offers can be cancelled",
				Code:    400,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to cancel offer",
			"error", err.Error(),
			"offer_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to cancel offer",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToResponse(offer))
}

// GetDeclineReasons handles GET /api/v1/decline-reasons
func (h *OfferHandler) GetDeclineReasons(c *fiber.Ctx) error {
	reasons, err := h.service.GetDeclineReasons(c.Context())
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to get decline reasons",
			"error", err.Error(),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get decline reasons",
			Code:    500,
		})
	}

	// Convert to response
	items := make([]dto.DeclineReasonResponse, 0, len(reasons))
	for _, reason := range reasons {
		items = append(items, dto.DeclineReasonResponse{
			ID:      reason.ID,
			Code:    reason.Code,
			Message: reason.Message,
		})
	}

	return c.JSON(items)
}
