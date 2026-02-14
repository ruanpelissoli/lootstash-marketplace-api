package dto

// UpdateFlairRequest represents a request to update profile flair
type UpdateFlairRequest struct {
	Flair string `json:"flair" validate:"required,oneof=none gold flame ice necro royal"`
}

// UpdateUsernameColorRequest represents a request to update username color
type UpdateUsernameColorRequest struct {
	Color string `json:"color" validate:"required"`
}

// ListingCountResponse contains the count of active listings
type ListingCountResponse struct {
	Count int `json:"count"`
}

// PriceHistoryTrade represents a single trade's offered items
type PriceHistoryTrade struct {
	OfferedItems []OfferedItemResponse `json:"offeredItems"`
}

// PriceHistoryDay represents all trades for a single date
type PriceHistoryDay struct {
	Date   string              `json:"date"`
	Trades []PriceHistoryTrade `json:"trades"`
}

// PriceHistoryResponse contains price history data grouped by date
type PriceHistoryResponse struct {
	Data []PriceHistoryDay `json:"data"`
}
