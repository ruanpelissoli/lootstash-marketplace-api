package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/games/d2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/uptrace/bun"
)

type listingRepository struct {
	db *database.BunDB
}

// NewListingRepository creates a new listing repository
func NewListingRepository(db *database.BunDB) ListingRepository {
	return &listingRepository{db: db}
}

func (r *listingRepository) Create(ctx context.Context, listing *models.Listing) error {
	_, err := r.db.DB().NewInsert().
		Model(listing).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create listing",
			"error", err.Error(),
			"seller_id", listing.SellerID,
		)
	}
	return err
}

func (r *listingRepository) GetByID(ctx context.Context, id string) (*models.Listing, error) {
	listing := new(models.Listing)
	err := r.db.DB().NewSelect().
		Model(listing).
		Where("l.id = ?", id).
		Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Debug("listing not found or error",
			"listing_id", id,
			"error", err.Error(),
		)
		return nil, err
	}
	return listing, nil
}

func (r *listingRepository) GetByIDWithSeller(ctx context.Context, id string) (*models.Listing, error) {
	listing := new(models.Listing)
	err := r.db.DB().NewSelect().
		Model(listing).
		Relation("Seller").
		Where("l.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return listing, nil
}

func (r *listingRepository) Update(ctx context.Context, listing *models.Listing) error {
	_, err := r.db.DB().NewUpdate().
		Model(listing).
		WherePK().
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to update listing",
			"error", err.Error(),
			"listing_id", listing.ID,
		)
	}
	return err
}

func (r *listingRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.DB().NewDelete().
		Model((*models.Listing)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to delete listing",
			"error", err.Error(),
			"listing_id", id,
		)
	}
	return err
}

func (r *listingRepository) List(ctx context.Context, filter ListingFilter) ([]*models.Listing, int, error) {
	var listings []*models.Listing

	query := r.db.DB().NewSelect().
		Model(&listings).
		Relation("Seller").
		Where("l.status = ?", "active").
		// Exclude listings that have an active trade
		Where("NOT EXISTS (SELECT 1 FROM d2.trades t WHERE t.listing_id = l.id AND t.status = ?)", "active")

	// Apply filters
	query = r.applyFilters(query, filter)

	// Get total count
	count, err := query.Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count listings",
			"error", err.Error(),
		)
		return nil, 0, err
	}

	// Apply sorting
	query = r.applySorting(query, filter)

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	err = query.Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to list listings",
			"error", err.Error(),
		)
		return nil, 0, err
	}

	return listings, count, nil
}

