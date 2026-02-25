package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testSupabaseURL = "https://supabase.example.com"

// newTradeServiceForTest creates a TradeServiceNew wired up with mocks for testing.
// It returns the service and all underlying mocks so callers can set expectations.
type tradeTestHarness struct {
	svc             *TradeServiceNew
	tradeRepo       *mocks.MockTradeRepository
	listingRepo     *mocks.MockListingRepository
	offerRepo       *mocks.MockOfferRepository
	transactionRepo *mocks.MockTransactionRepository
	ratingRepo      *mocks.MockRatingRepository
	chatRepo        *mocks.MockChatRepository
	messageRepo     *mocks.MockMessageRepository
	notifRepo       *mocks.MockNotificationRepository
	notifService    *NotificationService
	profileRepo     *mocks.MockProfileRepository
	profileService  *ProfileService
	listingRepoSvc  *mocks.MockListingRepository
	listingService  *ListingService
}

func newTradeTestHarness() *tradeTestHarness {
	tradeRepo := new(mocks.MockTradeRepository)
	listingRepo := new(mocks.MockListingRepository)
	offerRepo := new(mocks.MockOfferRepository)
	transactionRepo := new(mocks.MockTransactionRepository)
	ratingRepo := new(mocks.MockRatingRepository)
	chatRepo := new(mocks.MockChatRepository)
	messageRepo := new(mocks.MockMessageRepository)
	notifRepo := new(mocks.MockNotificationRepository)
	notifService := NewNotificationService(notifRepo, nil)
	profileRepo := new(mocks.MockProfileRepository)
	profileService := NewProfileService(profileRepo, nil, nil)
	listingRepoSvc := new(mocks.MockListingRepository)
	listingService := NewListingService(listingRepoSvc, profileService, nil)

	svc := NewTradeServiceNew(
		nil, tradeRepo, listingRepo, offerRepo,
		transactionRepo, ratingRepo, chatRepo, messageRepo,
		notifService, profileService, listingService,
		nil, testSupabaseURL,
	)

	return &tradeTestHarness{
		svc:             svc,
		tradeRepo:       tradeRepo,
		listingRepo:     listingRepo,
		offerRepo:       offerRepo,
		transactionRepo: transactionRepo,
		ratingRepo:      ratingRepo,
		chatRepo:        chatRepo,
		messageRepo:     messageRepo,
		notifRepo:       notifRepo,
		notifService:    notifService,
		profileRepo:     profileRepo,
		profileService:  profileService,
		listingRepoSvc:  listingRepoSvc,
		listingService:  listingService,
	}
}

func makeOfferedItemsJSON() json.RawMessage {
	items := []map[string]interface{}{
		{"name": "Ber", "type": "rune", "quantity": 1},
	}
	data, _ := json.Marshal(items)
	return data
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestTradeGetByID_Seller_Success(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)

	result, err := h.svc.GetByID(ctx, testTradeID, testSellerID)

	require.NoError(t, err)
	assert.Equal(t, testTradeID, result.ID)
	assert.Equal(t, testSellerID, result.SellerID)
	h.tradeRepo.AssertExpectations(t)
}

func TestTradeGetByID_Buyer_Success(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)

	result, err := h.svc.GetByID(ctx, testTradeID, testBuyerID)

	require.NoError(t, err)
	assert.Equal(t, testTradeID, result.ID)
	assert.Equal(t, testBuyerID, result.BuyerID)
	h.tradeRepo.AssertExpectations(t)
}

func TestTradeGetByID_NonParticipant(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)

	result, err := h.svc.GetByID(ctx, testTradeID, "stranger-999")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrForbidden)
	h.tradeRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Complete (dual-acknowledgment)
// ---------------------------------------------------------------------------

