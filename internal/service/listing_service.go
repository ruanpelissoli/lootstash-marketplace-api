package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/games/d2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

const (
	listingCacheTTL      = 15 * time.Minute
	listingDTOCacheTTL   = 1 * time.Hour
	maxRecentListings    = 20
	FreeListingLimit     = 10
)

// ListingService handles listing business logic
type ListingService struct {
	repo             repository.ListingRepository
	profileService   *ProfileService
	redis            *cache.RedisClient
	invalidator      *cache.Invalidator
	wishlistService  *WishlistService
	statsService     *StatsService
}

// NewListingService creates a new listing service
func NewListingService(repo repository.ListingRepository, profileService *ProfileService, redis *cache.RedisClient) *ListingService {
	return &ListingService{
		repo:           repo,
		profileService: profileService,
		redis:          redis,
		invalidator:    cache.NewInvalidator(redis),
	}
}

// SetWishlistService sets the wishlist service for matching on listing creation
func (s *ListingService) SetWishlistService(ws *WishlistService) {
	s.wishlistService = ws
}

// SetStatsService sets the stats service for cache refresh on listing events
func (s *ListingService) SetStatsService(ss *StatsService) {
	s.statsService = ss
}

// ErrListingLimitReached indicates a free user has reached their active listing limit
var ErrListingLimitReached = fmt.Errorf("listing limit reached")

// Create creates a new listing
func (s *ListingService) Create(ctx context.Context, sellerID string, req *dto.CreateListingRequest) (*models.Listing, error) {
	log := logger.FromContext(ctx)
	log.Info("creating new listing",
		"seller_id", sellerID,
		"name", req.Name,
		"item_type", req.ItemType,
		"rarity", req.Rarity,
		"category", req.Category,
		"game", req.Game,
	)

	// Check listing limit for free users
	profile, err := s.profileService.GetByID(ctx, sellerID)
	if err != nil {
		log.Error("failed to get seller profile", "error", err.Error(), "seller_id", sellerID)
		return nil, err
	}
	if !profile.IsPremium {
		count, err := s.repo.CountActiveBySellerID(ctx, sellerID)
		if err != nil {
			log.Error("failed to count active listings", "error", err.Error(), "seller_id", sellerID)
			return nil, err
		}
		log.Debug("checking listing limit for free user", "current_count", count, "limit", FreeListingLimit)
		if count >= FreeListingLimit {
			log.Warn("listing limit reached for free user", "seller_id", sellerID, "count", count)
			return nil, ErrListingLimitReached
		}
	}

	// Deduplicate platforms
	seen := make(map[string]bool)
	var uniquePlatforms []string
	for _, p := range req.Platforms {
		if !seen[p] {
			seen[p] = true
			uniquePlatforms = append(uniquePlatforms, p)
		}
	}

	listing := &models.Listing{
		ID:             uuid.New().String(),
		SellerID:       sellerID,
		Name:           req.Name,
		ItemType:       req.ItemType,
		Rarity:         req.Rarity,
		Category:       req.Category,
		Stats:          req.Stats,
		Suffixes:       req.Suffixes,
		Runes:          req.Runes,
		AskingFor:      req.AskingFor,
		Game:           req.Game,
		Ladder:         req.Ladder,
		Hardcore:       req.Hardcore,
		IsNonRotw:      req.IsNonRotw,
		Platforms:      uniquePlatforms,
		Region:         req.Region,
		SellerTimezone: profile.Timezone,
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(30 * 24 * time.Hour),
	}

	if req.ImageURL != "" {
		listing.ImageURL = &req.ImageURL
	}
	if req.AskingPrice != "" {
		listing.AskingPrice = &req.AskingPrice
	}
	if req.Notes != "" {
		listing.Notes = &req.Notes
	}
	if req.RuneOrder != "" {
		listing.RuneOrder = &req.RuneOrder
	}
	if req.BaseItemCode != "" {
		listing.BaseItemCode = &req.BaseItemCode
	}
	if req.BaseItemName != "" {
		listing.BaseItemName = &req.BaseItemName
	}
	if req.CatalogItemID != "" {
		listing.CatalogItemID = &req.CatalogItemID
	}

	if err := s.repo.Create(ctx, listing); err != nil {
		log.Error("failed to create listing in database", "error", err.Error(), "listing_id", listing.ID)
		return nil, err
	}

	log.Info("listing created successfully",
		"listing_id", listing.ID,
		"seller_id", sellerID,
		"name", listing.Name,
		"rarity", listing.Rarity,
		"category", listing.Category,
	)
	fmt.Printf("[LISTING] Created listing: id=%s name=%s seller=%s\n", listing.ID, listing.Name, sellerID)

	// Push to recent listings cache
	listing.Seller = profile
	s.pushToRecentListings(ctx, listing)

	// Refresh home stats (activeListings changed)
	if s.statsService != nil {
		go s.statsService.RefreshHomeStats(context.Background())
	}

	// Trigger async wishlist matching
	if s.wishlistService != nil {
		log.Info("triggering async wishlist matching",
			"listing_id", listing.ID,
			"listing_name", listing.Name,
		)
		fmt.Printf("[LISTING] Triggering wishlist matching for listing: id=%s name=%s\n", listing.ID, listing.Name)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("[WISHLIST] PANIC in wishlist matching: %v\n%s\n", r, debug.Stack())
					log.Error("panic in wishlist matching",
						"error", fmt.Sprintf("%v", r),
						"listing_id", listing.ID,
					)
				}
			}()
			fmt.Printf("[WISHLIST] Starting async wishlist matching for listing: id=%s\n", listing.ID)
			s.wishlistService.CheckAndNotifyMatches(context.Background(), listing)
			fmt.Printf("[WISHLIST] Completed wishlist matching for listing: id=%s\n", listing.ID)
		}()
	} else {
		log.Warn("wishlist service not configured, skipping wishlist matching",
			"listing_id", listing.ID,
		)
		fmt.Printf("[LISTING] WARNING: wishlist service is nil, skipping matching for listing: id=%s\n", listing.ID)
	}

	return listing, nil
}

