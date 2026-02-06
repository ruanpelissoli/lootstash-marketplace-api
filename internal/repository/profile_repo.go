package repository

import (
	"context"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

type profileRepository struct {
	db *database.BunDB
}

// NewProfileRepository creates a new profile repository
func NewProfileRepository(db *database.BunDB) ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) GetByID(ctx context.Context, id string) (*models.Profile, error) {
	profile := new(models.Profile)
	err := r.db.DB().NewSelect().
		Model(profile).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Debug("profile not found or error",
			"profile_id", id,
			"error", err.Error(),
		)
		return nil, err
	}
	return profile, nil
}

func (r *profileRepository) GetByStripeCustomerID(ctx context.Context, customerID string) (*models.Profile, error) {
	profile := new(models.Profile)
	err := r.db.DB().NewSelect().
		Model(profile).
		Where("stripe_customer_id = ?", customerID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (r *profileRepository) GetByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*models.Profile, error) {
	profile := new(models.Profile)
	err := r.db.DB().NewSelect().
		Model(profile).
		Where("stripe_subscription_id = ?", subscriptionID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (r *profileRepository) Update(ctx context.Context, profile *models.Profile) error {
	_, err := r.db.DB().NewUpdate().
		Model(profile).
		WherePK().
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to update profile",
			"error", err.Error(),
			"profile_id", profile.ID,
		)
	}
	return err
}

func (r *profileRepository) GetEmailByID(ctx context.Context, id string) (string, error) {
	var email string
	err := r.db.DB().NewSelect().
		TableExpr("auth.users").
		Column("email").
		Where("id = ?", id).
		Scan(ctx, &email)
	if err != nil {
		logger.FromContext(ctx).Debug("failed to get email from auth.users",
			"user_id", id,
			"error", err.Error(),
		)
		return "", err
	}
	return email, nil
}

func (r *profileRepository) UpdateLastActiveAt(ctx context.Context, userID string) error {
	_, err := r.db.DB().NewUpdate().
		Model((*models.Profile)(nil)).
		Set("last_active_at = ?", time.Now()).
		Where("id = ?", userID).
		Exec(ctx)
	return err
}
