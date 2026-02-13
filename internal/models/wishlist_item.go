package models

import (
	"time"

	"github.com/uptrace/bun"
)

// WishlistItem represents a wishlist entry for a desired item
type WishlistItem struct {
	bun.BaseModel `bun:"table:d2.wishlist_items,alias:wi"`

	ID           string          `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID       string          `bun:"user_id,type:uuid,notnull"`
	Name         string          `bun:"name,notnull"`
	Category     *string         `bun:"category"`
	Rarity       *string         `bun:"rarity"`
	ImageURL     *string         `bun:"image_url"`
	StatCriteria []StatCriterion `bun:"stat_criteria,type:jsonb,default:'[]'"`
	Game         string          `bun:"game,default:'diablo2'"`
	Ladder       *bool           `bun:"ladder"`
	Hardcore     *bool           `bun:"hardcore"`
	IsNonRotw    *bool           `bun:"is_non_rotw"`
	Platform     *string         `bun:"platform"`
	Region       *string         `bun:"region"`
	Status       string          `bun:"status,default:'active'"`
	CreatedAt    time.Time       `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt    time.Time       `bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	User *Profile `bun:"rel:belongs-to,join:user_id=id"`
}

// StatCriterion represents a single stat filter criterion
type StatCriterion struct {
	Code     string `json:"code"`
	MinValue *int   `json:"minValue,omitempty"`
	MaxValue *int   `json:"maxValue,omitempty"`
}