func (r *listingRepository) ListBySellerID(ctx context.Context, sellerID string, status string, offset, limit int) ([]*models.Listing, int, error) {
	var listings []*models.Listing

	query := r.db.DB().NewSelect().
		Model(&listings).
		Where("l.seller_id = ?", sellerID)

	if status != "" {
		query = query.Where("l.status = ?", status)
	}

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	query = query.Order("l.created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	return listings, count, nil
}

func (r *listingRepository) CountByListingID(ctx context.Context, listingID string) (int, error) {
	count, err := r.db.DB().NewSelect().
		Model((*models.Offer)(nil)).
		Where("listing_id = ?", listingID).
		Where("status NOT IN (?)", bun.In([]string{"rejected", "cancelled"})).
		Count(ctx)
	return count, err
}

func (r *listingRepository) applyFilters(query *bun.SelectQuery, filter ListingFilter) *bun.SelectQuery {
	if filter.SellerID != "" {
		query = query.Where("l.seller_id = ?", filter.SellerID)
	}

	if filter.Query != "" {
		searchPattern := "%" + strings.ToLower(filter.Query) + "%"
		query = query.Where("LOWER(l.name) LIKE ?", searchPattern)
	}

	if filter.Game != "" {
		query = query.Where("l.game = ?", filter.Game)
	}

	if filter.Ladder != nil {
		query = query.Where("l.ladder = ?", *filter.Ladder)
	}

	if filter.Hardcore != nil {
		query = query.Where("l.hardcore = ?", *filter.Hardcore)
	}

	if filter.IsNonRotw != nil {
		query = query.Where("l.is_non_rotw = ?", *filter.IsNonRotw)
	}

	if filter.Platform != "" {
		query = query.Where("l.platform = ?", filter.Platform)
	}

	if filter.Region != "" {
		query = query.Where("l.region = ?", filter.Region)
	}

	if filter.Category != "" {
		query = query.Where("l.category = ? OR l.category LIKE ? OR l.category IN (?)",
			filter.Category,
			"% "+filter.Category,
			bun.In(d2.GetSubcategories(filter.Category)),
		)
	}

	if filter.Rarity != "" {
		query = query.Where("l.rarity = ?", filter.Rarity)
	}

	if filter.CatalogItemID != "" {
		query = query.Where("l.catalog_item_id = ?", filter.CatalogItemID)
	}

	// Apply affix filters (JSONB queries)
	for _, af := range filter.AffixFilters {
		query = r.applyAffixFilter(query, af)
	}

	// Apply asking_for filters (JSONB queries)
	for _, af := range filter.AskingForFilters {
		query = r.applyAskingForFilter(query, af)
	}

	return query
}

func (r *listingRepository) applyAffixFilter(query *bun.SelectQuery, af AffixFilter) *bun.SelectQuery {
	// Check if this is a skill tree code that needs param matching
	// Items store skill tree bonuses as {code: "skilltab", param: "N", value: X}
	if skillTabParam := d2.GetSkillTabParam(af.Code); skillTabParam != "" {
		// Skill tree filter: match code="skilltab" AND param=N
		if af.MinValue != nil && af.MaxValue != nil {
			query = query.Where(
				"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' = 'skilltab' AND elem->>'param' = ? AND (elem->>'value')::int >= ? AND (elem->>'value')::int <= ?)",
				skillTabParam, *af.MinValue, *af.MaxValue,
			)
		} else if af.MinValue != nil {
			query = query.Where(
				"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' = 'skilltab' AND elem->>'param' = ? AND (elem->>'value')::int >= ?)",
				skillTabParam, *af.MinValue,
			)
		} else if af.MaxValue != nil {
			query = query.Where(
				"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' = 'skilltab' AND elem->>'param' = ? AND (elem->>'value')::int <= ?)",
				skillTabParam, *af.MaxValue,
			)
		} else {
			// Just check if the skill tree exists
			query = query.Where(
				"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' = 'skilltab' AND elem->>'param' = ?)",
				skillTabParam,
			)
		}
		return query
	}

	// Expand code to all aliases (canonical + game codes)
	// This allows filtering to match listings regardless of which code system was used
	codes := d2.ExpandStatCode(af.Code)

	if af.MinValue != nil && af.MaxValue != nil {
		query = query.Where(
			"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' IN (?) AND (elem->>'value')::int >= ? AND (elem->>'value')::int <= ?)",
			bun.In(codes), *af.MinValue, *af.MaxValue,
		)
	} else if af.MinValue != nil {
		query = query.Where(
			"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' IN (?) AND (elem->>'value')::int >= ?)",
			bun.In(codes), *af.MinValue,
		)
	} else if af.MaxValue != nil {
		query = query.Where(
			"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' IN (?) AND (elem->>'value')::int <= ?)",
			bun.In(codes), *af.MaxValue,
		)
	} else {
		// Just check if the affix exists
		query = query.Where(
			"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' IN (?)",
			bun.In(codes),
		)
	}

	return query
}

func (r *listingRepository) applyAskingForFilter(query *bun.SelectQuery, af AskingForFilter) *bun.SelectQuery {
	namePattern := "%" + strings.ToLower(af.Name) + "%"

	if af.Type != "" && af.MinQuantity != nil {
		query = query.Where(
			"(SELECT COUNT(*) FROM jsonb_array_elements(l.asking_for) elem WHERE LOWER(elem->>'name') LIKE ? AND elem->>'type' = ?) >= ?",
			namePattern, af.Type, *af.MinQuantity,
		)
	} else if af.Type != "" {
		query = query.Where(
			"EXISTS (SELECT 1 FROM jsonb_array_elements(l.asking_for) elem WHERE LOWER(elem->>'name') LIKE ? AND elem->>'type' = ?)",
			namePattern, af.Type,
		)
	} else if af.MinQuantity != nil {
		query = query.Where(
			"(SELECT COUNT(*) FROM jsonb_array_elements(l.asking_for) elem WHERE LOWER(elem->>'name') LIKE ?) >= ?",
			namePattern, *af.MinQuantity,
		)
	} else {
		query = query.Where(
			"EXISTS (SELECT 1 FROM jsonb_array_elements(l.asking_for) elem WHERE LOWER(elem->>'name') LIKE ?)",
			namePattern,
		)
	}

	return query
}

func (r *listingRepository) applySorting(query *bun.SelectQuery, filter ListingFilter) *bun.SelectQuery {
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}

	// Validate sort field to prevent SQL injection
	validSortFields := map[string]string{
		"created_at":   "l.created_at",
		"name":         "l.name",
		"asking_price": "l.asking_price",
	}

	sortField, ok := validSortFields[sortBy]
	if !ok {
		sortField = "l.created_at"
	}

	sortOrder := strings.ToUpper(filter.SortOrder)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}

	// JOIN profiles so we can sort premium sellers first
	query = query.Join("JOIN d2.profiles AS p ON p.id = l.seller_id")

	return query.OrderExpr(fmt.Sprintf("p.is_premium DESC, %s %s", sortField, sortOrder))
}

