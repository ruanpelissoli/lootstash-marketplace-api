package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---------------------------------------------------------------------------
// CountUnread
// ---------------------------------------------------------------------------

func TestCountUnread_WritesToCache(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	redisClient, mr := newTestRedisReal(t)
	svc := NewNotificationService(notifRepo, redisClient)

	ctx := context.Background()
	userID := testUserID
	expectedCount := 7

	notifRepo.On("CountUnread", ctx, userID).Return(expectedCount, nil)

	count, err := svc.CountUnread(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, expectedCount, count)

	// Verify the key was written to Redis
	cacheKey := cache.NotificationCountKey(userID)
	val, err := mr.Get(cacheKey)
	assert.NoError(t, err)

	var cachedCount int
	err = json.Unmarshal([]byte(val), &cachedCount)
	assert.NoError(t, err)
	assert.Equal(t, expectedCount, cachedCount)

	notifRepo.AssertExpectations(t)
}

func TestCountUnread_CacheHit(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	redisClient, mr := newTestRedisReal(t)
	svc := NewNotificationService(notifRepo, redisClient)

	ctx := context.Background()
	userID := testUserID
	cachedCount := 42

	// Pre-set value in miniredis
	cacheKey := cache.NotificationCountKey(userID)
	data, _ := json.Marshal(cachedCount)
	mr.Set(cacheKey, string(data))

	count, err := svc.CountUnread(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, cachedCount, count)

	// Repo should NOT have been called
	notifRepo.AssertNotCalled(t, "CountUnread", mock.Anything, mock.Anything)
}

// ---------------------------------------------------------------------------
// MarkAsRead
// ---------------------------------------------------------------------------

func TestMarkAsRead_Success(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	redisClient, _ := newTestRedisReal(t)
	svc := NewNotificationService(notifRepo, redisClient)

	ctx := context.Background()
	userID := testUserID
	notificationIDs := []string{"notif-1", "notif-2"}

	notifRepo.On("MarkAsRead", ctx, notificationIDs, userID).Return(nil)

	err := svc.MarkAsRead(ctx, userID, notificationIDs)
	assert.NoError(t, err)

	notifRepo.AssertExpectations(t)
}

func TestMarkAsRead_InvalidatesCache(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	redisClient, mr := newTestRedisReal(t)
	svc := NewNotificationService(notifRepo, redisClient)

	ctx := context.Background()
	userID := testUserID
	notificationIDs := []string{"notif-1"}

	// Pre-set count in cache
	cacheKey := cache.NotificationCountKey(userID)
	mr.Set(cacheKey, "5")

	// Verify it exists
	assert.True(t, mr.Exists(cacheKey))

	notifRepo.On("MarkAsRead", ctx, notificationIDs, userID).Return(nil)

	err := svc.MarkAsRead(ctx, userID, notificationIDs)
	assert.NoError(t, err)

	// Verify cache was invalidated
	assert.False(t, mr.Exists(cacheKey))

	notifRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestNotificationCreate_SetsIDAndTimestamp(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	redisClient, _ := newTestRedisReal(t)
	svc := NewNotificationService(notifRepo, redisClient)

	ctx := context.Background()
	notification := &models.Notification{
		UserID: testUserID,
		Type:   models.NotificationTypeNewMessage,
		Title:  "Test",
	}

	notifRepo.On("Create", ctx, notification).Return(nil)

	before := time.Now().Add(-1 * time.Second)
	err := svc.Create(ctx, notification)
	after := time.Now().Add(1 * time.Second)

	assert.NoError(t, err)

	// Verify ID is a valid UUID
	assert.NotEmpty(t, notification.ID)
	_, parseErr := uuid.Parse(notification.ID)
	assert.NoError(t, parseErr)

	// Verify CreatedAt is recent
	assert.True(t, notification.CreatedAt.After(before), "CreatedAt should be after test start")
	assert.True(t, notification.CreatedAt.Before(after), "CreatedAt should be before test end")

	notifRepo.AssertExpectations(t)
}

func TestNotificationCreate_InvalidatesCountCache(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	redisClient, mr := newTestRedisReal(t)
	svc := NewNotificationService(notifRepo, redisClient)

	ctx := context.Background()

	// Pre-set count in cache
	cacheKey := cache.NotificationCountKey(testUserID)
	mr.Set(cacheKey, "10")
	assert.True(t, mr.Exists(cacheKey))

	notification := &models.Notification{
		UserID: testUserID,
		Type:   models.NotificationTypeNewMessage,
		Title:  "Test",
	}

	notifRepo.On("Create", ctx, notification).Return(nil)

	err := svc.Create(ctx, notification)
	assert.NoError(t, err)

	// Verify cache was invalidated
	assert.False(t, mr.Exists(cacheKey))

	notifRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Notify methods
// ---------------------------------------------------------------------------

func TestNotifyOfferReceived(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	svc := NewNotificationService(notifRepo, newTestRedis())

	ctx := context.Background()
	userID := testUserID
	offerID := testOfferID
	itemName := "Shako"

	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).
		Run(func(args mock.Arguments) {
			n := args.Get(1).(*models.Notification)
			assert.Equal(t, userID, n.UserID)
			assert.Equal(t, models.NotificationTypeTradeRequestReceived, n.Type)
			assert.Equal(t, "New Offer", n.Title)
			assert.NotNil(t, n.Body)
			assert.Contains(t, *n.Body, itemName)
			assert.NotNil(t, n.ReferenceType)
			assert.Equal(t, "offer", *n.ReferenceType)
			assert.NotNil(t, n.ReferenceID)
			assert.Equal(t, offerID, *n.ReferenceID)
		}).Return(nil)

	err := svc.NotifyOfferReceived(ctx, userID, offerID, itemName)
	assert.NoError(t, err)

	notifRepo.AssertExpectations(t)
}

func TestNotifyOfferAccepted(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	svc := NewNotificationService(notifRepo, newTestRedis())

	ctx := context.Background()
	userID := testUserID
	offerID := testOfferID
	itemName := "Enigma"

	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).
		Run(func(args mock.Arguments) {
			n := args.Get(1).(*models.Notification)
			assert.Equal(t, userID, n.UserID)
			assert.Equal(t, models.NotificationTypeTradeRequestAccepted, n.Type)
			assert.Equal(t, "Offer Accepted", n.Title)
			assert.NotNil(t, n.Body)
			assert.Contains(t, *n.Body, itemName)
			assert.NotNil(t, n.ReferenceType)
			assert.Equal(t, "offer", *n.ReferenceType)
			assert.NotNil(t, n.ReferenceID)
			assert.Equal(t, offerID, *n.ReferenceID)
		}).Return(nil)

	err := svc.NotifyOfferAccepted(ctx, userID, offerID, itemName)
	assert.NoError(t, err)

	notifRepo.AssertExpectations(t)
}

