package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Chat represents a conversation linked to a trade
type Chat struct {
	bun.BaseModel `bun:"table:d2.chats,alias:c"`

	ID        string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	TradeID   string    `bun:"trade_id,type:uuid,notnull"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Trade    *Trade     `bun:"rel:belongs-to,join:trade_id=id"`
	Messages []*Message `bun:"rel:has-many,join:id=chat_id"`
}
