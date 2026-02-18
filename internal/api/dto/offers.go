package dto

import (
	"encoding/json"
	"time"
)

// OfferResponse represents an offer
type OfferResponse struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	ListingID     string                 `json:"listingId,omitempty"`
	Listing       *ListingResponse       `json:"listing,omitempty"`
	ServiceID     string                 `json:"serviceId,omitempty"`
	Service       *ServiceResponse       `json:"service,omitempty"`
	RequesterID   string                 `json:"requesterId"`
	Requester     *ProfileResponse       `json:"requester,omitempty"`
	OfferedItems  json.RawMessage        `json:"offeredItems"`
	Message       string                 `json:"message,omitempty"`
	Status        string                 `json:"status"`
	DeclineReason *DeclineReasonResponse `json:"declineReason,omitempty"`
	DeclineNote   string                 `json:"declineNote,omitempty"`
	TradeID       *string                `json:"tradeId,omitempty"`
	ServiceRunID  *string                `json:"serviceRunId,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
	AcceptedAt    *time.Time             `json:"acceptedAt,omitempty"`
}

// OfferDetailResponse includes additional details for a single offer
type OfferDetailResponse struct {
	OfferResponse
}

// CreateOfferRequest represents a request to create an offer
type CreateOfferRequest struct {
	Type         string          `json:"type" validate:"required,oneof=item service"`
	ListingID    *string         `json:"listingId,omitempty" validate:"omitempty,uuid"`
	ServiceID    *string         `json:"serviceId,omitempty" validate:"omitempty,uuid"`
	OfferedItems json.RawMessage `json:"offeredItems" validate:"required"`
	Message      string          `json:"message,omitempty" validate:"omitempty,max=500"`
}

// RejectOfferRequest represents a request to reject an offer
type RejectOfferRequest struct {
	DeclineReasonID int    `json:"declineReasonId" validate:"required,min=1"`
	DeclineNote     string `json:"declineNote,omitempty" validate:"omitempty,max=200"`
}

// DeclineReasonResponse represents a decline reason
type DeclineReasonResponse struct {
	ID      int    `json:"id"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// OffersFilterRequest represents filter parameters for offers
type OffersFilterRequest struct {
	Status    string `query:"status"`    // pending, accepted, rejected, cancelled
	Role      string `query:"role"`      // buyer, seller, all
	Type      string `query:"type"`      // item, service, all
	ListingID string `query:"listingId"` // Filter by listing ID
	ServiceID string `query:"serviceId"` // Filter by service ID
	Pagination
}

// AcceptOfferResponse represents the response when accepting an offer
type AcceptOfferResponse struct {
	Offer        *OfferResponse `json:"offer"`
	TradeID      string         `json:"tradeId,omitempty"`
	ServiceRunID string         `json:"serviceRunId,omitempty"`
	ChatID       string         `json:"chatId"`
}
