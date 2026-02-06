package v1

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/middleware"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// BattleNetHandler handles Battle.net OAuth requests
type BattleNetHandler struct {
	service   *service.BattleNetService
	validator *validator.Validate
}

// NewBattleNetHandler creates a new Battle.net handler
func NewBattleNetHandler(service *service.BattleNetService) *BattleNetHandler {
	return &BattleNetHandler{
		service:   service,
		validator: validator.New(),
	}
}

// Link handles POST /api/v1/me/battlenet/link
func (h *BattleNetHandler) Link(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.BattleNetLinkRequest
	if err := c.BodyParser(&req); err != nil {
		// Body is optional, use defaults
		req = dto.BattleNetLinkRequest{}
	}

	if err := h.validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
			Code:    400,
		})
	}

	authURL, err := h.service.GetAuthorizationURL(c.Context(), userID, req.Region)
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to generate authorization URL",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to initiate Battle.net linking",
			Code:    500,
		})
	}

	return c.JSON(dto.BattleNetLinkResponse{
		AuthorizationURL: authURL,
	})
}

// Callback handles POST /api/v1/me/battlenet/callback
func (h *BattleNetHandler) Callback(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.BattleNetCallbackRequest
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

	profile, err := h.service.HandleCallback(c.Context(), userID, req.Code, req.State)
	if err != nil {
		log := logger.FromContext(c.UserContext())

		// Check for duplicate Battle.net account
		if strings.Contains(err.Error(), "already linked") {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{
				Error:   "conflict",
				Message: "This Battle.net account is already linked to another user",
				Code:    409,
			})
		}

		// Check for invalid/expired state
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "mismatch") {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid or expired authorization state",
				Code:    400,
			})
		}

		log.Error("failed to handle Battle.net callback",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to complete Battle.net linking",
			Code:    500,
		})
	}

	return c.JSON(dto.BattleNetCallbackResponse{
		Success:     true,
		BattleTag:   profile.GetBattleTag(),
		BattleNetID: *profile.BattleNetID,
		LinkedAt:    *profile.BattleNetLinkedAt,
	})
}

// Unlink handles DELETE /api/v1/me/battlenet
func (h *BattleNetHandler) Unlink(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	err := h.service.Unlink(c.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Profile not found",
				Code:    404,
			})
		}

		if strings.Contains(err.Error(), "no Battle.net account linked") {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "No Battle.net account linked",
				Code:    400,
			})
		}

		logger.FromContext(c.UserContext()).Error("failed to unlink Battle.net",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to unlink Battle.net account",
			Code:    500,
		})
	}

	return c.JSON(dto.BattleNetUnlinkResponse{
		Success: true,
		Message: "Battle.net account unlinked successfully",
	})
}
