package dto

import (
	"encoding/json"
	"time"
)

// ServiceResponse represents a single service in a provider card
type ServiceResponse struct {
	ID          string          `json:"id"`
	ServiceType string          `json:"serviceType"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	AskingPrice string          `json:"askingPrice,omitempty"`
	AskingFor   json.RawMessage `json:"askingFor,omitempty"`
	Game        string          `json:"game"`
	Ladder      bool            `json:"ladder"`
	Hardcore    bool            `json:"hardcore"`
	IsNonRotw   bool            `json:"isNonRotw"`
	Platforms   []string        `json:"platforms"`
	Region      string          `json:"region"`
	Notes       string          `json:"notes,omitempty"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// ProviderCardResponse represents a provider with all their services
type ProviderCardResponse struct {
	Provider *ProfileResponse  `json:"provider"`
	Services []ServiceResponse `json:"services"`
}

// CreateServiceRequest represents a request to create a service
type CreateServiceRequest struct {
	ServiceType string          `json:"serviceType" validate:"required,oneof=rush crush grush sockets waypoints ubers colossal_ancients"`
	Name        string          `json:"name" validate:"required,min=1,max=100"`
	Description string          `json:"description,omitempty" validate:"omitempty,max=2000"`
	AskingPrice string          `json:"askingPrice,omitempty" validate:"omitempty,max=100"`
	AskingFor   json.RawMessage `json:"askingFor,omitempty"`
	Notes       string          `json:"notes,omitempty" validate:"omitempty,max=500"`
	Game        string          `json:"game" validate:"required,min=1,max=20"`
	Ladder      bool            `json:"ladder"`
	Hardcore    bool            `json:"hardcore"`
	IsNonRotw   bool            `json:"isNonRotw"`
	Platforms   []string        `json:"platforms" validate:"required,min=1,dive,oneof=pc xbox playstation switch"`
	Region      string          `json:"region" validate:"required,oneof=americas europe asia"`
}

// UpdateServiceRequest represents a request to update a service
type UpdateServiceRequest struct {
	Name        *string         `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string         `json:"description,omitempty" validate:"omitempty,max=2000"`
	AskingPrice *string         `json:"askingPrice,omitempty" validate:"omitempty,max=100"`
	AskingFor   json.RawMessage `json:"askingFor,omitempty"`
	Notes       *string         `json:"notes,omitempty" validate:"omitempty,max=500"`
	Platforms   []string        `json:"platforms,omitempty" validate:"omitempty,min=1,dive,oneof=pc xbox playstation switch"`
	Region      *string         `json:"region,omitempty" validate:"omitempty,oneof=americas europe asia"`
}

// SearchServicesRequest represents service search/filter parameters via JSON body
type SearchServicesRequest struct {
	ServiceType []string `json:"serviceType"`
	Game        string   `json:"game"`
	Ladder      *bool    `json:"ladder"`
	Hardcore    *bool    `json:"hardcore"`
	IsNonRotw   *bool    `json:"isNonRotw"`
	Platforms   []string `json:"platforms"`
	Region      string   `json:"region"`
	Page        int      `json:"page"`
	PerPage     int      `json:"perPage"`
}

// ServiceProvidersFilterRequest represents filter parameters for provider listing
type ServiceProvidersFilterRequest struct {
	ServiceType string `query:"serviceType"`
	Game        string `query:"game"`
	Ladder      *bool  `query:"ladder"`
	Hardcore    *bool  `query:"hardcore"`
	IsNonRotw   *bool  `query:"isNonRotw"`
	Platforms   string `query:"platforms"`
	Region      string `query:"region"`
	Pagination
}
