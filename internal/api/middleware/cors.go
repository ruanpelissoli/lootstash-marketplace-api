package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORSConfig holds CORS middleware configuration
type CORSConfig struct {
	AllowedOrigins string
}

// NewCORSMiddleware creates a CORS middleware
func NewCORSMiddleware(config CORSConfig) fiber.Handler {
	// AllowCredentials cannot be true when AllowOrigins is "*"
	allowCredentials := config.AllowedOrigins != "*"

	return cors.New(cors.Config{
		AllowOrigins:     config.AllowedOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Request-ID",
		AllowCredentials: allowCredentials,
		ExposeHeaders:    "X-Request-ID,X-RateLimit-Limit,X-RateLimit-Remaining,X-RateLimit-Reset",
		MaxAge:           86400, // 24 hours
	})
}
