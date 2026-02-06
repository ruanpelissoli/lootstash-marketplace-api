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

func (r *transactionRepository) GetTradeVolume(ctx context.Context, itemName string, days int) ([]TradeVolumePoint, error) {
	var results []TradeVolumePoint
	err := r.db.DB().NewSelect().
		ColumnExpr("DATE(tx.created_at) AS date").
		ColumnExpr("COUNT(*) AS volume").
		TableExpr("d2.transactions AS tx").
		Where("tx.item_name ILIKE ?", itemName).
		Where("tx.created_at >= NOW() - INTERVAL '1 day' * ?", days).
		GroupExpr("DATE(tx.created_at)").
		OrderExpr("date ASC").
		Scan(ctx, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}
