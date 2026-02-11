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

// TradeVolumePoint represents a single data point of trade volume
type TradeVolumePoint struct {
	Date   string `json:"date"`
	Volume int    `json:"volume"`
}

// TradeVolumeResponse contains trade volume history data
type TradeVolumeResponse struct {
	Data []TradeVolumePoint `json:"data"`
}
