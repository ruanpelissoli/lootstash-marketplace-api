package mocks

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/stretchr/testify/mock"
)

// MockProfileRepository is a mock implementation of repository.ProfileRepository
type MockProfileRepository struct {
	mock.Mock
}

func (m *MockProfileRepository) GetByID(ctx context.Context, id string) (*models.Profile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Profile), args.Error(1)
}

func (m *MockProfileRepository) GetByUsername(ctx context.Context, username string) (*models.Profile, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Profile), args.Error(1)
}

func (m *MockProfileRepository) GetByStripeCustomerID(ctx context.Context, customerID string) (*models.Profile, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Profile), args.Error(1)
}

func (m *MockProfileRepository) GetByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*models.Profile, error) {
	args := m.Called(ctx, subscriptionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Profile), args.Error(1)
}

func (m *MockProfileRepository) Update(ctx context.Context, profile *models.Profile) error {
	args := m.Called(ctx, profile)
	return args.Error(0)
}

func (m *MockProfileRepository) GetEmailByID(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}

func (m *MockProfileRepository) UpdateLastActiveAt(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockListingRepository is a mock implementation of repository.ListingRepository
type MockListingRepository struct {
	mock.Mock
}

func (m *MockListingRepository) Create(ctx context.Context, listing *models.Listing) error {
	args := m.Called(ctx, listing)
	return args.Error(0)
}

func (m *MockListingRepository) GetByID(ctx context.Context, id string) (*models.Listing, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Listing), args.Error(1)
}

func (m *MockListingRepository) GetByIDWithSeller(ctx context.Context, id string) (*models.Listing, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Listing), args.Error(1)
}

func (m *MockListingRepository) Update(ctx context.Context, listing *models.Listing) error {
	args := m.Called(ctx, listing)
	return args.Error(0)
}

func (m *MockListingRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockListingRepository) List(ctx context.Context, filter repository.ListingFilter) ([]*models.Listing, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Listing), args.Int(1), args.Error(2)
}

func (m *MockListingRepository) ListBySellerID(ctx context.Context, sellerID string, status string, offset, limit int) ([]*models.Listing, int, error) {
	args := m.Called(ctx, sellerID, status, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Listing), args.Int(1), args.Error(2)
}

func (m *MockListingRepository) CountByListingID(ctx context.Context, listingID string) (int, error) {
	args := m.Called(ctx, listingID)
	return args.Int(0), args.Error(1)
}

func (m *MockListingRepository) CountActiveBySellerID(ctx context.Context, sellerID string) (int, error) {
	args := m.Called(ctx, sellerID)
	return args.Int(0), args.Error(1)
}

func (m *MockListingRepository) IncrementViews(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockListingRepository) CountActive(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockListingRepository) CancelOldestActiveListings(ctx context.Context, sellerID string, keepCount int) (int, error) {
	args := m.Called(ctx, sellerID, keepCount)
	return args.Int(0), args.Error(1)
}

// MockStatsRepository is a mock implementation of repository.StatsRepository
type MockStatsRepository struct {
	mock.Mock
}

func (m *MockStatsRepository) GetMarketplaceStats(ctx context.Context) (*repository.MarketplaceStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.MarketplaceStats), args.Error(1)
}

// MockOfferRepository is a mock implementation of repository.OfferRepository
type MockOfferRepository struct {
	mock.Mock
}

func (m *MockOfferRepository) Create(ctx context.Context, offer *models.Offer) error {
	args := m.Called(ctx, offer)
	return args.Error(0)
}

func (m *MockOfferRepository) GetByID(ctx context.Context, id string) (*models.Offer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Offer), args.Error(1)
}

func (m *MockOfferRepository) GetByIDWithRelations(ctx context.Context, id string) (*models.Offer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Offer), args.Error(1)
}

func (m *MockOfferRepository) Update(ctx context.Context, offer *models.Offer) error {
	args := m.Called(ctx, offer)
	return args.Error(0)
}

func (m *MockOfferRepository) List(ctx context.Context, filter repository.OfferFilter) ([]*models.Offer, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Offer), args.Int(1), args.Error(2)
}

func (m *MockOfferRepository) GetDeclineReasons(ctx context.Context) ([]*models.DeclineReason, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DeclineReason), args.Error(1)
}

func (m *MockOfferRepository) GetDeclineReasonByID(ctx context.Context, id int) (*models.DeclineReason, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DeclineReason), args.Error(1)
}

// MockTradeRepository is a mock implementation of repository.TradeRepository
type MockTradeRepository struct {
	mock.Mock
}

func (m *MockTradeRepository) Create(ctx context.Context, trade *models.Trade) error {
	args := m.Called(ctx, trade)
	return args.Error(0)
}

func (m *MockTradeRepository) GetByID(ctx context.Context, id string) (*models.Trade, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Trade), args.Error(1)
}

func (m *MockTradeRepository) GetByIDWithRelations(ctx context.Context, id string) (*models.Trade, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Trade), args.Error(1)
}

