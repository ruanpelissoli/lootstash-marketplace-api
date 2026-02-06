package models

import (
	"time"

	"github.com/uptrace/bun"
)

// MarketplaceStats represents pre-aggregated marketplace statistics
// stored in d2.marketplace_stats table for Supabase Realtime subscriptions
type MarketplaceStats struct {
	bun.BaseModel `bun:"table:d2.marketplace_stats,alias:ms"`

	ID                     string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ActiveListings         int       `bun:"active_listings,notnull,default:0"`
	TradesToday            int       `bun:"trades_today,notnull,default:0"`
	AvgResponseTimeMinutes float64   `bun:"avg_response_time_minutes,default:5.0"`
	UpdatedAt              time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
