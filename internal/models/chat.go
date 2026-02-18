package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Chat represents a conversation linked to a trade or service run
type Chat struct {
	bun.BaseModel `bun:"table:d2.chats,alias:c"`

	ID           string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	TradeID      *string   `bun:"trade_id,type:uuid"`
	ServiceRunID *string   `bun:"service_run_id,type:uuid"`
	CreatedAt    time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt    time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Trade      *Trade       `bun:"rel:belongs-to,join:trade_id=id"`
	ServiceRun *ServiceRun  `bun:"rel:belongs-to,join:service_run_id=id"`
	Messages   []*Message   `bun:"rel:has-many,join:id=chat_id"`
}

// GetTradeID returns the trade ID or empty string
func (c *Chat) GetTradeID() string {
	if c.TradeID != nil {
		return *c.TradeID
	}
	return ""
}

// GetServiceRunID returns the service run ID or empty string
func (c *Chat) GetServiceRunID() string {
	if c.ServiceRunID != nil {
		return *c.ServiceRunID
	}
	return ""
}

// IsTradeChat returns true if this chat is linked to a trade
func (c *Chat) IsTradeChat() bool {
	return c.TradeID != nil && *c.TradeID != ""
}

// IsServiceRunChat returns true if this chat is linked to a service run
func (c *Chat) IsServiceRunChat() bool {
	return c.ServiceRunID != nil && *c.ServiceRunID != ""
}