func TestTradeComplete_SellerConfirmsFirst_PartialConfirmation(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	chat := testChat(testChatID, nil, nil)
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		withTradeChat(chat),
	)

	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)
	h.tradeRepo.On("Update", ctx, mock.AnythingOfType("*models.Trade")).Return(nil)
	h.messageRepo.On("Create", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	h.notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	resultTrade, resultTx, err := h.svc.Complete(ctx, testTradeID, testSellerID)

	require.NoError(t, err)
	assert.Equal(t, "active", resultTrade.Status, "trade should remain active after one confirmation")
	assert.NotNil(t, resultTrade.SellerConfirmedAt, "seller should be confirmed")
	assert.Nil(t, resultTrade.BuyerConfirmedAt, "buyer should NOT be confirmed yet")
	assert.Nil(t, resultTx, "no transaction until both confirm")

	h.tradeRepo.AssertExpectations(t)
	h.messageRepo.AssertExpectations(t)
	// Should NOT have created a transaction
	h.transactionRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestTradeComplete_BothConfirm_FullCompletion(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	offeredItems := makeOfferedItemsJSON()
	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, &listing.ID, withOfferStatus("accepted"))
	offer.OfferedItems = offeredItems

	// Seller already confirmed
	sellerConfirmedAt := time.Now().Add(-5 * time.Minute)
	chat := testChat(testChatID, nil, nil)
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		withTradeOffer(offer),
		withTradeListing(listing),
		withTradeChat(chat),
		func(t *models.Trade) { t.SellerConfirmedAt = &sellerConfirmedAt },
	)

	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)
	h.tradeRepo.On("Update", ctx, mock.AnythingOfType("*models.Trade")).Return(nil)
	h.messageRepo.On("Create", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	h.offerRepo.On("Update", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)
	h.listingRepo.On("GetByID", ctx, testListingID).Return(listing, nil)
	h.listingRepo.On("Update", ctx, mock.AnythingOfType("*models.Listing")).Return(nil)
	h.transactionRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).Return(nil)
	h.notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	// Buyer confirms => both confirmed => finalize
	resultTrade, resultTx, err := h.svc.Complete(ctx, testTradeID, testBuyerID)

	require.NoError(t, err)
	assert.Equal(t, "completed", resultTrade.Status)
	assert.NotNil(t, resultTrade.CompletedAt)
	assert.NotNil(t, resultTrade.SellerConfirmedAt)
	assert.NotNil(t, resultTrade.BuyerConfirmedAt)
	assert.NotNil(t, resultTx)
	assert.Equal(t, listing.Name, resultTx.ItemName)

	h.tradeRepo.AssertExpectations(t)
	h.transactionRepo.AssertExpectations(t)
	h.offerRepo.AssertExpectations(t)
	h.listingRepo.AssertExpectations(t)
}

func TestTradeComplete_AlreadyConfirmed_ReturnsError(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	// Seller already confirmed
	sellerConfirmedAt := time.Now().Add(-5 * time.Minute)
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		func(t *models.Trade) { t.SellerConfirmedAt = &sellerConfirmedAt },
	)

	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)

	resultTrade, resultTx, err := h.svc.Complete(ctx, testTradeID, testSellerID)

	assert.Nil(t, resultTrade)
	assert.Nil(t, resultTx)
	assert.ErrorIs(t, err, ErrAlreadyConfirmed)
	h.tradeRepo.AssertExpectations(t)
}

func TestTradeComplete_Idempotent(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		withTradeStatus("completed"),
	)

	existingTx := testTransaction(testTransactionID, testSellerID, testBuyerID)

	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)
	h.transactionRepo.On("GetByTradeID", ctx, testTradeID).Return(existingTx, nil)

	resultTrade, resultTx, err := h.svc.Complete(ctx, testTradeID, testSellerID)

	require.NoError(t, err)
	assert.Equal(t, "completed", resultTrade.Status)
	assert.Equal(t, testTransactionID, resultTx.ID)

	// Should NOT have called Update or Create since it's already completed
	h.tradeRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	h.transactionRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	h.tradeRepo.AssertExpectations(t)
	h.transactionRepo.AssertExpectations(t)
}

func TestTradeComplete_NotActive(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	// Cancelled trade is neither completed nor active
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		withTradeStatus("cancelled"),
	)

	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)

	resultTrade, resultTx, err := h.svc.Complete(ctx, testTradeID, testSellerID)

	assert.Nil(t, resultTrade)
	assert.Nil(t, resultTx)
	assert.ErrorIs(t, err, ErrInvalidState)
	h.tradeRepo.AssertExpectations(t)
}

