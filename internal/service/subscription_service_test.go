package service

import (
	"context"
	"testing"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---------------------------------------------------------------------------
// Helper: create a SubscriptionService with mock repos
// ---------------------------------------------------------------------------

func newTestSubscriptionService(
	profileRepo *mocks.MockProfileRepository,
	billingRepo *mocks.MockBillingEventRepository,
	txnRepo *mocks.MockTransactionRepository,
	wishlistRepo *mocks.MockWishlistRepository,
	listingRepo *mocks.MockListingRepository,
	config StripeConfig,
) *SubscriptionService {
	return NewSubscriptionService(
		profileRepo,
		billingRepo,
		txnRepo,
		wishlistRepo,
		listingRepo,
		newTestRedis(),
		config,
	)
}

func defaultStripeConfig() StripeConfig {
	return StripeConfig{
		SecretKey:     "",
		WebhookSecret: "",
		PriceID:       "price_default",
		SuccessURL:    "https://example.com/success",
		CancelURL:     "https://example.com/cancel",
	}
}

// ---------------------------------------------------------------------------
// isAllowedPriceID
// ---------------------------------------------------------------------------

func TestIsAllowedPriceID_DefaultOnly(t *testing.T) {
	config := defaultStripeConfig()
	// No AllowedPriceIDs configured — only the default PriceID should match
	svc := newTestSubscriptionService(
		new(mocks.MockProfileRepository),
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		config,
	)

	assert.True(t, svc.isAllowedPriceID("price_default"))
	assert.False(t, svc.isAllowedPriceID("price_other"))
	assert.False(t, svc.isAllowedPriceID(""))
}

func TestIsAllowedPriceID_WithAllowedList(t *testing.T) {
	config := defaultStripeConfig()
	config.AllowedPriceIDs = []string{"price_a", "price_b", "price_c"}
	svc := newTestSubscriptionService(
		new(mocks.MockProfileRepository),
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		config,
	)

	assert.True(t, svc.isAllowedPriceID("price_a"))
	assert.True(t, svc.isAllowedPriceID("price_b"))
	assert.True(t, svc.isAllowedPriceID("price_c"))
	// The default PriceID is NOT in the allowed list, so it should be rejected
	assert.False(t, svc.isAllowedPriceID("price_default"))
}

func TestIsAllowedPriceID_NotInList(t *testing.T) {
	config := defaultStripeConfig()
	config.AllowedPriceIDs = []string{"price_a", "price_b"}
	svc := newTestSubscriptionService(
		new(mocks.MockProfileRepository),
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		config,
	)

	assert.False(t, svc.isAllowedPriceID("price_unknown"))
	assert.False(t, svc.isAllowedPriceID(""))
	assert.False(t, svc.isAllowedPriceID("price_default"))
}

// ---------------------------------------------------------------------------
// UpdateFlair
// ---------------------------------------------------------------------------

func TestUpdateFlair_PremiumUser_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := newTestSubscriptionService(
		profileRepo,
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	profile := testProfile(testUserID, withPremium)

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)
	profileRepo.On("Update", ctx, profile).Return(nil)

	err := svc.UpdateFlair(ctx, testUserID, "gold")
	assert.NoError(t, err)
	assert.NotNil(t, profile.ProfileFlair)
	assert.Equal(t, "gold", *profile.ProfileFlair)

	profileRepo.AssertExpectations(t)
}

