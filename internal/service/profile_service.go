package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/storage"
)

const profileCacheTTL = 1 * time.Hour

// ProfileService handles profile business logic
type ProfileService struct {
	repo            repository.ProfileRepository
	transactionRepo repository.TransactionRepository
	redis           *cache.RedisClient
	invalidator     *cache.Invalidator
	storage         storage.Storage
}

// NewProfileService creates a new profile service
func NewProfileService(repo repository.ProfileRepository, redis *cache.RedisClient, stor storage.Storage) *ProfileService {
	return &ProfileService{
		repo:        repo,
		redis:       redis,
		invalidator: cache.NewInvalidator(redis),
		storage:     stor,
	}
}

// SetTransactionRepository sets the transaction repository for sales queries
func (s *ProfileService) SetTransactionRepository(repo repository.TransactionRepository) {
	s.transactionRepo = repo
}

// GetByID retrieves a profile by ID with caching
func (s *ProfileService) GetByID(ctx context.Context, id string) (*models.Profile, error) {
	// Try cache first
	cacheKey := cache.ProfileKey(id)
	cached, err := s.redis.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var profile models.Profile
		if json.Unmarshal([]byte(cached), &profile) == nil {
			return &profile, nil
		}
	}

	// Fetch from database
	profile, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(profile); err == nil {
		_ = s.redis.Set(ctx, cacheKey, string(data), profileCacheTTL)
	}

	// Cache DTO version for frontend direct access
	s.CacheProfileDTO(ctx, profile)

	return profile, nil
}

// GetByUsername retrieves a profile by username with caching
func (s *ProfileService) GetByUsername(ctx context.Context, username string) (*models.Profile, error) {
	// Try cache first
	cacheKey := cache.ProfileUsernameKey(strings.ToLower(username))
	cached, err := s.redis.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var profile models.Profile
		if json.Unmarshal([]byte(cached), &profile) == nil {
			return &profile, nil
		}
	}

	// Fetch from database
	profile, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(profile); err == nil {
		_ = s.redis.Set(ctx, cacheKey, string(data), profileCacheTTL)
	}

	return profile, nil
}

// GetByIdentifier retrieves a profile by UUID or username
func (s *ProfileService) GetByIdentifier(ctx context.Context, identifier string) (*models.Profile, error) {
	if _, err := uuid.Parse(identifier); err == nil {
		return s.GetByID(ctx, identifier)
	}
	return s.GetByUsername(ctx, identifier)
}

// Update updates a user's profile
func (s *ProfileService) Update(ctx context.Context, userID string, req *dto.UpdateProfileRequest) (*models.Profile, error) {
	profile, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.DisplayName != nil {
		profile.DisplayName = req.DisplayName
	}
	if req.AvatarURL != nil {
		profile.AvatarURL = req.AvatarURL
	}
	if req.Timezone != nil {
		profile.Timezone = req.Timezone
	}
	if req.PreferredLadder != nil {
		profile.PreferredLadder = req.PreferredLadder
	}
	if req.PreferredHardcore != nil {
		profile.PreferredHardcore = req.PreferredHardcore
	}
	if req.PreferredPlatforms != nil {
		profile.PreferredPlatforms = req.PreferredPlatforms
	}
	if req.PreferredRegion != nil {
		profile.PreferredRegion = req.PreferredRegion
	}

	if err := s.repo.Update(ctx, profile); err != nil {
		return nil, err
	}

	// Invalidate cache
	_ = s.invalidator.InvalidateProfile(ctx, userID)
	_ = s.invalidator.InvalidateProfileDTO(ctx, userID)
	if profile.Username != "" {
		_ = s.invalidator.InvalidateProfileByUsername(ctx, strings.ToLower(profile.Username))
	}

	return profile, nil
}

