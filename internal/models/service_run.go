package models

import (
	"time"

	"github.com/uptrace/bun"
)

// ServiceRun represents an active service engagement between provider and client
type ServiceRun struct {
	bun.BaseModel `bun:"table:d2.service_runs,alias:sr"`

	ID           string     `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ServiceID    string     `bun:"service_id,type:uuid,notnull"`
	OfferID      string     `bun:"offer_id,type:uuid,notnull"`
	ProviderID   string     `bun:"provider_id,type:uuid,notnull"`
	ClientID     string     `bun:"client_id,type:uuid,notnull"`
	Status       string     `bun:"status,notnull,default:'active'"`
	CancelReason *string    `bun:"cancel_reason"`
	CancelledBy  *string    `bun:"cancelled_by,type:uuid"`
	CreatedAt    time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt    time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	CompletedAt  *time.Time `bun:"completed_at"`
	CancelledAt  *time.Time `bun:"cancelled_at"`

	// Relations
	Service  *Service `bun:"rel:belongs-to,join:service_id=id"`
	Offer    *Offer   `bun:"rel:belongs-to,join:offer_id=id"`
	Provider *Profile `bun:"rel:belongs-to,join:provider_id=id"`
	Client   *Profile `bun:"rel:belongs-to,join:client_id=id"`
	Chat     *Chat    `bun:"rel:has-one,join:id=service_run_id"`
}

// IsActive returns true if the service run is active
func (sr *ServiceRun) IsActive() bool {
	return sr.Status == "active"
}

// IsCompleted returns true if the service run is completed
func (sr *ServiceRun) IsCompleted() bool {
	return sr.Status == "completed"
}

// IsCancelled returns true if the service run is cancelled
func (sr *ServiceRun) IsCancelled() bool {
	return sr.Status == "cancelled"
}

// GetCancelReason returns the cancel reason or empty string
func (sr *ServiceRun) GetCancelReason() string {
	if sr.CancelReason != nil {
		return *sr.CancelReason
	}
	return ""
}

// GetCancelledBy returns the ID of who cancelled, or empty string
func (sr *ServiceRun) GetCancelledBy() string {
	if sr.CancelledBy != nil {
		return *sr.CancelledBy
	}
	return ""
}
