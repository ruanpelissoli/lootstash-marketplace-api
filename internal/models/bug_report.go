package models

import (
	"time"

	"github.com/uptrace/bun"
)

// BugReport represents a user-submitted bug report
type BugReport struct {
	bun.BaseModel `bun:"table:d2.bug_reports,alias:br"`

	ID          string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID      string    `bun:"user_id,type:uuid,notnull"`
	Title       string    `bun:"title,notnull"`
	Description string    `bun:"description,notnull"`
	Status      string    `bun:"status,notnull,default:'open'"`
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Reporter *Profile `bun:"rel:belongs-to,join:user_id=id"`
}
