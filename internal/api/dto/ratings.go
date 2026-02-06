package dto

import "time"

// RatingResponse represents a user rating
type RatingResponse struct {
	ID            string           `json:"id"`
	TransactionID string           `json:"transactionId"`
	RaterID       string           `json:"raterId"`
	Rater         *ProfileResponse `json:"rater,omitempty"`
	RatedID       string           `json:"ratedId"`
	Stars         int              `json:"stars"`
	Comment       string           `json:"comment,omitempty"`
	CreatedAt     time.Time        `json:"createdAt"`
}

// CreateRatingRequest represents a request to rate a trade
type CreateRatingRequest struct {
	TransactionID string `json:"transactionId" validate:"required,uuid"`
	Stars         int    `json:"stars" validate:"required,min=1,max=5"`
	Comment       string `json:"comment,omitempty" validate:"omitempty,max=500"`
}

// RatingsFilterRequest represents filter parameters for ratings
type RatingsFilterRequest struct {
	Pagination
}

// TransactionResponse represents a completed transaction
type TransactionResponse struct {
	ID             string           `json:"id"`
	TradeRequestID string           `json:"tradeRequestId,omitempty"`
	ListingID      string           `json:"listingId,omitempty"`
	SellerID       string           `json:"sellerId"`
	Seller         *ProfileResponse `json:"seller,omitempty"`
	BuyerID        string           `json:"buyerId"`
	Buyer          *ProfileResponse `json:"buyer,omitempty"`
	ItemName       string           `json:"itemName"`
	CreatedAt      time.Time        `json:"createdAt"`
	MyRating       *RatingResponse  `json:"myRating,omitempty"`
	TheirRating    *RatingResponse  `json:"theirRating,omitempty"`
}
