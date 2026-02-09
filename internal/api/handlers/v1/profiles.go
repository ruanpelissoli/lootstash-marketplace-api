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

// ProfileHandler handles profile-related requests
type ProfileHandler struct {
	service   *service.ProfileService
	validator *validator.Validate
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(service *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		service:   service,
		validator: validator.New(),
	}
}

// GetByID handles GET /api/v1/profiles/:id
func (h *ProfileHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Profile ID is required",
			Code:    400,
		})
	}

	profile, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Profile not found",
				Code:    404,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get profile",
			"error", err.Error(),
			"profile_id", id,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get profile",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToResponse(profile))
}

// GetMe handles GET /api/v1/me
func (h *ProfileHandler) GetMe(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	profile, err := h.service.GetByID(c.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Profile not found",
				Code:    404,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get my profile",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get profile",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToMyProfileResponse(profile))
}

// UpdateMe handles PATCH /api/v1/me
func (h *ProfileHandler) UpdateMe(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.UpdateProfileRequest
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

	profile, err := h.service.Update(c.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Profile not found",
				Code:    404,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to update profile",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update profile",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToMyProfileResponse(profile))
}

// UploadPicture handles POST /api/v1/me/picture
func (h *ProfileHandler) UploadPicture(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// Parse multipart form with 2MB limit
	file, err := c.FormFile("picture")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "No picture file provided",
			Code:    400,
		})
	}

	// Validate file size (2MB max)
	const maxSize = 2 * 1024 * 1024 // 2MB
	if file.Size > maxSize {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "File too large. Maximum size is 2MB",
			Code:    400,
		})
	}

	// Validate content type
	contentType := file.Header.Get("Content-Type")
	if !isValidImageType(contentType) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid file type. Allowed: PNG, JPEG, WebP",
			Code:    400,
		})
	}

	// Read file data
	f, err := file.Open()
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to open uploaded file",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to process file",
			Code:    500,
		})
	}
	defer f.Close()

	data := make([]byte, file.Size)
	if _, err := f.Read(data); err != nil {
		logger.FromContext(c.UserContext()).Error("failed to read uploaded file",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to process file",
			Code:    500,
		})
	}

	// Upload and update profile
	avatarURL, err := h.service.UploadProfilePicture(c.Context(), userID, data, contentType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Profile not found",
				Code:    404,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to upload profile picture",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to upload picture",
			Code:    500,
		})
	}

	return c.JSON(dto.UploadPictureResponse{
		AvatarURL: avatarURL,
	})
}

// isValidImageType checks if the content type is an allowed image type
func isValidImageType(contentType string) bool {
	switch contentType {
	case "image/png", "image/jpeg", "image/webp":
		return true
	default:
		return false
	}
}

// GetSales handles GET /api/v1/profiles/:id/sales
func (h *ProfileHandler) GetSales(c *fiber.Ctx) error {
	profileID := c.Params("id")
	if profileID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Profile ID is required",
			Code:    400,
		})
	}

	var filter dto.SalesFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	// Verify profile exists
	_, err := h.service.GetByID(c.Context(), profileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Profile not found",
				Code:    404,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get profile",
			"error", err.Error(),
			"profile_id", profileID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get profile",
			Code:    500,
		})
	}

	// Get sales
	response, err := h.service.GetSales(c.Context(), profileID, filter.GetOffset(), filter.GetLimit())
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to get sales",
			"error", err.Error(),
			"profile_id", profileID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get sales",
			Code:    500,
		})
	}

	return c.JSON(response)
}
