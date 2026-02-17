package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

// Listing represents an item or service listing for trade
type Listing struct {
	bun.BaseModel `bun:"table:d2.listings,alias:l"`

	ID          string          `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	SellerID    string          `bun:"seller_id,type:uuid,notnull"`
	ListingType string          `bun:"listing_type,notnull,default:'item'"`
	Name        string          `bun:"name,notnull"`
	ItemType    string          `bun:"item_type"`
	Rarity      string          `bun:"rarity,nullzero"`
	ImageURL    *string         `bun:"image_url"`
	Category    string          `bun:"category"`
	ServiceType *string         `bun:"service_type"`
	Description *string         `bun:"description"`
	Stats        json.RawMessage `bun:"stats,type:jsonb,default:'[]'"`
	Suffixes     json.RawMessage `bun:"suffixes,type:jsonb,default:'[]'"`
	Runes        json.RawMessage `bun:"runes,type:jsonb,default:'[]'"`
	RuneOrder    *string         `bun:"rune_order"`
	BaseItemCode  *string         `bun:"base_item_code"`
	BaseItemName  *string         `bun:"base_item_name"`
	CatalogItemID *string         `bun:"catalog_item_id"`
	AskingFor    json.RawMessage `bun:"asking_for,type:jsonb,default:'[]'"`
	AskingPrice *string         `bun:"asking_price"`
	Notes       *string         `bun:"notes"`
	Game        string          `bun:"game,notnull,default:'diablo2'"`
	Ladder      bool            `bun:"ladder"`
	Hardcore    bool            `bun:"hardcore,default:false"`
	IsNonRotw   bool            `bun:"is_non_rotw,default:false"`
	Platforms   []string        `bun:"platforms,array,default:'{pc}'"`
	Region         string          `bun:"region,default:'americas'"`
	SellerTimezone *string         `bun:"seller_timezone"`
	Status         string          `bun:"status,notnull,default:'active'"`
	Views       int             `bun:"views,default:0"`
	CreatedAt   time.Time       `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time       `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	ExpiresAt   time.Time       `bun:"expires_at,nullzero"`

	// Relations
	Seller *Profile `bun:"rel:belongs-to,join:seller_id=id"`
}

// IsActive returns true if the listing is active
func (l *Listing) IsActive() bool {
	return l.Status == "active"
}

// GetImageURL returns the image URL or empty string
func (l *Listing) GetImageURL() string {
	if l.ImageURL != nil {
		return *l.ImageURL
	}
	return ""
}

// GetAskingPrice returns the asking price or empty string
func (l *Listing) GetAskingPrice() string {
	if l.AskingPrice != nil {
		return *l.AskingPrice
	}
	return ""
}

// GetNotes returns the notes or empty string
func (l *Listing) GetNotes() string {
	if l.Notes != nil {
		return *l.Notes
	}
	return ""
}

// GetRuneOrder returns the rune order or empty string
func (l *Listing) GetRuneOrder() string {
	if l.RuneOrder != nil {
		return *l.RuneOrder
	}
	return ""
}

// GetBaseItemCode returns the base item code or empty string
func (l *Listing) GetBaseItemCode() string {
	if l.BaseItemCode != nil {
		return *l.BaseItemCode
	}
	return ""
}

// GetBaseItemName returns the base item name or empty string
func (l *Listing) GetBaseItemName() string {
	if l.BaseItemName != nil {
		return *l.BaseItemName
	}
	return ""
}

// GetCatalogItemID returns the catalog item ID or empty string
func (l *Listing) GetCatalogItemID() string {
	if l.CatalogItemID != nil {
		return *l.CatalogItemID
	}
	return ""
}

// GetSellerTimezone returns the seller timezone or empty string
func (l *Listing) GetSellerTimezone() string {
	if l.SellerTimezone != nil {
		return *l.SellerTimezone
	}
	return ""
}

// IsService returns true if the listing is a service listing
func (l *Listing) IsService() bool {
	return l.ListingType == "service"
}

// GetServiceType returns the service type or empty string
func (l *Listing) GetServiceType() string {
	if l.ServiceType != nil {
		return *l.ServiceType
	}
	return ""
}

// GetDescription returns the description or empty string
func (l *Listing) GetDescription() string {
	if l.Description != nil {
		return *l.Description
	}
	return ""
}
