package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

const (
	// activityThrottleTTL is how often we update last_active_at per user
	// This prevents database writes on every single request
	activityThrottleTTL = 1 * time.Minute
)

// activityKey returns the Redis key for tracking user activity throttle
func activityKey(userID string) string {
	return fmt.Sprintf("activity:%s", userID)
}

// ActivityTracker creates middleware that updates user's last_active_at timestamp
// It throttles updates to once per minute to avoid excessive database writes
// If Redis is unavailable, updates happen on every request (no throttling)
func ActivityTracker(profileRepo repository.ProfileRepository, redis *cache.RedisClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user ID from context (set by auth middleware)
		userID := GetUserID(c)
		if userID == "" {
			// Not authenticated, skip activity tracking
			return c.Next()
		}

		// Check if we've already updated recently (throttle)
		// If Redis is unavailable, skip throttle check (allow update)
		if redis != nil && redis.IsAvailable() {
			key := activityKey(userID)
			exists, err := redis.Exists(c.Context(), key)
			if err == nil && exists {
				// Already updated within throttle period, skip
				return c.Next()
			}
		}

		// Update last_active_at in database (async to not block request)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := profileRepo.UpdateLastActiveAt(ctx, userID); err != nil {
				// Log error but don't fail the request
				fmt.Printf("[ACTIVITY] Error updating last_active_at for user %s: %v\n", userID, err)
				return
			}

			// Set throttle key in Redis (if available)
			if redis != nil && redis.IsAvailable() {
				_ = redis.Set(ctx, activityKey(userID), "1", activityThrottleTTL)
			}
		}()

		return c.Next()
	}
}
