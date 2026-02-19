package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

// Test constants
const (
	testUserID        = "user-111"
	testSellerID      = "seller-222"
	testBuyerID       = "buyer-333"
	testProviderID    = "provider-444"
	testClientID      = "client-555"
	testListingID     = "listing-aaa"
	testOfferID       = "offer-bbb"
	testTradeID       = "trade-ccc"
	testChatID        = "chat-ddd"
	testMessageID     = "msg-eee"
	testTransactionID = "txn-fff"
	testRatingID      = "rating-ggg"
	testServiceID     = "service-hhh"
	testServiceRunID  = "servicerun-iii"
	testWishlistID    = "wishlist-jjj"
)

// newTestRedis returns a nil RedisClient (all cache ops are no-ops).
// Use for tests that don't need to assert cache behavior.
func newTestRedis() *cache.RedisClient {
	return nil
}

// newTestRedisReal returns a RedisClient backed by miniredis for cache assertion tests.
// Returns the client and the miniredis server (for inspecting keys/values).
func newTestRedisReal(t *testing.T) (*cache.RedisClient, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() { client.Close() })

	return cache.NewRedisClientFromClient(client), mr
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int {
	return &i
}

// boolPtr returns a pointer to a bool.
func boolPtr(b bool) *bool {
	return &b
}

