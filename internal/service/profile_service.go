package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/storage"
)

const profileCacheTTL = 1 * time.Hour

// ProfileService handles profile business logic
type ProfileService struct {
	repo        repository.ProfileRepository
	redis       *cache.RedisClient
	invalidator *cache.Invalidator
	storage     storage.Storage
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

	return profile, nil
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

	if err := s.repo.Update(ctx, profile); err != nil {
		return nil, err
	}

	// Invalidate cache
	_ = s.invalidator.InvalidateProfile(ctx, userID)

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
		ProfileFlair:  profile.GetProfileFlair(),
		CreatedAt:     profile.CreatedAt,
	}
}

// ToMyProfileResponse converts a profile model to a my profile DTO response
func (s *ProfileService) ToMyProfileResponse(profile *models.Profile) *dto.MyProfileResponse {
	return &dto.MyProfileResponse{
		ProfileResponse:   *s.ToResponse(profile),
		BattleNetLinked:   profile.IsBattleNetLinked(),
		BattleNetLinkedAt: profile.BattleNetLinkedAt,
		UpdatedAt:         profile.UpdatedAt,
	}
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

	return avatarURL, nil
}