func TestNotifyOfferRejected(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	svc := NewNotificationService(notifRepo, newTestRedis())

	ctx := context.Background()
	userID := testUserID
	offerID := testOfferID
	itemName := "Griffon's Eye"

	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).
		Run(func(args mock.Arguments) {
			n := args.Get(1).(*models.Notification)
			assert.Equal(t, userID, n.UserID)
			assert.Equal(t, models.NotificationTypeTradeRequestRejected, n.Type)
			assert.Equal(t, "Offer Rejected", n.Title)
			assert.NotNil(t, n.Body)
			assert.Contains(t, *n.Body, itemName)
			assert.NotNil(t, n.ReferenceType)
			assert.Equal(t, "offer", *n.ReferenceType)
			assert.NotNil(t, n.ReferenceID)
			assert.Equal(t, offerID, *n.ReferenceID)
		}).Return(nil)

	err := svc.NotifyOfferRejected(ctx, userID, offerID, itemName)
	assert.NoError(t, err)

	notifRepo.AssertExpectations(t)
}

func TestNotifyTradeCompleted(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	svc := NewNotificationService(notifRepo, newTestRedis())

	ctx := context.Background()
	userID := testUserID
	tradeID := testTradeID
	itemName := "Jah Rune"

	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).
		Run(func(args mock.Arguments) {
			n := args.Get(1).(*models.Notification)
			assert.Equal(t, userID, n.UserID)
			assert.Equal(t, models.NotificationTypeTradeRequestAccepted, n.Type)
			assert.Equal(t, "Trade Completed", n.Title)
			assert.NotNil(t, n.Body)
			assert.Contains(t, *n.Body, itemName)
			assert.NotNil(t, n.ReferenceType)
			assert.Equal(t, "trade", *n.ReferenceType)
			assert.NotNil(t, n.ReferenceID)
			assert.Equal(t, tradeID, *n.ReferenceID)
		}).Return(nil)

	err := svc.NotifyTradeCompleted(ctx, userID, tradeID, itemName)
	assert.NoError(t, err)

	notifRepo.AssertExpectations(t)
}

func TestNotifyTradeCancelled(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	svc := NewNotificationService(notifRepo, newTestRedis())

	ctx := context.Background()
	userID := testUserID
	tradeID := testTradeID
	itemName := "Ber Rune"

	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).
		Run(func(args mock.Arguments) {
			n := args.Get(1).(*models.Notification)
			assert.Equal(t, userID, n.UserID)
			assert.Equal(t, models.NotificationTypeTradeRequestRejected, n.Type)
			assert.Equal(t, "Trade Cancelled", n.Title)
			assert.NotNil(t, n.Body)
			assert.Contains(t, *n.Body, itemName)
			assert.NotNil(t, n.ReferenceType)
			assert.Equal(t, "trade", *n.ReferenceType)
			assert.NotNil(t, n.ReferenceID)
			assert.Equal(t, tradeID, *n.ReferenceID)
		}).Return(nil)

	err := svc.NotifyTradeCancelled(ctx, userID, tradeID, itemName)
	assert.NoError(t, err)

	notifRepo.AssertExpectations(t)
}

