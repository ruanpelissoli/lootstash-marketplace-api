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
	if i == nil || i.redis == nil {
		return nil
	}
	return i.redis.Del(ctx, ProfileKey(id))
}

// InvalidateProfileByUsername removes a profile cached by username
func (i *Invalidator) InvalidateProfileByUsername(ctx context.Context, username string) error {
	if i == nil || i.redis == nil {
		return nil
	}
	return i.redis.Del(ctx, ProfileUsernameKey(username))
}

// InvalidateListing removes a specific listing from cache
func (i *Invalidator) InvalidateListing(ctx context.Context, id string) error {
	if i == nil || i.redis == nil {
		return nil
	}
	return i.redis.Del(ctx, ListingKey(id))
}

// InvalidateNotificationCount removes notification count from cache
func (i *Invalidator) InvalidateNotificationCount(ctx context.Context, userID string) error {
	if i == nil || i.redis == nil {
		return nil
	}
	return i.redis.Del(ctx, NotificationCountKey(userID))
}

// InvalidateDeclineReasons removes decline reasons from cache
func (i *Invalidator) InvalidateDeclineReasons(ctx context.Context) error {
	if i == nil || i.redis == nil {
		return nil
	}
	return i.redis.Del(ctx, DeclineReasonsKey())
}

// InvalidateProfileDTO removes a profile DTO cache entry
func (i *Invalidator) InvalidateProfileDTO(ctx context.Context, id string) error {
	if i == nil || i.redis == nil {
		return nil
	}
	return i.redis.Del(ctx, ProfileDTOKey(id))
}

// InvalidateListingDTO removes a listing DTO cache entry
func (i *Invalidator) InvalidateListingDTO(ctx context.Context, id string) error {
	if i == nil || i.redis == nil {
		return nil
	}
	return i.redis.Del(ctx, ListingDTOKey(id))
}

// InvalidateAllListings removes all listing-related cache entries
func (i *Invalidator) InvalidateAllListings(ctx context.Context) error {
	if i == nil || i.redis == nil {
		return nil
	}
	return i.redis.DeleteByPattern(ctx, ListingPattern())
}

// InvalidateService removes a specific service from cache
func (i *Invalidator) InvalidateService(ctx context.Context, id string) error {
	if i == nil || i.redis == nil {
		return nil
	}
	_ = i.redis.Del(ctx, ServiceKey(id))
	return i.redis.Del(ctx, ServiceDTOKey(id))
}

// InvalidateServiceProviders removes service providers cache for a game
func (i *Invalidator) InvalidateServiceProviders(ctx context.Context, game string) error {
	if i == nil || i.redis == nil {
		return nil
	}
	return i.redis.Del(ctx, ServiceProvidersKey(game))
}
