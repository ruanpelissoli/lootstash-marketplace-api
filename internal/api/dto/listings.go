package dto

import (
	"encoding/json"
	"time"
)

// ListingCardResponse represents a listing in card/list view (lightweight)
type ListingCardResponse struct {
	ID             string           `json:"id"`
	SellerID       string           `json:"sellerId"`
	Seller         *ProfileResponse `json:"seller,omitempty"`
	Name           string           `json:"name"`
	ItemType       string           `json:"itemType,omitempty"`
	Rarity         string           `json:"rarity,omitempty"`
	ImageURL       string           `json:"imageUrl,omitempty"`
	Stats          []ItemStat       `json:"stats,omitempty"`
	CatalogItemID  string           `json:"catalogItemId,omitempty"`
	AskingFor      json.RawMessage  `json:"askingFor,omitempty"`
	AskingPrice    string           `json:"askingPrice,omitempty"`
	Amount         int              `json:"amount"`
	Game           string           `json:"game"`
	Ladder         bool             `json:"ladder"`
	Hardcore       bool             `json:"hardcore"`
	IsNonRotw      bool             `json:"isNonRotw"`
	Platforms      []string         `json:"platforms"`
	Region         string           `json:"region"`
	SellerTimezone string           `json:"sellerTimezone,omitempty"`
	Views          int              `json:"views"`
	IsBoosted      bool             `json:"isBoosted"`
	CreatedAt      time.Time        `json:"createdAt"`
}

// ListingResponse represents a listing with full details
type ListingResponse struct {
	ID             string           `json:"id"`
	SellerID       string           `json:"sellerId"`
	Seller         *ProfileResponse `json:"seller,omitempty"`
	Name           string           `json:"name"`
	ItemType       string           `json:"itemType,omitempty"`
	Rarity         string           `json:"rarity,omitempty"`
	ImageURL       string           `json:"imageUrl,omitempty"`
	Category       string           `json:"category,omitempty"`
	Stats          []ItemStat       `json:"stats,omitempty"`
	Suffixes       json.RawMessage  `json:"suffixes,omitempty"`
	Runes          []RuneInfo       `json:"runes,omitempty"`
	RuneOrder      string           `json:"runeOrder,omitempty"`
	BaseItemCode   string           `json:"baseItemCode,omitempty"`
	BaseItemName   string           `json:"baseItemName,omitempty"`
	CatalogItemID  string           `json:"catalogItemId,omitempty"`
	AskingFor      json.RawMessage  `json:"askingFor,omitempty"`
	AskingPrice    string           `json:"askingPrice,omitempty"`
	Amount         int              `json:"amount"`
	Notes          string           `json:"notes,omitempty"`
	Game           string           `json:"game"`
	Ladder         bool             `json:"ladder"`
	Hardcore       bool             `json:"hardcore"`
	IsNonRotw      bool             `json:"isNonRotw"`
	Platforms      []string         `json:"platforms"`
	Region         string           `json:"region"`
	SellerTimezone string           `json:"sellerTimezone,omitempty"`
	Status         string           `json:"status"`
	Views          int              `json:"views"`
	IsBoosted      bool             `json:"isBoosted"`
	CreatedAt      time.Time        `json:"createdAt"`
	ExpiresAt      time.Time        `json:"expiresAt,omitempty"`
}

// ListingDetailResponse represents a listing with full details
type ListingDetailResponse struct {
	ListingResponse
	UpdatedAt  time.Time `json:"updatedAt"`
	TradeCount int       `json:"tradeCount"`
}

// CreateListingRequest represents a request to create a listing
type CreateListingRequest struct {
	Name          string          `json:"name" validate:"required,min=1,max=100"`
	ItemType      string          `json:"itemType" validate:"required,min=1,max=50"`
	Rarity        string          `json:"rarity" validate:"required,oneof=normal superior magic rare unique set runeword"`
	ImageURL      string          `json:"imageUrl,omitempty" validate:"omitempty,url"`
	Category      string          `json:"category" validate:"omitempty,max=50"`
	Stats         json.RawMessage `json:"stats,omitempty"`
	Suffixes      json.RawMessage `json:"suffixes,omitempty"`
	Runes         json.RawMessage `json:"runes,omitempty"`
	RuneOrder     string          `json:"runeOrder,omitempty"`
	BaseItemCode  string          `json:"baseItemCode,omitempty"`
	BaseItemName  string          `json:"baseItemName,omitempty"`
	CatalogItemID string          `json:"catalogItemId,omitempty"`
	AskingFor     json.RawMessage `json:"askingFor,omitempty"`
	AskingPrice   string          `json:"askingPrice,omitempty" validate:"omitempty,max=100"`
	Amount        *int            `json:"amount,omitempty" validate:"omitempty,min=1"`
	Notes         string          `json:"notes,omitempty" validate:"omitempty,max=500"`
	Game          string          `json:"game" validate:"required,min=1,max=20"`
	Ladder        bool            `json:"ladder"`
	Hardcore      bool            `json:"hardcore"`
	IsNonRotw     bool            `json:"isNonRotw"`
	Platforms     []string        `json:"platforms" validate:"required,min=1,dive,oneof=pc xbox playstation switch"`
	Region        string          `json:"region" validate:"required,oneof=americas europe asia"`
}