// GetByID retrieves a listing by ID with caching
func (s *ListingService) GetByID(ctx context.Context, id string) (*models.Listing, error) {
	// Try cache first
	cacheKey := cache.ListingKey(id)
	cached, err := s.redis.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var listing models.Listing
		if json.Unmarshal([]byte(cached), &listing) == nil {
			return &listing, nil
		}
	}

	// Fetch from database
	listing, err := s.repo.GetByIDWithSeller(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(listing); err == nil {
		_ = s.redis.Set(ctx, cacheKey, string(data), listingCacheTTL)
	}

	// Cache DTO version for frontend direct access
	s.cacheListingDTO(ctx, listing)

	return listing, nil
}

// Update updates a listing
func (s *ListingService) Update(ctx context.Context, id string, userID string, req *dto.UpdateListingRequest) (*models.Listing, error) {
	listing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if listing.SellerID != userID {
		return nil, ErrForbidden
	}

	// Apply updates
	if req.AskingFor != nil {
		listing.AskingFor = req.AskingFor
	}
	if req.AskingPrice != nil {
		listing.AskingPrice = req.AskingPrice
	}
	if req.Notes != nil {
		listing.Notes = req.Notes
	}
	if req.Status != nil {
		listing.Status = *req.Status
	}

	if err := s.repo.Update(ctx, listing); err != nil {
		return nil, err
	}

	// Invalidate cache
	_ = s.invalidator.InvalidateListing(ctx, id)
	_ = s.invalidator.InvalidateListingDTO(ctx, id)

	return listing, nil
}

// Delete cancels a listing
func (s *ListingService) Delete(ctx context.Context, id string, userID string) error {
	listing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify ownership
	if listing.SellerID != userID {
		return ErrForbidden
	}

	// Soft delete by setting status to cancelled
	listing.Status = "cancelled"
	if err := s.repo.Update(ctx, listing); err != nil {
		return err
	}

	// Invalidate cache
	_ = s.invalidator.InvalidateListing(ctx, id)
	_ = s.invalidator.InvalidateListingDTO(ctx, id)

	// Refresh home stats (activeListings changed)
	if s.statsService != nil {
		go s.statsService.RefreshHomeStats(context.Background())
	}

	return nil
}

