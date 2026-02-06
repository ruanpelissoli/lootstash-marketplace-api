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

// RatingHandler handles rating-related requests
type RatingHandler struct {
	service   *service.RatingService
	validator *validator.Validate
}

// NewRatingHandler creates a new rating handler
func NewRatingHandler(service *service.RatingService) *RatingHandler {
	return &RatingHandler{
		service:   service,
		validator: validator.New(),
	}
}

// Create handles POST /api/v1/ratings
func (h *RatingHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.CreateRatingRequest
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

	rating, err := h.service.Create(c.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Transaction not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You are not a participant in this transaction",
				Code:    403,
			})
		}
		if errors.Is(err, service.ErrAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{
				Error:   "conflict",
				Message: "You have already rated this transaction",
				Code:    409,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to create rating",
			"error", err.Error(),
			"user_id", userID,
			"transaction_id", req.TransactionID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create rating",
			Code:    500,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(h.service.ToResponse(rating))
}

// GetByProfileID handles GET /api/v1/profiles/:id/ratings
func (h *RatingHandler) GetByProfileID(c *fiber.Ctx) error {
	profileID := c.Params("id")

	var filter dto.RatingsFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	ratings, count, err := h.service.GetByUserID(c.Context(), profileID, filter.GetOffset(), filter.GetLimit())
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to get ratings",
			"error", err.Error(),
			"profile_id", profileID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get ratings",
			Code:    500,
		})
	}

	// Convert to response
	items := make([]dto.RatingResponse, 0, len(ratings))
	for _, rating := range ratings {
		items = append(items, *h.service.ToResponse(rating))
	}

	return c.JSON(dto.NewPaginatedResponse(items, filter.Page, filter.GetLimit(), count))
}
