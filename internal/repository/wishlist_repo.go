package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type wishlistRepository struct {
	db *database.BunDB
}

// NewWishlistRepository creates a new wishlist repository
func NewWishlistRepository(db *database.BunDB) WishlistRepository {
	return &wishlistRepository{db: db}
}

func (r *wishlistRepository) Create(ctx context.Context, item *models.WishlistItem) error {
	_, err := r.db.DB().NewInsert().
		Model(item).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create wishlist item",
			"error", err.Error(),
			"user_id", item.UserID,
		)
	}
	return err
}

func (r *wishlistRepository) GetByID(ctx context.Context, id string) (*models.WishlistItem, error) {
	item := new(models.WishlistItem)
	err := r.db.DB().NewSelect().
		Model(item).
		Where("wi.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *wishlistRepository) Update(ctx context.Context, item *models.WishlistItem) error {
	item.UpdatedAt = time.Now()
	_, err := r.db.DB().NewUpdate().
		Model(item).
		WherePK().
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to update wishlist item",
			"error", err.Error(),
			"wishlist_id", item.ID,
		)
	}
	return err
}

func (r *wishlistRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.DB().NewUpdate().
		Model((*models.WishlistItem)(nil)).
		Set("status = ?", "deleted").
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to soft-delete wishlist item",
			"error", err.Error(),
			"wishlist_id", id,
		)
	}
	return err
}

func (r *wishlistRepository) ListByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.WishlistItem, int, error) {
	var items []*models.WishlistItem

	query := r.db.DB().NewSelect().
		Model(&items).
		Where("wi.user_id = ?", userID).
		Where("wi.status != ?", "deleted").
		Order("wi.created_at DESC")

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, nil
}

func (r *wishlistRepository) CountActiveByUserID(ctx context.Context, userID string) (int, error) {
	count, err := r.db.DB().NewSelect().
		Model((*models.WishlistItem)(nil)).
		Where("user_id = ?", userID).
		Where("status = ?", "active").
		Count(ctx)
	return count, err
}

func (r *wishlistRepository) DeleteAllByUserID(ctx context.Context, userID string) (int, error) {
	res, err := r.db.DB().NewUpdate().
		Model((*models.WishlistItem)(nil)).
		Set("status = ?", "deleted").
		Set("updated_at = ?", time.Now()).
		Where("user_id = ?", userID).
		Where("status != ?", "deleted").
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to delete all wishlist items for user",
			"error", err.Error(),
			"user_id", userID,
		)
		return 0, err
	}
	rowsAffected, _ := res.RowsAffected()
	return int(rowsAffected), nil
}

func (r *wishlistRepository) FindMatchingItems(ctx context.Context, listing *models.Listing) ([]*models.WishlistItem, error) {
	log := logger.FromContext(ctx)
	var items []*models.WishlistItem

	fmt.Printf("[WISHLIST-REPO] FindMatchingItems called\n")
	fmt.Printf("[WISHLIST-REPO] Listing: id=%s name=%s name_lower=%s game=%s seller=%s\n",
		listing.ID, listing.Name, strings.ToLower(listing.Name), listing.Game, listing.SellerID)
	fmt.Printf("[WISHLIST-REPO] Filters: ladder=%v hardcore=%v is_non_rotw=%v platforms=%v region=%s category=%s rarity=%s\n",
		listing.Ladder, listing.Hardcore, listing.IsNonRotw, listing.Platforms, listing.Region, listing.Category, listing.Rarity)

	log.Info("searching for matching wishlist items",
		"listing_id", listing.ID,
		"listing_name", listing.Name,
		"listing_name_lower", strings.ToLower(listing.Name),
		"listing_game", listing.Game,
		"listing_seller_id", listing.SellerID,
		"listing_ladder", listing.Ladder,
		"listing_hardcore", listing.Hardcore,
		"listing_platforms", listing.Platforms,
		"listing_region", listing.Region,
		"listing_category", listing.Category,
		"listing_rarity", listing.Rarity,
	)

	query := r.db.DB().NewSelect().
		Model(&items).
		Where("wi.status = ?", "active").
		Where("wi.game = ?", listing.Game).
		Where("LOWER(wi.name) = ?", strings.ToLower(listing.Name)).
		Where("wi.user_id != ?", listing.SellerID)

	// NULL filter fields act as wildcards — only filter when the wishlist field is NOT NULL
	query = query.Where("(wi.ladder IS NULL OR wi.ladder = ?)", listing.Ladder)
	query = query.Where("(wi.hardcore IS NULL OR wi.hardcore = ?)", listing.Hardcore)
	query = query.Where("(wi.is_non_rotw IS NULL OR wi.is_non_rotw = ?)", listing.IsNonRotw)
	query = query.Where("(wi.platform IS NULL OR wi.platform = ANY(?))", pgdialect.Array(listing.Platforms))
	query = query.Where("(wi.category IS NULL OR wi.category = ?)", listing.Category)
	query = query.Where("(wi.rarity IS NULL OR wi.rarity = ?)", listing.Rarity)

	fmt.Printf("[WISHLIST-REPO] Executing query...\n")
	err := query.Scan(ctx)
	if err != nil {
		fmt.Printf("[WISHLIST-REPO] Query ERROR: %v\n", err)
		log.Error("failed to find matching wishlist items",
			"error", err.Error(),
			"listing_id", listing.ID,
		)
		return nil, err
	}

	fmt.Printf("[WISHLIST-REPO] Query completed: found %d matching wishlist items\n", len(items))

	log.Info("wishlist query completed",
		"listing_id", listing.ID,
		"listing_name", listing.Name,
		"matching_items_count", len(items),
	)

	// Log each matching wishlist item found
	for i, item := range items {
		fmt.Printf("[WISHLIST-REPO] Match %d: id=%s user=%s name=%s category=%v rarity=%v criteria=%d\n",
			i, item.ID, item.UserID, item.Name, item.Category, item.Rarity, len(item.StatCriteria))
		log.Info("found matching wishlist item",
			"index", i,
			"wishlist_id", item.ID,
			"wishlist_user_id", item.UserID,
			"wishlist_name", item.Name,
			"wishlist_category", item.Category,
			"wishlist_rarity", item.Rarity,
			"stat_criteria_count", len(item.StatCriteria),
		)
	}

	return items, nil
}
