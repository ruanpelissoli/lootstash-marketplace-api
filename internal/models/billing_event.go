package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

// BillingEvent represents a Stripe billing event record
type BillingEvent struct {
	bun.BaseModel `bun:"table:d2.billing_events,alias:be"`

	ID            string          `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID        string          `bun:"user_id,type:uuid,notnull"`
	StripeEventID string          `bun:"stripe_event_id,notnull"`
	EventType     string          `bun:"event_type,notnull"`
	AmountCents   *int            `bun:"amount_cents"`
	Currency      *string         `bun:"currency"`
	InvoiceURL    *string         `bun:"invoice_url"`
	CreatedAt     time.Time       `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	Metadata      json.RawMessage `bun:"metadata,type:jsonb"`
}
