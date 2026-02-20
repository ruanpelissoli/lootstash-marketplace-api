package dto

import "time"

// ProfileResponse represents a public user profile
type ProfileResponse struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	DisplayName   string    `json:"displayName,omitempty"`
	AvatarURL     string    `json:"avatarUrl,omitempty"`
	BattleTag     string    `json:"battleTag,omitempty"`
	TotalTrades   int       `json:"totalTrades"`
	AverageRating float64   `json:"averageRating"`
	RatingCount   int       `json:"ratingCount"`
	IsPremium     bool      `json:"isPremium"`
	IsAdmin       bool      `json:"isAdmin"`
	ProfileFlair  string    `json:"profileFlair,omitempty"`
	UsernameColor string    `json:"usernameColor,omitempty"`
	Timezone      string    `json:"timezone,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

// MyProfileResponse represents the current user's profile with additional fields
type MyProfileResponse struct {
	ProfileResponse
	BattleNetLinked    bool       `json:"battleNetLinked"`
	BattleNetLinkedAt  *time.Time `json:"battleNetLinkedAt,omitempty"`
	PreferredLadder    *bool      `json:"preferredLadder,omitempty"`
	PreferredHardcore  *bool      `json:"preferredHardcore,omitempty"`
	PreferredPlatforms []string   `json:"preferredPlatforms,omitempty"`
	PreferredRegion    string     `json:"preferredRegion,omitempty"`
	PreferredNonRotw   *bool      `json:"preferredNonRotw,omitempty"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	DisplayName        *string  `json:"displayName" validate:"omitempty,min=1,max=50"`
	AvatarURL          *string  `json:"avatarUrl" validate:"omitempty,url"`
	Timezone           *string  `json:"timezone" validate:"omitempty,max=100"`
	PreferredLadder    *bool    `json:"preferredLadder"`
	PreferredHardcore  *bool    `json:"preferredHardcore"`
	PreferredPlatforms []string `json:"preferredPlatforms" validate:"omitempty,dive,oneof=pc xbox playstation switch"`
	PreferredRegion    *string  `json:"preferredRegion" validate:"omitempty,oneof=americas europe asia"`
	PreferredNonRotw   *bool    `json:"preferredNonRotw"`
}

// UploadPictureResponse represents the response after uploading a profile picture
type UploadPictureResponse struct {
	AvatarURL string `json:"avatarUrl"`
}

// SoldItemStat represents a stat on a sold item
type SoldItemStat struct {
	Code        string `json:"code"`
	Value       *int   `json:"value,omitempty"`
	DisplayText string `json:"displayText,omitempty"`
	IsVariable  bool   `json:"isVariable,omitempty"`
}

// SoldItemInfo represents the item info in a sale
type SoldItemInfo struct {
	Name     string         `json:"name"`
	BaseName string         `json:"baseName,omitempty"`
	ImageURL string         `json:"imageUrl,omitempty"`
	ItemType string         `json:"itemType"`
	Rarity   string         `json:"rarity"`
	Stats    []SoldItemStat `json:"stats,omitempty"`
}

// SoldForItem represents what the item sold for
type SoldForItem struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	ImageURL string `json:"imageUrl,omitempty"`
}

// SaleBuyerInfo represents the buyer info in a sale
type SaleBuyerInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
}

// SaleReview represents the review left by the buyer
type SaleReview struct {
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// SoldItem represents a completed sale
type SoldItem struct {
	ID          string        `json:"id"`
	CompletedAt time.Time     `json:"completedAt"`
	Item        SoldItemInfo  `json:"item"`
	SoldFor     []SoldForItem `json:"soldFor"`
	Buyer       SaleBuyerInfo `json:"buyer"`
	Review      *SaleReview   `json:"review,omitempty"`
}

// SalesResponse represents the response for the sales endpoint
type SalesResponse struct {
	Sales   []SoldItem `json:"sales"`
	Total   int        `json:"total"`
	HasMore bool       `json:"hasMore"`
}

// SalesFilterRequest represents filter parameters for sales
type SalesFilterRequest struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

// GetLimit returns the limit with defaults
func (r *SalesFilterRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	if r.Limit > 100 {
		return 100
	}
	return r.Limit
}

// GetOffset returns the offset
func (r *SalesFilterRequest) GetOffset() int {
	if r.Offset < 0 {
		return 0
	}
	return r.Offset
}