// UpdateListingRequest represents a request to update a listing
type UpdateListingRequest struct {
	AskingFor   json.RawMessage `json:"askingFor,omitempty"`
	AskingPrice *string         `json:"askingPrice,omitempty" validate:"omitempty,max=100"`
	Notes       *string         `json:"notes,omitempty" validate:"omitempty,max=500"`
	Status      *string         `json:"status,omitempty" validate:"omitempty,oneof=active cancelled"`
}

// RefreshListingRequest represents a request to refresh (bump) a listing
type RefreshListingRequest struct {
	AskingFor json.RawMessage `json:"askingFor,omitempty"`
}

// ListingFilterRequest represents listing filter parameters
type ListingFilterRequest struct {
	SellerID         string `query:"sellerId"`
	Q                string `query:"q"`
	CatalogItemID    string `query:"catalogItemId"`
	Game             string `query:"game"`
	Ladder           *bool  `query:"ladder"`
	Hardcore         *bool  `query:"hardcore"`
	IsNonRotw        *bool  `query:"isNonRotw"`
	Platforms        string `query:"platforms"`
	Region           string `query:"region"`
	Categories       string `query:"categories"`
	Rarity           string `query:"rarity"`
	AffixFilters     string `query:"affixFilters"`
	AskingForFilters string `query:"askingForFilters"`
	SortBy           string `query:"sortBy"`
	SortOrder        string `query:"sortOrder"`
	Pagination
}

// AffixFilter represents a filter for item affixes
type AffixFilter struct {
	Code     string `json:"code"`
	MinValue *int   `json:"minValue,omitempty"`
	MaxValue *int   `json:"maxValue,omitempty"`
}

// AskingForFilter represents a filter for what sellers are asking for
type AskingForFilter struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Type        string `json:"type,omitempty" validate:"omitempty,max=50"`
	MinQuantity *int   `json:"minQuantity,omitempty" validate:"omitempty,min=1"`
	MaxQuantity *int   `json:"maxQuantity,omitempty" validate:"omitempty,min=1"`
}

// ItemStat represents a stat on a listing item with display name
type ItemStat struct {
	Code        string `json:"code"`
	Value       *int   `json:"value,omitempty"`
	Min         *int   `json:"min,omitempty"`
	Max         *int   `json:"max,omitempty"`
	Param       string `json:"param,omitempty"`
	DisplayText string `json:"displayText"`
	IsVariable  bool   `json:"isVariable,omitempty"`
}

// RuneInfo represents a rune with its display information
type RuneInfo struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl"`
}

// SearchListingsRequest represents listing search/filter parameters via JSON body
type SearchListingsRequest struct {
	Q               string           `json:"q"`
	CatalogItemID   string           `json:"catalogItemId"`
	Game            string           `json:"game"`
	Ladder          *bool            `json:"ladder"`
	Hardcore        *bool            `json:"hardcore"`
	IsNonRotw       *bool            `json:"nonRotw"`
	Platforms       []string         `json:"platforms"`
	Region          string           `json:"region"`
	Categories      []string         `json:"categories"`
	Rarity          string           `json:"rarity"`
	SellerID        string           `json:"sellerId"`
	AffixFilters    []AffixFilter    `json:"affixFilters"`
	AskingForFilter *AskingForFilter `json:"askingForFilter"`
	SortBy          string           `json:"sortBy"`
	SortOrder       string           `json:"sortOrder"`
	Page            int              `json:"page"`
	PerPage         int              `json:"perPage"`
}

// MyListingsFilterRequest represents filter parameters for user's own listings
type MyListingsFilterRequest struct {
	Status string `query:"status"`
	Pagination
}

// MarketplaceStatsResponse represents marketplace statistics
type MarketplaceStatsResponse struct {
	ActiveListings         int       `json:"activeListings"`
	TradesToday            int       `json:"tradesToday"`
	OnlineSellers          int       `json:"onlineSellers"`
	AvgResponseTimeMinutes float64   `json:"avgResponseTimeMinutes"`
	LastUpdated            time.Time `json:"lastUpdated"`
}
