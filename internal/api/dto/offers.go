package dto

import (
	"encoding/json"
	"time"
)

// OfferResponse represents an offer
type OfferResponse struct {
	ID            string                 `json:"id"`
	ListingID     string                 `json:"listingId"`
	Listing       *ListingResponse       `json:"listing,omitempty"`
	RequesterID   string                 `json:"requesterId"`
	Requester     *ProfileResponse       `json:"requester,omitempty"`
	OfferedItems  json.RawMessage        `json:"offeredItems"`
	Message       string                 `json:"message,omitempty"`
	Status        string                 `json:"status"`
	DeclineReason *DeclineReasonResponse `json:"declineReason,omitempty"`
	DeclineNote   string                 `json:"declineNote,omitempty"`
	TradeID       *string                `json:"tradeId,omitempty"`
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
	ListingID    string          `json:"listingId" validate:"required,uuid"`
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
	ListingID string `query:"listingId"` // Filter by listing ID (for sellers to see offers on their listing)
	Pagination
}

// AcceptOfferResponse represents the response when accepting an offer
type AcceptOfferResponse struct {
	Offer   *OfferResponse `json:"offer"`
	TradeID string         `json:"tradeId"`
	ChatID  string         `json:"chatId"`
}