// List retrieves listings with filters
func (s *ListingService) List(ctx context.Context, req *dto.ListingFilterRequest) ([]*models.Listing, int, error) {
	// Parse affix filters
	var affixFilters []repository.AffixFilter
	if len(req.AffixFilters) > 0 {
		var dtoFilters []dto.AffixFilter
		if json.Unmarshal(req.AffixFilters, &dtoFilters) == nil {
			for _, f := range dtoFilters {
				affixFilters = append(affixFilters, repository.AffixFilter{
					Code:     f.Code,
					MinValue: f.MinValue,
					MaxValue: f.MaxValue,
				})
			}
		}
	}

	// Parse asking_for filters
	var askingForFilters []repository.AskingForFilter
	if len(req.AskingForFilters) > 0 {
		var dtoFilters []dto.AskingForFilter
		if json.Unmarshal(req.AskingForFilters, &dtoFilters) == nil {
			for _, f := range dtoFilters {
				askingForFilters = append(askingForFilters, repository.AskingForFilter{
					Name:        f.Name,
					Type:        f.Type,
					MinQuantity: f.MinQuantity,
				})
			}
		}
	}

	filter := repository.ListingFilter{
		SellerID:         req.SellerID,
		Query:            req.Q,
		Game:             req.Game,
		Ladder:           req.Ladder,
		Hardcore:         req.Hardcore,
		IsNonRotw:        req.IsNonRotw,
		Platforms:        parsePlatforms(req.Platforms),
		Region:           req.Region,
		Category:         req.Category,
		Rarity:           req.Rarity,
		AffixFilters:     affixFilters,
		AskingForFilters: askingForFilters,
		SortBy:           req.SortBy,
		SortOrder:        req.SortOrder,
		Offset:           req.GetOffset(),
		Limit:            req.GetLimit(),
	}

	return s.repo.List(ctx, filter)
}

// ListByFilter retrieves listings using a pre-built filter
func (s *ListingService) ListByFilter(ctx context.Context, filter repository.ListingFilter) ([]*models.Listing, int, error) {
	return s.repo.List(ctx, filter)
}

// ListBySellerID retrieves listings for a specific seller
func (s *ListingService) ListBySellerID(ctx context.Context, sellerID string, status string, offset, limit int) ([]*models.Listing, int, error) {
	return s.repo.ListBySellerID(ctx, sellerID, status, offset, limit)
}

// GetTradeCount returns the number of active trade requests for a listing
func (s *ListingService) GetTradeCount(ctx context.Context, listingID string) (int, error) {
	return s.repo.CountByListingID(ctx, listingID)
}

// ToCardResponse converts a listing model to a lightweight card DTO for list views
func (s *ListingService) ToCardResponse(listing *models.Listing) *dto.ListingCardResponse {
	resp := &dto.ListingCardResponse{
		ID:             listing.ID,
		SellerID:       listing.SellerID,
		Name:           listing.Name,
		ItemType:       listing.ItemType,
		Rarity:         listing.Rarity,
		ImageURL:       listing.GetImageURL(),
		Stats:          s.transformCardStats(listing.Stats),
		CatalogItemID:  listing.GetCatalogItemID(),
		AskingFor:      listing.AskingFor,
		AskingPrice:    listing.GetAskingPrice(),
		Game:           listing.Game,
		Ladder:         listing.Ladder,
		Hardcore:       listing.Hardcore,
		IsNonRotw:      listing.IsNonRotw,
		Platforms:      listing.Platforms,
		Region:         listing.Region,
		SellerTimezone: listing.GetSellerTimezone(),
		Views:          listing.Views,
		CreatedAt:      listing.CreatedAt,
	}

	if listing.Seller != nil {
		resp.Seller = s.profileService.ToResponse(listing.Seller)
	}

	return resp
}

