package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/games/d2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/storage"
)

// ErrInsufficientQuantity indicates not enough available quantity
var ErrInsufficientQuantity = fmt.Errorf("insufficient available quantity")

// ErrVisionNotConfigured indicates the Vision API is not configured
var ErrVisionNotConfigured = fmt.Errorf("vision import not configured")

// ErrNotListed indicates the stash item has no active listings
var ErrNotListed = fmt.Errorf("stash item is not listed")

const (
	maxImportFiles          = 10
	stashItemCacheTTL       = 15 * time.Minute
	stashFilterResultCacheTTL = 20 * time.Second
)

// StashService handles stash item business logic
type StashService struct {
	repo           repository.StashItemRepository
	listingRepo    repository.ListingRepository
	profileService *ProfileService
	listingService *ListingService
	redis          *cache.RedisClient
	invalidator    *cache.Invalidator
	storage        storage.Storage
	visionClient   *VisionClient
	catalogClient  *CatalogClient
}

// NewStashService creates a new stash service
func NewStashService(
	repo repository.StashItemRepository,
	listingRepo repository.ListingRepository,
	profileService *ProfileService,
	redis *cache.RedisClient,
	stor storage.Storage,
	anthropicAPIKey string,
	catalogAPIURL string,
) *StashService {
	s := &StashService{
		repo:           repo,
		listingRepo:    listingRepo,
		profileService: profileService,
		redis:          redis,
		invalidator:    cache.NewInvalidator(redis),
		storage:        stor,
	}
	if anthropicAPIKey != "" {
		s.visionClient = NewVisionClient(anthropicAPIKey)
	}
	if catalogAPIURL != "" {
		s.catalogClient = NewCatalogClient(catalogAPIURL)
	}
	return s
}

// SetListingService sets the listing service reference for cross-service calls
func (s *StashService) SetListingService(ls *ListingService) {
	s.listingService = ls
}

// Create adds a new item to the user's stash
func (s *StashService) Create(ctx context.Context, userID string, req *dto.CreateStashItemRequest) (*models.StashItem, error) {
	log := logger.FromContext(ctx)

	item := &models.StashItem{
		ID:        uuid.New().String(),
		UserID:    userID,
		ItemType:  req.ItemType,
		Rarity:    req.Rarity,
		Category:  req.Category,
		Stats:     req.Stats,
		Suffixes:  req.Suffixes,
		Runes:     req.Runes,
		Quantity:  1,
		Game:      req.Game,
		Ladder:    req.Ladder,
		Hardcore:  req.Hardcore,

		Platforms: req.Platforms,
		Region:    req.Region,
		Source:    "manual",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.Name != "" {
		item.Name = &req.Name
	}
	if req.ImageURL != "" {
		item.ImageURL = &req.ImageURL
	}
	if req.RuneOrder != "" {
		item.RuneOrder = &req.RuneOrder
	}
	if req.BaseItemCode != "" {
		item.BaseItemCode = &req.BaseItemCode
	}
	if req.BaseItemName != "" {
		item.BaseItemName = &req.BaseItemName
	}
	if req.CatalogItemID != "" {
		item.CatalogItemID = &req.CatalogItemID
	}

	// Only stackable items can have quantity > 1
	if req.Quantity != nil && *req.Quantity > 1 {
		if item.IsStackable() {
			item.Quantity = *req.Quantity
		}
		// Non-stackable items stay at quantity=1
	}

	if err := s.repo.Create(ctx, item); err != nil {
		log.Error("failed to create stash item", "error", err.Error(), "user_id", userID)
		return nil, err
	}

	_ = s.invalidator.InvalidateStashList(ctx, userID)

	log.Info("stash item created", "id", item.ID, "user_id", userID, "item_type", item.ItemType)
	return item, nil
}

// Delete removes a stash item or a specific portion of it.
// deleteListed controls which portion to remove for split (partially listed) stackable items:
//   - nil:   delete everything (cancel listing + hard delete) — default for non-split items
//   - true:  delete only the listed portion (cancel listing, decrement qty by listed amount)
//   - false: delete only the unlisted portion (decrement qty by unlisted amount)
//
// If the operation removes all remaining quantity, the item is fully deleted.
func (s *StashService) Delete(ctx context.Context, id string, userID string, deleteListed *bool) error {
	item, err := s.repo.GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		return err
	}

	log := logger.FromContext(ctx)

	// Fetch linked listing (1:1 relationship)
	var linkedListing *models.Listing
	listedAmount := 0
	if s.listingService != nil {
		listing, fetchErr := s.listingRepo.GetActiveListingByStashItemID(ctx, item.ID)
		if fetchErr == nil && listing != nil {
			linkedListing = listing
			listedAmount = listing.Amount
		}
	}

	unlistedAmount := item.Quantity - listedAmount

	cancelListing := func() {
		if linkedListing != nil {
			if err := s.listingService.Delete(ctx, linkedListing.ID, userID); err != nil {
				log.Error("failed to cancel linked listing",
					"error", err.Error(),
					"listing_id", linkedListing.ID,
					"stash_item_id", item.ID,
				)
			}
		}
	}

	switch {
	case deleteListed != nil && *deleteListed:
		// Delete the LISTED portion only: cancel listing, reduce quantity
		cancelListing()
		if unlistedAmount > 0 && listedAmount > 0 {
			_ = s.repo.DecrementQuantity(ctx, id, listedAmount)
		} else {
			// Everything was listed — delete entire item
			if err := s.repo.Delete(ctx, item.ID); err != nil {
				return err
			}
		}

	case deleteListed != nil && !*deleteListed:
		// Delete the UNLISTED portion only: reduce quantity, keep listed
		if listedAmount > 0 && unlistedAmount > 0 {
			_ = s.repo.DecrementQuantity(ctx, id, unlistedAmount)
		} else {
			// Nothing listed — delete entire item
			cancelListing()
			if err := s.repo.Delete(ctx, item.ID); err != nil {
				return err
			}
		}

	default:
		// No param — delete everything
		cancelListing()
		if err := s.repo.Delete(ctx, item.ID); err != nil {
			return err
		}
	}

	_ = s.invalidator.InvalidateStashItem(ctx, id)
	_ = s.invalidator.InvalidateStashList(ctx, userID)
	return nil
}

