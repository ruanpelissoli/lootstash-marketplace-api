package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// CacheControl returns middleware that sets Cache-Control headers on successful responses.
func CacheControl(maxAgeSeconds int) fiber.Handler {
	value := fmt.Sprintf("public, max-age=%d, s-maxage=%d", maxAgeSeconds, maxAgeSeconds)
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if c.Response().StatusCode() >= 200 && c.Response().StatusCode() < 300 {
			c.Set("Cache-Control", value)
		}
		return err
	}
}