// ToResponse converts a listing model to a full DTO response
func (s *ListingService) ToResponse(listing *models.Listing) *dto.ListingResponse {
	resp := &dto.ListingResponse{
		ID:             listing.ID,
		SellerID:       listing.SellerID,
		Name:           listing.Name,
		ItemType:       listing.ItemType,
		Rarity:         listing.Rarity,
		ImageURL:       listing.GetImageURL(),
		Category:       listing.Category,
		Stats:          s.transformAllStats(listing.Stats),
		Suffixes:       listing.Suffixes,
		Runes:          s.transformRunes(listing.Runes),
		RuneOrder:      listing.GetRuneOrder(),
		BaseItemCode:   listing.GetBaseItemCode(),
		BaseItemName:   listing.GetBaseItemName(),
		CatalogItemID:  listing.GetCatalogItemID(),
		AskingFor:      listing.AskingFor,
		AskingPrice:    listing.GetAskingPrice(),
		Notes:          listing.GetNotes(),
		Game:           listing.Game,
		Ladder:         listing.Ladder,
		Hardcore:       listing.Hardcore,
		IsNonRotw:      listing.IsNonRotw,
		Platforms:      listing.Platforms,
		Region:         listing.Region,
		SellerTimezone: listing.GetSellerTimezone(),
		Status:         listing.Status,
		Views:          listing.Views,
		CreatedAt:      listing.CreatedAt,
		ExpiresAt:      listing.ExpiresAt,
	}

	if listing.Seller != nil {
		resp.Seller = s.profileService.ToResponse(listing.Seller)
	}

	return resp
}

// rawStat represents the raw stat format from frontend (with mixed value types)
type rawStat struct {
	Code        string      `json:"code"`
	Value       interface{} `json:"value,omitempty"`
	Min         *int        `json:"min,omitempty"`
	Max         *int        `json:"max,omitempty"`
	Param       string      `json:"param,omitempty"`
	DisplayText string      `json:"displayText,omitempty"`
	IsVariable  bool        `json:"isVariable,omitempty"`
}

// extractNumericValue attempts to extract an integer from an interface{}
// Returns nil if the value cannot be parsed as a number
func extractNumericValue(val interface{}) *int {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case float64:
		// JSON numbers are parsed as float64
		intVal := int(v)
		return &intVal
	case int:
		return &v
	case string:
		// Try to extract number from string like "+40% Increased Attack Speed"
		re := regexp.MustCompile(`[+-]?\d+`)
		matches := re.FindString(v)
		if matches != "" {
			if num, err := strconv.Atoi(matches); err == nil {
				return &num
			}
		}
	}

	return nil
}

// transformCardStats converts raw JSON stats to DTOs, returning only variable stats for card display
func (s *ListingService) transformCardStats(rawStats json.RawMessage) []dto.ItemStat {
	return s.doTransformStats(rawStats, true)
}

// transformAllStats converts raw JSON stats to DTOs, returning all stats with isVariable flag
func (s *ListingService) transformAllStats(rawStats json.RawMessage) []dto.ItemStat {
	return s.doTransformStats(rawStats, false)
}

// doTransformStats converts raw JSON stats to DTOs with display text
// When variableOnly is true, only stats with isVariable=true are included (for card views)
// When variableOnly is false, all stats are included with the isVariable flag preserved (for detail views)
func (s *ListingService) doTransformStats(rawStats json.RawMessage, variableOnly bool) []dto.ItemStat {
	if len(rawStats) == 0 {
		return nil
	}

	var stats []rawStat
	if err := json.Unmarshal(rawStats, &stats); err != nil {
		return []dto.ItemStat{}
	}

	if len(stats) == 0 {
		return []dto.ItemStat{}
	}

	result := make([]dto.ItemStat, 0, len(stats))
	for _, stat := range stats {
		if variableOnly && !stat.IsVariable {
			continue
		}
		numericValue := extractNumericValue(stat.Value)

		// Use stored displayText from catalog-api, fallback to code:value for legacy data
		displayText := stat.DisplayText
		if displayText == "" {
			if numericValue != nil {
				displayText = fmt.Sprintf("%s: %d", stat.Code, *numericValue)
			} else {
				displayText = stat.Code
			}
		}

		result = append(result, dto.ItemStat{
			Code:        stat.Code,
			Value:       numericValue,
			Min:         stat.Min,
			Max:         stat.Max,
			Param:       stat.Param,
			DisplayText: displayText,
			IsVariable:  stat.IsVariable,
		})
	}

	return result
}