func TestTradeComplete_NotParticipant(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)

	resultTrade, resultTx, err := h.svc.Complete(ctx, testTradeID, "stranger-999")

	assert.Nil(t, resultTrade)
	assert.Nil(t, resultTx)
	assert.ErrorIs(t, err, ErrForbidden)
	h.tradeRepo.AssertExpectations(t)
}

func TestTradeComplete_SellerConfirms_NotifiesBuyer(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	sellerName := "SellerDude"
	seller := testProfile(testSellerID)
	seller.DisplayName = &sellerName

	chat := testChat(testChatID, nil, nil)
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		withTradeSeller(seller),
		withTradeChat(chat),
	)

	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)
	h.tradeRepo.On("Update", ctx, mock.AnythingOfType("*models.Trade")).Return(nil)
	h.messageRepo.On("Create", ctx, mock.AnythingOfType("*models.Message")).Return(nil)

	// Capture the notification to verify recipient
	h.notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).
		Run(func(args mock.Arguments) {
			notif := args.Get(1).(*models.Notification)
			// Seller confirms => buyer gets notified to confirm
			assert.Equal(t, testBuyerID, notif.UserID)
			assert.Equal(t, "Trade Confirmation", notif.Title)
		}).
		Return(nil)

	_, _, err := h.svc.Complete(ctx, testTradeID, testSellerID)

	require.NoError(t, err)
	h.notifRepo.AssertExpectations(t)
}

func TestTradeComplete_PostsSystemChatMessage(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	chat := testChat(testChatID, nil, nil)
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		withTradeChat(chat),
	)

	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)
	h.tradeRepo.On("Update", ctx, mock.AnythingOfType("*models.Trade")).Return(nil)
	h.notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	// Capture the system message
	h.messageRepo.On("Create", ctx, mock.AnythingOfType("*models.Message")).
		Run(func(args mock.Arguments) {
			msg := args.Get(1).(*models.Message)
			assert.Equal(t, testChatID, msg.ChatID)
			assert.Equal(t, "trade_update", msg.MessageType)
			assert.Contains(t, msg.Content, "Seller has confirmed")
			assert.Contains(t, msg.Content, "Waiting for buyer")
		}).
		Return(nil)

	_, _, err := h.svc.Complete(ctx, testTradeID, testSellerID)

	require.NoError(t, err)
	h.messageRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Cancel
// ---------------------------------------------------------------------------

func TestTradeCancel_Success(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID, withListingStatus("pending"))
	offer := testOffer(testOfferID, testBuyerID, &listing.ID, withOfferStatus("accepted"))

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		withTradeOffer(offer),
		withTradeListing(listing),
	)

	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)
	h.tradeRepo.On("Update", ctx, mock.AnythingOfType("*models.Trade")).Return(nil)
	h.offerRepo.On("Update", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)
	h.listingRepo.On("GetByID", ctx, testListingID).Return(listing, nil)
	h.listingRepo.On("Update", ctx, mock.AnythingOfType("*models.Listing")).Return(nil)
	h.notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	result, err := h.svc.Cancel(ctx, testTradeID, testSellerID, "")

	require.NoError(t, err)
	assert.Equal(t, "cancelled", result.Status)
	assert.NotNil(t, result.CancelledAt)
	assert.Equal(t, testSellerID, *result.CancelledBy)

	// Listing should be re-activated because its status was "pending" (not completed/cancelled)
	assert.Equal(t, "active", listing.Status)

	h.tradeRepo.AssertExpectations(t)
	h.offerRepo.AssertExpectations(t)
	h.listingRepo.AssertExpectations(t)
}

func TestTradeCancel_NotActive(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		withTradeStatus("completed"),
	)

	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)

	result, err := h.svc.Cancel(ctx, testTradeID, testSellerID, "changed my mind")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrInvalidState)
	h.tradeRepo.AssertExpectations(t)
}

func TestTradeCancel_NotParticipant(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)

	result, err := h.svc.Cancel(ctx, testTradeID, "stranger-999", "")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrForbidden)
	h.tradeRepo.AssertExpectations(t)
}

