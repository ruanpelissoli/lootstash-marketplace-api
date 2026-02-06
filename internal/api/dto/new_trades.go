package dto

import (
	"time"
)

// OfferedItemResponse represents an item offered in a trade
type OfferedItemResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	ImageURL string `json:"imageUrl,omitempty"`
	Quantity int    `json:"quantity"`
}

// TradeResponse represents an active trade
type TradeResponse struct {
	ID            string                `json:"id"`
	OfferID       string                `json:"offerId"`
	ListingID     string                `json:"listingId"`
	Listing       *ListingResponse      `json:"listing,omitempty"`
	SellerID      string                `json:"sellerId"`
	Seller        *ProfileResponse      `json:"seller,omitempty"`
	BuyerID       string                `json:"buyerId"`
	Buyer         *ProfileResponse      `json:"buyer,omitempty"`
	OfferedItems  []OfferedItemResponse `json:"offeredItems,omitempty"`
	Status        string                `json:"status"`
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

// TradeDetailResponse includes additional details for a single trade
type TradeDetailResponse struct {
	TradeResponse
	CanComplete bool `json:"canComplete"`
	CanCancel   bool `json:"canCancel"`
	CanMessage  bool `json:"canMessage"`
}

// TradesFilterRequest represents filter parameters for trades
type TradesFilterRequest struct {
	Status string `query:"status"` // active, completed, cancelled
	Pagination
}

// CancelTradeRequest represents a request to cancel a trade
type CancelTradeRequest struct {
	Reason string `json:"reason,omitempty" validate:"omitempty,max=500"`
}

// CompleteTradeResponse represents the response when completing a trade
type CompleteTradeResponse struct {
	Trade         *TradeResponse `json:"trade"`
	TransactionID string         `json:"transactionId"`
}
