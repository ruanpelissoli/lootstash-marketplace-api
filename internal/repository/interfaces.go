package repository

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

// ProfileRepository defines the interface for profile data access
type ProfileRepository interface {
	GetByID(ctx context.Context, id string) (*models.Profile, error)
	GetByStripeCustomerID(ctx context.Context, customerID string) (*models.Profile, error)
	GetByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*models.Profile, error)
	Update(ctx context.Context, profile *models.Profile) error
	GetEmailByID(ctx context.Context, id string) (string, error)
	UpdateLastActiveAt(ctx context.Context, userID string) error
}

// ListingRepository defines the interface for listing data access
type ListingRepository interface {
	Create(ctx context.Context, listing *models.Listing) error
	GetByID(ctx context.Context, id string) (*models.Listing, error)
	GetByIDWithSeller(ctx context.Context, id string) (*models.Listing, error)
	Update(ctx context.Context, listing *models.Listing) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter ListingFilter) ([]*models.Listing, int, error)
	ListBySellerID(ctx context.Context, sellerID string, status string, offset, limit int) ([]*models.Listing, int, error)
	CountByListingID(ctx context.Context, listingID string) (int, error)
	CountActiveBySellerID(ctx context.Context, sellerID string) (int, error)
	IncrementViews(ctx context.Context, id string) error
	CountActive(ctx context.Context) (int, error)
}

// StatsRepository defines the interface for marketplace stats data access
type StatsRepository interface {
	GetMarketplaceStats(ctx context.Context) (*MarketplaceStats, error)
}

// MarketplaceStats holds aggregated marketplace statistics
type MarketplaceStats struct {
	ActiveListings         int
	TradesToday            int
	OnlineSellers          int
	AvgResponseTimeMinutes float64
}

// ListingFilter represents listing query parameters
type ListingFilter struct {
	SellerID         string
	Query            string
	CatalogItemID    string
	Game             string
	Ladder           *bool
	Hardcore         *bool
	Platform         string
	Region           string
	Category         string
	Rarity           string
	AffixFilters     []AffixFilter
	AskingForFilters []AskingForFilter
	SortBy           string
	SortOrder        string
	Offset           int
	Limit            int
}

// AffixFilter represents an affix filter for JSONB queries
type AffixFilter struct {
	Code     string
	MinValue *int
	MaxValue *int
}

// AskingForFilter represents an asking_for filter for JSONB queries
type AskingForFilter struct {
	Name        string
	Type        string
	MinQuantity *int
}

// OfferRepository defines the interface for offer data access
type OfferRepository interface {
	Create(ctx context.Context, offer *models.Offer) error
	GetByID(ctx context.Context, id string) (*models.Offer, error)
	GetByIDWithRelations(ctx context.Context, id string) (*models.Offer, error)
	Update(ctx context.Context, offer *models.Offer) error
	List(ctx context.Context, filter OfferFilter) ([]*models.Offer, int, error)
	GetDeclineReasons(ctx context.Context) ([]*models.DeclineReason, error)
	GetDeclineReasonByID(ctx context.Context, id int) (*models.DeclineReason, error)
}

// OfferFilter represents offer query parameters
type OfferFilter struct {
	UserID    string // Required for permission filtering (unless ListingID is set by owner)
	Role      string // buyer, seller, all
	Status    string // pending, accepted, rejected, cancelled
	ListingID string // Filter by specific listing
	Offset    int
	Limit     int
}

// TradeRepository defines the interface for trade data access
type TradeRepository interface {
	Create(ctx context.Context, trade *models.Trade) error
	GetByID(ctx context.Context, id string) (*models.Trade, error)
	GetByIDWithRelations(ctx context.Context, id string) (*models.Trade, error)
	GetByOfferID(ctx context.Context, offerID string) (*models.Trade, error)
	Update(ctx context.Context, trade *models.Trade) error
	List(ctx context.Context, filter TradeFilter) ([]*models.Trade, int, error)
	HasActiveTradeForListing(ctx context.Context, listingID string) (bool, error)
}