func TestTradeCancel_WithReason(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, &listing.ID, withOfferStatus("accepted"))

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		withTradeOffer(offer),
		withTradeListing(listing),
	)

	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)
	h.tradeRepo.On("Update", ctx, mock.AnythingOfType("*models.Trade")).Return(nil)
	h.offerRepo.On("Update", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)
	h.listingRepo.On("GetByID", ctx, testListingID).Return(listing, nil)
	h.listingRepo.On("Update", ctx, mock.AnythingOfType("*models.Listing")).Return(nil)
	h.notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	cancelReason := "Item was misrepresented"
	result, err := h.svc.Cancel(ctx, testTradeID, testBuyerID, cancelReason)

	require.NoError(t, err)
	assert.Equal(t, "cancelled", result.Status)
	assert.NotNil(t, result.CancelReason)
	assert.Equal(t, cancelReason, *result.CancelReason)
	assert.NotNil(t, result.CancelledBy)
	assert.Equal(t, testBuyerID, *result.CancelledBy)

	h.tradeRepo.AssertExpectations(t)
}

func TestTradeCancel_ListingAlreadyCompleted_NoReactivation(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID, withListingStatus("completed"))
	offer := testOffer(testOfferID, testBuyerID, &listing.ID, withOfferStatus("accepted"))

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		withTradeOffer(offer),
		withTradeListing(listing),
	)

	h.tradeRepo.On("GetByIDWithRelations", ctx, testTradeID).Return(trade, nil)
	h.tradeRepo.On("Update", ctx, mock.AnythingOfType("*models.Trade")).Return(nil)
	h.offerRepo.On("Update", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)
	h.listingRepo.On("GetByID", ctx, testListingID).Return(listing, nil)
	h.notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	result, err := h.svc.Cancel(ctx, testTradeID, testSellerID, "")

	require.NoError(t, err)
	assert.Equal(t, "cancelled", result.Status)

	// Listing should NOT be updated because it was already "completed"
	h.listingRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	h.tradeRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// generateItemImageURL
// ---------------------------------------------------------------------------

func TestGenerateItemImageURL_Rune(t *testing.T) {
	h := newTradeTestHarness()

	url := h.svc.generateItemImageURL("Ber", "rune")

	assert.Equal(t, "https://supabase.example.com/storage/v1/object/public/d2-items/runes/ber.png", url)
}

func TestGenerateItemImageURL_Unique(t *testing.T) {
	h := newTradeTestHarness()

	url := h.svc.generateItemImageURL("Harlequin Crest", "unique")

	assert.Equal(t, "https://supabase.example.com/storage/v1/object/public/d2-items/uniques/harlequin-crest.png", url)
}

func TestGenerateItemImageURL_DefaultType(t *testing.T) {
	h := newTradeTestHarness()

	url := h.svc.generateItemImageURL("Some Item", "unknown")

	assert.Equal(t, "https://supabase.example.com/storage/v1/object/public/d2-items/items/some-item.png", url)
}

func TestGenerateItemImageURL_NormalizesSpecialChars(t *testing.T) {
	h := newTradeTestHarness()

	url := h.svc.generateItemImageURL("Tal Rasha's Guardianship!", "set")

	// "Tal Rasha's Guardianship!" -> lowercase "tal rasha's guardianship!" -> spaces to hyphens
	// "tal-rasha's-guardianship!" -> remove special chars -> "tal-rashas-guardianship"
	assert.Equal(t, "https://supabase.example.com/storage/v1/object/public/d2-items/sets/tal-rashas-guardianship.png", url)
}

func TestGenerateItemImageURL_EmptySupabaseURL(t *testing.T) {
	svc := NewTradeServiceNew(
		nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, "",
	)

	url := svc.generateItemImageURL("Ber", "rune")

	assert.Equal(t, "", url)
}

// ---------------------------------------------------------------------------
// transformOfferedItems
// ---------------------------------------------------------------------------

