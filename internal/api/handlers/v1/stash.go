package v1

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/middleware"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// StashHandler handles stash-related requests
type StashHandler struct {
	service   *service.StashService
	validator *validator.Validate
}

// NewStashHandler creates a new stash handler
func NewStashHandler(service *service.StashService) *StashHandler {
	return &StashHandler{
		service:   service,
		validator: validator.New(),
	}
}

// Search handles POST /api/v1/stash/search
func (h *StashHandler) Search(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.SearchStashRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	items, count, err := h.service.Search(c.Context(), userID, &req)
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to search stash items",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to search stash items",
			Code:    500,
		})
	}

	responses := make([]dto.StashItemResponse, 0, len(items))
	for _, item := range items {
		for _, resp := range h.service.ToListResponses(c.Context(), item) {
			responses = append(responses, *resp)
		}
	}

	pag := dto.Pagination{Page: req.Page, PerPage: req.PerPage}
	return c.JSON(dto.NewPaginatedResponse(responses, pag.Page, pag.GetLimit(), count))
}

// List handles GET /api/v1/stash
func (h *StashHandler) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var filter dto.StashFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	items, count, err := h.service.List(c.Context(), userID, &filter)
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to list stash items",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list stash items",
			Code:    500,
		})
	}

	responses := make([]dto.StashItemResponse, 0, len(items))
	for _, item := range items {
		for _, resp := range h.service.ToListResponses(c.Context(), item) {
			responses = append(responses, *resp)
		}
	}

	return c.JSON(dto.NewPaginatedResponse(responses, filter.Page, filter.GetLimit(), count))
}

// Create handles POST /api/v1/stash
func (h *StashHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.CreateStashItemRequest
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
		logger.FromContext(c.UserContext()).Error("failed to create stash item",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create stash item",
			Code:    500,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(h.service.ToResponse(c.Context(), item))
}

// GetByID handles GET /api/v1/stash/:id
func (h *StashHandler) GetByID(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Stash item ID is required",
			Code:    400,
		})
	}

	item, err := h.service.GetByID(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Stash item not found",
				Code:    404,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get stash item",
			"error", err.Error(),
			"id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get stash item",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToResponse(c.Context(), item))
}

// Delete handles DELETE /api/v1/stash/:id
// Accepts optional ?listed=true|false query param for split-row deletion:
//   - listed=true:  delete only the listed portion (cancel listings, keep unlisted stacks)
//   - listed=false: delete only the unlisted portion (keep listed stacks)
//   - omitted:      delete everything
func (h *StashHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	var deleteListed *bool
	if listedParam := c.Query("listed"); listedParam != "" {
		val := listedParam == "true"
		deleteListed = &val
	}

	err := h.service.Delete(c.Context(), id, userID, deleteListed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Stash item not found",
				Code:    404,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to delete stash item",
			"error", err.Error(),
			"id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete stash item",
			Code:    500,
		})
	}

	return c.JSON(dto.SuccessResponse{Success: true, Message: "Stash item deleted"})
}

// Unlist handles POST /api/v1/stash/:id/unlist
func (h *StashHandler) Unlist(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	err := h.service.Unlist(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Stash item not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrNotListed) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{
				Error:   "not_listed",
				Message: "This stash item has no active listings",
				Code:    409,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to unlist stash item",
			"error", err.Error(),
			"id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to unlist stash item",
			Code:    500,
		})
	}

	return c.JSON(dto.SuccessResponse{Success: true, Message: "Listings cancelled"})
}

// AdjustQuantity handles POST /api/v1/stash/:id/quantity
func (h *StashHandler) AdjustQuantity(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	var req dto.AdjustQuantityRequest
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

	item, err := h.service.AdjustQuantity(c.Context(), id, userID, req.Quantity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Stash item not found or was deleted (quantity reached 0)",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrInvalidState) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{
				Error:   "invalid_state",
				Message: "Only stackable items (runes, gems, misc) support quantity adjustment",
				Code:    409,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to adjust stash quantity",
			"error", err.Error(),
			"id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to adjust quantity",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToResponse(c.Context(), item))
}

// CreateListing handles POST /api/v1/stash/list
func (h *StashHandler) CreateListing(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.CreateListingFromStashRequest
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

	listing, err := h.service.CreateListingFromStash(c.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Stash item not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrInsufficientQuantity) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{
				Error:   "insufficient_quantity",
				Message: "Not enough available quantity to list this item",
				Code:    409,
			})
		}
		if errors.Is(err, service.ErrListingLimitReached) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "listing_limit_reached",
				Message: fmt.Sprintf("Free users can have at most %d active listings. Upgrade to premium for unlimited listings.", service.FreeListingLimit),
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to create listing from stash",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create listing from stash",
			Code:    500,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":   true,
		"listingId": listing.ID,
	})
}

// BulkImportPreview handles POST /api/v1/stash/import/preview
func (h *StashHandler) BulkImportPreview(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid multipart form data",
			Code:    400,
		})
	}

	files := form.File["images"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "At least one image file is required",
			Code:    400,
		})
	}

	game := c.FormValue("game", "diablo2")
	ladder := c.FormValue("ladder", "false") == "true"
	hardcore := c.FormValue("hardcore", "false") == "true"
	region := c.FormValue("region", "americas")
	platformsRaw := c.FormValue("platforms", "pc")
	platforms := strings.Split(platformsRaw, ",")

	resp, err := h.service.BulkImportPreview(c.Context(), userID, files, game, ladder, hardcore, region, platforms)
	if err != nil {
		if errors.Is(err, service.ErrPremiumRequired) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "premium_required",
				Message: "Bulk import is a premium feature. Upgrade to premium to use it.",
				Code:    403,
			})
		}
		if errors.Is(err, service.ErrVisionNotConfigured) {
			return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{
				Error:   "service_unavailable",
				Message: "Image import is not configured on this server",
				Code:    503,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to process import preview",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to process import",
			Code:    500,
		})
	}

	return c.JSON(resp)
}

// BulkImportConfirm handles POST /api/v1/stash/import/confirm
func (h *StashHandler) BulkImportConfirm(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.BulkImportConfirmRequest
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

	items, err := h.service.BulkImportConfirm(c.Context(), userID, req.Items)
	if err != nil {
		if errors.Is(err, service.ErrPremiumRequired) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "premium_required",
				Message: "Bulk import is a premium feature. Upgrade to premium to use it.",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to confirm import",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to save imported items",
			Code:    500,
		})
	}

	responses := make([]dto.StashItemResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, *h.service.ToResponse(c.Context(), item))
	}

	return c.Status(fiber.StatusCreated).JSON(responses)
}