// ToResponse converts a profile model to a DTO response
func (s *ProfileService) ToResponse(profile *models.Profile) *dto.ProfileResponse {
	return &dto.ProfileResponse{
		ID:            profile.ID,
		Username:      profile.Username,
		DisplayName:   profile.GetDisplayName(),
		AvatarURL:     profile.GetAvatarURL(),
		BattleTag:     profile.GetBattleTag(),
		TotalTrades:   profile.TotalTrades,
		AverageRating: profile.AverageRating,
		RatingCount:   profile.RatingCount,
		IsPremium:     profile.IsPremium,
		IsAdmin:       profile.IsAdmin,
		ProfileFlair:  profile.GetProfileFlair(),
		UsernameColor: profile.GetUsernameColor(),
		Timezone:      profile.GetTimezone(),
		CreatedAt:     profile.CreatedAt,
	}
}

// ToMyProfileResponse converts a profile model to a my profile DTO response
func (s *ProfileService) ToMyProfileResponse(profile *models.Profile) *dto.MyProfileResponse {
	return &dto.MyProfileResponse{
		ProfileResponse:    *s.ToResponse(profile),
		BattleNetLinked:    profile.IsBattleNetLinked(),
		BattleNetLinkedAt:  profile.BattleNetLinkedAt,
		PreferredLadder:    profile.PreferredLadder,
		PreferredHardcore:  profile.PreferredHardcore,
		PreferredPlatforms: profile.PreferredPlatforms,
		PreferredRegion:    profile.GetPreferredRegion(),
		UpdatedAt:          profile.UpdatedAt,
	}
}

// IsAdmin checks if a user has admin privileges using the cached profile
func (s *ProfileService) IsAdmin(ctx context.Context, userID string) (bool, error) {
	profile, err := s.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return profile.IsAdmin, nil
}

// UploadProfilePicture uploads a profile picture and updates the profile
func (s *ProfileService) UploadProfilePicture(ctx context.Context, userID string, data []byte, contentType string) (string, error) {
	if s.storage == nil {
		return "", fmt.Errorf("storage not configured")
	}

	// Determine file extension from content type
	var ext string
	switch contentType {
	case "image/png":
		ext = "png"
	case "image/jpeg":
		ext = "jpg"
	case "image/webp":
		ext = "webp"
	default:
		return "", fmt.Errorf("unsupported content type: %s", contentType)
	}

	// Generate storage path: {userID}.{ext}
	storagePath := fmt.Sprintf("%s.%s", userID, ext)

	// Upload to storage
	avatarURL, err := s.storage.UploadImage(ctx, storagePath, data, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	// Update profile with new avatar URL
	profile, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get profile: %w", err)
	}

	profile.AvatarURL = &avatarURL
	if err := s.repo.Update(ctx, profile); err != nil {
		return "", fmt.Errorf("failed to update profile: %w", err)
	}

	// Invalidate cache
	_ = s.invalidator.InvalidateProfile(ctx, userID)
	_ = s.invalidator.InvalidateProfileDTO(ctx, userID)
	if profile.Username != "" {
		_ = s.invalidator.InvalidateProfileByUsername(ctx, strings.ToLower(profile.Username))
	}

	return avatarURL, nil
}

// GetSales retrieves completed sales for a seller
func (s *ProfileService) GetSales(ctx context.Context, sellerID string, offset, limit int) (*dto.SalesResponse, error) {
	if s.transactionRepo == nil {
		return nil, fmt.Errorf("transaction repository not configured")
	}

	records, total, err := s.transactionRepo.GetSalesBySeller(ctx, sellerID, offset, limit)
	if err != nil {
		return nil, err
	}

	sales := make([]dto.SoldItem, 0, len(records))
	for _, record := range records {
		sale := s.saleRecordToDTO(record)
		sales = append(sales, sale)
	}

	hasMore := offset+len(sales) < total

	return &dto.SalesResponse{
		Sales:   sales,
		Total:   total,
		HasMore: hasMore,
	}, nil
}

