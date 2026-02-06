package cache

import "fmt"

const (
	// Key prefixes
	prefixProfile           = "profile"
	prefixListing           = "listing"
	prefixNotificationCount = "notification:count"
	prefixDeclineReasons    = "decline:reasons"
	prefixRateLimit         = "ratelimit"
	prefixMarketplaceStats  = "marketplace:stats"
)

// Profile cache keys
func ProfileKey(id string) string {
	return fmt.Sprintf("%s:%s", prefixProfile, id)
}

func ProfilePattern() string {
	return fmt.Sprintf("%s:*", prefixProfile)
}

// Listing cache keys
func ListingKey(id string) string {
	return fmt.Sprintf("%s:%s", prefixListing, id)
}

func ListingPattern() string {
	return fmt.Sprintf("%s:*", prefixListing)
}

// Notification cache keys
func NotificationCountKey(userID string) string {
	return fmt.Sprintf("%s:%s", prefixNotificationCount, userID)
}

func NotificationCountPattern() string {
	return fmt.Sprintf("%s:*", prefixNotificationCount)
}

// Decline reasons cache key (single key for all reasons)
func DeclineReasonsKey() string {
	return prefixDeclineReasons
}

// Rate limiting keys
func RateLimitKey(ip, endpoint string) string {
	return fmt.Sprintf("%s:%s:%s", prefixRateLimit, ip, endpoint)
}

// Marketplace stats cache key
func MarketplaceStatsKey() string {
	return prefixMarketplaceStats
}
