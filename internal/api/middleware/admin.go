package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// AdminMiddleware creates middleware that restricts access to admin users only
func AdminMiddleware(profileService *service.ProfileService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := GetUserID(c)
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Error:   "unauthorized",
				Message: "Authentication required",
				Code:    401,
			})
		}

		isAdmin, err := profileService.IsAdmin(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to verify admin status",
				Code:    500,
			})
		}

		if !isAdmin {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "Admin access required",
				Code:    403,
			})
		}

		return c.Next()
	}
}
