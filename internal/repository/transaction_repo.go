package repository

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

type transactionRepository struct {
	db *database.BunDB
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(db *database.BunDB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	_, err := r.db.DB().NewInsert().
		Model(transaction).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create transaction",
			"error", err.Error(),
			"trade_id", transaction.TradeID,
		)
	}
	return err
}

func (r *transactionRepository) GetByID(ctx context.Context, id string) (*models.Transaction, error) {
	transaction := new(models.Transaction)
	err := r.db.DB().NewSelect().
		Model(transaction).
		Relation("Seller").
		Relation("Buyer").
		Where("tx.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return transaction, nil
}

func (r *transactionRepository) GetByTradeID(ctx context.Context, tradeID string) (*models.Transaction, error) {
	transaction := new(models.Transaction)
	err := r.db.DB().NewSelect().
		Model(transaction).
		Relation("Seller").
		Relation("Buyer").
		Where("tx.trade_id = ?", tradeID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return transaction, nil
}

func (r *transactionRepository) GetPriceHistory(ctx context.Context, itemName string, days int) ([]PriceHistoryRecord, error) {
	var results []PriceHistoryRecord
	err := r.db.DB().NewSelect().
		ColumnExpr("DATE(tx.created_at) AS date").
		ColumnExpr("tx.offered_items").
		TableExpr("d2.transactions AS tx").
		Join("INNER JOIN d2.trades AS t ON t.id = tx.trade_id").
		Where("tx.item_name ILIKE ?", itemName).
		Where("tx.created_at >= NOW() - INTERVAL '1 day' * ?", days).
		Where("t.status = ?", "completed").
		OrderExpr("date ASC, tx.created_at ASC").
		Scan(ctx, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *transactionRepository) GetSalesBySeller(ctx context.Context, sellerID string, offset, limit int) ([]SaleRecord, int, error) {
	// Count total sales for pagination
	count, err := r.db.DB().NewSelect().
		TableExpr("d2.transactions AS tx").
		Join("INNER JOIN d2.trades AS t ON t.id = tx.trade_id").
		Where("tx.seller_id = ?", sellerID).
		Where("t.status = ?", "completed").
		Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count sales",
			"error", err.Error(),
			"seller_id", sellerID,
		)
		return nil, 0, err
	}

	// Fetch sales with related data
	var results []SaleRecord
	err = r.db.DB().NewSelect().
		ColumnExpr("tx.id AS transaction_id").
		ColumnExpr("t.completed_at").
		ColumnExpr("tx.item_name").
		ColumnExpr("l.item_type").
		ColumnExpr("l.rarity").
		ColumnExpr("l.image_url").
		ColumnExpr("l.base_item_name AS base_name").
		ColumnExpr("l.stats").
		ColumnExpr("tx.offered_items").
		ColumnExpr("buyer.id AS buyer_id").
		ColumnExpr("COALESCE(buyer.display_name, buyer.username) AS buyer_name").
		ColumnExpr("buyer.avatar_url AS buyer_avatar").
		ColumnExpr("r.stars AS review_rating").
		ColumnExpr("r.comment AS review_comment").
		ColumnExpr("r.created_at AS reviewed_at").
		TableExpr("d2.transactions AS tx").
		Join("INNER JOIN d2.trades AS t ON t.id = tx.trade_id").
		Join("INNER JOIN d2.listings AS l ON l.id = tx.listing_id").
		Join("INNER JOIN d2.profiles AS buyer ON buyer.id = tx.buyer_id").
		Join("LEFT JOIN d2.ratings AS r ON r.transaction_id = tx.id AND r.rater_id = tx.buyer_id").
		Where("tx.seller_id = ?", sellerID).
		Where("t.status = ?", "completed").
		Order("t.completed_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx, &results)
	if err != nil {
		logger.FromContext(ctx).Error("failed to get sales",
			"error", err.Error(),
			"seller_id", sellerID,
		)
		return nil, 0, err
	}

	return results, count, nil
}
