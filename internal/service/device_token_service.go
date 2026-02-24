package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

// DeviceTokenService handles device token business logic
type DeviceTokenService struct {
	repo repository.DeviceTokenRepository
}

// NewDeviceTokenService creates a new device token service
func NewDeviceTokenService(repo repository.DeviceTokenRepository) *DeviceTokenService {
	return &DeviceTokenService{repo: repo}
}

// Register registers or updates a device token for a user
func (s *DeviceTokenService) Register(ctx context.Context, userID string, req *dto.RegisterDeviceTokenRequest) error {
	token := &models.DeviceToken{
		ID:            uuid.New().String(),
		UserID:        userID,
		ExpoPushToken: req.ExpoPushToken,
		Platform:      req.Platform,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if req.DeviceName != "" {
		token.DeviceName = &req.DeviceName
	}

	return s.repo.Upsert(ctx, token)
}

// Unregister removes a device token for a user
func (s *DeviceTokenService) Unregister(ctx context.Context, userID string, req *dto.UnregisterDeviceTokenRequest) error {
	return s.repo.DeleteByToken(ctx, userID, req.ExpoPushToken)
}

// GetByUserID retrieves all device tokens for a user
func (s *DeviceTokenService) GetByUserID(ctx context.Context, userID string) ([]*models.DeviceToken, error) {
	return s.repo.GetByUserID(ctx, userID)
}
