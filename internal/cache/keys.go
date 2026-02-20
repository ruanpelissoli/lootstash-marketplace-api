package cache

import "fmt"

const (
	// Key prefixes
	prefixProfile           = "profile"
	prefixProfileDTO        = "profile:dto"
	prefixListing           = "listing"
	prefixListingDTO        = "listing:dto"
	prefixNotificationCount = "notification:count"
	prefixDeclineReasons    = "decline:reasons"
	prefixRateLimit         = "ratelimit"
	prefixMarketplaceStats   = "marketplace:stats"
	prefixHomeStats          = "home:stats"
	prefixHomeRecent         = "home:recent"
	prefixHomeRecentServices = "home:recent:services"
	prefixService            = "service"
	prefixServiceDTO         = "service:dto"
	prefixServiceProviders   = "service:providers"
	prefixFilterResults      = "filter:results"
)

// Profile cache keys
func ProfileKey(id string) string {
	return fmt.Sprintf("%s:%s", prefixProfile, id)
}

func ProfileUsernameKey(username string) string {
	return fmt.Sprintf("%s:username:%s", prefixProfile, username)
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

// Profile DTO cache key (camelCase JSON for frontend)
func ProfileDTOKey(id string) string {
	return fmt.Sprintf("%s:%s", prefixProfileDTO, id)
}

// Listing DTO cache key (camelCase JSON for frontend)
func ListingDTOKey(id string) string {
	return fmt.Sprintf("%s:%s", prefixListingDTO, id)
}

// HomeStatsKey returns the home stats cache key
func HomeStatsKey() string {
	return prefixHomeStats
}

// HomeRecentKey returns the home recent listings cache key
func HomeRecentKey() string {
	return prefixHomeRecent
}

// HomeRecentServicesKey returns the home recent services cache key
func HomeRecentServicesKey() string {
	return prefixHomeRecentServices
}

// ServiceKey returns the service cache key
func ServiceKey(id string) string {
	return fmt.Sprintf("%s:%s", prefixService, id)
}

// ServiceDTOKey returns the service DTO cache key
func ServiceDTOKey(id string) string {
	return fmt.Sprintf("%s:%s", prefixServiceDTO, id)
}

// ServiceProvidersKey returns the service providers cache key for a game
func ServiceProvidersKey(game string) string {
	return fmt.Sprintf("%s:%s", prefixServiceProviders, game)
}

// FilterResultsKey returns the cache key for a filter result hash
func FilterResultsKey(hash string) string {
	return fmt.Sprintf("%s:%s", prefixFilterResults, hash)
}

// FilterResultsPattern returns the pattern for all filter result keys
func FilterResultsPattern() string {
	return fmt.Sprintf("%s:*", prefixFilterResults)
}
