package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeTradeRequestReceived NotificationType = "trade_request_received"
	NotificationTypeTradeRequestAccepted NotificationType = "trade_request_accepted"
	NotificationTypeTradeRequestRejected NotificationType = "trade_request_rejected"
	NotificationTypeNewMessage           NotificationType = "new_message"
	NotificationTypeRatingReceived       NotificationType = "rating_received"
	NotificationTypeWishlistMatch        NotificationType = "wishlist_match"
)

// Notification represents a user notification
type Notification struct {
	bun.BaseModel `bun:"table:d2.notifications,alias:n"`

	ID            string          `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID        string          `bun:"user_id,type:uuid,notnull"`
	Type          NotificationType `bun:"type,notnull"`
	Title         string          `bun:"title,notnull"`
	Body          *string         `bun:"body"`
	ReferenceType *string         `bun:"reference_type"`
	ReferenceID   *string         `bun:"reference_id,type:uuid"`
	Read          bool            `bun:"read,default:false"`
	ReadAt        *time.Time      `bun:"read_at"`
	Metadata      json.RawMessage `bun:"metadata,type:jsonb,default:'{}'"`
	CreatedAt     time.Time       `bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	User *Profile `bun:"rel:belongs-to,join:user_id=id"`
}

// GetBody returns the body or empty string
func (n *Notification) GetBody() string {
	if n.Body != nil {
		return *n.Body
	}
	return ""
}

// GetReferenceType returns the reference type or empty string
func (n *Notification) GetReferenceType() string {
	if n.ReferenceType != nil {
		return *n.ReferenceType
	}
	return ""
}

// GetReferenceID returns the reference ID or empty string
func (n *Notification) GetReferenceID() string {
	if n.ReferenceID != nil {
		return *n.ReferenceID
	}
	return ""
}
