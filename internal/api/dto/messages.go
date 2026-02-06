package dto

import "time"

// MessageResponse represents a chat message
type MessageResponse struct {
	ID          string           `json:"id"`
	ChatID      string           `json:"chatId"`
	SenderID    string           `json:"senderId"`
	Sender      *ProfileResponse `json:"sender,omitempty"`
	Content     string           `json:"content"`
	MessageType string           `json:"messageType"`
	ReadAt      *time.Time       `json:"readAt,omitempty"`
	CreatedAt   time.Time        `json:"createdAt"`
}

// MessagesFilterRequest represents filter parameters for messages
type MessagesFilterRequest struct {
	After  *time.Time `query:"after"`
	Before *time.Time `query:"before"`
	Pagination
}
