package repository

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

type chatRepository struct {
	db *database.BunDB
}

// NewChatRepository creates a new chat repository
func NewChatRepository(db *database.BunDB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) Create(ctx context.Context, chat *models.Chat) error {
	_, err := r.db.DB().NewInsert().
		Model(chat).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create chat",
			"error", err.Error(),
			"trade_id", chat.TradeID,
		)
	}
	return err
}

func (r *chatRepository) GetByID(ctx context.Context, id string) (*models.Chat, error) {
	chat := new(models.Chat)
	err := r.db.DB().NewSelect().
		Model(chat).
		Where("c.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return chat, nil
}

func (r *chatRepository) GetByIDWithTrade(ctx context.Context, id string) (*models.Chat, error) {
	chat := new(models.Chat)
	err := r.db.DB().NewSelect().
		Model(chat).
		Relation("Trade").
		Relation("Trade.Seller").
		Relation("Trade.Buyer").
		Relation("Trade.Listing").
		Where("c.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return chat, nil
}

func (r *chatRepository) GetByTradeID(ctx context.Context, tradeID string) (*models.Chat, error) {
	chat := new(models.Chat)
	err := r.db.DB().NewSelect().
		Model(chat).
		Where("c.trade_id = ?", tradeID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return chat, nil
}

func (r *chatRepository) Update(ctx context.Context, chat *models.Chat) error {
	_, err := r.db.DB().NewUpdate().
		Model(chat).
		WherePK().
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to update chat",
			"error", err.Error(),
			"chat_id", chat.ID,
		)
	}
	return err
}
