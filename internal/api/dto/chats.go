package dto

import "time"

// ChatResponse represents a chat
type ChatResponse struct {
	ID           string    `json:"id"`
	TradeID      string    `json:"tradeId,omitempty"`
	ServiceRunID string    `json:"serviceRunId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ChatDetailResponse includes additional details for a single chat
type ChatDetailResponse struct {
	ChatResponse
	Trade      *TradeResponse      `json:"trade,omitempty"`
	ServiceRun *ServiceRunResponse `json:"serviceRun,omitempty"`
}

// SendChatMessageRequest represents a request to send a message in a chat
type SendChatMessageRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

// MarkChatMessagesReadRequest represents a request to mark messages as read
type MarkChatMessagesReadRequest struct {
	MessageIDs []string `json:"messageIds" validate:"required,min=1,dive,uuid"`
}
