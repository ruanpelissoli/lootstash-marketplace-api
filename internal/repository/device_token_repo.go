package repository

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

type deviceTokenRepository struct {
	db *database.BunDB
}

// NewDeviceTokenRepository creates a new device token repository
func NewDeviceTokenRepository(db *database.BunDB) DeviceTokenRepository {
	return &deviceTokenRepository{db: db}
}

func (r *deviceTokenRepository) Upsert(ctx context.Context, token *models.DeviceToken) error {
	_, err := r.db.DB().NewInsert().
		Model(token).
		On("CONFLICT (user_id, expo_push_token) DO UPDATE").
		Set("device_name = EXCLUDED.device_name").
		Set("platform = EXCLUDED.platform").
		Set("updated_at = CURRENT_TIMESTAMP").
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to upsert device token",
			"error", err.Error(),
			"user_id", token.UserID,
		)
	}
	return err
}

func (r *deviceTokenRepository) DeleteByToken(ctx context.Context, userID string, expoPushToken string) error {
	_, err := r.db.DB().NewDelete().
		Model((*models.DeviceToken)(nil)).
		Where("user_id = ?", userID).
		Where("expo_push_token = ?", expoPushToken).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to delete device token",
			"error", err.Error(),
			"user_id", userID,
		)
	}
	return err
}

func (r *deviceTokenRepository) GetByUserID(ctx context.Context, userID string) ([]*models.DeviceToken, error) {
	var tokens []*models.DeviceToken
	err := r.db.DB().NewSelect().
		Model(&tokens).
		Where("dt.user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to get device tokens",
			"error", err.Error(),
			"user_id", userID,
		)
	}
	return tokens, err
}

func (r *deviceTokenRepository) DeleteAllByUserID(ctx context.Context, userID string) error {
	_, err := r.db.DB().NewDelete().
		Model((*models.DeviceToken)(nil)).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to delete all device tokens",
			"error", err.Error(),
			"user_id", userID,
		)
	}
	return err
}
