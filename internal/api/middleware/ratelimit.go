package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/storage/redis/v3"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
)

// RateLimitConfig holds rate limit configuration
type RateLimitConfig struct {
	RedisURL    string
	RedisClient *cache.RedisClient // Optional: use existing client to check availability
	Max         int                // Max requests per window
	Window      time.Duration      // Time window
}

// DefaultRateLimitConfig returns sensible defaults
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Max:    100,
		Window: time.Minute,
	}
}

// NewRateLimitMiddleware creates a Redis-backed rate limiter
// If Redis is not available, returns a no-op middleware
func NewRateLimitMiddleware(config RateLimitConfig) fiber.Handler {
	// If Redis client is provided and not available, skip rate limiting
	if config.RedisClient != nil && !config.RedisClient.IsAvailable() {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	// If no Redis URL provided, skip rate limiting
	if config.RedisURL == "" {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	// Create Redis storage for rate limiting
	storage := redis.New(redis.Config{
		URL:   fmt.Sprintf("redis://%s", config.RedisURL),
		Reset: false,
	})

	return limiter.New(limiter.Config{
		Max:        config.Max,
		Expiration: config.Window,
		Storage:    storage,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use user ID if authenticated, otherwise use IP
			userID := GetUserID(c)
			if userID != "" {
				return fmt.Sprintf("user:%s", userID)
			}
			return fmt.Sprintf("ip:%s", c.IP())
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please try again later.",
				"code":    429,
			})
		},
	})
}

// StrictRateLimitMiddleware creates a stricter rate limiter for sensitive endpoints
// If Redis is not available, returns a no-op middleware
func StrictRateLimitMiddleware(config RateLimitConfig) fiber.Handler {
	// If Redis client is provided and not available, skip rate limiting
	if config.RedisClient != nil && !config.RedisClient.IsAvailable() {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	// If no Redis URL provided, skip rate limiting
	if config.RedisURL == "" {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	storage := redis.New(redis.Config{
		URL:   fmt.Sprintf("redis://%s", config.RedisURL),
		Reset: false,
	})

	return limiter.New(limiter.Config{
		Max:        config.Max / 5, // 5x stricter
		Expiration: config.Window,
		Storage:    storage,
		KeyGenerator: func(c *fiber.Ctx) string {
			userID := GetUserID(c)
			endpoint := c.Path()
			if userID != "" {
				return fmt.Sprintf("strict:user:%s:%s", userID, endpoint)
			}
			return fmt.Sprintf("strict:ip:%s:%s", c.IP(), endpoint)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests to this endpoint. Please try again later.",
				"code":    429,
			})
		},
	})
}