// TradeFilter represents trade query parameters
type TradeFilter struct {
	UserID string // Required for permission filtering
	Status string // active, completed, cancelled
	Offset int
	Limit  int
}

// ChatRepository defines the interface for chat data access
type ChatRepository interface {
	Create(ctx context.Context, chat *models.Chat) error
	GetByID(ctx context.Context, id string) (*models.Chat, error)
	GetByIDWithTrade(ctx context.Context, id string) (*models.Chat, error)
	GetByTradeID(ctx context.Context, tradeID string) (*models.Chat, error)
	Update(ctx context.Context, chat *models.Chat) error
}

// MessageRepository defines the interface for message data access
type MessageRepository interface {
	Create(ctx context.Context, message *models.Message) error
	GetByChatID(ctx context.Context, chatID string, offset, limit int) ([]*models.Message, int, error)
	MarkAsRead(ctx context.Context, messageIDs []string, userID string) error
	MarkAllAsReadInChat(ctx context.Context, chatID string, userID string) error
	CountUnread(ctx context.Context, userID string) (int, error)
}

// NotificationRepository defines the interface for notification data access
type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	GetByUserID(ctx context.Context, userID string, unreadOnly bool, notificationType string, offset, limit int) ([]*models.Notification, int, error)
	CountUnread(ctx context.Context, userID string) (int, error)
	MarkAsRead(ctx context.Context, notificationIDs []string, userID string) error
}

// TransactionRepository defines the interface for transaction data access
type TransactionRepository interface {
	Create(ctx context.Context, transaction *models.Transaction) error
	GetByID(ctx context.Context, id string) (*models.Transaction, error)
	GetByTradeID(ctx context.Context, tradeID string) (*models.Transaction, error)
	GetTradeVolume(ctx context.Context, itemName string, days int) ([]TradeVolumePoint, error)
	GetSalesBySeller(ctx context.Context, sellerID string, offset, limit int) ([]SaleRecord, int, error)
}

// SaleRecord represents a completed sale with all related data
type SaleRecord struct {
	TransactionID string
	CompletedAt   interface{} // time.Time
	ItemName      string
	ItemType      string
	Rarity        string
	ImageURL      *string
	BaseName      *string
	Stats         []byte // json.RawMessage
	OfferedItems  []byte // json.RawMessage
	BuyerID       string
	BuyerName     string
	BuyerAvatar   *string
	ReviewRating  *int
	ReviewComment *string
	ReviewedAt    interface{} // *time.Time
}

// TradeVolumePoint represents a single day's trade volume
type TradeVolumePoint struct {
	Date   string
	Volume int
}

// BillingEventRepository defines the interface for billing event data access
type BillingEventRepository interface {
	Create(ctx context.Context, event *models.BillingEvent) error
	GetByUserID(ctx context.Context, userID string) ([]*models.BillingEvent, error)
	ExistsByStripeEventID(ctx context.Context, stripeEventID string) (bool, error)
}

// WishlistRepository defines the interface for wishlist data access
type WishlistRepository interface {
	Create(ctx context.Context, item *models.WishlistItem) error
	GetByID(ctx context.Context, id string) (*models.WishlistItem, error)
	Update(ctx context.Context, item *models.WishlistItem) error
	Delete(ctx context.Context, id string) error
	ListByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.WishlistItem, int, error)
	CountActiveByUserID(ctx context.Context, userID string) (int, error)
	FindMatchingItems(ctx context.Context, listing *models.Listing) ([]*models.WishlistItem, error)
}

// RatingRepository defines the interface for rating data access
type RatingRepository interface {
	Create(ctx context.Context, rating *models.Rating) error
	GetByTransactionID(ctx context.Context, transactionID string) ([]*models.Rating, error)
	GetByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.Rating, int, error)
	Exists(ctx context.Context, transactionID, raterID string) (bool, error)
}