// Unlist cancels the active listing linked to a stash item without deleting the stash item itself
func (s *StashService) Unlist(ctx context.Context, id string, userID string) error {
	item, err := s.repo.GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		return err
	}

	if s.listingService == nil {
		return nil
	}

	listing, err := s.listingRepo.GetActiveListingByStashItemID(ctx, item.ID)
	if err != nil || listing == nil {
		return ErrNotListed
	}

	if err := s.listingService.Delete(ctx, listing.ID, userID); err != nil {
		return err
	}

	_ = s.invalidator.InvalidateStashList(ctx, userID)
	return nil
}

// AdjustQuantity increments or decrements the quantity of a stash item
func (s *StashService) AdjustQuantity(ctx context.Context, id string, userID string, delta int) (*models.StashItem, error) {
	item, err := s.repo.GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if !item.IsStackable() {
		return nil, ErrInvalidState
	}

	if delta > 0 {
		if err := s.repo.IncrementQuantity(ctx, id, delta); err != nil {
			return nil, err
		}
	} else if delta < 0 {
		if err := s.repo.DecrementQuantity(ctx, id, -delta); err != nil {
			return nil, err
		}
		// Delete if quantity reached zero
		_ = s.repo.DeleteIfZeroQuantity(ctx, id)
	}

	_ = s.invalidator.InvalidateStashItem(ctx, id)
	_ = s.invalidator.InvalidateStashList(ctx, userID)

	// Re-fetch to return updated item
	updated, err := s.repo.GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		// Item may have been deleted if quantity reached 0
		return nil, err
	}
	return updated, nil
}

// stashFilterCacheResult wraps stash items and count for cache serialization
type stashFilterCacheResult struct {
	Items []*models.StashItem `json:"items"`
	Count int                 `json:"count"`
}

// List returns paginated, filtered stash items for a user with caching
func (s *StashService) List(ctx context.Context, userID string, req *dto.StashFilterRequest) ([]*models.StashItem, int, error) {
	var affixFilters []repository.AffixFilter
	if req.AffixFilters != "" {
		var dtoFilters []dto.AffixFilter
		if json.Unmarshal([]byte(req.AffixFilters), &dtoFilters) == nil {
			for _, f := range dtoFilters {
				affixFilters = append(affixFilters, repository.AffixFilter{
					Code:     f.Code,
					MinValue: f.MinValue,
					MaxValue: f.MaxValue,
				})
			}
		}
	}

	filter := repository.StashFilter{
		Query:         req.Q,
		Categories:    parseCSV(req.Categories),
		Rarities:      parseCSV(req.Rarities),
		Listed:        req.Listed,
		CatalogItemID: req.CatalogItemID,
		AffixFilters:  affixFilters,
		SortBy:        req.SortBy,
		SortOrder:     req.SortOrder,
		Offset:        req.GetOffset(),
		Limit:         req.GetLimit(),
	}

	// Build cache key from user + filter params
	params := map[string]interface{}{
		"q":        filter.Query,
		"cats":     filter.Categories,
		"rarities": filter.Rarities,
		"listed":   filter.Listed,
		"catalog":  filter.CatalogItemID,
		"affixes":  filter.AffixFilters,
		"sortBy":   filter.SortBy,
		"sortOrd":  filter.SortOrder,
		"offset":   filter.Offset,
		"limit":    filter.Limit,
	}
	cacheKey := cache.StashFilterResultsKey(userID, cache.HashFilter(params))

	// Try cache first
	if cached, err := s.redis.Get(ctx, cacheKey); err == nil && cached != "" {
		var result stashFilterCacheResult
		if json.Unmarshal([]byte(cached), &result) == nil {
			return result.Items, result.Count, nil
		}
	}

	// Cache miss — query database
	items, count, err := s.repo.ListByUserID(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}

	// Cache the result
	result := stashFilterCacheResult{Items: items, Count: count}
	if data, marshalErr := json.Marshal(result); marshalErr == nil {
		_ = s.redis.Set(ctx, cacheKey, string(data), stashFilterResultCacheTTL)
	}

	return items, count, nil
}

// Search returns paginated, filtered stash items using JSON body params (POST)
func (s *StashService) Search(ctx context.Context, userID string, req *dto.SearchStashRequest) ([]*models.StashItem, int, error) {
	var affixFilters []repository.AffixFilter
	for _, f := range req.AffixFilters {
		affixFilters = append(affixFilters, repository.AffixFilter{
			Code:     f.Code,
			MinValue: f.MinValue,
			MaxValue: f.MaxValue,
		})
	}

	pag := dto.Pagination{Page: req.Page, PerPage: req.PerPage}

	filter := repository.StashFilter{
		Query:         req.Q,
		Categories:    req.Categories,
		Rarities:      req.Rarities,
		Listed:        req.Listed,
		CatalogItemID: req.CatalogItemID,
		AffixFilters:  affixFilters,
		SortBy:        req.SortBy,
		SortOrder:     req.SortOrder,
		Offset:        pag.GetOffset(),
		Limit:         pag.GetLimit(),
	}

	// Build cache key from user + filter params
	params := map[string]interface{}{
		"q":        filter.Query,
		"cats":     filter.Categories,
		"rarities": filter.Rarities,
		"listed":   filter.Listed,
		"catalog":  filter.CatalogItemID,
		"affixes":  filter.AffixFilters,
		"sortBy":   filter.SortBy,
		"sortOrd":  filter.SortOrder,
		"offset":   filter.Offset,
		"limit":    filter.Limit,
	}
	cacheKey := cache.StashFilterResultsKey(userID, cache.HashFilter(params))

	// Try cache first
	if cached, err := s.redis.Get(ctx, cacheKey); err == nil && cached != "" {
		var result stashFilterCacheResult
		if json.Unmarshal([]byte(cached), &result) == nil {
			return result.Items, result.Count, nil
		}
	}

	// Cache miss — query database
	items, count, err := s.repo.ListByUserID(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}

	// Cache the result
	result := stashFilterCacheResult{Items: items, Count: count}
	if data, marshalErr := json.Marshal(result); marshalErr == nil {
		_ = s.redis.Set(ctx, cacheKey, string(data), stashFilterResultCacheTTL)
	}

	return items, count, nil
}

