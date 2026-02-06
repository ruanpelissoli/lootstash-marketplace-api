package repository

import (
	"context"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/uptrace/bun"
)

type messageRepository struct {
	db *database.BunDB
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(db *database.BunDB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, message *models.Message) error {
	_, err := r.db.DB().NewInsert().
		Model(message).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create message",
			"error", err.Error(),
			"chat_id", message.ChatID,
			"sender_id", message.SenderID,
		)
	}
	return err
}

func (r *messageRepository) GetByChatID(ctx context.Context, chatID string, offset, limit int) ([]*models.Message, int, error) {
	var messages []*models.Message

	query := r.db.DB().NewSelect().
		Model(&messages).
		Relation("Sender").
		Where("m.chat_id = ?", chatID)

	count, err := query.Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count messages",
			"error", err.Error(),
			"chat_id", chatID,
		)
		return nil, 0, err
	}

	query = query.Order("m.created_at ASC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err = query.Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to get messages",
			"error", err.Error(),
			"chat_id", chatID,
		)
		return nil, 0, err
	}

	return messages, count, nil
}

func (r *messageRepository) MarkAsRead(ctx context.Context, messageIDs []string, userID string) error {
	now := time.Now()
	_, err := r.db.DB().NewUpdate().
		Model((*models.Message)(nil)).
		Set("read_at = ?", now).
		Where("id IN (?)", bun.In(messageIDs)).
		Where("sender_id != ?", userID). // Can't mark own messages as read
		Where("read_at IS NULL").
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to mark messages as read",
			"error", err.Error(),
			"user_id", userID,
		)
	}
	return err
}

func (r *messageRepository) MarkAllAsReadInChat(ctx context.Context, chatID string, userID string) error {
	now := time.Now()
	_, err := r.db.DB().NewUpdate().
		Model((*models.Message)(nil)).
		Set("read_at = ?", now).
		Where("chat_id = ?", chatID).
		Where("sender_id != ?", userID). // Can't mark own messages as read
		Where("read_at IS NULL").
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to mark all messages as read in chat",
			"error", err.Error(),
			"chat_id", chatID,
			"user_id", userID,
		)
	}
	return err
}

func (r *messageRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	// Count unread messages where user is not the sender
	// and user is a participant in the trade (via chat)
	count, err := r.db.DB().NewSelect().
		Model((*models.Message)(nil)).
		Where("sender_id != ?", userID).
		Where("read_at IS NULL").
		Where(`chat_id IN (
			SELECT c.id FROM d2.chats c
			JOIN d2.trades t ON t.id = c.trade_id
			WHERE t.seller_id = ? OR t.buyer_id = ?
		)`, userID, userID).
		Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count unread messages",
			"error", err.Error(),
			"user_id", userID,
		)
	}
	return count, err
}
