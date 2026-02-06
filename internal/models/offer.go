package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

// Offer represents an offer on a listing (renamed from TradeRequest)
type Offer struct {
	bun.BaseModel `bun:"table:d2.offers,alias:o"`

	ID              string          `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ListingID       string          `bun:"listing_id,type:uuid,notnull"`
	RequesterID     string          `bun:"requester_id,type:uuid,notnull"`
	OfferedItems    json.RawMessage `bun:"offered_items,type:jsonb,notnull,default:'[]'"`
	Message         *string         `bun:"message"`
	Status          string          `bun:"status,notnull,default:'pending'"`
	DeclineReasonID *int            `bun:"decline_reason_id"`
	DeclineNote     *string         `bun:"decline_note"`
	CreatedAt       time.Time       `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt       time.Time       `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	AcceptedAt      *time.Time      `bun:"accepted_at"`

	// Relations
	Listing       *Listing       `bun:"rel:belongs-to,join:listing_id=id"`
	Requester     *Profile       `bun:"rel:belongs-to,join:requester_id=id"`
	DeclineReason *DeclineReason `bun:"rel:belongs-to,join:decline_reason_id=id"`
	Trade         *Trade         `bun:"rel:has-one,join:id=offer_id"`
}

// GetMessage returns the message or empty string
func (o *Offer) GetMessage() string {
	if o.Message != nil {
		return *o.Message
	}
	return ""
}

// GetDeclineNote returns the decline note or empty string
func (o *Offer) GetDeclineNote() string {
	if o.DeclineNote != nil {
		return *o.DeclineNote
	}
	return ""
}

// IsPending returns true if the offer is pending
func (o *Offer) IsPending() bool {
	return o.Status == "pending"
}

// IsAccepted returns true if the offer is accepted
func (o *Offer) IsAccepted() bool {
	return o.Status == "accepted"
}

// IsRejected returns true if the offer is rejected
func (o *Offer) IsRejected() bool {
	return o.Status == "rejected"
}

// IsCancelled returns true if the offer is cancelled
func (o *Offer) IsCancelled() bool {
	return o.Status == "cancelled"
}

// DeclineReason represents a predefined decline reason
type DeclineReason struct {
	bun.BaseModel `bun:"table:d2.decline_reasons,alias:dr"`

	ID      int    `bun:"id,pk,autoincrement"`
	Code    string `bun:"code,notnull,unique"`
	Message string `bun:"message,notnull"`
	Active  bool   `bun:"active,default:true"`
}
