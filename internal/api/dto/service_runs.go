package dto

import (
	"encoding/json"
	"time"
)

// ServiceRunResponse represents a service run
type ServiceRunResponse struct {
	ID            string           `json:"id"`
	ServiceID     string           `json:"serviceId"`
	Service       *ServiceResponse `json:"service,omitempty"`
	OfferID       string           `json:"offerId"`
	ProviderID    string           `json:"providerId"`
	Provider      *ProfileResponse `json:"provider,omitempty"`
	ClientID      string           `json:"clientId"`
	Client        *ProfileResponse `json:"client,omitempty"`
	OfferedItems  json.RawMessage  `json:"offeredItems,omitempty"`
	Status        string           `json:"status"`
	CancelReason  string           `json:"cancelReason,omitempty"`
	CancelledBy   string           `json:"cancelledBy,omitempty"`
	ChatID        *string          `json:"chatId,omitempty"`
	TransactionID *string          `json:"transactionId,omitempty"`
	CanRate       bool             `json:"canRate"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
	CompletedAt   *time.Time       `json:"completedAt,omitempty"`
	CancelledAt   *time.Time       `json:"cancelledAt,omitempty"`
}

// ServiceRunDetailResponse includes additional details
type ServiceRunDetailResponse struct {
	ServiceRunResponse
	CanComplete bool `json:"canComplete"`
	CanCancel   bool `json:"canCancel"`
	CanMessage  bool `json:"canMessage"`
}

// ServiceRunsFilterRequest represents filter parameters for service runs
type ServiceRunsFilterRequest struct {
	Status string `query:"status"` // active, completed, cancelled
	Role   string `query:"role"`   // provider, client, all
	Pagination
}

// CancelServiceRunRequest represents a request to cancel a service run
type CancelServiceRunRequest struct {
	Reason string `json:"reason,omitempty" validate:"omitempty,max=500"`
}
