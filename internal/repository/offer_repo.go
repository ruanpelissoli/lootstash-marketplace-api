package repository

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

type offerRepository struct {
	db *database.BunDB
}

// NewOfferRepository creates a new offer repository
func NewOfferRepository(db *database.BunDB) OfferRepository {
	return &offerRepository{db: db}
}

func (r *offerRepository) Create(ctx context.Context, offer *models.Offer) error {
	_, err := r.db.DB().NewInsert().
		Model(offer).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create offer",
			"error", err.Error(),
			"requester_id", offer.RequesterID,
		)
	}
	return err
}

func (r *offerRepository) GetByID(ctx context.Context, id string) (*models.Offer, error) {
	offer := new(models.Offer)
	err := r.db.DB().NewSelect().
		Model(offer).
		Where("o.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return offer, nil
}

func (r *offerRepository) GetByIDWithRelations(ctx context.Context, id string) (*models.Offer, error) {
	offer := new(models.Offer)
	err := r.db.DB().NewSelect().
		Model(offer).
		Relation("Listing").
		Relation("Listing.Seller").
		Relation("Service").
		Relation("Service.Provider").
		Relation("Requester").
		Relation("DeclineReason").
		Relation("Trade").
		Relation("ServiceRun").
		Where("o.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return offer, nil
}

func (r *offerRepository) Update(ctx context.Context, offer *models.Offer) error {
	_, err := r.db.DB().NewUpdate().
		Model(offer).
		WherePK().
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to update offer",
			"error", err.Error(),
			"offer_id", offer.ID,
		)
	}
	return err
}

func (r *offerRepository) List(ctx context.Context, filter OfferFilter) ([]*models.Offer, int, error) {
	var offers []*models.Offer

	query := r.db.DB().NewSelect().
		Model(&offers).
		Relation("Listing").
		Relation("Listing.Seller").
		Relation("Service").
		Relation("Service.Provider").
		Relation("Requester").
		Relation("DeclineReason").
		Relation("Trade").
		Relation("ServiceRun")

	// Filter by offer type
	switch filter.Type {
	case "item":
		query = query.Where("o.type = ?", "item")
	case "service":
		query = query.Where("o.type = ?", "service")
		// "all" or empty: no type filter
	}

	// If filtering by listingId, check that the user is the listing owner
	if filter.ListingID != "" {
		query = query.Where("o.listing_id = ?", filter.ListingID)
		if filter.UserID != "" {
			query = query.Where("EXISTS (SELECT 1 FROM d2.listings l WHERE l.id = o.listing_id AND l.seller_id = ?)", filter.UserID)
		}
	} else if filter.ServiceID != "" {
		query = query.Where("o.service_id = ?", filter.ServiceID)
		if filter.UserID != "" {
			query = query.Where("EXISTS (SELECT 1 FROM d2.services s WHERE s.id = o.service_id AND s.provider_id = ?)", filter.UserID)
		}
	} else {
		// Filter by role
		switch filter.Role {
		case "buyer":
			query = query.Where("o.requester_id = ?", filter.UserID)
		case "seller":
			// User is seller (listing owner) or provider (service owner)
			query = query.Where(
				"EXISTS (SELECT 1 FROM d2.listings l WHERE l.id = o.listing_id AND l.seller_id = ?) OR EXISTS (SELECT 1 FROM d2.services s WHERE s.id = o.service_id AND s.provider_id = ?)",
				filter.UserID, filter.UserID,
			)
		default:
			// All offers where user is buyer, seller, or provider
			query = query.Where(
				"o.requester_id = ? OR EXISTS (SELECT 1 FROM d2.listings l WHERE l.id = o.listing_id AND l.seller_id = ?) OR EXISTS (SELECT 1 FROM d2.services s WHERE s.id = o.service_id AND s.provider_id = ?)",
				filter.UserID, filter.UserID, filter.UserID,
			)
		}
	}

	if filter.Status != "" {
		query = query.Where("o.status = ?", filter.Status)
	}

	count, err := query.Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count offers",
			"error", err.Error(),
			"user_id", filter.UserID,
		)
		return nil, 0, err
	}

	query = query.Order("o.created_at DESC")

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	err = query.Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to list offers",
			"error", err.Error(),
			"user_id", filter.UserID,
		)
		return nil, 0, err
	}

	return offers, count, nil
}

func (r *offerRepository) GetDeclineReasons(ctx context.Context) ([]*models.DeclineReason, error) {
	var reasons []*models.DeclineReason
	err := r.db.DB().NewSelect().
		Model(&reasons).
		Where("active = ?", true).
		Order("id ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return reasons, nil
}

func (r *offerRepository) GetDeclineReasonByID(ctx context.Context, id int) (*models.DeclineReason, error) {
	reason := new(models.DeclineReason)
	err := r.db.DB().NewSelect().
		Model(reason).
		Where("id = ?", id).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return reason, nil
}