func TestUpdateFlair_FreeUser(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := newTestSubscriptionService(
		profileRepo,
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	profile := testProfile(testUserID) // not premium

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	err := svc.UpdateFlair(ctx, testUserID, "gold")
	assert.ErrorIs(t, err, ErrForbidden)

	// Update should NOT have been called
	profileRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUpdateFlair_None_ClearsFlair(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := newTestSubscriptionService(
		profileRepo,
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	existingFlair := "gold"
	profile := testProfile(testUserID, withPremium)
	profile.ProfileFlair = &existingFlair

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)
	profileRepo.On("Update", ctx, profile).Return(nil)

	err := svc.UpdateFlair(ctx, testUserID, "none")
	assert.NoError(t, err)
	assert.Nil(t, profile.ProfileFlair)

	profileRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// UpdateUsernameColor
// ---------------------------------------------------------------------------

func TestUpdateUsernameColor_PremiumUser_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := newTestSubscriptionService(
		profileRepo,
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	profile := testProfile(testUserID, withPremium)

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)
	profileRepo.On("Update", ctx, profile).Return(nil)

	err := svc.UpdateUsernameColor(ctx, testUserID, "#FFD700")
	assert.NoError(t, err)
	assert.NotNil(t, profile.UsernameColor)
	assert.Equal(t, "#FFD700", *profile.UsernameColor)

	profileRepo.AssertExpectations(t)
}

func TestUpdateUsernameColor_FreeUser(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := newTestSubscriptionService(
		profileRepo,
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	profile := testProfile(testUserID) // not premium

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	err := svc.UpdateUsernameColor(ctx, testUserID, "#FFD700")
	assert.ErrorIs(t, err, ErrForbidden)

	profileRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUpdateUsernameColor_None_ClearsColor(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := newTestSubscriptionService(
		profileRepo,
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	existingColor := "#FFD700"
	profile := testProfile(testUserID, withPremium)
	profile.UsernameColor = &existingColor

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)
	profileRepo.On("Update", ctx, profile).Return(nil)

	err := svc.UpdateUsernameColor(ctx, testUserID, "none")
	assert.NoError(t, err)
	assert.Nil(t, profile.UsernameColor)

	profileRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetSubscriptionInfo
// ---------------------------------------------------------------------------

func TestGetSubscriptionInfo_Premium(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := newTestSubscriptionService(
		profileRepo,
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	periodEnd := time.Now().Add(30 * 24 * time.Hour)
	flair := "flame"
	color := "#FF4500"
	profile := testProfile(testUserID, withPremium)
	profile.SubscriptionStatus = "active"
	profile.SubscriptionCurrentPeriodEnd = &periodEnd
	profile.CancelAtPeriodEnd = false
	profile.ProfileFlair = &flair
	profile.UsernameColor = &color

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	result, err := svc.GetSubscriptionInfo(ctx, testUserID)
	assert.NoError(t, err)
	assert.True(t, result.IsPremium)
	assert.Equal(t, "active", result.SubscriptionStatus)
	assert.NotNil(t, result.CurrentPeriodEnd)
	assert.Equal(t, periodEnd, *result.CurrentPeriodEnd)
	assert.False(t, result.CancelAtPeriodEnd)
	assert.Equal(t, "flame", result.ProfileFlair)
	assert.Equal(t, "#FF4500", result.UsernameColor)

	profileRepo.AssertExpectations(t)
}

func TestGetSubscriptionInfo_Free(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := newTestSubscriptionService(
		profileRepo,
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	profile := testProfile(testUserID) // not premium

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	result, err := svc.GetSubscriptionInfo(ctx, testUserID)
	assert.NoError(t, err)
	assert.False(t, result.IsPremium)
	assert.Empty(t, result.ProfileFlair)
	assert.Empty(t, result.UsernameColor)
	assert.Nil(t, result.CurrentPeriodEnd)
	assert.False(t, result.CancelAtPeriodEnd)

	profileRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetPriceHistory
// ---------------------------------------------------------------------------

func TestGetPriceHistory_PremiumUser_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	txnRepo := new(mocks.MockTransactionRepository)
	svc := newTestSubscriptionService(
		profileRepo,
		new(mocks.MockBillingEventRepository),
		txnRepo,
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	profile := testProfile(testUserID, withPremium)

	records := []repository.PriceHistoryRecord{
		{
			Date:         "2024-01-15",
			OfferedItems: []byte(`[{"name":"Sol","type":"rune","quantity":1}]`),
		},
		{
			Date:         "2024-01-15",
			OfferedItems: []byte(`[{"name":"Ist","type":"rune","quantity":2}]`),
		},
		{
			Date:         "2024-01-16",
			OfferedItems: []byte(`[{"name":"Ber","type":"rune","quantity":1}]`),
		},
	}

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)
	txnRepo.On("GetPriceHistory", ctx, "Shako", 30).Return(records, nil)

	result, err := svc.GetPriceHistory(ctx, testUserID, "Shako", 30)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should be grouped into 2 days
	assert.Len(t, result.Data, 2)

	// First day: 2024-01-15 with 2 trades
	assert.Equal(t, "2024-01-15", result.Data[0].Date)
	assert.Len(t, result.Data[0].Trades, 2)
	assert.Equal(t, "Sol", result.Data[0].Trades[0].OfferedItems[0].Name)
	assert.Equal(t, 1, result.Data[0].Trades[0].OfferedItems[0].Quantity)
	assert.Equal(t, "Ist", result.Data[0].Trades[1].OfferedItems[0].Name)
	assert.Equal(t, 2, result.Data[0].Trades[1].OfferedItems[0].Quantity)

	// Second day: 2024-01-16 with 1 trade
	assert.Equal(t, "2024-01-16", result.Data[1].Date)
	assert.Len(t, result.Data[1].Trades, 1)
	assert.Equal(t, "Ber", result.Data[1].Trades[0].OfferedItems[0].Name)

	profileRepo.AssertExpectations(t)
	txnRepo.AssertExpectations(t)
}

func TestGetPriceHistory_FreeUser(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := newTestSubscriptionService(
		profileRepo,
		new(mocks.MockBillingEventRepository),
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	profile := testProfile(testUserID) // not premium

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	result, err := svc.GetPriceHistory(ctx, testUserID, "Shako", 30)
	assert.ErrorIs(t, err, ErrForbidden)
	assert.Nil(t, result)

	profileRepo.AssertExpectations(t)
}

func TestGetPriceHistory_DefaultsDays(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	txnRepo := new(mocks.MockTransactionRepository)
	svc := newTestSubscriptionService(
		profileRepo,
		new(mocks.MockBillingEventRepository),
		txnRepo,
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	profile := testProfile(testUserID, withPremium)

	// When days=0, the service should default to 30
	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil).Once()
	txnRepo.On("GetPriceHistory", ctx, "Shako", 30).Return([]repository.PriceHistoryRecord{}, nil).Once()

	result, err := svc.GetPriceHistory(ctx, testUserID, "Shako", 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Data)

	// When days>90, the service should also default to 30
	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil).Once()
	txnRepo.On("GetPriceHistory", ctx, "Shako", 30).Return([]repository.PriceHistoryRecord{}, nil).Once()

	result, err = svc.GetPriceHistory(ctx, testUserID, "Shako", 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// When days is negative, should also default to 30
	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil).Once()
	txnRepo.On("GetPriceHistory", ctx, "Shako", 30).Return([]repository.PriceHistoryRecord{}, nil).Once()

	result, err = svc.GetPriceHistory(ctx, testUserID, "Shako", -5)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	profileRepo.AssertExpectations(t)
	txnRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetBillingHistory
// ---------------------------------------------------------------------------

func TestGetBillingHistory_Success(t *testing.T) {
	billingRepo := new(mocks.MockBillingEventRepository)
	svc := newTestSubscriptionService(
		new(mocks.MockProfileRepository),
		billingRepo,
		new(mocks.MockTransactionRepository),
		new(mocks.MockWishlistRepository),
		new(mocks.MockListingRepository),
		defaultStripeConfig(),
	)

	ctx := context.Background()
	now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)

	amountCents := 999
	currency := "usd"
	invoiceURL := "https://stripe.com/invoice/123"

	events := []*models.BillingEvent{
		{
			ID:            "be-001",
			UserID:        testUserID,
			StripeEventID: "evt_001",
			EventType:     "invoice.payment_succeeded",
			AmountCents:   &amountCents,
			Currency:      &currency,
			InvoiceURL:    &invoiceURL,
			CreatedAt:     now,
		},
		{
			ID:            "be-002",
			UserID:        testUserID,
			StripeEventID: "evt_002",
			EventType:     "customer.subscription.deleted",
			AmountCents:   nil,
			Currency:      nil,
			InvoiceURL:    nil,
			CreatedAt:     now.Add(24 * time.Hour),
		},
	}

	billingRepo.On("GetByUserID", ctx, testUserID).Return(events, nil)

	result, err := svc.GetBillingHistory(ctx, testUserID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Data, 2)

	// First entry: payment succeeded with amount
	entry0 := result.Data[0]
	assert.Equal(t, "be-001", entry0.ID)
	assert.Equal(t, "2024-06-15", entry0.Date)
	assert.Equal(t, "Payment Succeeded", entry0.Description)
	assert.Equal(t, "9.99 usd", entry0.Amount)
	assert.Equal(t, "succeeded", entry0.Status)
	assert.Equal(t, "https://stripe.com/invoice/123", entry0.InvoiceURL)

	// Second entry: subscription deleted with no amount
	entry1 := result.Data[1]
	assert.Equal(t, "be-002", entry1.ID)
	assert.Equal(t, "2024-06-16", entry1.Date)
	assert.Equal(t, "Subscription Cancelled", entry1.Description)
	assert.Equal(t, "", entry1.Amount)
	assert.Equal(t, "cancelled", entry1.Status)
	assert.Equal(t, "", entry1.InvoiceURL)

	billingRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// billingDisplayName
// ---------------------------------------------------------------------------

func TestBillingDisplayName(t *testing.T) {
	tests := []struct {
		eventType string
		expected  string
	}{
		{"checkout.session.completed", "Subscription Started"},
		{"customer.subscription.updated", "Subscription Updated"},
		{"customer.subscription.deleted", "Subscription Cancelled"},
		{"invoice.payment_succeeded", "Payment Succeeded"},
		{"invoice.payment_failed", "Payment Failed"},
		// Unknown event type returns the raw type
		{"some.unknown.event", "some.unknown.event"},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.eventType, func(t *testing.T) {
			assert.Equal(t, tc.expected, billingDisplayName(tc.eventType))
		})
	}
}

// ---------------------------------------------------------------------------
// billingStatus
// ---------------------------------------------------------------------------

func TestBillingStatus(t *testing.T) {
	tests := []struct {
		eventType string
		expected  string
	}{
		{"checkout.session.completed", "completed"},
		{"customer.subscription.updated", "updated"},
		{"customer.subscription.deleted", "cancelled"},
		{"invoice.payment_succeeded", "succeeded"},
		{"invoice.payment_failed", "failed"},
		// Unknown event type returns the raw type
		{"some.unknown.event", "some.unknown.event"},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.eventType, func(t *testing.T) {
			assert.Equal(t, tc.expected, billingStatus(tc.eventType))
		})
	}
}
