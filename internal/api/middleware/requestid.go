package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
)

const (
	// RequestIDHeader is the HTTP header for request ID
	RequestIDHeader = "X-Request-ID"
)

// RequestID middleware generates a unique request ID for each request
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if request ID was provided in header
		requestID := c.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to response header
		c.Set(RequestIDHeader, requestID)

		// Store request ID in context for logging
		ctx := logger.WithRequestID(c.Context(), requestID)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// GetRequestID retrieves the request ID from context
func GetRequestID(c *fiber.Ctx) string {
	if reqID, ok := c.UserContext().Value(logger.RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}