func TestTransformOfferedItems_WithImageURL(t *testing.T) {
	h := newTradeTestHarness()

	items := []map[string]interface{}{
		{
			"id":       "item-1",
			"name":     "Ber",
			"type":     "rune",
			"imageUrl": "https://custom.example.com/ber.png",
			"quantity": 2,
		},
	}
	rawJSON, _ := json.Marshal(items)

	result := h.svc.transformOfferedItems(rawJSON)

	require.Len(t, result, 1)
	assert.Equal(t, "Ber", result[0].Name)
	assert.Equal(t, "rune", result[0].Type)
	assert.Equal(t, 2, result[0].Quantity)
	// Should use the provided imageUrl, not generate one
	assert.Equal(t, "https://custom.example.com/ber.png", result[0].ImageURL)
}

func TestTransformOfferedItems_GeneratesImageURL(t *testing.T) {
	h := newTradeTestHarness()

	items := []map[string]interface{}{
		{
			"id":       "item-1",
			"name":     "Jah",
			"type":     "rune",
			"quantity": 1,
		},
	}
	rawJSON, _ := json.Marshal(items)

	result := h.svc.transformOfferedItems(rawJSON)

	require.Len(t, result, 1)
	assert.Equal(t, "Jah", result[0].Name)
	assert.Equal(t, "rune", result[0].Type)
	assert.Equal(t, 1, result[0].Quantity)
	// Should generate the URL since imageUrl was not provided
	assert.Equal(t, "https://supabase.example.com/storage/v1/object/public/d2-items/runes/jah.png", result[0].ImageURL)
}

func TestTransformOfferedItems_Empty(t *testing.T) {
	h := newTradeTestHarness()

	// nil input
	result := h.svc.transformOfferedItems(nil)
	assert.Nil(t, result)

	// empty slice input
	result = h.svc.transformOfferedItems(json.RawMessage("[]"))
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// ToDetailResponse
// ---------------------------------------------------------------------------

func TestTradeToDetailResponse_ActiveTrade(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)

	detail := h.svc.ToDetailResponse(ctx, trade, testSellerID)

	assert.Equal(t, testTradeID, detail.ID)
	assert.Equal(t, "active", detail.Status)
	assert.True(t, detail.CanComplete, "seller should be able to complete an active trade")
	assert.True(t, detail.CanCancel, "should be able to cancel an active trade")
	assert.True(t, detail.CanMessage, "should be able to message in an active trade")
	assert.False(t, detail.CanRate, "cannot rate an active trade")
}

func TestTradeToDetailResponse_CompletedTrade(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		withTradeStatus("completed"),
	)

	existingTx := testTransaction(testTransactionID, testSellerID, testBuyerID)

	h.transactionRepo.On("GetByTradeID", ctx, testTradeID).Return(existingTx, nil)
	// User has not rated yet
	h.ratingRepo.On("Exists", ctx, testTransactionID, testSellerID).Return(false, nil)

	detail := h.svc.ToDetailResponse(ctx, trade, testSellerID)

	assert.Equal(t, testTradeID, detail.ID)
	assert.Equal(t, "completed", detail.Status)
	assert.False(t, detail.CanComplete, "cannot complete an already completed trade")
	assert.False(t, detail.CanCancel, "cannot cancel a completed trade")
	assert.False(t, detail.CanMessage, "cannot message in a completed trade")
	assert.True(t, detail.CanRate, "seller who hasn't rated should be able to rate")
	assert.NotNil(t, detail.TransactionID)
	assert.Equal(t, testTransactionID, *detail.TransactionID)

	h.transactionRepo.AssertExpectations(t)
	h.ratingRepo.AssertExpectations(t)
}

func TestTradeToDetailResponse_SellerAlreadyConfirmed(t *testing.T) {
	h := newTradeTestHarness()
	ctx := context.Background()

	sellerConfirmedAt := time.Now().Add(-5 * time.Minute)
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID,
		func(t *models.Trade) { t.SellerConfirmedAt = &sellerConfirmedAt },
	)

	detail := h.svc.ToDetailResponse(ctx, trade, testSellerID)

	assert.Equal(t, "active", detail.Status)
	assert.False(t, detail.CanComplete, "seller already confirmed, cannot complete again")
	assert.True(t, detail.CanCancel, "can still cancel an active trade")

	// But the buyer should still be able to complete
	detailBuyer := h.svc.ToDetailResponse(ctx, trade, testBuyerID)
	assert.True(t, detailBuyer.CanComplete, "buyer has not confirmed yet, should be able to complete")
}
