package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/games/d2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

const maxActiveWishlistItems = 10

// ErrWishlistLimitReached indicates a premium user has reached their wishlist item limit
var ErrWishlistLimitReached = fmt.Errorf("wishlist limit reached")

// ErrPremiumRequired indicates the feature requires a premium subscription
var ErrPremiumRequired = fmt.Errorf("premium required")

// WishlistService handles wishlist business logic
type WishlistService struct {
	repo                repository.WishlistRepository
	profileService      *ProfileService
	notificationService *NotificationService
}

// NewWishlistService creates a new wishlist service
func NewWishlistService(
	repo repository.WishlistRepository,
	profileService *ProfileService,
	notificationService *NotificationService,
) *WishlistService {
	return &WishlistService{
		repo:                repo,
		profileService:      profileService,
		notificationService: notificationService,
	}
}

// Create creates a new wishlist item
func (s *WishlistService) Create(ctx context.Context, userID string, req *dto.CreateWishlistItemRequest) (*models.WishlistItem, error) {
	// Check premium status
	profile, err := s.profileService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !profile.IsPremium {
		return nil, ErrPremiumRequired
	}

	// Check active limit
	count, err := s.repo.CountActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= maxActiveWishlistItems {
		return nil, ErrWishlistLimitReached
	}

	// Convert stat criteria
	var statCriteria []models.StatCriterion
	for _, sc := range req.StatCriteria {
		statCriteria = append(statCriteria, models.StatCriterion{
			Code:     sc.Code,
			Name:     sc.Name,
			MinValue: sc.MinValue,
			MaxValue: sc.MaxValue,
		})
	}

	item := &models.WishlistItem{
		ID:            uuid.New().String(),
		UserID:        userID,
		Name:          req.Name,
		Category:      req.Category,
		Rarity:        req.Rarity,
		ImageURL:      req.ImageURL,
		CatalogItemID: req.CatalogItemID,
		StatCriteria:  statCriteria,
		Game:          req.Game,
		Ladder:        req.Ladder,
		Hardcore:      req.Hardcore,
		IsNonRotw:     req.IsNonRotw,
		Platforms:     req.Platforms,
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

// List retrieves wishlist items for a user
func (s *WishlistService) List(ctx context.Context, userID string, offset, limit int) ([]*models.WishlistItem, int, error) {
	// Check premium status
	profile, err := s.profileService.GetByID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	if !profile.IsPremium {
		return nil, 0, ErrPremiumRequired
	}

	return s.repo.ListByUserID(ctx, userID, offset, limit)
}

// Update updates a wishlist item
func (s *WishlistService) Update(ctx context.Context, id string, userID string, req *dto.UpdateWishlistItemRequest) (*models.WishlistItem, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if item.UserID != userID {
		return nil, ErrForbidden
	}

	// Apply updates
	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Category != nil {
		item.Category = req.Category
	}
	if req.Rarity != nil {
		item.Rarity = req.Rarity
	}
	if req.ImageURL != nil {
		item.ImageURL = req.ImageURL
	}
	if req.CatalogItemID != nil {
		item.CatalogItemID = req.CatalogItemID
	}
	if req.StatCriteria != nil {
		var statCriteria []models.StatCriterion
		for _, sc := range req.StatCriteria {
			statCriteria = append(statCriteria, models.StatCriterion{
				Code:     sc.Code,
				Name:     sc.Name,
				MinValue: sc.MinValue,
				MaxValue: sc.MaxValue,
			})
		}
		item.StatCriteria = statCriteria
	}
	if req.Game != nil {
		item.Game = *req.Game
	}
	if req.Ladder != nil {
		item.Ladder = req.Ladder
	}
	if req.Hardcore != nil {
		item.Hardcore = req.Hardcore
	}
	if req.IsNonRotw != nil {
		item.IsNonRotw = req.IsNonRotw
	}
	if req.Platforms != nil {
		item.Platforms = req.Platforms
	}
	if req.Status != nil {
		item.Status = *req.Status
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

// Delete soft-deletes a wishlist item
func (s *WishlistService) Delete(ctx context.Context, id string, userID string) error {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify ownership
	if item.UserID != userID {
		return ErrForbidden
	}

	return s.repo.Delete(ctx, id)
}

// CheckAndNotifyMatches finds wishlist items matching a listing and sends notifications
func (s *WishlistService) CheckAndNotifyMatches(ctx context.Context, listing *models.Listing) {
	log := logger.FromContext(ctx)

	fmt.Printf("[WISHLIST] CheckAndNotifyMatches called for listing: id=%s name=%s game=%s\n", listing.ID, listing.Name, listing.Game)

	log.Info("starting wishlist matching for new listing",
		"listing_id", listing.ID,
		"listing_name", listing.Name,
		"listing_rarity", listing.Rarity,
		"listing_category", listing.Category,
		"listing_game", listing.Game,
	)

	fmt.Printf("[WISHLIST] Calling repo.FindMatchingItems for listing: id=%s name=%s\n", listing.ID, listing.Name)
	candidates, err := s.repo.FindMatchingItems(ctx, listing)
	if err != nil {
		fmt.Printf("[WISHLIST] ERROR finding matching items: %v\n", err)
		log.Error("failed to find matching wishlist items",
			"error", err.Error(),
			"listing_id", listing.ID,
		)
		return
	}

	fmt.Printf("[WISHLIST] Found %d candidates for listing: id=%s name=%s\n", len(candidates), listing.ID, listing.Name)

	if len(candidates) == 0 {
		log.Info("no wishlist candidates found for listing",
			"listing_id", listing.ID,
			"listing_name", listing.Name,
		)
		fmt.Printf("[WISHLIST] No candidates found - done\n")
		return
	}

	log.Info("found wishlist candidates for listing",
		"listing_id", listing.ID,
		"listing_name", listing.Name,
		"candidate_count", len(candidates),
	)

	// Parse listing stats once
	fmt.Printf("[WISHLIST] Parsing listing stats (raw length: %d bytes)\n", len(listing.Stats))
	var listingStats []listingStat
	if len(listing.Stats) > 0 {
		if err := json.Unmarshal(listing.Stats, &listingStats); err != nil {
			fmt.Printf("[WISHLIST] ERROR parsing listing stats: %v\n", err)
			log.Error("failed to parse listing stats for wishlist matching",
				"error", err.Error(),
				"listing_id", listing.ID,
			)
			return
		}
		fmt.Printf("[WISHLIST] Parsed %d stats from JSON\n", len(listingStats))
	} else {
		fmt.Printf("[WISHLIST] Listing has no stats (empty)\n")
	}

	// Build stat lookup map: code -> value
	statMap := make(map[string]int)
	for _, stat := range listingStats {
		fmt.Printf("[WISHLIST] Stat: code=%s value=%v (type=%T)\n", stat.Code, stat.Value, stat.Value)
		if numVal := extractNumericValue(stat.Value); numVal != nil {
			statMap[stat.Code] = *numVal
			fmt.Printf("[WISHLIST] Added to statMap: %s = %d\n", stat.Code, *numVal)
		} else {
			fmt.Printf("[WISHLIST] Could not extract numeric value for code=%s\n", stat.Code)
		}
	}

	fmt.Printf("[WISHLIST] Final statMap: %v\n", statMap)
	log.Info("parsed listing stats for wishlist matching",
		"listing_id", listing.ID,
		"stat_count", len(statMap),
		"stat_codes", getStatCodes(statMap),
	)

	matched := 0
	for _, candidate := range candidates {
		fmt.Printf("[WISHLIST] Evaluating candidate: id=%s user=%s name=%s criteria_count=%d\n",
			candidate.ID, candidate.UserID, candidate.Name, len(candidate.StatCriteria))
		for i, sc := range candidate.StatCriteria {
			fmt.Printf("[WISHLIST] Candidate criterion %d: code=%s min=%v max=%v\n", i, sc.Code, sc.MinValue, sc.MaxValue)
		}
		log.Info("evaluating wishlist candidate",
			"listing_id", listing.ID,
			"wishlist_id", candidate.ID,
			"wishlist_user_id", candidate.UserID,
			"wishlist_name", candidate.Name,
			"stat_criteria_count", len(candidate.StatCriteria),
		)
		if s.matchesStatCriteria(candidate.StatCriteria, statMap, log) {
			matched++
			fmt.Printf("[WISHLIST] MATCH! Calling sendWishlistNotification for user=%s listing=%s\n", candidate.UserID, listing.ID)
			log.Info("wishlist item MATCHED listing - sending notification",
				"listing_id", listing.ID,
				"listing_name", listing.Name,
				"wishlist_id", candidate.ID,
				"wishlist_user_id", candidate.UserID,
				"wishlist_name", candidate.Name,
			)
			s.sendWishlistNotification(ctx, candidate, listing)
			fmt.Printf("[WISHLIST] sendWishlistNotification completed for user=%s\n", candidate.UserID)
		} else {
			log.Info("wishlist item did NOT match listing stats",
				"listing_id", listing.ID,
				"wishlist_id", candidate.ID,
				"wishlist_name", candidate.Name,
			)
		}
	}

	log.Info("wishlist matching complete",
		"listing_id", listing.ID,
		"listing_name", listing.Name,
		"candidates_evaluated", len(candidates),
		"matches_found", matched,
	)
	fmt.Printf("[WISHLIST] Matching complete: listing=%s candidates=%d matches=%d\n", listing.ID, len(candidates), matched)
}

// getStatCodes extracts stat codes from the stat map for logging
func getStatCodes(statMap map[string]int) []string {
	codes := make([]string, 0, len(statMap))
	for code := range statMap {
		codes = append(codes, code)
	}
	return codes
}

// listingStat represents a stat entry in listing stats JSON
type listingStat struct {
	Code  string      `json:"code"`
	Value interface{} `json:"value,omitempty"`
}

// matchesStatCriteria checks if listing stats satisfy all wishlist stat criteria
func (s *WishlistService) matchesStatCriteria(criteria []models.StatCriterion, statMap map[string]int, log *slog.Logger) bool {
	fmt.Printf("[WISHLIST-MATCH] Checking %d stat criteria against %d listing stats\n", len(criteria), len(statMap))
	fmt.Printf("[WISHLIST-MATCH] Listing stats: %v\n", statMap)

	if len(criteria) == 0 {
		fmt.Printf("[WISHLIST-MATCH] No stat criteria - auto match!\n")
		return true
	}

	for i, c := range criteria {
		fmt.Printf("[WISHLIST-MATCH] Criterion %d: code=%s min=%v max=%v\n", i, c.Code, c.MinValue, c.MaxValue)

		// Expand the criterion code to all aliases (canonical + game codes)
		codes := d2.ExpandStatCode(c.Code)
		fmt.Printf("[WISHLIST-MATCH] Expanded codes to search: %v\n", codes)

		var value int
		var found bool
		for _, code := range codes {
			if v, exists := statMap[code]; exists {
				value = v
				found = true
				fmt.Printf("[WISHLIST-MATCH] Found matching stat: code=%s value=%d\n", code, v)
				break
			}
		}

		if !found {
			fmt.Printf("[WISHLIST-MATCH] FAIL: Stat code %s NOT FOUND in listing (searched: %v, available: %v)\n",
				c.Code, codes, getStatCodes(statMap))
			return false
		}

		if c.MinValue != nil && value < *c.MinValue {
			fmt.Printf("[WISHLIST-MATCH] FAIL: Stat %s value %d is BELOW minimum %d\n", c.Code, value, *c.MinValue)
			return false
		}
		if c.MaxValue != nil && value > *c.MaxValue {
			fmt.Printf("[WISHLIST-MATCH] FAIL: Stat %s value %d is ABOVE maximum %d\n", c.Code, value, *c.MaxValue)
			return false
		}
		fmt.Printf("[WISHLIST-MATCH] OK: Stat %s value %d passes (min=%v max=%v)\n", c.Code, value, c.MinValue, c.MaxValue)
	}

	fmt.Printf("[WISHLIST-MATCH] All criteria passed - MATCH!\n")
	return true
}

func (s *WishlistService) sendWishlistNotification(ctx context.Context, wishlistItem *models.WishlistItem, listing *models.Listing) {
	fmt.Printf("[NOTIFICATION] sendWishlistNotification called: user=%s wishlist=%s listing=%s\n",
		wishlistItem.UserID, wishlistItem.ID, listing.ID)

	log := logger.FromContext(ctx)
	refType := "listing"
	notification := &models.Notification{
		UserID:        wishlistItem.UserID,
		Type:          models.NotificationTypeWishlistMatch,
		Title:         "Wishlist Match Found",
		Body:          strPtr(fmt.Sprintf("A new listing for \"%s\" matches your wishlist!", listing.Name)),
		ReferenceType: &refType,
		ReferenceID:   &listing.ID,
	}

	fmt.Printf("[NOTIFICATION] Creating notification: user_id=%s type=%s title=%s ref_type=%s ref_id=%s\n",
		notification.UserID, notification.Type, notification.Title, refType, listing.ID)

	log.Info("creating wishlist match notification",
		"user_id", wishlistItem.UserID,
		"wishlist_id", wishlistItem.ID,
		"wishlist_name", wishlistItem.Name,
		"listing_id", listing.ID,
		"listing_name", listing.Name,
	)

	if err := s.notificationService.Create(ctx, notification); err != nil {
		fmt.Printf("[NOTIFICATION] ERROR creating notification: %v\n", err)
		log.Error("failed to send wishlist match notification",
			"error", err.Error(),
			"wishlist_id", wishlistItem.ID,
			"listing_id", listing.ID,
		)
	} else {
		fmt.Printf("[NOTIFICATION] SUCCESS: Notification created for user=%s, notification_id=%s\n",
			wishlistItem.UserID, notification.ID)
		log.Info("wishlist match notification sent successfully",
			"user_id", wishlistItem.UserID,
			"wishlist_id", wishlistItem.ID,
			"listing_id", listing.ID,
		)
	}
}

// ToResponse converts a wishlist item model to a DTO response
func (s *WishlistService) ToResponse(item *models.WishlistItem) *dto.WishlistItemResponse {
	var statCriteria []dto.StatCriterionDTO
	for _, sc := range item.StatCriteria {
		statCriteria = append(statCriteria, dto.StatCriterionDTO{
			Code:     sc.Code,
			Name:     sc.Name,
			MinValue: sc.MinValue,
			MaxValue: sc.MaxValue,
		})
	}

	return &dto.WishlistItemResponse{
		ID:            item.ID,
		UserID:        item.UserID,
		Name:          item.Name,
		Category:      item.Category,
		Rarity:        item.Rarity,
		ImageURL:      item.ImageURL,
		CatalogItemID: item.CatalogItemID,
		StatCriteria:  statCriteria,
		Game:          item.Game,
		Ladder:        item.Ladder,
		Hardcore:      item.Hardcore,
		IsNonRotw:     item.IsNonRotw,
		Platforms:     item.Platforms,
		Status:        item.Status,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}
