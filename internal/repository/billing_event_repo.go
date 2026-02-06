package repository

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

type billingEventRepository struct {
	db *database.BunDB
}

// NewBillingEventRepository creates a new billing event repository
func NewBillingEventRepository(db *database.BunDB) BillingEventRepository {
	return &billingEventRepository{db: db}
}

func (r *billingEventRepository) Create(ctx context.Context, event *models.BillingEvent) error {
	_, err := r.db.DB().NewInsert().
		Model(event).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create billing event",
			"error", err.Error(),
			"user_id", event.UserID,
			"stripe_event_id", event.StripeEventID,
		)
	}
	return err
}

func (r *billingEventRepository) GetByUserID(ctx context.Context, userID string) ([]*models.BillingEvent, error) {
	var events []*models.BillingEvent
	err := r.db.DB().NewSelect().
		Model(&events).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(50).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *billingEventRepository) ExistsByStripeEventID(ctx context.Context, stripeEventID string) (bool, error) {
	exists, err := r.db.DB().NewSelect().
		Model((*models.BillingEvent)(nil)).
		Where("stripe_event_id = ?", stripeEventID).
		Exists(ctx)
	return exists, err
}