// GetByID retrieves a stash item with ownership check and caching
func (s *StashService) GetByID(ctx context.Context, id string, userID string) (*models.StashItem, error) {
	// Try cache first
	cacheKey := cache.StashItemKey(id)
	if cached, err := s.redis.Get(ctx, cacheKey); err == nil && cached != "" {
		var item models.StashItem
		if json.Unmarshal([]byte(cached), &item) == nil && item.UserID == userID {
			return &item, nil
		}
	}

	// Cache miss — query database
	item, err := s.repo.GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(item); err == nil {
		_ = s.redis.Set(ctx, cacheKey, string(data), stashItemCacheTTL)
	}

	return item, nil
}

// CreateListingFromStash creates a listing from a stash item
func (s *StashService) CreateListingFromStash(ctx context.Context, userID string, req *dto.CreateListingFromStashRequest) (*models.Listing, error) {
	// Get stash item with ownership check
	item, err := s.repo.GetByIDAndUserID(ctx, req.StashItemID, userID)
	if err != nil {
		return nil, err
	}

	// Determine amount to list
	amount := 1
	if req.Amount != nil {
		amount = *req.Amount
	}

	// Check available quantity
	listedAmount, err := s.repo.GetListedAmountForStashItem(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	availableQty := item.Quantity - listedAmount
	if amount > availableQty {
		return nil, ErrInsufficientQuantity
	}

	// Build listing request from stash item
	listingReq := &dto.CreateListingRequest{
		Name:          item.GetDisplayName(),
		ItemType:      item.ItemType,
		Rarity:        item.Rarity,
		ImageURL:      item.GetImageURL(),
		Category:      item.Category,
		Stats:         item.Stats,
		Suffixes:      item.Suffixes,
		Runes:         item.Runes,
		RuneOrder:     item.GetRuneOrder(),
		BaseItemCode:  item.GetBaseItemCode(),
		BaseItemName:  item.GetBaseItemName(),
		CatalogItemID: item.GetCatalogItemID(),
		Amount:        &amount,
		Game:          item.Game,
		Ladder:        item.Ladder,
		Hardcore:      item.Hardcore,

		Platforms:     item.Platforms,
		Region:        item.Region,
	}

	if req.AskingFor != nil {
		listingReq.AskingFor = req.AskingFor
	}
	if req.AskingPrice != "" {
		listingReq.AskingPrice = req.AskingPrice
	}
	if req.Notes != "" {
		listingReq.Notes = req.Notes
	}

	// Link to existing stash item so ListingService.Create() skips auto-creation
	listingReq.StashItemID = item.ID

	// Create the listing via listing service (handles limit checks, wishlist matching, etc.)
	listing, err := s.listingService.Create(ctx, userID, listingReq)
	if err != nil {
		return nil, err
	}

	_ = s.invalidator.InvalidateStashList(ctx, userID)

	return listing, nil
}

// BulkImportPreview parses screenshots and returns preview data without saving to DB
func (s *StashService) BulkImportPreview(ctx context.Context, userID string, files []*multipart.FileHeader, game string, ladder, hardcore bool, region string, platforms []string) (*dto.BulkImportPreviewResponse, error) {
	// TODO: Re-enable premium gate when ready
	// profile, err := s.profileService.GetByID(ctx, userID)
	// if err != nil {
	// 	return nil, err
	// }
	// if !profile.IsPremium {
	// 	return nil, ErrPremiumRequired
	// }

	if s.visionClient == nil || !s.visionClient.IsConfigured() {
		return nil, ErrVisionNotConfigured
	}

	if len(files) > maxImportFiles {
		return nil, fmt.Errorf("maximum %d files allowed", maxImportFiles)
	}

	log := logger.FromContext(ctx)

	// fileResult holds the outcome of processing a single file (either an item or an error)
	type fileResult struct {
		item *dto.ParsedStashItem
		err  *dto.ImportError
	}

	results := make([]fileResult, len(files))
	var wg sync.WaitGroup

	for i, fh := range files {
		wg.Add(1)
		go func(idx int, fh *multipart.FileHeader) {
			defer wg.Done()

			ct := fh.Header.Get("Content-Type")
			if !isImageContentType(ct) {
				results[idx] = fileResult{err: &dto.ImportError{
					Index:   idx,
					File:    fh.Filename,
					Message: "Invalid file type. Only JPEG, PNG, GIF, and WebP images are supported.",
				}}
				return
			}

			// Read file data
			f, err := fh.Open()
			if err != nil {
				results[idx] = fileResult{err: &dto.ImportError{
					Index:   idx,
					File:    fh.Filename,
					Message: "Failed to read file",
				}}
				return
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				results[idx] = fileResult{err: &dto.ImportError{
					Index:   idx,
					File:    fh.Filename,
					Message: "Failed to read file data",
				}}
				return
			}

			// Catalog-backed flow: Vision identifies → catalog provides full data
			if s.catalogClient != nil && s.catalogClient.IsConfigured() {
				identification, err := s.visionClient.IdentifyItem(ctx, data, ct)
				if err != nil {
					results[idx] = fileResult{err: &dto.ImportError{
						Index:   idx,
						File:    fh.Filename,
						Message: fmt.Sprintf("Failed to identify item: %s", err.Error()),
					}}
					return
				}

				// Magic/rare items have random stats — catalog doesn't have these as named items.
				// Fall back to full Vision parse for stats, then enrich with catalog base data.
				if identification.TypeHint == "magic" || identification.TypeHint == "rare" {
					parsed, err := s.parseMagicRareItem(ctx, data, ct, identification)
					if err != nil {
						results[idx] = fileResult{err: &dto.ImportError{
							Index:   idx,
							File:    fh.Filename,
							Message: fmt.Sprintf("Failed to parse item: %s", err.Error()),
						}}
						return
					}
					results[idx] = fileResult{item: parsed}
					return
				}

				parsed, err := s.catalogLookup(ctx, identification)
				if err != nil {
					// Catalog has no match — fall back to full Vision parse
					log.Warn("catalog lookup failed, falling back to vision",
						"name", identification.Name,
						"error", err.Error(),
					)
					fallback, fbErr := s.visionClient.ParseItemScreenshot(ctx, data, ct)
					if fbErr != nil {
						results[idx] = fileResult{err: &dto.ImportError{
							Index:   idx,
							File:    fh.Filename,
							Message: fmt.Sprintf("Failed to parse item: %s", fbErr.Error()),
						}}
						return
					}
					// Upload screenshot so the fallback item gets an image URL
					var fbImageURL string
					if s.storage != nil {
						path := fmt.Sprintf("stash/%s/%s-%s", userID, uuid.New().String(), fh.Filename)
						url, uploadErr := s.storage.UploadImage(ctx, path, data, ct)
						if uploadErr == nil {
							fbImageURL = url
						}
					}

					fbParsed := dto.ParsedStashItem{
						ItemType:     fallback.ItemType,
						Rarity:       fallback.Rarity,
						Category:     fallback.Category,
						Stats:        normalizeStatCodes(fallback.Stats),
						BaseItemCode: fallback.BaseItemCode,
						BaseItemName: fallback.BaseItemName,
						Quantity:     identification.Quantity,
						IsStackable:  fallback.IsStackable,
						ImageURL:     fbImageURL,
					}

					// Prefer identification.Name (from IdentifyItem, tuned for name accuracy)
					// over fallback.Name (from ParseItemScreenshot, which often returns empty)
					if identification.Name != "" {
						fbParsed.Name = identification.Name
					} else if fallback.Name != "" {
						fbParsed.Name = fallback.Name
					}

					// Apply type hint as rarity when available (IdentifyItem is reliable for rarity)
					if identification.TypeHint != "" {
						fbParsed.Rarity = identification.TypeHint
					}

					results[idx] = fileResult{item: &fbParsed}
					return
				}

				// Read blue stats from the screenshot via Vision (preserves screenshot order,
				// only magical properties). Annotate with catalog range info for isVariable.
				visionResult, visionErr := s.visionClient.ParseItemScreenshot(ctx, data, ct)
				if visionErr == nil && visionResult.Stats != nil {
					parsed.Stats = annotateWithCatalogRanges(visionResult.Stats, parsed.Stats)
				}

				// Normalize any remaining Vision-guessed codes (e.g. "skilltab" → semantic)
				parsed.Stats = normalizeStatCodes(parsed.Stats)

				// Superior, magic, and rare items: all stats are variable
				if parsed.Rarity == "superior" || parsed.Rarity == "magic" || parsed.Rarity == "rare" {
					parsed.Stats = markAllStatsVariable(parsed.Stats)
				}

				// Runeword names in the catalog may use internal codes (e.g. "HTMLRuneword_Spirit").
				// Override with the name Vision read from the screenshot.
				if identification.TypeHint == "runeword" && identification.Name != "" {
					parsed.Name = identification.Name
				}

				// Set quantity from Vision identification
				parsed.Quantity = identification.Quantity
				// Set isStackable based on category
				parsed.IsStackable = isStackableCategory(parsed.Category)

				results[idx] = fileResult{item: parsed}
				return
			}

			// Fallback: Vision-only flow (no catalog configured)
			var imageURL string
			if s.storage != nil {
				path := fmt.Sprintf("stash/%s/%s-%s", userID, uuid.New().String(), fh.Filename)
				url, err := s.storage.UploadImage(ctx, path, data, ct)
				if err == nil {
					imageURL = url
				}
			}

			result, err := s.visionClient.ParseItemScreenshot(ctx, data, ct)
			if err != nil {
				results[idx] = fileResult{err: &dto.ImportError{
					Index:   idx,
					File:    fh.Filename,
					Message: fmt.Sprintf("Failed to parse item: %s", err.Error()),
				}}
				return
			}

			parsed := dto.ParsedStashItem{
				ItemType:     result.ItemType,
				Rarity:       result.Rarity,
				Category:     result.Category,
				Stats:        normalizeStatCodes(result.Stats),
				BaseItemCode: result.BaseItemCode,
				BaseItemName: result.BaseItemName,
				ImageURL:     imageURL,
				Quantity:     result.Quantity,
				IsStackable:  result.IsStackable,
			}
			if result.Name != "" {
				parsed.Name = result.Name
			}

			results[idx] = fileResult{item: &parsed}
		}(i, fh)
	}

	wg.Wait()

	// Collect results in original file order
	resp := &dto.BulkImportPreviewResponse{
		Items:  make([]dto.ParsedStashItem, 0, len(files)),
		Errors: make([]dto.ImportError, 0),
	}
	for _, r := range results {
		if r.err != nil {
			resp.Errors = append(resp.Errors, *r.err)
		} else if r.item != nil {
			resp.Items = append(resp.Items, *r.item)
		}
	}

	return resp, nil
}

// catalogLookup searches the catalog API and returns a fully populated ParsedStashItem
func (s *StashService) catalogLookup(ctx context.Context, identification *VisionIdentification) (*dto.ParsedStashItem, error) {
	log := logger.FromContext(ctx)

	// 1. Strip "Superior " prefix from base items before searching catalog
	searchName := identification.Name
	isSuperior := strings.HasPrefix(searchName, "Superior ")
	if isSuperior {
		searchName = strings.TrimPrefix(searchName, "Superior ")
	}
	searchName = normalizeSearchQuery(searchName)

	searchResp, err := s.catalogClient.Search(ctx, searchName)
	if err != nil {
		return nil, fmt.Errorf("catalog search failed: %w", err)
	}
	if len(searchResp.Items) == 0 {
		return nil, fmt.Errorf("no catalog results for: %s", searchName)
	}

	// 2. Find best match — requires an exact name match
	best, found := findBestMatch(searchResp.Items, searchName, identification.TypeHint)
	if !found {
		return nil, fmt.Errorf("no exact catalog match for: %s", searchName)
	}

	// 3. Map search result type to lowercase for detail fetch
	mappedType := strings.ToLower(best.Type)

	// 4. Fetch full detail
	detail, err := s.catalogClient.GetItemDetail(ctx, mappedType, best.ID.String())
	if err != nil {
		return nil, fmt.Errorf("catalog detail fetch failed: %w", err)
	}

	// 5. Map detail to ParsedStashItem
	parsed, err := s.catalogClient.mapDetailToParsedItem(mappedType, detail, best)
	if err != nil {
		return nil, fmt.Errorf("failed to map catalog detail: %w", err)
	}

	// If original item was "Superior", override rarity
	if isSuperior && mappedType == "base" {
		parsed.Rarity = "superior"
	}

	// 6. Runeword base item resolution
	if mappedType == "runeword" && identification.BaseName != "" {
		baseSearch, err := s.catalogClient.Search(ctx, normalizeSearchQuery(identification.BaseName))
		if err == nil && len(baseSearch.Items) > 0 {
			for _, item := range baseSearch.Items {
				if strings.EqualFold(item.Type, "Base") {
					baseDetail, err := s.catalogClient.GetItemDetail(ctx, "base", item.ID.String())
					if err == nil {
						var wrapper map[string]json.RawMessage
						if json.Unmarshal(baseDetail, &wrapper) == nil {
							innerData := baseDetail
							if bd, ok := wrapper["base"]; ok {
								innerData = bd
							}
							var base catalogBaseDetail
							if json.Unmarshal(innerData, &base) == nil {
								parsed.BaseItemCode = base.Code
								parsed.BaseItemName = base.Name
								parsed.ItemType = base.Name
							}
						}
					}
					break
				}
			}
		} else if err != nil {
			log.Warn("failed to resolve runeword base item", "baseName", identification.BaseName, "error", err.Error())
		}
	}

	// 7. Socket resolution (non-runeword socketed items)
	if len(identification.Sockets) > 0 && mappedType != "runeword" {
		s.resolveSocketedItems(ctx, parsed, identification.Sockets)
	}

	return parsed, nil
}

// parseMagicRareItem handles magic/rare items by using full Vision parse for stats,
// then enriching with catalog base item data for category/image.
func (s *StashService) parseMagicRareItem(ctx context.Context, imageData []byte, contentType string, identification *VisionIdentification) (*dto.ParsedStashItem, error) {
	log := logger.FromContext(ctx)

	// Full Vision parse to get stats (magic/rare stats are random, only Vision can read them)
	result, err := s.visionClient.ParseItemScreenshot(ctx, imageData, contentType)
	if err != nil {
		return nil, fmt.Errorf("vision parse failed: %w", err)
	}

	// Use the full display name from IdentifyItem (e.g. "Fine Small Charm of Vita")
	// since ParseItemScreenshot typically returns null for magic/rare names.
	name := identification.Name
	if name == "" {
		name = result.Name
	}

	// Use identification.TypeHint for rarity — IdentifyItem is purpose-built for
	// color/rarity detection, while ParseItemScreenshot can misclassify (e.g. rare as magic).
	rarity := identification.TypeHint
	if rarity == "" {
		rarity = result.Rarity
	}

	parsed := &dto.ParsedStashItem{
		Name:         name,
		ItemType:     result.ItemType,
		Rarity:       rarity,
		Category:     result.Category,
		Stats:        markAllStatsVariable(normalizeStatCodes(result.Stats)),
		BaseItemCode: result.BaseItemCode,
		BaseItemName: result.BaseItemName,
		Quantity:     identification.Quantity,
		IsStackable:  result.IsStackable,
	}

	// Try to enrich with catalog base item data (category, image)
	baseName := identification.BaseName
	if baseName == "" {
		baseName = result.ItemType // fallback: Vision's itemType is often the base name
	}
	if baseName != "" && s.catalogClient != nil {
		baseItem := s.findBaseItem(ctx, baseName)
		// Fallback: if baseName didn't match and result.ItemType is different, try that
		if baseItem == nil && result.ItemType != "" && !strings.EqualFold(baseName, result.ItemType) {
			baseItem = s.findBaseItem(ctx, result.ItemType)
		}
		if baseItem != nil {
			parsed.Category = resolveCategory(baseItem.Category, baseItem.Subcategory)
			parsed.CatalogItemID = baseItem.ID.String()
			parsed.CatalogType = "base"
			if baseItem.ImageURL != "" {
				parsed.ImageURL = baseItem.ImageURL
			}
			// Fetch full base detail for code
			baseDetail, err := s.catalogClient.GetItemDetail(ctx, "base", baseItem.ID.String())
			if err == nil {
				var wrapper map[string]json.RawMessage
				if json.Unmarshal(baseDetail, &wrapper) == nil {
					innerData := baseDetail
					if bd, ok := wrapper["base"]; ok {
						innerData = bd
					}
					var base catalogBaseDetail
					if json.Unmarshal(innerData, &base) == nil {
						parsed.BaseItemCode = base.Code
						parsed.BaseItemName = base.Name
						parsed.ItemType = base.Name
					}
				}
			}
		} else {
			log.Warn("no base item found for magic/rare icon",
				"baseName", baseName,
				"itemType", result.ItemType,
				"name", identification.Name,
			)
		}
	}

	return parsed, nil
}

// findBaseItem searches the catalog for a base item by name, using type=base filter.
// Uses exact match first, then normalized name matching, then falls back to first Base result.
func (s *StashService) findBaseItem(ctx context.Context, query string) *CatalogSearchResult {
	query = normalizeSearchQuery(query)
	searchResp, err := s.catalogClient.SearchBases(ctx, query)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to look up base item", "query", query, "error", err.Error())
		return nil
	}

	// Exact name match (preferred)
	for i, item := range searchResp.Items {
		if strings.EqualFold(item.Type, "Base") && strings.EqualFold(item.Name, query) {
			return &searchResp.Items[i]
		}
	}
	// Normalized name match (handles punctuation variants)
	normQuery := normalizeName(query)
	for i, item := range searchResp.Items {
		if strings.EqualFold(item.Type, "Base") && normalizeName(item.Name) == normQuery {
			return &searchResp.Items[i]
		}
	}
	// Fallback: first Base result (search API already filtered by query relevance)
	for i, item := range searchResp.Items {
		if strings.EqualFold(item.Type, "Base") {
			return &searchResp.Items[i]
		}
	}
	return nil
}

