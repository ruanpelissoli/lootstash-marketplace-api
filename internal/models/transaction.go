package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

// Transaction represents a completed trade transaction
type Transaction struct {
	bun.BaseModel `bun:"table:d2.transactions,alias:tx"`

	ID           string          `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	TradeID      *string         `bun:"trade_id,type:uuid"`
	ListingID    *string         `bun:"listing_id,type:uuid"`
	SellerID     string          `bun:"seller_id,type:uuid,notnull"`
	BuyerID      string          `bun:"buyer_id,type:uuid,notnull"`
	ItemName     string          `bun:"item_name,notnull"`
	ItemDetails  json.RawMessage `bun:"item_details,type:jsonb"`
	OfferedItems json.RawMessage `bun:"offered_items,type:jsonb"`
	CreatedAt    time.Time       `bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Trade   *Trade   `bun:"rel:belongs-to,join:trade_id=id"`
	Listing *Listing `bun:"rel:belongs-to,join:listing_id=id"`
	Seller  *Profile `bun:"rel:belongs-to,join:seller_id=id"`
	Buyer   *Profile `bun:"rel:belongs-to,join:buyer_id=id"`
}

// GetTradeID returns the trade ID or empty string
func (t *Transaction) GetTradeID() string {
	if t.TradeID != nil {
		return *t.TradeID
	}
	return ""
}

// GetListingID returns the listing ID or empty string
func (t *Transaction) GetListingID() string {
	if t.ListingID != nil {
		return *t.ListingID
	}
	return ""
}
