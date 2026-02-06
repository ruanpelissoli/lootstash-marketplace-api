package dto

import (
	"encoding/json"
	"time"
)

// NotificationResponse represents a notification
type NotificationResponse struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Title         string          `json:"title"`
	Body          string          `json:"body,omitempty"`
	ReferenceType string          `json:"referenceType,omitempty"`
	ReferenceID   string          `json:"referenceId,omitempty"`
	Read          bool            `json:"read"`
	ReadAt        *time.Time      `json:"readAt,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
}

// NotificationCountResponse represents the unread notification count
type NotificationCountResponse struct {
	Count int `json:"count"`
}

// MarkNotificationsReadRequest represents a request to mark notifications as read
type MarkNotificationsReadRequest struct {
	NotificationIDs []string `json:"notificationIds" validate:"required,min=1,dive,uuid"`
}

// NotificationsFilterRequest represents filter parameters for notifications
type NotificationsFilterRequest struct {
	Unread *bool `query:"unread"`
	Type   string `query:"type"`
	Pagination
}