// resolveSocketedItems looks up socketed rune/gem names in the catalog and populates the Runes field
func (s *StashService) resolveSocketedItems(ctx context.Context, parsed *dto.ParsedStashItem, sockets []string) {
	var runes []dto.RuneInfo
	for _, socketName := range sockets {
		searchResp, err := s.catalogClient.Search(ctx, socketName)
		if err != nil || len(searchResp.Items) == 0 {
			continue
		}
		for _, item := range searchResp.Items {
			if strings.EqualFold(item.Type, "Rune") || strings.EqualFold(item.Type, "Gem") {
				runes = append(runes, dto.RuneInfo{
					Code:     "",
					Name:     item.Name,
					ImageURL: item.ImageURL,
				})
				break
			}
		}
	}
	if len(runes) > 0 {
		data, err := json.Marshal(runes)
		if err == nil {
			parsed.Runes = data
		}
	}
}

// findBestMatch selects the best catalog search result for the identified item.
// Returns (result, true) if a match was found, or (zero, false) if no name matched.
func findBestMatch(items []CatalogSearchResult, name string, typeHint string) (CatalogSearchResult, bool) {
	// Exact name + type match (preferred)
	for _, item := range items {
		if strings.EqualFold(item.Name, name) && strings.EqualFold(item.Type, typeHint) {
			return item, true
		}
	}
	// Exact name match (any type)
	for _, item := range items {
		if strings.EqualFold(item.Name, name) {
			return item, true
		}
	}
	// Normalized name + type match (handles apostrophe/dash/whitespace variants)
	normName := normalizeName(name)
	for _, item := range items {
		if normalizeName(item.Name) == normName && strings.EqualFold(item.Type, typeHint) {
			return item, true
		}
	}
	// Normalized name match (any type)
	for _, item := range items {
		if normalizeName(item.Name) == normName {
			return item, true
		}
	}
	// Contains match + type match (catalog may return qualified names like "Rainbow Facet (Die)")
	for _, item := range items {
		normItem := normalizeName(item.Name)
		if strings.EqualFold(item.Type, typeHint) && (strings.Contains(normItem, normName) || strings.Contains(normName, normItem)) {
			return item, true
		}
	}
	// Single result of matching type (search API returns by relevance — trust the top result)
	if typeHint != "" {
		var match *CatalogSearchResult
		count := 0
		for i, item := range items {
			if strings.EqualFold(item.Type, typeHint) {
				match = &items[i]
				count++
			}
		}
		if count == 1 {
			return *match, true
		}
	}
	// No match — do NOT fall back to an unrelated item
	return CatalogSearchResult{}, false
}

