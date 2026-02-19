package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

// Service represents a standalone service offered by a provider
type Service struct {
	bun.BaseModel `bun:"table:d2.services,alias:s"`

	ID          string          `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ProviderID  string          `bun:"provider_id,type:uuid,notnull"`
	ServiceType string          `bun:"service_type,notnull"`
	Name        string          `bun:"name,notnull"`
	Description *string         `bun:"description"`
	AskingPrice *string         `bun:"asking_price"`
	AskingFor   json.RawMessage `bun:"asking_for,type:jsonb,default:'[]'"`
	Game        string          `bun:"game,notnull,default:'diablo2'"`
	Ladder      bool            `bun:"ladder"`
	Hardcore    bool            `bun:"hardcore,default:false"`
	IsNonRotw   bool            `bun:"is_non_rotw,default:false"`
	Platforms   []string        `bun:"platforms,array,default:'{pc}'"`
	Region      string          `bun:"region,default:'americas'"`
	Notes       *string         `bun:"notes"`
	Status      string          `bun:"status,notnull,default:'active'"`
	CreatedAt   time.Time       `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time       `bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Provider *Profile `bun:"rel:belongs-to,join:provider_id=id"`
}

// IsActive returns true if the service is active
func (s *Service) IsActive() bool {
	return s.Status == "active"
}

// IsPaused returns true if the service is paused
func (s *Service) IsPaused() bool {
	return s.Status == "paused"
}

// GetDescription returns the description or empty string
func (s *Service) GetDescription() string {
	if s.Description != nil {
		return *s.Description
	}
	return ""
}

// GetAskingPrice returns the asking price or empty string
func (s *Service) GetAskingPrice() string {
	if s.AskingPrice != nil {
		return *s.AskingPrice
	}
	return ""
}

// GetNotes returns the notes or empty string
func (s *Service) GetNotes() string {
	if s.Notes != nil {
		return *s.Notes
	}
	return ""
}
