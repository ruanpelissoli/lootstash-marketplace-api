package dto

import "time"

// BattleNetLinkRequest represents a request to initiate Battle.net OAuth flow
type BattleNetLinkRequest struct {
	Region string `json:"region,omitempty" validate:"omitempty,oneof=us eu kr tw"`
}

// BattleNetLinkResponse represents the response with the authorization URL
type BattleNetLinkResponse struct {
	AuthorizationURL string `json:"authorizationUrl"`
}

// BattleNetCallbackRequest represents the OAuth callback data from frontend
type BattleNetCallbackRequest struct {
	Code  string `json:"code" validate:"required"`
	State string `json:"state" validate:"required"`
}

// BattleNetCallbackResponse represents the response after successful linking
type BattleNetCallbackResponse struct {
	Success     bool      `json:"success"`
	BattleTag   string    `json:"battleTag"`
	BattleNetID int64     `json:"battleNetId"`
	LinkedAt    time.Time `json:"linkedAt"`
}

// BattleNetUnlinkResponse represents the response after unlinking
type BattleNetUnlinkResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
