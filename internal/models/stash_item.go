package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

// StashItem represents an item in a user's stash (inventory)
type StashItem struct {
	bun.BaseModel `bun:"table:d2.stash_items,alias:si"`

	ID            string          `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID        string          `bun:"user_id,type:uuid,notnull"`
	Name          *string         `bun:"name"`
	ItemType      string          `bun:"item_type,notnull"`
	Rarity        string          `bun:"rarity,nullzero"`
	ImageURL      *string         `bun:"image_url"`
	Category      string          `bun:"category"`
	Stats         json.RawMessage `bun:"stats,type:jsonb,default:'[]'"`
	Suffixes      json.RawMessage `bun:"suffixes,type:jsonb,default:'[]'"`
	Runes         json.RawMessage `bun:"runes,type:jsonb,default:'[]'"`
	RuneOrder     *string         `bun:"rune_order"`
	BaseItemCode  *string         `bun:"base_item_code"`
	BaseItemName  *string         `bun:"base_item_name"`
	CatalogItemID *string         `bun:"catalog_item_id"`
	Quantity      int             `bun:"quantity,notnull,default:1"`
	Game          string          `bun:"game,notnull,default:'diablo2'"`
	Ladder        bool            `bun:"ladder"`
	Hardcore      bool            `bun:"hardcore,default:false"`

	Platforms     []string        `bun:"platforms,array,default:'{pc}'"`
	Region        string          `bun:"region,default:'americas'"`
	Source        string          `bun:"source,default:'manual'"`
	CreatedAt     time.Time       `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time       `bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	User *Profile `bun:"rel:belongs-to,join:user_id=id"`
}

// IsStackable returns true for categories that support quantity > 1
func (si *StashItem) IsStackable() bool {
	switch si.Category {
	case "rune", "runes", "gem", "gems", "misc":
		return true
	case "amethyst", "diamond", "emerald", "ruby", "sapphire", "topaz", "skull":
		return true
	default:
		return false
	}
}

// GetDisplayName returns Name if set, otherwise ItemType
func (si *StashItem) GetDisplayName() string {
	if si.Name != nil && *si.Name != "" {
		return *si.Name
	}
	return si.ItemType
}

// GetName returns the name or empty string
func (si *StashItem) GetName() string {
	if si.Name != nil {
		return *si.Name
	}
	return ""
}

// GetImageURL returns the image URL or empty string
func (si *StashItem) GetImageURL() string {
	if si.ImageURL != nil {
		return *si.ImageURL
	}
	return ""
}

// GetRuneOrder returns the rune order or empty string
func (si *StashItem) GetRuneOrder() string {
	if si.RuneOrder != nil {
		return *si.RuneOrder
	}
	return ""
}

// GetBaseItemCode returns the base item code or empty string
func (si *StashItem) GetBaseItemCode() string {
	if si.BaseItemCode != nil {
		return *si.BaseItemCode
	}
	return ""
}

// GetBaseItemName returns the base item name or empty string
func (si *StashItem) GetBaseItemName() string {
	if si.BaseItemName != nil {
		return *si.BaseItemName
	}
	return ""
}

// GetCatalogItemID returns the catalog item ID or empty string
func (si *StashItem) GetCatalogItemID() string {
	if si.CatalogItemID != nil {
		return *si.CatalogItemID
	}
	return ""
}
