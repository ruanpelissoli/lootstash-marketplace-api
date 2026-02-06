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

// TradeHandlerNew handles trade-related requests
type TradeHandlerNew struct {
	service   *service.TradeServiceNew
	validator *validator.Validate
}

// NewTradeHandlerNew creates a new trade handler
func NewTradeHandlerNew(service *service.TradeServiceNew) *TradeHandlerNew {
	return &TradeHandlerNew{
		service:   service,
		validator: validator.New(),
	}
}

// List handles GET /api/v1/trades
func (h *TradeHandlerNew) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var filter dto.TradesFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	trades, count, err := h.service.List(c.Context(), userID, filter.Status, filter.GetOffset(), filter.GetLimit())
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to list trades",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list trades",
			Code:    500,
		})
	}

	// Convert to response with user-specific rating info
	items := make([]dto.TradeResponse, 0, len(trades))
	for _, trade := range trades {
		items = append(items, *h.service.ToResponseWithUser(c.Context(), trade, userID))
	}

	return c.JSON(dto.NewPaginatedResponse(items, filter.Page, filter.GetLimit(), count))
}

// GetByID handles GET /api/v1/trades/:id
func (h *TradeHandlerNew) GetByID(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	trade, err := h.service.GetByID(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Trade not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have access to this trade",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get trade",
			"error", err.Error(),
			"trade_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get trade",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToDetailResponse(c.Context(), trade, userID))
}

// Complete handles POST /api/v1/trades/:id/complete
func (h *TradeHandlerNew) Complete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	trade, transaction, err := h.service.Complete(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Trade not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You are not a participant in this trade",
				Code:    403,
			})
		}
		if errors.Is(err, service.ErrInvalidState) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "Trade is not active",
				Code:    400,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to complete trade",
			"error", err.Error(),
			"trade_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to complete trade",
			Code:    500,
		})
	}

	return c.JSON(dto.CompleteTradeResponse{
		Trade:         h.service.ToResponse(trade),
		TransactionID: transaction.ID,
	})
}

// Cancel handles POST /api/v1/trades/:id/cancel
func (h *TradeHandlerNew) Cancel(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	var req dto.CancelTradeRequest
	if err := c.BodyParser(&req); err != nil {
		// It's OK if body is empty
		req = dto.CancelTradeRequest{}
	}

	trade, err := h.service.Cancel(c.Context(), id, userID, req.Reason)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Trade not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You are not a participant in this trade",
				Code:    403,
			})
		}
		if errors.Is(err, service.ErrInvalidState) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "Only active trades can be cancelled",
				Code:    400,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to cancel trade",
			"error", err.Error(),
			"trade_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to cancel trade",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToResponse(trade))
}
