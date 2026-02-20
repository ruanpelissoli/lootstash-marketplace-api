package dto

import "time"

// CreateWishlistItemRequest represents a request to create a wishlist item
type CreateWishlistItemRequest struct {
	Name         string              `json:"name" validate:"required,min=1,max=100"`
	Category     *string             `json:"category,omitempty" validate:"omitempty,max=50"`
	Rarity       *string             `json:"rarity,omitempty" validate:"omitempty,max=50"`
	ImageURL     *string             `json:"imageUrl,omitempty" validate:"omitempty,url,max=500"`
	StatCriteria []StatCriterionDTO  `json:"statCriteria,omitempty"`
	Game         string              `json:"game" validate:"required,min=1,max=20"`
	Ladder       *bool               `json:"ladder,omitempty"`
	Hardcore     *bool               `json:"hardcore,omitempty"`
	IsNonRotw    *bool               `json:"isNonRotw,omitempty"`
	Platforms    []string            `json:"platforms,omitempty" validate:"omitempty,dive,oneof=pc xbox playstation switch"`
}

// UpdateWishlistItemRequest represents a request to update a wishlist item
type UpdateWishlistItemRequest struct {
	Name         *string             `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Category     *string             `json:"category,omitempty" validate:"omitempty,max=50"`
	Rarity       *string             `json:"rarity,omitempty" validate:"omitempty,max=50"`
	ImageURL     *string             `json:"imageUrl,omitempty" validate:"omitempty,url,max=500"`
	StatCriteria []StatCriterionDTO  `json:"statCriteria,omitempty"`
	Game         *string             `json:"game,omitempty" validate:"omitempty,min=1,max=20"`
	Ladder       *bool               `json:"ladder,omitempty"`
	Hardcore     *bool               `json:"hardcore,omitempty"`
	IsNonRotw    *bool               `json:"isNonRotw,omitempty"`
	Platforms    []string            `json:"platforms,omitempty" validate:"omitempty,dive,oneof=pc xbox playstation switch"`
	Status       *string             `json:"status,omitempty" validate:"omitempty,oneof=active paused"`
}

// StatCriterionDTO represents a stat filter criterion in API requests/responses
type StatCriterionDTO struct {
	Code     string `json:"code" validate:"required"`
	Name     string `json:"name,omitempty"`
	MinValue *int   `json:"minValue,omitempty"`
	MaxValue *int   `json:"maxValue,omitempty"`
}

// WishlistItemResponse represents a wishlist item in API responses
type WishlistItemResponse struct {
	ID           string             `json:"id"`
	UserID       string             `json:"userId"`
	Name         string             `json:"name"`
	Category     *string            `json:"category,omitempty"`
	Rarity       *string            `json:"rarity,omitempty"`
	ImageURL     *string            `json:"imageUrl,omitempty"`
	StatCriteria []StatCriterionDTO `json:"statCriteria,omitempty"`
	Game         string             `json:"game"`
	Ladder       *bool              `json:"ladder,omitempty"`
	Hardcore     *bool              `json:"hardcore,omitempty"`
	IsNonRotw    *bool              `json:"isNonRotw,omitempty"`
	Platforms    []string           `json:"platforms,omitempty"`
	Status       string             `json:"status"`
	CreatedAt    time.Time          `json:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt"`
}

// WishlistFilterRequest represents wishlist filter parameters
type WishlistFilterRequest struct {
	Pagination
}
