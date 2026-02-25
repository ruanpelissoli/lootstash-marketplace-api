package repository

import (
	"context"
	"fmt"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/uptrace/bun"
)

type stashItemRepository struct {
	db *database.BunDB
}

// NewStashItemRepository creates a new stash item repository
func NewStashItemRepository(db *database.BunDB) StashItemRepository {
	return &stashItemRepository{db: db}
}

func (r *stashItemRepository) Create(ctx context.Context, item *models.StashItem) error {
	_, err := r.db.DB().NewInsert().
		Model(item).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create stash item",
			"error", err.Error(),
			"user_id", item.UserID,
		)
	}
	return err
}

func (r *stashItemRepository) CreateBatch(ctx context.Context, items []*models.StashItem) error {
	if len(items) == 0 {
		return nil
	}
	_, err := r.db.DB().NewInsert().
		Model(&items).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to batch create stash items",
			"error", err.Error(),
			"count", len(items),
		)
	}
	return err
}

func (r *stashItemRepository) GetByID(ctx context.Context, id string) (*models.StashItem, error) {
	item := new(models.StashItem)
	err := r.db.DB().NewSelect().
		Model(item).
		Where("si.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *stashItemRepository) GetByIDAndUserID(ctx context.Context, id string, userID string) (*models.StashItem, error) {
	item := new(models.StashItem)
	err := r.db.DB().NewSelect().
		Model(item).
		Where("si.id = ?", id).
		Where("si.user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *stashItemRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.DB().NewDelete().
		Model((*models.StashItem)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to delete stash item",
			"error", err.Error(),
			"id", id,
		)
	}
	return err
}

func (r *stashItemRepository) ListByUserID(ctx context.Context, userID string, filter StashFilter) ([]*models.StashItem, int, error) {
	var items []*models.StashItem

	query := r.db.DB().NewSelect().
		Model(&items).
		Where("si.user_id = ?", userID)

	// Text search
	if filter.Query != "" {
		searchTerm := "%" + filter.Query + "%"
		query = query.Where("(si.name ILIKE ? OR si.item_type ILIKE ?)", searchTerm, searchTerm)
	}

	// Category filter
	if len(filter.Categories) > 0 {
		query = query.Where("si.category IN (?)", bun.In(filter.Categories))
	}

	// Rarity filter (multi-select)
	if len(filter.Rarities) > 0 {
		query = query.Where("si.rarity IN (?)", bun.In(filter.Rarities))
	}

	// Listed/unlisted status filter
	if filter.Listed != nil {
		if *filter.Listed {
			// Only items that have an active/pending listing
			query = query.Where("EXISTS (SELECT 1 FROM d2.listings l WHERE l.stash_item_id = si.id AND l.status IN ('active', 'pending'))")
		} else {
			// Only items that do NOT have an active/pending listing
			query = query.Where("NOT EXISTS (SELECT 1 FROM d2.listings l WHERE l.stash_item_id = si.id AND l.status IN ('active', 'pending'))")
		}
	}

	// Catalog item ID filter
	if filter.CatalogItemID != "" {
		query = query.Where("si.catalog_item_id = ?", filter.CatalogItemID)
	}

	// Affix filters (JSONB query on stats)
	for i, af := range filter.AffixFilters {
		if af.Code == "" {
			continue
		}
		alias := fmt.Sprintf("stat_%d", i)
		query = query.Join(
			fmt.Sprintf("JOIN LATERAL jsonb_array_elements(si.stats) AS %s ON TRUE", alias),
		)
		query = query.Where(fmt.Sprintf("%s->>'code' = ?", alias), af.Code)
		if af.MinValue != nil {
			query = query.Where(fmt.Sprintf("(%s->>'value')::int >= ?", alias), *af.MinValue)
		}
		if af.MaxValue != nil {
			query = query.Where(fmt.Sprintf("(%s->>'value')::int <= ?", alias), *af.MaxValue)
		}
	}

	// Count before pagination
	count, err := query.Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count stash items",
			"error", err.Error(),
			"user_id", userID,
		)
		return nil, 0, err
	}

	// Sorting
	sortBy := "si.created_at"
	switch filter.SortBy {
	case "name":
		sortBy = "COALESCE(si.name, si.item_type)"
	case "rarity":
		sortBy = "si.rarity"
	case "quantity":
		sortBy = "si.quantity"
	case "created_at":
		sortBy = "si.created_at"
	}
	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	query = query.OrderExpr(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	} else {
		query = query.Limit(20)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	err = query.Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to list stash items",
			"error", err.Error(),
			"user_id", userID,
		)
		return nil, 0, err
	}

	return items, count, nil
}

func (r *stashItemRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	count, err := r.db.DB().NewSelect().
		Model((*models.StashItem)(nil)).
		Where("user_id = ?", userID).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *stashItemRepository) DecrementQuantity(ctx context.Context, id string, amount int) error {
	result, err := r.db.DB().NewUpdate().
		Model((*models.StashItem)(nil)).
		Set("quantity = quantity - ?", amount).
		Set("updated_at = now()").
		Where("id = ?", id).
		Where("quantity >= ?", amount).
		Exec(ctx)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("insufficient quantity")
	}
	return nil
}

func (r *stashItemRepository) IncrementQuantity(ctx context.Context, id string, amount int) error {
	_, err := r.db.DB().NewUpdate().
		Model((*models.StashItem)(nil)).
		Set("quantity = quantity + ?", amount).
		Set("updated_at = now()").
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (r *stashItemRepository) DeleteIfZeroQuantity(ctx context.Context, id string) error {
	_, err := r.db.DB().NewDelete().
		Model((*models.StashItem)(nil)).
		Where("id = ?", id).
		Where("quantity <= 0").
		Exec(ctx)
	return err
}

// stackCategoryVariants returns both singular and plural forms for stackable categories.
func stackCategoryVariants(category string) []string {
	switch category {
	case "rune":
		return []string{"rune", "runes"}
	case "runes":
		return []string{"rune", "runes"}
	case "gem":
		return []string{"gem", "gems"}
	case "gems":
		return []string{"gem", "gems"}
	default:
		return []string{category}
	}
}

func (r *stashItemRepository) FindMatchingForStack(ctx context.Context, userID, name, itemType, category string, catalogItemID *string) (*models.StashItem, error) {
	item := new(models.StashItem)
	categories := stackCategoryVariants(category)
	query := r.db.DB().NewSelect().
		Model(item).
		Where("si.user_id = ?", userID).
		Where("si.item_type = ?", itemType).
		Where("si.category IN (?)", bun.In(categories))

	if name != "" {
		query = query.Where("si.name = ?", name)
	} else {
		query = query.Where("si.name IS NULL")
	}

	if catalogItemID != nil {
		query = query.Where("si.catalog_item_id = ?", *catalogItemID)
	} else {
		query = query.Where("si.catalog_item_id IS NULL")
	}

	err := query.OrderExpr("si.created_at ASC").Limit(1).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *stashItemRepository) GetListedAmountForStashItem(ctx context.Context, stashItemID string) (int, error) {
	var amount int
	err := r.db.DB().NewSelect().
		TableExpr("d2.listings").
		ColumnExpr("COALESCE(SUM(amount), 0)").
		Where("stash_item_id = ?", stashItemID).
		Where("status IN ('active', 'pending')").
		Scan(ctx, &amount)
	if err != nil {
		return 0, err
	}
	return amount, nil
}
