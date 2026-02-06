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

// WishlistHandler handles wishlist-related requests
type WishlistHandler struct {
	service   *service.WishlistService
	validator *validator.Validate
}

// NewWishlistHandler creates a new wishlist handler
func NewWishlistHandler(service *service.WishlistService) *WishlistHandler {
	return &WishlistHandler{
		service:   service,
		validator: validator.New(),
	}
}

// Create handles POST /api/v1/wishlist
func (h *WishlistHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.CreateWishlistItemRequest
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

	item, err := h.service.Create(c.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrPremiumRequired) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "premium_required",
				Message: "Wishlist is a premium feature. Upgrade to premium to use it.",
				Code:    403,
			})
		}
		if errors.Is(err, service.ErrWishlistLimitReached) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "wishlist_limit_reached",
				Message: "You can have at most 10 active wishlist items.",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to create wishlist item",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create wishlist item",
			Code:    500,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(h.service.ToResponse(item))
}

// List handles GET /api/v1/wishlist
func (h *WishlistHandler) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var filter dto.WishlistFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	items, count, err := h.service.List(c.Context(), userID, filter.GetOffset(), filter.GetLimit())
	if err != nil {
		if errors.Is(err, service.ErrPremiumRequired) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "premium_required",
				Message: "Wishlist is a premium feature. Upgrade to premium to use it.",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to list wishlist items",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list wishlist items",
			Code:    500,
		})
	}

	responses := make([]dto.WishlistItemResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, *h.service.ToResponse(item))
	}

	return c.JSON(dto.NewPaginatedResponse(responses, filter.Page, filter.GetLimit(), count))
}

// Update handles PATCH /api/v1/wishlist/:id
func (h *WishlistHandler) Update(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	var req dto.UpdateWishlistItemRequest
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

	item, err := h.service.Update(c.Context(), id, userID, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Wishlist item not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You can only update your own wishlist items",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to update wishlist item",
			"error", err.Error(),
			"wishlist_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update wishlist item",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToResponse(item))
}

// Delete handles DELETE /api/v1/wishlist/:id
func (h *WishlistHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	err := h.service.Delete(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Wishlist item not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You can only delete your own wishlist items",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to delete wishlist item",
			"error", err.Error(),
			"wishlist_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete wishlist item",
			Code:    500,
		})
	}

	return c.JSON(dto.SuccessResponse{Success: true, Message: "Wishlist item deleted"})
}
