package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

const notificationCountCacheTTL = 1 * time.Minute

// NotificationService handles notification business logic
type NotificationService struct {
	repo        repository.NotificationRepository
	redis       *cache.RedisClient
	invalidator *cache.Invalidator
}

// NewNotificationService creates a new notification service
func NewNotificationService(repo repository.NotificationRepository, redis *cache.RedisClient) *NotificationService {
	return &NotificationService{
		repo:        repo,
		redis:       redis,
		invalidator: cache.NewInvalidator(redis),
	}
}

// GetByUserID retrieves notifications for a user
func (s *NotificationService) GetByUserID(ctx context.Context, userID string, unreadOnly bool, notificationType string, offset, limit int) ([]*models.Notification, int, error) {
	return s.repo.GetByUserID(ctx, userID, unreadOnly, notificationType, offset, limit)
}

// CountUnread returns the count of unread notifications with caching
func (s *NotificationService) CountUnread(ctx context.Context, userID string) (int, error) {
	// Try cache first
	cacheKey := cache.NotificationCountKey(userID)
	cached, err := s.redis.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var count int
		if json.Unmarshal([]byte(cached), &count) == nil {
			return count, nil
		}
	}

	// Fetch from database
	count, err := s.repo.CountUnread(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Cache the result
	if data, err := json.Marshal(count); err == nil {
		_ = s.redis.Set(ctx, cacheKey, string(data), notificationCountCacheTTL)
	}

	return count, nil
}

// MarkAsRead marks notifications as read
func (s *NotificationService) MarkAsRead(ctx context.Context, userID string, notificationIDs []string) error {
	if err := s.repo.MarkAsRead(ctx, notificationIDs, userID); err != nil {
		return err
	}

	// Invalidate count cache
	_ = s.invalidator.InvalidateNotificationCount(ctx, userID)

	return nil
}

// Create creates a new notification
func (s *NotificationService) Create(ctx context.Context, notification *models.Notification) error {
	notification.ID = uuid.New().String()
	notification.CreatedAt = time.Now()

	fmt.Printf("[NOTIFICATION-SVC] Creating notification: id=%s user_id=%s type=%s title=%s\n",
		notification.ID, notification.UserID, notification.Type, notification.Title)

	if err := s.repo.Create(ctx, notification); err != nil {
		fmt.Printf("[NOTIFICATION-SVC] ERROR inserting into database: %v\n", err)
		logger.FromContext(ctx).Error("failed to create notification",
			"error", err.Error(),
			"notification_id", notification.ID,
			"user_id", notification.UserID,
		)
		return err
	}

	fmt.Printf("[NOTIFICATION-SVC] SUCCESS: Notification inserted into database: id=%s user_id=%s\n",
		notification.ID, notification.UserID)

	// Invalidate count cache
	_ = s.invalidator.InvalidateNotificationCount(ctx, notification.UserID)

	return nil
}

// NotifyOfferReceived notifies a seller of a new offer
func (s *NotificationService) NotifyOfferReceived(ctx context.Context, userID string, offerID string, itemName string) error {
	refType := "offer"
	notification := &models.Notification{
		UserID:        userID,
		Type:          models.NotificationTypeTradeRequestReceived,
		Title:         "New Offer",
		Body:          strPtr(fmt.Sprintf("You received a new offer for %s", itemName)),
		ReferenceType: &refType,
		ReferenceID:   &offerID,
	}
	return s.Create(ctx, notification)
}

// NotifyOfferAccepted notifies a requester their offer was accepted
func (s *NotificationService) NotifyOfferAccepted(ctx context.Context, userID string, offerID string, itemName string) error {
	refType := "offer"
	notification := &models.Notification{
		UserID:        userID,
		Type:          models.NotificationTypeTradeRequestAccepted,
		Title:         "Offer Accepted",
		Body:          strPtr(fmt.Sprintf("Your offer for %s was accepted! You can now start trading.", itemName)),
		ReferenceType: &refType,
		ReferenceID:   &offerID,
	}
	return s.Create(ctx, notification)
}

// NotifyOfferRejected notifies a requester their offer was rejected
func (s *NotificationService) NotifyOfferRejected(ctx context.Context, userID string, offerID string, itemName string) error {
	refType := "offer"
	notification := &models.Notification{
		UserID:        userID,
		Type:          models.NotificationTypeTradeRequestRejected,
		Title:         "Offer Rejected",
		Body:          strPtr(fmt.Sprintf("Your offer for %s was rejected.", itemName)),
		ReferenceType: &refType,
		ReferenceID:   &offerID,
	}
	return s.Create(ctx, notification)
}

// NotifyTradeCompleted notifies a user the trade was completed
func (s *NotificationService) NotifyTradeCompleted(ctx context.Context, userID string, tradeID string, itemName string) error {
	refType := "trade"
	notification := &models.Notification{
		UserID:        userID,
		Type:          models.NotificationTypeTradeRequestAccepted, // Using existing type
		Title:         "Trade Completed",
		Body:          strPtr(fmt.Sprintf("Your trade for %s has been completed!", itemName)),
		ReferenceType: &refType,
		ReferenceID:   &tradeID,
	}
	return s.Create(ctx, notification)
}

// NotifyTradeCancelled notifies a user the trade was cancelled
func (s *NotificationService) NotifyTradeCancelled(ctx context.Context, userID string, tradeID string, itemName string) error {
	refType := "trade"
	notification := &models.Notification{
		UserID:        userID,
		Type:          models.NotificationTypeTradeRequestRejected, // Using existing type
		Title:         "Trade Cancelled",
		Body:          strPtr(fmt.Sprintf("The trade for %s has been cancelled.", itemName)),
		ReferenceType: &refType,
		ReferenceID:   &tradeID,
	}
	return s.Create(ctx, notification)
}

// NotifyNewMessage notifies a user of a new message
func (s *NotificationService) NotifyNewMessage(ctx context.Context, userID string, tradeID string, senderName string) error {
	refType := "trade"
	notification := &models.Notification{
		UserID:        userID,
		Type:          models.NotificationTypeNewMessage,
		Title:         "New Message",
		Body:          strPtr(fmt.Sprintf("New message from %s", senderName)),
		ReferenceType: &refType,
		ReferenceID:   &tradeID,
	}
	return s.Create(ctx, notification)
}

// NotifyRatingReceived notifies a user they received a rating
func (s *NotificationService) NotifyRatingReceived(ctx context.Context, userID string, transactionID string, stars int) error {
	refType := "transaction"
	notification := &models.Notification{
		UserID:        userID,
		Type:          models.NotificationTypeRatingReceived,
		Title:         "New Rating",
		Body:          strPtr(fmt.Sprintf("You received a %d-star rating", stars)),
		ReferenceType: &refType,
		ReferenceID:   &transactionID,
	}
	return s.Create(ctx, notification)
}

// ToResponse converts a notification model to a DTO response
func (s *NotificationService) ToResponse(notification *models.Notification) *dto.NotificationResponse {
	return &dto.NotificationResponse{
		ID:            notification.ID,
		Type:          string(notification.Type),
		Title:         notification.Title,
		Body:          notification.GetBody(),
		ReferenceType: notification.GetReferenceType(),
		ReferenceID:   notification.GetReferenceID(),
		Read:          notification.Read,
		ReadAt:        notification.ReadAt,
		Metadata:      notification.Metadata,
		CreatedAt:     notification.CreatedAt,
	}
}

func strPtr(s string) *string {
	return &s
}
