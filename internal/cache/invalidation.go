package cache

import (
	"context"
)

// Invalidator handles cache invalidation strategies
type Invalidator struct {
	redis *RedisClient
}

// NewInvalidator creates a new cache invalidator
func NewInvalidator(redis *RedisClient) *Invalidator {
	return &Invalidator{redis: redis}
}

// InvalidateProfile removes a specific profile from cache
func (i *Invalidator) InvalidateProfile(ctx context.Context, id string) error {
	return i.redis.Del(ctx, ProfileKey(id))
}

// InvalidateListing removes a specific listing from cache
func (i *Invalidator) InvalidateListing(ctx context.Context, id string) error {
	return i.redis.Del(ctx, ListingKey(id))
}

// InvalidateNotificationCount removes notification count from cache
func (i *Invalidator) InvalidateNotificationCount(ctx context.Context, userID string) error {
	return i.redis.Del(ctx, NotificationCountKey(userID))
}

// InvalidateDeclineReasons removes decline reasons from cache
func (i *Invalidator) InvalidateDeclineReasons(ctx context.Context) error {
	return i.redis.Del(ctx, DeclineReasonsKey())
}

// InvalidateAllListings removes all listing-related cache entries
func (i *Invalidator) InvalidateAllListings(ctx context.Context) error {
	return i.redis.DeleteByPattern(ctx, ListingPattern())
}
