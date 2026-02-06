package repository

import (
	"context"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/uptrace/bun"
)

type notificationRepository struct {
	db *database.BunDB
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *database.BunDB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	_, err := r.db.DB().NewInsert().
		Model(notification).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create notification",
			"error", err.Error(),
			"user_id", notification.UserID,
			"type", notification.Type,
		)
	}
	return err
}

func (r *notificationRepository) GetByUserID(ctx context.Context, userID string, unreadOnly bool, notificationType string, offset, limit int) ([]*models.Notification, int, error) {
	var notifications []*models.Notification

	query := r.db.DB().NewSelect().
		Model(&notifications).
		Where("n.user_id = ?", userID)

	if unreadOnly {
		query = query.Where("n.read = ?", false)
	}

	if notificationType != "" {
		query = query.Where("n.type = ?", notificationType)
	}

	count, err := query.Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count notifications",
			"error", err.Error(),
			"user_id", userID,
		)
		return nil, 0, err
	}

	query = query.Order("n.created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err = query.Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to get notifications",
			"error", err.Error(),
			"user_id", userID,
		)
		return nil, 0, err
	}

	return notifications, count, nil
}

func (r *notificationRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	count, err := r.db.DB().NewSelect().
		Model((*models.Notification)(nil)).
		Where("user_id = ?", userID).
		Where("read = ?", false).
		Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count unread notifications",
			"error", err.Error(),
			"user_id", userID,
		)
	}
	return count, err
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, notificationIDs []string, userID string) error {
	now := time.Now()
	_, err := r.db.DB().NewUpdate().
		Model((*models.Notification)(nil)).
		Set("read = ?", true).
		Set("read_at = ?", now).
		Where("id IN (?)", bun.In(notificationIDs)).
		Where("user_id = ?", userID).
		Where("read = ?", false).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to mark notifications as read",
			"error", err.Error(),
			"user_id", userID,
		)
	}
	return err
}
