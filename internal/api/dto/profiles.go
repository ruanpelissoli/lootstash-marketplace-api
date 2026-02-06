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
	ProfileFlair  string    `json:"profileFlair,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

// MyProfileResponse represents the current user's profile with additional fields
type MyProfileResponse struct {
	ProfileResponse
	BattleNetLinked   bool       `json:"battleNetLinked"`
	BattleNetLinkedAt *time.Time `json:"battleNetLinkedAt,omitempty"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	DisplayName *string `json:"displayName" validate:"omitempty,min=1,max=50"`
	AvatarURL   *string `json:"avatarUrl" validate:"omitempty,url"`
}

// UploadPictureResponse represents the response after uploading a profile picture
type UploadPictureResponse struct {
	AvatarURL string `json:"avatarUrl"`
}
