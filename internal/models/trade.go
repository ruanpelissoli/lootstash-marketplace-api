package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Trade represents an active negotiation after an offer is accepted
type Trade struct {
	bun.BaseModel `bun:"table:d2.trades,alias:t"`

	ID           string     `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	OfferID      string     `bun:"offer_id,type:uuid,notnull"`
	ListingID    string     `bun:"listing_id,type:uuid,notnull"`
	SellerID     string     `bun:"seller_id,type:uuid,notnull"`
	BuyerID      string     `bun:"buyer_id,type:uuid,notnull"`
	Status       string     `bun:"status,notnull,default:'active'"`
	CancelReason *string    `bun:"cancel_reason"`
	CancelledBy  *string    `bun:"cancelled_by,type:uuid"`
	CreatedAt    time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt    time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	CompletedAt  *time.Time `bun:"completed_at"`
	CancelledAt  *time.Time `bun:"cancelled_at"`

	// Relations
	Offer   *Offer   `bun:"rel:belongs-to,join:offer_id=id"`
	Listing *Listing `bun:"rel:belongs-to,join:listing_id=id"`
	Seller  *Profile `bun:"rel:belongs-to,join:seller_id=id"`
	Buyer   *Profile `bun:"rel:belongs-to,join:buyer_id=id"`
	Chat    *Chat    `bun:"rel:has-one,join:id=trade_id"`
}

// IsActive returns true if the trade is active
func (t *Trade) IsActive() bool {
	return t.Status == "active"
}

// IsCompleted returns true if the trade is completed
func (t *Trade) IsCompleted() bool {
	return t.Status == "completed"
}

// IsCancelled returns true if the trade is cancelled
func (t *Trade) IsCancelled() bool {
	return t.Status == "cancelled"
}

// GetCancelReason returns the cancel reason or empty string
func (t *Trade) GetCancelReason() string {
	if t.CancelReason != nil {
		return *t.CancelReason
	}
	return ""
}

// GetCancelledBy returns the ID of who cancelled, or empty string
func (t *Trade) GetCancelledBy() string {
	if t.CancelledBy != nil {
		return *t.CancelledBy
	}
	return ""
}
