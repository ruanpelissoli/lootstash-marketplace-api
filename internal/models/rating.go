package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Rating represents a user rating for a completed trade
type Rating struct {
	bun.BaseModel `bun:"table:d2.ratings,alias:r"`

	ID            string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	TransactionID string    `bun:"transaction_id,type:uuid,notnull"`
	RaterID       string    `bun:"rater_id,type:uuid,notnull"`
	RatedID       string    `bun:"rated_id,type:uuid,notnull"`
	Stars         int       `bun:"stars,notnull"`
	Comment       *string   `bun:"comment"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Transaction *Transaction `bun:"rel:belongs-to,join:transaction_id=id"`
	Rater       *Profile     `bun:"rel:belongs-to,join:rater_id=id"`
	Rated       *Profile     `bun:"rel:belongs-to,join:rated_id=id"`
}

// GetComment returns the comment or empty string
func (r *Rating) GetComment() string {
	if r.Comment != nil {
		return *r.Comment
	}
	return ""
}