func TestNotifyNewMessage(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	svc := NewNotificationService(notifRepo, newTestRedis())

	ctx := context.Background()
	userID := testUserID
	referenceID := testChatID
	senderName := "TradeMaster99"

	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).
		Run(func(args mock.Arguments) {
			n := args.Get(1).(*models.Notification)
			assert.Equal(t, userID, n.UserID)
			assert.Equal(t, models.NotificationTypeNewMessage, n.Type)
			assert.Equal(t, "New Message", n.Title)
			assert.NotNil(t, n.Body)
			assert.Contains(t, *n.Body, senderName)
			assert.NotNil(t, n.ReferenceType)
			assert.Equal(t, "chat", *n.ReferenceType)
			assert.NotNil(t, n.ReferenceID)
			assert.Equal(t, referenceID, *n.ReferenceID)
		}).Return(nil)

	err := svc.NotifyNewMessage(ctx, userID, referenceID, senderName)
	assert.NoError(t, err)

	notifRepo.AssertExpectations(t)
}

func TestNotifyRatingReceived(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	svc := NewNotificationService(notifRepo, newTestRedis())

	ctx := context.Background()
	userID := testUserID
	transactionID := testTransactionID
	stars := 5

	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).
		Run(func(args mock.Arguments) {
			n := args.Get(1).(*models.Notification)
			assert.Equal(t, userID, n.UserID)
			assert.Equal(t, models.NotificationTypeRatingReceived, n.Type)
			assert.Equal(t, "New Rating", n.Title)
			assert.NotNil(t, n.Body)
			assert.Contains(t, *n.Body, fmt.Sprintf("%d-star", stars))
			assert.NotNil(t, n.ReferenceType)
			assert.Equal(t, "transaction", *n.ReferenceType)
			assert.NotNil(t, n.ReferenceID)
			assert.Equal(t, transactionID, *n.ReferenceID)
		}).Return(nil)

	err := svc.NotifyRatingReceived(ctx, userID, transactionID, stars)
	assert.NoError(t, err)

	notifRepo.AssertExpectations(t)
}

func TestNotifyServiceRunCreated(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	svc := NewNotificationService(notifRepo, newTestRedis())

	ctx := context.Background()
	userID := testUserID
	serviceRunID := testServiceRunID
	serviceName := "Normal Rush"

	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).
		Run(func(args mock.Arguments) {
			n := args.Get(1).(*models.Notification)
			assert.Equal(t, userID, n.UserID)
			assert.Equal(t, models.NotificationTypeServiceRunCreated, n.Type)
			assert.Equal(t, "Service Run Started", n.Title)
			assert.NotNil(t, n.Body)
			assert.Contains(t, *n.Body, serviceName)
			assert.NotNil(t, n.ReferenceType)
			assert.Equal(t, "service_run", *n.ReferenceType)
			assert.NotNil(t, n.ReferenceID)
			assert.Equal(t, serviceRunID, *n.ReferenceID)
		}).Return(nil)

	err := svc.NotifyServiceRunCreated(ctx, userID, serviceRunID, serviceName)
	assert.NoError(t, err)

	notifRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ToResponse
// ---------------------------------------------------------------------------

func TestNotificationToResponse(t *testing.T) {
	notifRepo := new(mocks.MockNotificationRepository)
	svc := NewNotificationService(notifRepo, newTestRedis())

	now := time.Now()
	readAt := now.Add(-5 * time.Minute)
	metadata := json.RawMessage(`{"key":"value"}`)
	refType := "offer"
	refID := testOfferID
	body := "You received a new offer for Shako"

	notification := &models.Notification{
		ID:            "notif-123",
		UserID:        testUserID,
		Type:          models.NotificationTypeTradeRequestReceived,
		Title:         "New Offer",
		Body:          &body,
		ReferenceType: &refType,
		ReferenceID:   &refID,
		Read:          true,
		ReadAt:        &readAt,
		Metadata:      metadata,
		CreatedAt:     now,
	}

	resp := svc.ToResponse(notification)

	assert.Equal(t, "notif-123", resp.ID)
	assert.Equal(t, "trade_request_received", resp.Type)
	assert.Equal(t, "New Offer", resp.Title)
	assert.Equal(t, body, resp.Body)
	assert.Equal(t, "offer", resp.ReferenceType)
	assert.Equal(t, testOfferID, resp.ReferenceID)
	assert.True(t, resp.Read)
	assert.NotNil(t, resp.ReadAt)
	assert.Equal(t, readAt, *resp.ReadAt)
	assert.Equal(t, metadata, resp.Metadata)
	assert.Equal(t, now, resp.CreatedAt)
}
