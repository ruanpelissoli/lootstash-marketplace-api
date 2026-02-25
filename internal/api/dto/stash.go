package dto

import (
	"encoding/json"
	"time"
)

// CreateStashItemRequest represents a request to add an item to the stash
type CreateStashItemRequest struct {
	Name          string          `json:"name,omitempty" validate:"omitempty,max=100"`
	ItemType      string          `json:"itemType" validate:"required,min=1,max=50"`
	Rarity        string          `json:"rarity" validate:"required,oneof=normal superior magic rare unique set runeword crafted"`
	ImageURL      string          `json:"imageUrl,omitempty" validate:"omitempty,url"`
	Category      string          `json:"category" validate:"omitempty,max=50"`
	Stats         json.RawMessage `json:"stats,omitempty"`
	Suffixes      json.RawMessage `json:"suffixes,omitempty"`
	Runes         json.RawMessage `json:"runes,omitempty"`
	RuneOrder     string          `json:"runeOrder,omitempty"`
	BaseItemCode  string          `json:"baseItemCode,omitempty"`
	BaseItemName  string          `json:"baseItemName,omitempty"`
	CatalogItemID string          `json:"catalogItemId,omitempty"`
	Quantity      *int            `json:"quantity,omitempty" validate:"omitempty,min=1"`
	Game          string          `json:"game" validate:"required,min=1,max=20"`
	Ladder        bool            `json:"ladder"`
	Hardcore      bool            `json:"hardcore"`
	Platforms     []string        `json:"platforms" validate:"required,min=1,dive,oneof=pc xbox playstation switch"`
	Region        string          `json:"region" validate:"required,oneof=americas europe asia"`
}

// AdjustQuantityRequest represents a request to adjust stash item quantity
type AdjustQuantityRequest struct {
	Quantity int `json:"quantity" validate:"required,ne=0"`
}

// StashFilterRequest represents stash filter query parameters (GET)
type StashFilterRequest struct {
	Q             string `query:"q"`
	Categories    string `query:"categories"`
	Rarities      string `query:"rarities"`
	Listed        *bool  `query:"listed"`
	CatalogItemID string `query:"catalogItemId"`
	AffixFilters  string `query:"affixFilters"`
	SortBy        string `query:"sortBy"`
	SortOrder     string `query:"sortOrder"`
	Pagination
}

// SearchStashRequest represents stash search/filter parameters via JSON body
type SearchStashRequest struct {
	Q             string        `json:"q"`
	Categories    []string      `json:"categories"`
	Rarities      []string      `json:"rarities"`
	Listed        *bool         `json:"listed"`
	CatalogItemID string        `json:"catalogItemId"`
	AffixFilters  []AffixFilter `json:"affixFilters"`
	SortBy        string        `json:"sortBy"`
	SortOrder     string        `json:"sortOrder"`
	Page          int           `json:"page"`
	PerPage       int           `json:"perPage"`
}

// CreateListingFromStashRequest represents a request to create a listing from a stash item
type CreateListingFromStashRequest struct {
	StashItemID string          `json:"stashItemId" validate:"required,uuid"`
	Amount      *int            `json:"amount,omitempty" validate:"omitempty,min=1"`
	AskingFor   json.RawMessage `json:"askingFor,omitempty"`
	AskingPrice string          `json:"askingPrice,omitempty" validate:"omitempty,max=100"`
	Notes       string          `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// StashItemResponse represents a stash item in API responses
type StashItemResponse struct {
	ID                string          `json:"id"`
	UserID            string          `json:"userId"`
	Name              string          `json:"name,omitempty"`
	ItemType          string          `json:"itemType"`
	DisplayName       string          `json:"displayName"`
	Rarity            string          `json:"rarity,omitempty"`
	ImageURL          string          `json:"imageUrl,omitempty"`
	Category          string          `json:"category,omitempty"`
	Stats             []ItemStat      `json:"stats,omitempty"`
	Suffixes          json.RawMessage `json:"suffixes,omitempty"`
	Runes             []RuneInfo      `json:"runes,omitempty"`
	RuneOrder         string          `json:"runeOrder,omitempty"`
	BaseItemCode      string          `json:"baseItemCode,omitempty"`
	BaseItemName      string          `json:"baseItemName,omitempty"`
	CatalogItemID     string          `json:"catalogItemId,omitempty"`
	Quantity    int             `json:"quantity"`
	IsListed    bool            `json:"isListed"`
	IsStackable bool            `json:"isStackable"`
	AskingPrice string          `json:"askingPrice,omitempty"`
	AskingFor   json.RawMessage `json:"askingFor,omitempty"`
	Game              string          `json:"game"`
	Ladder            bool            `json:"ladder"`
	Hardcore          bool            `json:"hardcore"`
	Platforms         []string        `json:"platforms"`
	Region            string          `json:"region"`
	Source            string          `json:"source"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

// BulkImportPreviewResponse represents the result of parsing screenshots
type BulkImportPreviewResponse struct {
	Items  []ParsedStashItem `json:"items"`
	Errors []ImportError     `json:"errors,omitempty"`
}

// ParsedStashItem represents a parsed item from Vision API / catalog lookup
type ParsedStashItem struct {
	Name          string          `json:"name,omitempty"`
	ItemType      string          `json:"itemType"`
	Rarity        string          `json:"rarity"`
	Category      string          `json:"category"`
	Stats         json.RawMessage `json:"stats,omitempty"`
	Suffixes      json.RawMessage `json:"suffixes,omitempty"`
	Runes         json.RawMessage `json:"runes,omitempty"`
	RuneOrder     string          `json:"runeOrder,omitempty"`
	BaseItemCode  string          `json:"baseItemCode,omitempty"`
	BaseItemName  string          `json:"baseItemName,omitempty"`
	CatalogItemID string          `json:"catalogItemId,omitempty"`
	CatalogType   string          `json:"catalogType,omitempty"`
	ImageURL      string          `json:"imageUrl,omitempty"`
	Quantity      int             `json:"quantity"`
	IsStackable   bool            `json:"isStackable"`
}

// BulkImportConfirmRequest represents a request to save reviewed/corrected items
type BulkImportConfirmRequest struct {
	Items []CreateStashItemRequest `json:"items" validate:"required,min=1,dive"`
}

// ImportError represents an error during import for a specific file
type ImportError struct {
	Index   int    `json:"index"`
	File    string `json:"file"`
	Message string `json:"message"`
}
