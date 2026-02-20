package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Profile represents a user profile in the database
type Profile struct {
	bun.BaseModel `bun:"table:d2.profiles,alias:p"`

	ID                             string     `bun:"id,pk,type:uuid"`
	Username                       string     `bun:"username,notnull"`
	DisplayName                    *string    `bun:"display_name"`
	AvatarURL                      *string    `bun:"avatar_url"`
	TotalTrades                    int        `bun:"total_trades,default:0"`
	AverageRating                  float64    `bun:"average_rating,default:0"`
	RatingCount                    int        `bun:"rating_count,default:0"`
	BattleNetID                    *int64     `bun:"battle_net_id"`
	BattleTag                      *string    `bun:"battle_tag"`
	BattleNetLinkedAt              *time.Time `bun:"battle_net_linked_at"`
	IsPremium                      bool       `bun:"is_premium,default:false"`
	IsAdmin                        bool       `bun:"is_admin,default:false"`
	StripeCustomerID               *string    `bun:"stripe_customer_id"`
	StripeSubscriptionID           *string    `bun:"stripe_subscription_id"`
	SubscriptionStatus             string     `bun:"subscription_status,default:'none'"`
	SubscriptionCurrentPeriodEnd   *time.Time `bun:"subscription_current_period_end"`
	CancelAtPeriodEnd              bool       `bun:"cancel_at_period_end,default:false"`
	ProfileFlair                   *string    `bun:"profile_flair"`
	UsernameColor                  *string    `bun:"username_color"`
	Timezone                       *string    `bun:"timezone"`
	PreferredLadder                *bool      `bun:"preferred_ladder"`
	PreferredHardcore              *bool      `bun:"preferred_hardcore"`
	PreferredPlatforms             []string   `bun:"preferred_platforms,array"`
	PreferredRegion                *string    `bun:"preferred_region"`
	PreferredNonRotw               *bool      `bun:"preferred_non_rotw"`
	LastActiveAt                   time.Time  `bun:"last_active_at,nullzero,default:current_timestamp"`
	CreatedAt                      time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt                      time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

// GetDisplayName returns the display name or username if not set
func (p *Profile) GetDisplayName() string {
	if p.DisplayName != nil && *p.DisplayName != "" {
		return *p.DisplayName
	}
	return p.Username
}

// GetAvatarURL returns the avatar URL or empty string
func (p *Profile) GetAvatarURL() string {
	if p.AvatarURL != nil {
		return *p.AvatarURL
	}
	return ""
}

// GetBattleTag returns the BattleTag or empty string
func (p *Profile) GetBattleTag() string {
	if p.BattleTag != nil {
		return *p.BattleTag
	}
	return ""
}

// IsBattleNetLinked returns true if the profile has a linked Battle.net account
func (p *Profile) IsBattleNetLinked() bool {
	return p.BattleNetID != nil
}

// GetProfileFlair returns the profile flair or empty string
func (p *Profile) GetProfileFlair() string {
	if p.ProfileFlair != nil {
		return *p.ProfileFlair
	}
	return ""
}

// GetUsernameColor returns the username color or empty string
func (p *Profile) GetUsernameColor() string {
	if p.UsernameColor != nil {
		return *p.UsernameColor
	}
	return ""
}

// GetTimezone returns the timezone or empty string
func (p *Profile) GetTimezone() string {
	if p.Timezone != nil {
		return *p.Timezone
	}
	return ""
}

// GetPreferredRegion returns the preferred region or empty string
func (p *Profile) GetPreferredRegion() string {
	if p.PreferredRegion != nil {
		return *p.PreferredRegion
	}
	return ""
}