// saleRecordToDTO converts a repository SaleRecord to a DTO SoldItem
func (s *ProfileService) saleRecordToDTO(record repository.SaleRecord) dto.SoldItem {
	// Parse completedAt
	var completedAt time.Time
	if t, ok := record.CompletedAt.(time.Time); ok {
		completedAt = t
	}

	// Build item info
	item := dto.SoldItemInfo{
		Name:     record.ItemName,
		ItemType: record.ItemType,
		Rarity:   record.Rarity,
	}
	if record.ImageURL != nil {
		item.ImageURL = *record.ImageURL
	}
	if record.BaseName != nil {
		item.BaseName = *record.BaseName
	}
	item.Stats = s.transformSaleStats(record.Stats)

	// Parse offered items (soldFor)
	soldFor := s.transformOfferedItems(record.OfferedItems)

	// Build buyer info
	buyer := dto.SaleBuyerInfo{
		ID:          record.BuyerID,
		DisplayName: record.BuyerName,
	}
	if record.BuyerAvatar != nil {
		buyer.AvatarURL = *record.BuyerAvatar
	}

	// Build review if present
	var review *dto.SaleReview
	if record.ReviewRating != nil {
		var reviewedAt time.Time
		if t, ok := record.ReviewedAt.(time.Time); ok {
			reviewedAt = t
		}
		review = &dto.SaleReview{
			Rating:    *record.ReviewRating,
			CreatedAt: reviewedAt,
		}
		if record.ReviewComment != nil {
			review.Comment = *record.ReviewComment
		}
	}

	return dto.SoldItem{
		ID:          record.TransactionID,
		CompletedAt: completedAt,
		Item:        item,
		SoldFor:     soldFor,
		Buyer:       buyer,
		Review:      review,
	}
}

// saleRawStat represents the raw stat format from the database
type saleRawStat struct {
	Code        string      `json:"code"`
	Value       interface{} `json:"value,omitempty"`
	DisplayText string      `json:"displayText,omitempty"`
	IsVariable  bool        `json:"isVariable,omitempty"`
}

// transformSaleStats converts raw JSON stats to DTOs for sales
func (s *ProfileService) transformSaleStats(rawStats []byte) []dto.SoldItemStat {
	if len(rawStats) == 0 {
		return nil
	}

	var stats []saleRawStat
	if err := json.Unmarshal(rawStats, &stats); err != nil {
		return nil
	}

	result := make([]dto.SoldItemStat, 0, len(stats))
	for _, stat := range stats {
		numericValue := extractSaleNumericValue(stat.Value)

		displayText := stat.DisplayText
		if displayText == "" {
			if numericValue != nil {
				displayText = fmt.Sprintf("%s: %d", stat.Code, *numericValue)
			} else {
				displayText = stat.Code
			}
		}

		result = append(result, dto.SoldItemStat{
			Code:        stat.Code,
			Value:       numericValue,
			DisplayText: displayText,
			IsVariable:  stat.IsVariable,
		})
	}

	return result
}

// extractSaleNumericValue extracts an integer from an interface{}
func extractSaleNumericValue(val interface{}) *int {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case float64:
		intVal := int(v)
		return &intVal
	case int:
		return &v
	case string:
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

// offeredItem represents an item in the offered_items JSON
type offeredItem struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	ImageURL string `json:"imageUrl,omitempty"`
}

// CacheProfileDTO caches the profile as a DTO (camelCase JSON) for frontend direct access
func (s *ProfileService) CacheProfileDTO(ctx context.Context, profile *models.Profile) {
	resp := s.ToResponse(profile)
	if data, err := json.Marshal(resp); err == nil {
		_ = s.redis.Set(ctx, cache.ProfileDTOKey(profile.ID), string(data), profileCacheTTL)
	}
}

// transformOfferedItems converts raw JSON offered items to DTOs
func (s *ProfileService) transformOfferedItems(rawItems []byte) []dto.SoldForItem {
	if len(rawItems) == 0 {
		return nil
	}

	var items []offeredItem
	if err := json.Unmarshal(rawItems, &items); err != nil {
		return nil
	}

	result := make([]dto.SoldForItem, 0, len(items))
	for _, item := range items {
		quantity := item.Quantity
		if quantity == 0 {
			quantity = 1
		}
		result = append(result, dto.SoldForItem{
			Type:     item.Type,
			Name:     item.Name,
			Quantity: quantity,
			ImageURL: item.ImageURL,
		})
	}

	return result
}
