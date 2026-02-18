package v1

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/middleware"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// ListingHandler handles listing-related requests
type ListingHandler struct {
	service   *service.ListingService
	validator *validator.Validate
}

// NewListingHandler creates a new listing handler
func NewListingHandler(service *service.ListingService) *ListingHandler {
	return &ListingHandler{
		service:   service,
		validator: validator.New(),
	}
}

// List handles GET /api/v1/listings
func (h *ListingHandler) List(c *fiber.Ctx) error {
	var filter dto.ListingFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	listings, count, err := h.service.List(c.Context(), &filter)
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to list listings",
			"error", err.Error(),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list listings",
			Code:    500,
		})
	}

	// Convert to card response for list view
	items := make([]dto.ListingCardResponse, 0, len(listings))
	for _, listing := range listings {
		items = append(items, *h.service.ToCardResponse(listing))
	}

	return c.JSON(dto.NewPaginatedResponse(items, filter.Page, filter.GetLimit(), count))
}

// Search handles POST /api/v1/listings/search
func (h *ListingHandler) Search(c *fiber.Ctx) error {
	var req dto.SearchListingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	// Apply pagination defaults/bounds
	pag := dto.Pagination{Page: req.Page, PerPage: req.PerPage}

	// Convert affix filters
	var affixFilters []repository.AffixFilter
	for _, f := range req.AffixFilters {
		affixFilters = append(affixFilters, repository.AffixFilter{
			Code:     f.Code,
			MinValue: f.MinValue,
			MaxValue: f.MaxValue,
		})
	}

	// Convert asking_for filter
	var askingForFilter *repository.AskingForFilter
	if req.AskingForFilter != nil {
		askingForFilter = &repository.AskingForFilter{
			Name:        req.AskingForFilter.Name,
			Type:        req.AskingForFilter.Type,
			MinQuantity: req.AskingForFilter.MinQuantity,
			MaxQuantity: req.AskingForFilter.MaxQuantity,
		}
	}

	filter := repository.ListingFilter{
		SellerID:        req.SellerID,
		Query:           req.Q,
		CatalogItemID:   req.CatalogItemID,
		Game:            req.Game,
		Ladder:          req.Ladder,
		Hardcore:        req.Hardcore,
		IsNonRotw:       req.IsNonRotw,
		Platforms:       req.Platforms,
		Region:          req.Region,
		Category:        req.Category,
		Rarity:          req.Rarity,
		AffixFilters:    affixFilters,
		AskingForFilter: askingForFilter,
		SortBy:          req.SortBy,
		SortOrder:       req.SortOrder,
		Offset:          pag.GetOffset(),
		Limit:           pag.GetLimit(),
	}

	listings, count, err := h.service.ListByFilter(c.Context(), filter)
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to search listings",
			"error", err.Error(),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to search listings",
			Code:    500,
		})
	}

	items := make([]dto.ListingCardResponse, 0, len(listings))
	for _, listing := range listings {
		items = append(items, *h.service.ToCardResponse(listing))
	}

	return c.JSON(dto.NewPaginatedResponse(items, pag.Page, pag.GetLimit(), count))
}

// GetByID handles GET /api/v1/listings/:id
func (h *ListingHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Listing ID is required",
			Code:    400,
		})
	}

	listing, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Listing not found",
				Code:    404,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get listing",
			"error", err.Error(),
			"listing_id", id,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get listing",
			Code:    500,
		})
	}

	// Increment view count asynchronously (don't block response)
	go func() {
		_ = h.service.IncrementViews(context.Background(), id)
	}()

	return c.JSON(h.service.ToDetailResponse(c.Context(), listing))
}

// Create handles POST /api/v1/listings
func (h *ListingHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.CreateListingRequest
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

	listing, err := h.service.Create(c.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrListingLimitReached) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "listing_limit_reached",
				Message: fmt.Sprintf("Free users can have at most %d active listings. Upgrade to premium for unlimited listings.", service.FreeListingLimit),
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to create listing",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create listing",
			Code:    500,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(h.service.ToCardResponse(listing))
}

// Update handles PATCH /api/v1/listings/:id
func (h *ListingHandler) Update(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	var req dto.UpdateListingRequest
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

	listing, err := h.service.Update(c.Context(), id, userID, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Listing not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You can only update your own listings",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to update listing",
			"error", err.Error(),
			"listing_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update listing",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToResponse(listing))
}

// Delete handles DELETE /api/v1/listings/:id
func (h *ListingHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	err := h.service.Delete(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Listing not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You can only delete your own listings",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to delete listing",
			"error", err.Error(),
			"listing_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete listing",
			Code:    500,
		})
	}

	return c.JSON(dto.SuccessResponse{Success: true, Message: "Listing cancelled"})
}

// ListMy handles GET /api/v1/my/listings
func (h *ListingHandler) ListMy(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var filter dto.MyListingsFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	listings, count, err := h.service.ListBySellerID(c.Context(), userID, filter.Status, filter.GetOffset(), filter.GetLimit())
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to list my listings",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list listings",
			Code:    500,
		})
	}

	// Convert to card response for list view
	items := make([]dto.ListingCardResponse, 0, len(listings))
	for _, listing := range listings {
		items = append(items, *h.service.ToCardResponse(listing))
	}

	return c.JSON(dto.NewPaginatedResponse(items, filter.Page, filter.GetLimit(), count))
}

// parsePlatformsFromString splits a comma-separated platform string into a slice
func parsePlatformsFromString(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