func (m *MockTradeRepository) GetByOfferID(ctx context.Context, offerID string) (*models.Trade, error) {
	args := m.Called(ctx, offerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Trade), args.Error(1)
}

func (m *MockTradeRepository) Update(ctx context.Context, trade *models.Trade) error {
	args := m.Called(ctx, trade)
	return args.Error(0)
}

func (m *MockTradeRepository) List(ctx context.Context, filter repository.TradeFilter) ([]*models.Trade, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Trade), args.Int(1), args.Error(2)
}

func (m *MockTradeRepository) HasActiveTradeForListing(ctx context.Context, listingID string) (bool, error) {
	args := m.Called(ctx, listingID)
	return args.Bool(0), args.Error(1)
}

// MockServiceRepository is a mock implementation of repository.ServiceRepository
type MockServiceRepository struct {
	mock.Mock
}

func (m *MockServiceRepository) Create(ctx context.Context, service *models.Service) error {
	args := m.Called(ctx, service)
	return args.Error(0)
}

func (m *MockServiceRepository) GetByID(ctx context.Context, id string) (*models.Service, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Service), args.Error(1)
}

func (m *MockServiceRepository) GetByIDWithProvider(ctx context.Context, id string) (*models.Service, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Service), args.Error(1)
}

func (m *MockServiceRepository) Update(ctx context.Context, service *models.Service) error {
	args := m.Called(ctx, service)
	return args.Error(0)
}

func (m *MockServiceRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServiceRepository) ListByProviderID(ctx context.Context, providerID string, offset, limit int) ([]*models.Service, int, error) {
	args := m.Called(ctx, providerID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Service), args.Int(1), args.Error(2)
}

func (m *MockServiceRepository) ListProviders(ctx context.Context, filter repository.ServiceProviderFilter) ([]repository.ProviderWithServices, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]repository.ProviderWithServices), args.Int(1), args.Error(2)
}

func (m *MockServiceRepository) GetProviderServices(ctx context.Context, providerID string) ([]*models.Service, error) {
	args := m.Called(ctx, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Service), args.Error(1)
}

func (m *MockServiceRepository) ExistsByProviderAndType(ctx context.Context, providerID string, serviceType string, game string) (bool, error) {
	args := m.Called(ctx, providerID, serviceType, game)
	return args.Bool(0), args.Error(1)
}

// MockServiceRunRepository is a mock implementation of repository.ServiceRunRepository
type MockServiceRunRepository struct {
	mock.Mock
}

func (m *MockServiceRunRepository) Create(ctx context.Context, serviceRun *models.ServiceRun) error {
	args := m.Called(ctx, serviceRun)
	return args.Error(0)
}

func (m *MockServiceRunRepository) GetByID(ctx context.Context, id string) (*models.ServiceRun, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServiceRun), args.Error(1)
}

func (m *MockServiceRunRepository) GetByIDWithRelations(ctx context.Context, id string) (*models.ServiceRun, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServiceRun), args.Error(1)
}

func (m *MockServiceRunRepository) Update(ctx context.Context, serviceRun *models.ServiceRun) error {
	args := m.Called(ctx, serviceRun)
	return args.Error(0)
}

func (m *MockServiceRunRepository) List(ctx context.Context, filter repository.ServiceRunFilter) ([]*models.ServiceRun, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.ServiceRun), args.Int(1), args.Error(2)
}

// MockChatRepository is a mock implementation of repository.ChatRepository
type MockChatRepository struct {
	mock.Mock
}

func (m *MockChatRepository) Create(ctx context.Context, chat *models.Chat) error {
	args := m.Called(ctx, chat)
	return args.Error(0)
}

func (m *MockChatRepository) GetByID(ctx context.Context, id string) (*models.Chat, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Chat), args.Error(1)
}

func (m *MockChatRepository) GetByIDWithContext(ctx context.Context, id string) (*models.Chat, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Chat), args.Error(1)
}

func (m *MockChatRepository) GetByTradeID(ctx context.Context, tradeID string) (*models.Chat, error) {
	args := m.Called(ctx, tradeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Chat), args.Error(1)
}

func (m *MockChatRepository) GetByServiceRunID(ctx context.Context, serviceRunID string) (*models.Chat, error) {
	args := m.Called(ctx, serviceRunID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Chat), args.Error(1)
}

func (m *MockChatRepository) Update(ctx context.Context, chat *models.Chat) error {
	args := m.Called(ctx, chat)
	return args.Error(0)
}

// MockMessageRepository is a mock implementation of repository.MessageRepository
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) Create(ctx context.Context, message *models.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessageRepository) GetByChatID(ctx context.Context, chatID string, offset, limit int) ([]*models.Message, int, error) {
	args := m.Called(ctx, chatID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Message), args.Int(1), args.Error(2)
}

