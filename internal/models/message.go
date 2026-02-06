package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Message represents a chat message in a trade
type Message struct {
	bun.BaseModel `bun:"table:d2.messages,alias:m"`

	ID          string     `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ChatID      string     `bun:"chat_id,type:uuid,notnull"`
	SenderID    string     `bun:"sender_id,type:uuid,notnull"`
	Content     string     `bun:"content,notnull"`
	MessageType string     `bun:"message_type,notnull,default:'text'"`
	ReadAt      *time.Time `bun:"read_at"`
	CreatedAt   time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Denormalized participant IDs for Realtime RLS (simpler policy evaluation)
	SellerID *string `bun:"seller_id,type:uuid"`
	BuyerID  *string `bun:"buyer_id,type:uuid"`

	// Relations
	Chat   *Chat    `bun:"rel:belongs-to,join:chat_id=id"`
	Sender *Profile `bun:"rel:belongs-to,join:sender_id=id"`
}

// IsRead returns true if the message has been read
func (m *Message) IsRead() bool {
	return m.ReadAt != nil
}

// IsSystemMessage returns true if this is a system message
func (m *Message) IsSystemMessage() bool {
	return m.MessageType == "system" || m.MessageType == "trade_update"
}