func (r *listingRepository) CountActiveBySellerID(ctx context.Context, sellerID string) (int, error) {
	count, err := r.db.DB().NewSelect().
		Model((*models.Listing)(nil)).
		Where("seller_id = ?", sellerID).
		Where("status = ?", "active").
		Count(ctx)
	return count, err
}

func (r *listingRepository) IncrementViews(ctx context.Context, id string) error {
	_, err := r.db.DB().NewUpdate().
		Model((*models.Listing)(nil)).
		Set("views = views + 1").
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to increment listing views",
			"error", err.Error(),
			"listing_id", id,
		)
	}
	return err
}

func (r *listingRepository) CountActive(ctx context.Context) (int, error) {
	count, err := r.db.DB().NewSelect().
		Model((*models.Listing)(nil)).
		Where("status = ?", "active").
		Count(ctx)
	return count, err
}

func (r *listingRepository) CancelOldestActiveListings(ctx context.Context, sellerID string, keepCount int) (int, error) {
	// Get IDs of the N most recent active listings to keep
	var keepIDs []string
	err := r.db.DB().NewSelect().
		Model((*models.Listing)(nil)).
		Column("id").
		Where("seller_id = ?", sellerID).
		Where("status = ?", "active").
		Order("created_at DESC").
		Limit(keepCount).
		Scan(ctx, &keepIDs)
	if err != nil {
		logger.FromContext(ctx).Error("failed to get listings to keep",
			"error", err.Error(),
			"seller_id", sellerID,
		)
		return 0, err
	}

	// Cancel all other active listings
	query := r.db.DB().NewUpdate().
		Model((*models.Listing)(nil)).
		Set("status = ?", "cancelled").
		Where("seller_id = ?", sellerID).
		Where("status = ?", "active")

	if len(keepIDs) > 0 {
		query = query.Where("id NOT IN (?)", bun.In(keepIDs))
	}

	res, err := query.Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to cancel oldest active listings",
			"error", err.Error(),
			"seller_id", sellerID,
		)
		return 0, err
	}

	rowsAffected, _ := res.RowsAffected()
	return int(rowsAffected), nil
}