func (m *MockMessageRepository) MarkAsRead(ctx context.Context, messageIDs []string, userID string) error {
	args := m.Called(ctx, messageIDs, userID)
	return args.Error(0)
}

func (m *MockMessageRepository) MarkAllAsReadInChat(ctx context.Context, chatID string, userID string) error {
	args := m.Called(ctx, chatID, userID)
	return args.Error(0)
}

func (m *MockMessageRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

// MockNotificationRepository is a mock implementation of repository.NotificationRepository
type MockNotificationRepository struct {
	mock.Mock
}

func (m *MockNotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationRepository) GetByUserID(ctx context.Context, userID string, unreadOnly bool, notificationType string, offset, limit int) ([]*models.Notification, int, error) {
	args := m.Called(ctx, userID, unreadOnly, notificationType, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Notification), args.Int(1), args.Error(2)
}

func (m *MockNotificationRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockNotificationRepository) MarkAsRead(ctx context.Context, notificationIDs []string, userID string) error {
	args := m.Called(ctx, notificationIDs, userID)
	return args.Error(0)
}

// MockTransactionRepository is a mock implementation of repository.TransactionRepository
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetByID(ctx context.Context, id string) (*models.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetByTradeID(ctx context.Context, tradeID string) (*models.Transaction, error) {
	args := m.Called(ctx, tradeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetByServiceRunID(ctx context.Context, serviceRunID string) (*models.Transaction, error) {
	args := m.Called(ctx, serviceRunID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetPriceHistory(ctx context.Context, itemName string, days int) ([]repository.PriceHistoryRecord, error) {
	args := m.Called(ctx, itemName, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.PriceHistoryRecord), args.Error(1)
}

func (m *MockTransactionRepository) GetSalesBySeller(ctx context.Context, sellerID string, offset, limit int) ([]repository.SaleRecord, int, error) {
	args := m.Called(ctx, sellerID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]repository.SaleRecord), args.Int(1), args.Error(2)
}

// MockWishlistRepository is a mock implementation of repository.WishlistRepository
type MockWishlistRepository struct {
	mock.Mock
}

func (m *MockWishlistRepository) Create(ctx context.Context, item *models.WishlistItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockWishlistRepository) GetByID(ctx context.Context, id string) (*models.WishlistItem, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WishlistItem), args.Error(1)
}

func (m *MockWishlistRepository) Update(ctx context.Context, item *models.WishlistItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockWishlistRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWishlistRepository) DeleteAllByUserID(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockWishlistRepository) ListByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.WishlistItem, int, error) {
	args := m.Called(ctx, userID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.WishlistItem), args.Int(1), args.Error(2)
}

func (m *MockWishlistRepository) CountActiveByUserID(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockWishlistRepository) FindMatchingItems(ctx context.Context, listing *models.Listing) ([]*models.WishlistItem, error) {
	args := m.Called(ctx, listing)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.WishlistItem), args.Error(1)
}

// MockBugReportRepository is a mock implementation of repository.BugReportRepository
type MockBugReportRepository struct {
	mock.Mock
}

func (m *MockBugReportRepository) Create(ctx context.Context, report *models.BugReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockBugReportRepository) GetByID(ctx context.Context, id string) (*models.BugReport, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BugReport), args.Error(1)
}

func (m *MockBugReportRepository) Update(ctx context.Context, report *models.BugReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockBugReportRepository) List(ctx context.Context, status string, offset, limit int) ([]*models.BugReport, int, error) {
	args := m.Called(ctx, status, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.BugReport), args.Int(1), args.Error(2)
}

// MockRatingRepository is a mock implementation of repository.RatingRepository
type MockRatingRepository struct {
	mock.Mock
}

func (m *MockRatingRepository) Create(ctx context.Context, rating *models.Rating) error {
	args := m.Called(ctx, rating)
	return args.Error(0)
}

func (m *MockRatingRepository) GetByTransactionID(ctx context.Context, transactionID string) ([]*models.Rating, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Rating), args.Error(1)
}

func (m *MockRatingRepository) GetByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.Rating, int, error) {
	args := m.Called(ctx, userID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Rating), args.Int(1), args.Error(2)
}

func (m *MockRatingRepository) Exists(ctx context.Context, transactionID, raterID string) (bool, error) {
	args := m.Called(ctx, transactionID, raterID)
	return args.Bool(0), args.Error(1)
}

// MockBillingEventRepository is a mock implementation of repository.BillingEventRepository
type MockBillingEventRepository struct {
	mock.Mock
}

func (m *MockBillingEventRepository) Create(ctx context.Context, event *models.BillingEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockBillingEventRepository) GetByUserID(ctx context.Context, userID string) ([]*models.BillingEvent, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.BillingEvent), args.Error(1)
}

func (m *MockBillingEventRepository) ExistsByStripeEventID(ctx context.Context, stripeEventID string) (bool, error) {
	args := m.Called(ctx, stripeEventID)
	return args.Bool(0), args.Error(1)
}