// testProfile creates a test profile with sensible defaults.
func testProfile(id string, opts ...func(*models.Profile)) *models.Profile {
	p := &models.Profile{
		ID:            id,
		Username:      "testuser_" + id[:8],
		DisplayName:   strPtr("Test User"),
		AvatarURL:     strPtr("https://example.com/avatar.png"),
		TotalTrades:   5,
		AverageRating: 4.5,
		RatingCount:   3,
		IsPremium:     false,
		IsAdmin:       false,
		CreatedAt:     time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:     time.Now(),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func withPremium(p *models.Profile)           { p.IsPremium = true }
func withAdmin(p *models.Profile)             { p.IsAdmin = true }
func withTimezone(tz string) func(*models.Profile) {
	return func(p *models.Profile) { p.Timezone = &tz }
}

// testListing creates a test listing with sensible defaults.
func testListing(id string, sellerID string, opts ...func(*models.Listing)) *models.Listing {
	l := &models.Listing{
		ID:        id,
		SellerID:  sellerID,
		Name:      "Shako",
		ItemType:  "unique",
		Rarity:    "unique",
		Category:  "helm",
		Game:      "diablo2",
		Ladder:    true,
		Hardcore:  false,
		IsNonRotw: false,
		Platforms: []string{"pc"},
		Region:    "americas",
		Status:    "active",
		Amount:    1,
		Views:     10,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(29 * 24 * time.Hour),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func withStats(stats json.RawMessage) func(*models.Listing) {
	return func(l *models.Listing) { l.Stats = stats }
}

func withRunes(runes json.RawMessage) func(*models.Listing) {
	return func(l *models.Listing) { l.Runes = runes }
}

func withSeller(seller *models.Profile) func(*models.Listing) {
	return func(l *models.Listing) { l.Seller = seller }
}

func withListingStatus(status string) func(*models.Listing) {
	return func(l *models.Listing) { l.Status = status }
}

// testOffer creates a test offer with sensible defaults (item type).
func testOffer(id string, requesterID string, listingID *string, opts ...func(*models.Offer)) *models.Offer {
	o := &models.Offer{
		ID:          id,
		Type:        "item",
		RequesterID: requesterID,
		ListingID:   listingID,
		Status:      "pending",
		CreatedAt:   time.Now().Add(-1 * time.Hour),
		UpdatedAt:   time.Now(),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func withOfferListing(listing *models.Listing) func(*models.Offer) {
	return func(o *models.Offer) { o.Listing = listing }
}

func withOfferService(svc *models.Service) func(*models.Offer) {
	return func(o *models.Offer) {
		o.Type = "service"
		o.Service = svc
		o.ServiceID = &svc.ID
		o.ListingID = nil
	}
}

func withOfferStatus(status string) func(*models.Offer) {
	return func(o *models.Offer) { o.Status = status }
}

// testTrade creates a test trade with sensible defaults.
func testTrade(id, offerID, listingID, sellerID, buyerID string, opts ...func(*models.Trade)) *models.Trade {
	t := &models.Trade{
		ID:        id,
		OfferID:   offerID,
		ListingID: listingID,
		SellerID:  sellerID,
		BuyerID:   buyerID,
		Status:    "active",
		CreatedAt: time.Now().Add(-30 * time.Minute),
		UpdatedAt: time.Now(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func withTradeStatus(status string) func(*models.Trade) {
	return func(t *models.Trade) {
		t.Status = status
		if status == "completed" {
			now := time.Now()
			t.CompletedAt = &now
		}
	}
}

func withTradeOffer(offer *models.Offer) func(*models.Trade) {
	return func(t *models.Trade) { t.Offer = offer }
}

func withTradeListing(listing *models.Listing) func(*models.Trade) {
	return func(t *models.Trade) { t.Listing = listing }
}

func withTradeSeller(seller *models.Profile) func(*models.Trade) {
	return func(t *models.Trade) { t.Seller = seller }
}

func withTradeBuyer(buyer *models.Profile) func(*models.Trade) {
	return func(t *models.Trade) { t.Buyer = buyer }
}

func withTradeChat(chat *models.Chat) func(*models.Trade) {
	return func(t *models.Trade) { t.Chat = chat }
}

// testChat creates a test chat.
func testChat(id string, tradeID *string, serviceRunID *string) *models.Chat {
	return &models.Chat{
		ID:           id,
		TradeID:      tradeID,
		ServiceRunID: serviceRunID,
		CreatedAt:    time.Now().Add(-20 * time.Minute),
		UpdatedAt:    time.Now(),
	}
}

func testChatWithTrade(id string, trade *models.Trade) *models.Chat {
	c := testChat(id, &trade.ID, nil)
	c.Trade = trade
	return c
}

func testChatWithServiceRun(id string, sr *models.ServiceRun) *models.Chat {
	c := testChat(id, nil, &sr.ID)
	c.ServiceRun = sr
	return c
}

// testTransaction creates a test transaction.
func testTransaction(id, sellerID, buyerID string) *models.Transaction {
	tradeID := testTradeID
	listingID := testListingID
	return &models.Transaction{
		ID:        id,
		TradeID:   &tradeID,
		ListingID: &listingID,
		SellerID:  sellerID,
		BuyerID:   buyerID,
		ItemName:  "Shako",
		CreatedAt: time.Now(),
	}
}

// testRating creates a test rating.
func testRating(id, transactionID, raterID, ratedID string, stars int) *models.Rating {
	return &models.Rating{
		ID:            id,
		TransactionID: transactionID,
		RaterID:       raterID,
		RatedID:       ratedID,
		Stars:         stars,
		CreatedAt:     time.Now(),
	}
}

// testNotification creates a test notification.
func testNotification(id, userID string, ntype models.NotificationType) *models.Notification {
	return &models.Notification{
		ID:        id,
		UserID:    userID,
		Type:      ntype,
		Title:     "Test Notification",
		CreatedAt: time.Now(),
	}
}

// testWishlistItem creates a test wishlist item.
func testWishlistItem(id, userID string, opts ...func(*models.WishlistItem)) *models.WishlistItem {
	w := &models.WishlistItem{
		ID:        id,
		UserID:    userID,
		Name:      "Shako",
		Game:      "diablo2",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

func withStatCriteria(criteria []models.StatCriterion) func(*models.WishlistItem) {
	return func(w *models.WishlistItem) { w.StatCriteria = criteria }
}

// testServiceModel creates a test service model.
func testServiceModel(id, providerID string, opts ...func(*models.Service)) *models.Service {
	s := &models.Service{
		ID:          id,
		ProviderID:  providerID,
		ServiceType: "rush",
		Name:        "Normal Rush",
		Game:        "diablo2",
		Platforms:   []string{"pc"},
		Region:      "americas",
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func withServiceStatus(status string) func(*models.Service) {
	return func(s *models.Service) { s.Status = status }
}

// testServiceRun creates a test service run.
func testServiceRun(id, serviceID, offerID, providerID, clientID string, opts ...func(*models.ServiceRun)) *models.ServiceRun {
	sr := &models.ServiceRun{
		ID:         id,
		ServiceID:  serviceID,
		OfferID:    offerID,
		ProviderID: providerID,
		ClientID:   clientID,
		Status:     "active",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	for _, opt := range opts {
		opt(sr)
	}
	return sr
}

func withServiceRunStatus(status string) func(*models.ServiceRun) {
	return func(sr *models.ServiceRun) {
		sr.Status = status
		if status == "completed" {
			now := time.Now()
			sr.CompletedAt = &now
		}
	}
}
