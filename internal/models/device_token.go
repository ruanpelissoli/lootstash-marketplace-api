package models

import (
	"time"

	"github.com/uptrace/bun"
)

// DeviceToken represents a registered mobile device for push notifications
type DeviceToken struct {
	bun.BaseModel `bun:"table:d2.device_tokens,alias:dt"`

	ID            string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID        string    `bun:"user_id,type:uuid,notnull"`
	ExpoPushToken string    `bun:"expo_push_token,notnull"`
	DeviceName    *string   `bun:"device_name"`
	Platform      string    `bun:"platform,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	User *Profile `bun:"rel:belongs-to,join:user_id=id"`
}