// normalizeName lowercases and normalizes punctuation variants for fuzzy name matching.
// Collapses curly/straight apostrophes, various dashes, and extra whitespace.
func normalizeName(s string) string {
	s = strings.ToLower(s)
	// Normalize apostrophe variants (curly ', ', `, ʼ) to straight '
	for _, ch := range []string{"\u2018", "\u2019", "\u0060", "\u02BC"} {
		s = strings.ReplaceAll(s, ch, "'")
	}
	// Normalize dash variants (en-dash, em-dash, minus) to hyphen
	for _, ch := range []string{"\u2013", "\u2014", "\u2212"} {
		s = strings.ReplaceAll(s, ch, "-")
	}
	// Collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// normalizeSearchQuery normalizes punctuation in search queries without lowercasing,
// so catalog API searches work even when Vision returns curly apostrophes or fancy dashes.
func normalizeSearchQuery(s string) string {
	// Curly apostrophes → straight
	for _, ch := range []string{"\u2018", "\u2019", "\u0060", "\u02BC"} {
		s = strings.ReplaceAll(s, ch, "'")
	}
	// Dash variants → hyphen
	for _, ch := range []string{"\u2013", "\u2014", "\u2212"} {
		s = strings.ReplaceAll(s, ch, "-")
	}
	return s
}

// isStackableCategory returns true if the category represents a stackable item type
func isStackableCategory(category string) bool {
	switch strings.ToLower(category) {
	case "runes", "rune", "gems", "gem", "misc":
		return true
	default:
		return false
	}
}

// annotateWithCatalogRanges uses Vision-read stats as the base (preserving screenshot
// order, blue stats only) and enriches them with isVariable/min/max from matching
// catalog affixes. Matching is done first by code (case-insensitive), then by
// display text keyword overlap as a fallback.
func annotateWithCatalogRanges(visionStats json.RawMessage, catalogStats json.RawMessage) json.RawMessage {
	var vision []catalogStatEntry
	if err := json.Unmarshal(visionStats, &vision); err != nil {
		return visionStats
	}

	if catalogStats == nil {
		return visionStats
	}

	var catalog []catalogStatEntry
	if json.Unmarshal(catalogStats, &catalog) == nil {
		// Build catalog lookup by lowercase code
		catalogByCode := make(map[string]*catalogStatEntry)
		for i := range catalog {
			catalogByCode[strings.ToLower(catalog[i].Code)] = &catalog[i]
		}

		for i := range vision {
			// 1. Try case-insensitive code match — also adopt catalog's authoritative code
			if cat, ok := catalogByCode[strings.ToLower(vision[i].Code)]; ok {
				vision[i].Code = cat.Code
				vision[i].IsVariable = cat.IsVariable
				vision[i].Min = cat.Min
				vision[i].Max = cat.Max
				continue
			}

			// 2. Fallback: match by display text keyword overlap — adopt catalog code + ranges
			if cat := matchByDisplayText(vision[i].DisplayText, catalog); cat != nil {
				vision[i].Code = cat.Code
				vision[i].IsVariable = cat.IsVariable
				vision[i].Min = cat.Min
				vision[i].Max = cat.Max
			}
		}
	}

	data, err := json.Marshal(vision)
	if err != nil {
		return visionStats
	}
	return data
}

// matchByDisplayText finds a catalog stat whose display text shares key terms
// with the given vision display text. Returns nil if no match found.
func matchByDisplayText(visionText string, catalog []catalogStatEntry) *catalogStatEntry {
	vNorm := normalizeStatText(visionText)
	if vNorm == "" {
		return nil
	}

	for i := range catalog {
		cNorm := normalizeStatText(catalog[i].DisplayText)
		if cNorm == "" {
			continue
		}
		// Check if one contains the other (handles "+35% Faster Cast Rate" vs "Faster Cast Rate")
		if strings.Contains(vNorm, cNorm) || strings.Contains(cNorm, vNorm) {
			return &catalog[i]
		}
	}
	return nil
}

// normalizeStatText strips numbers, symbols, and extra whitespace from a stat
// description, leaving only the alphabetic keywords for comparison.
func normalizeStatText(text string) string {
	var b strings.Builder
	prevSpace := true
	for _, r := range strings.ToLower(text) {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
			prevSpace = false
		} else if !prevSpace {
			b.WriteRune(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// markAllStatsVariable sets isVariable=true on every stat entry in the JSON array.
// Used for superior, magic, and rare items where all stats are considered variable.
func markAllStatsVariable(stats json.RawMessage) json.RawMessage {
	if stats == nil {
		return nil
	}
	var entries []catalogStatEntry
	if err := json.Unmarshal(stats, &entries); err != nil {
		return stats
	}
	for i := range entries {
		entries[i].IsVariable = true
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return stats
	}
	return data
}

// normalizeStatCodes scans a stats JSON array and replaces Vision-guessed codes
// with their semantic equivalents. For "skilltab" entries, infers the semantic code
// from displayText via d2.InferSemanticStatCode. No-op for non-skilltab codes.
func normalizeStatCodes(stats json.RawMessage) json.RawMessage {
	if stats == nil {
		return nil
	}
	var entries []catalogStatEntry
	if err := json.Unmarshal(stats, &entries); err != nil {
		return stats
	}
	changed := false
	for i := range entries {
		if semantic := d2.InferSemanticStatCode(entries[i].Code, entries[i].DisplayText); semantic != "" {
			entries[i].Code = semantic
			changed = true
		}
	}
	if !changed {
		return stats
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return stats
	}
	return data
}

// BulkImportConfirm saves reviewed/corrected items to the database
func (s *StashService) BulkImportConfirm(ctx context.Context, userID string, items []dto.CreateStashItemRequest) ([]*models.StashItem, error) {
	// TODO: Re-enable premium gate when ready
	// profile, err := s.profileService.GetByID(ctx, userID)
	// if err != nil {
	// 	return nil, err
	// }
	// if !profile.IsPremium {
	// 	return nil, ErrPremiumRequired
	// }

	log := logger.FromContext(ctx)

	created := make([]*models.StashItem, 0, len(items))

	for _, req := range items {
		item := &models.StashItem{
			ID:        uuid.New().String(),
			UserID:    userID,
			ItemType:  req.ItemType,
			Rarity:    req.Rarity,
			Category:  req.Category,
			Stats:     req.Stats,
			Suffixes:  req.Suffixes,
			Runes:     req.Runes,
			Quantity:  1,
			Game:      req.Game,
			Ladder:    req.Ladder,
			Hardcore:  req.Hardcore,
	
			Platforms: req.Platforms,
			Region:    req.Region,
			Source:    "import",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if req.Name != "" {
			item.Name = &req.Name
		}
		if req.ImageURL != "" {
			item.ImageURL = &req.ImageURL
		}
		if req.RuneOrder != "" {
			item.RuneOrder = &req.RuneOrder
		}
		if req.BaseItemCode != "" {
			item.BaseItemCode = &req.BaseItemCode
		}
		if req.BaseItemName != "" {
			item.BaseItemName = &req.BaseItemName
		}
		if req.CatalogItemID != "" {
			item.CatalogItemID = &req.CatalogItemID
		}
		if req.Quantity != nil && *req.Quantity > 1 && item.IsStackable() {
			item.Quantity = *req.Quantity
		}

		created = append(created, item)
	}

	if len(created) > 0 {
		if err := s.repo.CreateBatch(ctx, created); err != nil {
			log.Error("failed to batch create stash items", "error", err.Error(), "user_id", userID)
			return nil, err
		}
	}

	_ = s.invalidator.InvalidateStashList(ctx, userID)

	return created, nil
}

// HandleTradeCompletion processes stash updates when a trade is completed
func (s *StashService) HandleTradeCompletion(ctx context.Context, trade *models.Trade, listing *models.Listing, offer *models.Offer) {
	log := logger.FromContext(ctx)

	// 1. Seller's item: if listing was from stash, decrement seller's stash
	if listing.StashItemID != nil {
		if err := s.repo.DecrementQuantity(ctx, *listing.StashItemID, listing.Amount); err != nil {
			log.Error("failed to decrement seller stash quantity",
				"error", err.Error(),
				"stash_item_id", *listing.StashItemID,
			)
		}
		_ = s.repo.DeleteIfZeroQuantity(ctx, *listing.StashItemID)
		_ = s.invalidator.InvalidateStashItem(ctx, *listing.StashItemID)
		_ = s.invalidator.InvalidateStashList(ctx, trade.SellerID)
	}

	// 2. Buyer receives: create stash item for buyer from listing data
	buyerItem := &models.StashItem{
		ID:           uuid.New().String(),
		UserID:       trade.BuyerID,
		ItemType:     listing.ItemType,
		Rarity:       listing.Rarity,
		Category:     listing.Category,
		Stats:        listing.Stats,
		Suffixes:     listing.Suffixes,
		Runes:        listing.Runes,
		RuneOrder:    listing.RuneOrder,
		BaseItemCode: listing.BaseItemCode,
		BaseItemName: listing.BaseItemName,
		Quantity:     listing.Amount,
		Game:         listing.Game,
		Ladder:       listing.Ladder,
		Hardcore:     listing.Hardcore,

		Platforms:    listing.Platforms,
		Region:       listing.Region,
		Source:       "trade",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	// Use listing name as the item name
	listingName := listing.Name
	buyerItem.Name = &listingName
	buyerItem.ImageURL = listing.ImageURL
	buyerItem.CatalogItemID = listing.CatalogItemID

	stacked := false
	if buyerItem.IsStackable() {
		existing, err := s.repo.FindMatchingForStack(ctx, buyerItem.UserID, buyerItem.GetName(), buyerItem.ItemType, buyerItem.Category, buyerItem.CatalogItemID)
		if err == nil && existing != nil {
			if err := s.repo.IncrementQuantity(ctx, existing.ID, buyerItem.Quantity); err != nil {
				log.Error("failed to increment buyer stash quantity",
					"error", err.Error(),
					"stash_item_id", existing.ID,
				)
			} else {
				stacked = true
			}
		}
	}
	if !stacked {
		if err := s.repo.Create(ctx, buyerItem); err != nil {
			log.Error("failed to create buyer stash item from trade",
				"error", err.Error(),
				"trade_id", trade.ID,
				"buyer_id", trade.BuyerID,
			)
		}
	}
	_ = s.invalidator.InvalidateStashList(ctx, trade.BuyerID)

	// 3 & 4. Process offered items — parse JSONB for stash-linked items
	if offer != nil && len(offer.OfferedItems) > 0 {
		s.processOfferedItemsOnTradeComplete(ctx, trade, offer)
	}
}

// offeredItemWithStash extends the basic offered item with optional stash link
type offeredItemWithStash struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	ImageURL      string  `json:"imageUrl,omitempty"`
	Quantity      int     `json:"quantity"`
	StashItemID   string  `json:"stashItemId,omitempty"`
	Category      string  `json:"category,omitempty"`
	Rarity        string  `json:"rarity,omitempty"`
	CatalogItemID *string `json:"catalogItemId,omitempty"`
}

func (s *StashService) processOfferedItemsOnTradeComplete(ctx context.Context, trade *models.Trade, offer *models.Offer) {
	log := logger.FromContext(ctx)

	var items []offeredItemWithStash
	if err := json.Unmarshal(offer.OfferedItems, &items); err != nil {
		log.Error("failed to parse offered items", "error", err.Error(), "offer_id", offer.ID)
		return
	}

	for _, item := range items {
		// If linked to a stash item, fetch it to get authoritative fields
		var sourceStash *models.StashItem
		if item.StashItemID != "" {
			if src, err := s.repo.GetByID(ctx, item.StashItemID); err == nil {
				sourceStash = src
			}

			if err := s.repo.DecrementQuantity(ctx, item.StashItemID, item.Quantity); err != nil {
				log.Error("failed to decrement buyer stash for offered item",
					"error", err.Error(),
					"stash_item_id", item.StashItemID,
				)
			}
			_ = s.repo.DeleteIfZeroQuantity(ctx, item.StashItemID)
			_ = s.invalidator.InvalidateStashItem(ctx, item.StashItemID)
		}

		// Build seller item — prefer source stash fields, fall back to JSON fields
		category := item.Category
		rarity := item.Rarity
		var catalogItemID *string = item.CatalogItemID
		if sourceStash != nil {
			if category == "" {
				category = sourceStash.Category
			}
			if rarity == "" {
				rarity = sourceStash.Rarity
			}
			if catalogItemID == nil {
				catalogItemID = sourceStash.CatalogItemID
			}
		}

		// Normalize category from offered item type as last resort
		if category == "" {
			switch item.Type {
			case "rune":
				category = "runes"
			case "gem":
				category = "gems"
			default:
				category = item.Type
			}
		}

		// Create stash item for seller from offered item
		sellerItem := &models.StashItem{
			ID:            uuid.New().String(),
			UserID:        trade.SellerID,
			ItemType:      item.Type,
			Category:      category,
			Rarity:        rarity,
			CatalogItemID: catalogItemID,
			Quantity:      item.Quantity,
			Game:          "diablo2",
			Source:        "trade",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if item.Name != "" {
			sellerItem.Name = &item.Name
		}
		if item.ImageURL != "" {
			sellerItem.ImageURL = &item.ImageURL
		}

		// Try stacking for stackable items
		stacked := false
		if sellerItem.IsStackable() {
			existing, err := s.repo.FindMatchingForStack(ctx, sellerItem.UserID, sellerItem.GetName(), sellerItem.ItemType, sellerItem.Category, sellerItem.CatalogItemID)
			if err == nil && existing != nil {
				if err := s.repo.IncrementQuantity(ctx, existing.ID, sellerItem.Quantity); err != nil {
					log.Error("failed to increment seller stash quantity",
						"error", err.Error(),
						"stash_item_id", existing.ID,
					)
				} else {
					stacked = true
				}
			}
		}
		if !stacked {
			if err := s.repo.Create(ctx, sellerItem); err != nil {
				log.Error("failed to create seller stash item from offered items",
					"error", err.Error(),
					"trade_id", trade.ID,
					"seller_id", trade.SellerID,
				)
			}
		}
	}

	_ = s.invalidator.InvalidateStashList(ctx, trade.BuyerID)
	_ = s.invalidator.InvalidateStashList(ctx, trade.SellerID)
}

// getLinkedListing fetches the active listing for a stash item (1:1 relationship).
func (s *StashService) getLinkedListing(ctx context.Context, stashItemID string) *models.Listing {
	listing, err := s.listingRepo.GetActiveListingByStashItemID(ctx, stashItemID)
	if err != nil {
		return nil
	}
	return listing
}

// ToResponse converts a stash item model to a DTO response
func (s *StashService) ToResponse(ctx context.Context, item *models.StashItem) *dto.StashItemResponse {
	listing := s.getLinkedListing(ctx, item.ID)
	return s.buildResponse(item, item.Quantity, listing != nil, listing)
}

// ToListResponses converts a stash item into 1 or 2 response rows for list views.
// If part of the quantity is listed, it splits into an unlisted row and a listed row.
func (s *StashService) ToListResponses(ctx context.Context, item *models.StashItem) []*dto.StashItemResponse {
	listing := s.getLinkedListing(ctx, item.ID)

	if listing == nil {
		return []*dto.StashItemResponse{s.buildResponse(item, item.Quantity, false, nil)}
	}

	listedAmount := listing.Amount
	unlisted := item.Quantity - listedAmount

	// All listed — single row
	if unlisted <= 0 {
		return []*dto.StashItemResponse{s.buildResponse(item, listedAmount, true, listing)}
	}

	// Split into two rows: unlisted first, then listed
	return []*dto.StashItemResponse{
		s.buildResponse(item, unlisted, false, nil),
		s.buildResponse(item, listedAmount, true, listing),
	}
}

// buildResponse creates a StashItemResponse with the given quantity and listed flag.
// When a listing is provided and isListed is true, asking price/for are populated.
func (s *StashService) buildResponse(item *models.StashItem, quantity int, isListed bool, listing *models.Listing) *dto.StashItemResponse {
	resp := &dto.StashItemResponse{
		ID:            item.ID,
		UserID:        item.UserID,
		Name:          item.GetName(),
		ItemType:      item.ItemType,
		DisplayName:   item.GetDisplayName(),
		Rarity:        item.Rarity,
		ImageURL:      item.GetImageURL(),
		Category:      item.Category,
		Suffixes:      item.Suffixes,
		RuneOrder:     item.GetRuneOrder(),
		BaseItemCode:  item.GetBaseItemCode(),
		BaseItemName:  item.GetBaseItemName(),
		CatalogItemID: item.GetCatalogItemID(),
		Quantity:      quantity,
		IsListed:      isListed,
		IsStackable:   item.IsStackable(),
		Game:          item.Game,
		Ladder:        item.Ladder,
		Hardcore:      item.Hardcore,

		Platforms:     item.Platforms,
		Region:        item.Region,
		Source:        item.Source,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}

	if isListed && listing != nil {
		resp.AskingPrice = listing.GetAskingPrice()
		resp.AskingFor = listing.AskingFor
	}

	// Transform stats and runes using listing service helpers (same package)
	if s.listingService != nil {
		resp.Stats = s.listingService.transformAllStats(item.Stats)
		resp.Runes = s.listingService.transformRunes(item.Runes)
	}

	return resp
}

// parseCSV splits a comma-separated string into a slice
func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func isImageContentType(ct string) bool {
	switch ct {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}