// transformRunes converts raw JSON rune codes to RuneInfo DTOs
func (s *ListingService) transformRunes(rawRunes json.RawMessage) []dto.RuneInfo {
	if len(rawRunes) == 0 {
		return nil
	}

	var codes []string
	if err := json.Unmarshal(rawRunes, &codes); err != nil {
		return nil
	}

	result := make([]dto.RuneInfo, 0, len(codes))
	for _, code := range codes {
		result = append(result, dto.RuneInfo{
			Code:     code,
			Name:     d2.GetRuneName(code),
			ImageURL: d2.GetRuneImageURL(code),
		})
	}

	return result
}

// IncrementViews increments the view count for a listing
func (s *ListingService) IncrementViews(ctx context.Context, id string) error {
	if err := s.repo.IncrementViews(ctx, id); err != nil {
		return err
	}
	// Invalidate listing cache
	_ = s.invalidator.InvalidateListing(ctx, id)
	_ = s.invalidator.InvalidateListingDTO(ctx, id)
	return nil
}

// ToDetailResponse converts a listing model to a detailed DTO response
func (s *ListingService) ToDetailResponse(ctx context.Context, listing *models.Listing) *dto.ListingDetailResponse {
	tradeCount, _ := s.GetTradeCount(ctx, listing.ID)

	return &dto.ListingDetailResponse{
		ListingResponse: *s.ToResponse(listing),
		UpdatedAt:       listing.UpdatedAt,
		TradeCount:      tradeCount,
	}
}

// cacheListingDTO caches the listing as a DTO (camelCase JSON) for frontend direct access
func (s *ListingService) cacheListingDTO(ctx context.Context, listing *models.Listing) {
	resp := s.ToResponse(listing)
	if data, err := json.Marshal(resp); err == nil {
		_ = s.redis.Set(ctx, cache.ListingDTOKey(listing.ID), string(data), listingDTOCacheTTL)
	}
}

// pushToRecentListings adds a listing to the home:recent Redis list
func (s *ListingService) pushToRecentListings(ctx context.Context, listing *models.Listing) {
	cardResp := s.ToCardResponse(listing)
	data, err := json.Marshal(cardResp)
	if err != nil {
		return
	}
	_ = s.redis.LPush(ctx, cache.HomeRecentKey(), string(data))
	_ = s.redis.LTrim(ctx, cache.HomeRecentKey(), 0, int64(maxRecentListings-1))
}

// GetRecentListings returns recent listings from the home:recent cache
func (s *ListingService) GetRecentListings(ctx context.Context) ([]dto.ListingCardResponse, error) {
	items, err := s.redis.LRange(ctx, cache.HomeRecentKey(), 0, int64(maxRecentListings-1))
	if err != nil {
		return nil, err
	}

	results := make([]dto.ListingCardResponse, 0, len(items))
	for _, item := range items {
		var card dto.ListingCardResponse
		if json.Unmarshal([]byte(item), &card) == nil {
			results = append(results, card)
		}
	}

	return results, nil
}

// parsePlatforms splits a comma-separated platform string into a slice
func parsePlatforms(raw string) []string {
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

// WarmRecentListings populates the home:recent cache on startup
func (s *ListingService) WarmRecentListings(ctx context.Context) {
	filter := repository.ListingFilter{
		SortBy:    "created_at",
		SortOrder: "desc",
		Limit:     maxRecentListings,
	}

	listings, _, err := s.repo.List(ctx, filter)
	if err != nil {
		logger.Log.Warn("failed to warm recent listings", "error", err.Error())
		return
	}

	if len(listings) == 0 {
		return
	}

	// Delete existing key
	_ = s.redis.Del(ctx, cache.HomeRecentKey())

	// Push in reverse order so newest is at index 0
	for i := len(listings) - 1; i >= 0; i-- {
		cardResp := s.ToCardResponse(listings[i])
		data, err := json.Marshal(cardResp)
		if err != nil {
			continue
		}
		_ = s.redis.LPush(ctx, cache.HomeRecentKey(), string(data))
	}
}
