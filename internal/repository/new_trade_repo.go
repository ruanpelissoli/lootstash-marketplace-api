package repository

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

type tradeRepositoryNew struct {
	db *database.BunDB
}

// NewTradeRepositoryNew creates a new trade repository
func NewTradeRepositoryNew(db *database.BunDB) TradeRepository {
	return &tradeRepositoryNew{db: db}
}

func (r *tradeRepositoryNew) Create(ctx context.Context, trade *models.Trade) error {
	_, err := r.db.DB().NewInsert().
		Model(trade).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create trade",
			"error", err.Error(),
			"offer_id", trade.OfferID,
			"listing_id", trade.ListingID,
		)
	}
	return err
}

func (r *tradeRepositoryNew) GetByID(ctx context.Context, id string) (*models.Trade, error) {
	trade := new(models.Trade)
	err := r.db.DB().NewSelect().
		Model(trade).
		Where("t.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return trade, nil
}

func (r *tradeRepositoryNew) GetByIDWithRelations(ctx context.Context, id string) (*models.Trade, error) {
	trade := new(models.Trade)
	err := r.db.DB().NewSelect().
		Model(trade).
		Relation("Offer").
		Relation("Offer.Requester").
		Relation("Listing").
		Relation("Listing.Seller").
		Relation("Seller").
		Relation("Buyer").
		Relation("Chat").
		Where("t.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return trade, nil
}

func (r *tradeRepositoryNew) GetByOfferID(ctx context.Context, offerID string) (*models.Trade, error) {
	trade := new(models.Trade)
	err := r.db.DB().NewSelect().
		Model(trade).
		Where("t.offer_id = ?", offerID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return trade, nil
}

func (r *tradeRepositoryNew) Update(ctx context.Context, trade *models.Trade) error {
	_, err := r.db.DB().NewUpdate().
		Model(trade).
		WherePK().
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to update trade",
			"error", err.Error(),
			"trade_id", trade.ID,
		)
	}
	return err
}

func (r *tradeRepositoryNew) List(ctx context.Context, filter TradeFilter) ([]*models.Trade, int, error) {
	var trades []*models.Trade

	query := r.db.DB().NewSelect().
		Model(&trades).
		Relation("Offer").
		Relation("Offer.Requester").
		Relation("Listing").
		Relation("Listing.Seller").
		Relation("Seller").
		Relation("Buyer").
		Relation("Chat")

	// Filter by user (must be seller or buyer)
	if filter.UserID != "" {
		query = query.Where("t.seller_id = ? OR t.buyer_id = ?", filter.UserID, filter.UserID)
	}

	if filter.Status != "" {
		query = query.Where("t.status = ?", filter.Status)
	}

	count, err := query.Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count trades",
			"error", err.Error(),
			"user_id", filter.UserID,
		)
		return nil, 0, err
	}

	query = query.Order("t.created_at DESC")

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	err = query.Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to list trades",
			"error", err.Error(),
			"user_id", filter.UserID,
		)
		return nil, 0, err
	}

	return trades, count, nil
}

func (r *tradeRepositoryNew) HasActiveTradeForListing(ctx context.Context, listingID string) (bool, error) {
	count, err := r.db.DB().NewSelect().
		Model((*models.Trade)(nil)).
		Where("listing_id = ?", listingID).
		Where("status = ?", "active").
		Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to check active trade for listing",
			"error", err.Error(),
			"listing_id", listingID,
		)
		return false, err
	}
	return count > 0, nil
}
