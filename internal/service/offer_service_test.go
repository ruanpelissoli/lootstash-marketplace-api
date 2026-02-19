package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

func newOfferTestService() (
	*OfferService,
	*mocks.MockOfferRepository,
	*mocks.MockListingRepository,
	*mocks.MockServiceRepository,
	*mocks.MockTradeRepository,
	*mocks.MockChatRepository,
	*mocks.MockServiceRunRepository,
	*mocks.MockNotificationRepository,
) {
	offerRepo := new(mocks.MockOfferRepository)
	listingRepo := new(mocks.MockListingRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	tradeRepo := new(mocks.MockTradeRepository)
	chatRepo := new(mocks.MockChatRepository)
	serviceRunRepo := new(mocks.MockServiceRunRepository)
	notifRepo := new(mocks.MockNotificationRepository)

	notifService := NewNotificationService(notifRepo, nil)
	profileService := NewProfileService(nil, nil, nil)
	listingService := NewListingService(listingRepo, profileService, nil)
	serviceService := NewServiceService(serviceRepo, profileService, nil)

	svc := NewOfferService(
		nil, // db
		offerRepo,
		listingRepo,
		serviceRepo,
		tradeRepo,
		chatRepo,
		serviceRunRepo,
		notifService,
		profileService,
		listingService,
		serviceService,
		nil, // redis
	)

	return svc, offerRepo, listingRepo, serviceRepo, tradeRepo, chatRepo, serviceRunRepo, notifRepo
}

// ---------- Create Item Offer ----------

func TestCreateItemOffer_Success(t *testing.T) {
	svc, offerRepo, listingRepo, _, tradeRepo, _, _, notifRepo := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	listingRepo.On("GetByID", ctx, testListingID).Return(listing, nil)
	tradeRepo.On("HasActiveTradeForListing", ctx, testListingID).Return(false, nil)
	offerRepo.On("Create", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)
	// Notification: the service re-fetches listing for the name
	listingRepo.On("GetByID", ctx, testListingID).Return(listing, nil)
	notifRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	req := &dto.CreateOfferRequest{
		Type:         "item",
		ListingID:    strPtr(testListingID),
		OfferedItems: json.RawMessage(`[{"name":"Ber","quantity":1}]`),
	}

	offer, err := svc.Create(ctx, testBuyerID, req)

	require.NoError(t, err)
	assert.Equal(t, "pending", offer.Status)
	assert.Equal(t, "item", offer.Type)
	assert.Equal(t, testBuyerID, offer.RequesterID)
	assert.Equal(t, testListingID, *offer.ListingID)
	offerRepo.AssertCalled(t, "Create", ctx, mock.AnythingOfType("*models.Offer"))
}

func TestCreateItemOffer_MissingListingID(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	req := &dto.CreateOfferRequest{
		Type:         "item",
		ListingID:    nil,
		OfferedItems: json.RawMessage(`[]`),
	}

	_, err := svc.Create(ctx, testBuyerID, req)

	assert.ErrorIs(t, err, ErrInvalidState)
}

func TestCreateItemOffer_EmptyListingID(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	req := &dto.CreateOfferRequest{
		Type:         "item",
		ListingID:    strPtr(""),
		OfferedItems: json.RawMessage(`[]`),
	}

	_, err := svc.Create(ctx, testBuyerID, req)

	assert.ErrorIs(t, err, ErrInvalidState)
}

func TestCreateItemOffer_SelfAction(t *testing.T) {
	svc, _, listingRepo, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	listingRepo.On("GetByID", ctx, testListingID).Return(listing, nil)

	req := &dto.CreateOfferRequest{
		Type:         "item",
		ListingID:    strPtr(testListingID),
		OfferedItems: json.RawMessage(`[]`),
	}

	// Requester == seller
	_, err := svc.Create(ctx, testSellerID, req)

	assert.ErrorIs(t, err, ErrSelfAction)
}

func TestCreateItemOffer_InactiveListing(t *testing.T) {
	svc, _, listingRepo, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID, withListingStatus("completed"))
	listingRepo.On("GetByID", ctx, testListingID).Return(listing, nil)

	req := &dto.CreateOfferRequest{
		Type:         "item",
		ListingID:    strPtr(testListingID),
		OfferedItems: json.RawMessage(`[]`),
	}

	_, err := svc.Create(ctx, testBuyerID, req)

	assert.ErrorIs(t, err, ErrInvalidState)
}

func TestCreateItemOffer_ActiveTradeExists(t *testing.T) {
	svc, _, listingRepo, _, tradeRepo, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	listingRepo.On("GetByID", ctx, testListingID).Return(listing, nil)
	tradeRepo.On("HasActiveTradeForListing", ctx, testListingID).Return(true, nil)

	req := &dto.CreateOfferRequest{
		Type:         "item",
		ListingID:    strPtr(testListingID),
		OfferedItems: json.RawMessage(`[]`),
	}

	_, err := svc.Create(ctx, testBuyerID, req)

	assert.ErrorIs(t, err, ErrInvalidState)
}

func TestCreateItemOffer_NotifiesSeller(t *testing.T) {
	svc, offerRepo, listingRepo, _, tradeRepo, _, _, notifRepo := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	listingRepo.On("GetByID", ctx, testListingID).Return(listing, nil)
	tradeRepo.On("HasActiveTradeForListing", ctx, testListingID).Return(false, nil)
	offerRepo.On("Create", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)
	notifRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	req := &dto.CreateOfferRequest{
		Type:         "item",
		ListingID:    strPtr(testListingID),
		OfferedItems: json.RawMessage(`[]`),
	}

	_, err := svc.Create(ctx, testBuyerID, req)

	require.NoError(t, err)
	notifRepo.AssertCalled(t, "Create", ctx, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == testSellerID &&
			n.Type == models.NotificationTypeTradeRequestReceived
	}))
}

// ---------- Create Service Offer ----------

func TestCreateServiceOffer_Success(t *testing.T) {
	svc, offerRepo, _, serviceRepo, _, _, _, notifRepo := newOfferTestService()
	ctx := context.Background()

	service := testServiceModel(testServiceID, testProviderID)
	serviceRepo.On("GetByID", ctx, testServiceID).Return(service, nil)
	offerRepo.On("Create", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)
	// Re-fetch for notification
	serviceRepo.On("GetByID", ctx, testServiceID).Return(service, nil)
	notifRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	req := &dto.CreateOfferRequest{
		Type:         "service",
		ServiceID:    strPtr(testServiceID),
		OfferedItems: json.RawMessage(`[]`),
	}

	offer, err := svc.Create(ctx, testClientID, req)

	require.NoError(t, err)
	assert.Equal(t, "pending", offer.Status)
	assert.Equal(t, "service", offer.Type)
	assert.Equal(t, testClientID, offer.RequesterID)
	assert.Equal(t, testServiceID, *offer.ServiceID)
}

func TestCreateServiceOffer_MissingServiceID(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	req := &dto.CreateOfferRequest{
		Type:         "service",
		ServiceID:    nil,
		OfferedItems: json.RawMessage(`[]`),
	}

	_, err := svc.Create(ctx, testClientID, req)

	assert.ErrorIs(t, err, ErrInvalidState)
}

func TestCreateServiceOffer_EmptyServiceID(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	req := &dto.CreateOfferRequest{
		Type:         "service",
		ServiceID:    strPtr(""),
		OfferedItems: json.RawMessage(`[]`),
	}

	_, err := svc.Create(ctx, testClientID, req)

	assert.ErrorIs(t, err, ErrInvalidState)
}

func TestCreateServiceOffer_SelfAction(t *testing.T) {
	svc, _, _, serviceRepo, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	service := testServiceModel(testServiceID, testProviderID)
	serviceRepo.On("GetByID", ctx, testServiceID).Return(service, nil)

	req := &dto.CreateOfferRequest{
		Type:         "service",
		ServiceID:    strPtr(testServiceID),
		OfferedItems: json.RawMessage(`[]`),
	}

	// Requester == provider
	_, err := svc.Create(ctx, testProviderID, req)

	assert.ErrorIs(t, err, ErrSelfAction)
}

func TestCreateServiceOffer_InactiveService(t *testing.T) {
	svc, _, _, serviceRepo, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	service := testServiceModel(testServiceID, testProviderID, withServiceStatus("paused"))
	serviceRepo.On("GetByID", ctx, testServiceID).Return(service, nil)

	req := &dto.CreateOfferRequest{
		Type:         "service",
		ServiceID:    strPtr(testServiceID),
		OfferedItems: json.RawMessage(`[]`),
	}

	_, err := svc.Create(ctx, testClientID, req)

	assert.ErrorIs(t, err, ErrInvalidState)
}

// ---------- Accept Item Offer ----------

func TestAcceptItemOffer_CreatesTradeAndChat(t *testing.T) {
	svc, offerRepo, _, _, tradeRepo, chatRepo, _, notifRepo := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing))

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)
	offerRepo.On("Update", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)
	tradeRepo.On("HasActiveTradeForListing", ctx, testListingID).Return(false, nil)
	tradeRepo.On("Create", ctx, mock.AnythingOfType("*models.Trade")).Return(nil)
	chatRepo.On("Create", ctx, mock.AnythingOfType("*models.Chat")).Return(nil)
	notifRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	returnedOffer, trade, serviceRun, chat, err := svc.Accept(ctx, testOfferID, testSellerID)

	require.NoError(t, err)
	assert.Equal(t, "accepted", returnedOffer.Status)
	assert.NotNil(t, returnedOffer.AcceptedAt)
	assert.NotNil(t, trade)
	assert.Equal(t, "active", trade.Status)
	assert.Equal(t, testSellerID, trade.SellerID)
	assert.Equal(t, testBuyerID, trade.BuyerID)
	assert.Nil(t, serviceRun)
	assert.NotNil(t, chat)
	tradeRepo.AssertCalled(t, "Create", ctx, mock.AnythingOfType("*models.Trade"))
	chatRepo.AssertCalled(t, "Create", ctx, mock.AnythingOfType("*models.Chat"))
}

func TestAcceptItemOffer_NotOwner(t *testing.T) {
	svc, offerRepo, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing))

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)

	// Someone other than the seller tries to accept
	_, _, _, _, err := svc.Accept(ctx, testOfferID, "stranger-999")

	assert.ErrorIs(t, err, ErrForbidden)
}

func TestAcceptItemOffer_NotPending(t *testing.T) {
	svc, offerRepo, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, strPtr(testListingID),
		withOfferListing(listing),
		withOfferStatus("rejected"),
	)

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)

	_, _, _, _, err := svc.Accept(ctx, testOfferID, testSellerID)

	assert.ErrorIs(t, err, ErrInvalidState)
}

func TestAcceptItemOffer_ActiveTradeBlocks(t *testing.T) {
	svc, offerRepo, _, _, tradeRepo, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing))

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)
	offerRepo.On("Update", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)
	tradeRepo.On("HasActiveTradeForListing", ctx, testListingID).Return(true, nil)

	_, _, _, _, err := svc.Accept(ctx, testOfferID, testSellerID)

	assert.ErrorIs(t, err, ErrInvalidState)
}

// ---------- Accept Service Offer ----------

func TestAcceptServiceOffer_CreatesServiceRunAndChat(t *testing.T) {
	svc, offerRepo, _, _, _, chatRepo, serviceRunRepo, notifRepo := newOfferTestService()
	ctx := context.Background()

	service := testServiceModel(testServiceID, testProviderID)
	offer := testOffer(testOfferID, testClientID, nil, withOfferService(service))

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)
	offerRepo.On("Update", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)
	serviceRunRepo.On("Create", ctx, mock.AnythingOfType("*models.ServiceRun")).Return(nil)
	chatRepo.On("Create", ctx, mock.AnythingOfType("*models.Chat")).Return(nil)
	notifRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	returnedOffer, trade, serviceRun, chat, err := svc.Accept(ctx, testOfferID, testProviderID)

	require.NoError(t, err)
	assert.Equal(t, "accepted", returnedOffer.Status)
	assert.NotNil(t, returnedOffer.AcceptedAt)
	assert.Nil(t, trade)
	assert.NotNil(t, serviceRun)
	assert.Equal(t, "active", serviceRun.Status)
	assert.Equal(t, testProviderID, serviceRun.ProviderID)
	assert.Equal(t, testClientID, serviceRun.ClientID)
	assert.NotNil(t, chat)
	serviceRunRepo.AssertCalled(t, "Create", ctx, mock.AnythingOfType("*models.ServiceRun"))
	chatRepo.AssertCalled(t, "Create", ctx, mock.AnythingOfType("*models.Chat"))
}

// ---------- Reject ----------

func TestRejectOffer_Success(t *testing.T) {
	svc, offerRepo, _, _, _, _, _, notifRepo := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing))

	declineReason := &models.DeclineReason{ID: 1, Code: "low_offer", Message: "Offer too low"}

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)
	offerRepo.On("GetDeclineReasonByID", ctx, 1).Return(declineReason, nil)
	offerRepo.On("Update", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)
	notifRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	req := &dto.RejectOfferRequest{
		DeclineReasonID: 1,
		DeclineNote:     "Need more runes",
	}

	result, err := svc.Reject(ctx, testOfferID, testSellerID, req)

	require.NoError(t, err)
	assert.Equal(t, "rejected", result.Status)
	assert.Equal(t, intPtr(1), result.DeclineReasonID)
	assert.Equal(t, "Need more runes", *result.DeclineNote)
}

func TestRejectOffer_NotOwner(t *testing.T) {
	svc, offerRepo, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing))

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)

	req := &dto.RejectOfferRequest{DeclineReasonID: 1}

	// Buyer (the requester) tries to reject -- not the seller
	_, err := svc.Reject(ctx, testOfferID, testBuyerID, req)

	assert.ErrorIs(t, err, ErrForbidden)
}

func TestRejectOffer_NotPending(t *testing.T) {
	svc, offerRepo, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, strPtr(testListingID),
		withOfferListing(listing),
		withOfferStatus("accepted"),
	)

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)

	req := &dto.RejectOfferRequest{DeclineReasonID: 1}

	_, err := svc.Reject(ctx, testOfferID, testSellerID, req)

	assert.ErrorIs(t, err, ErrInvalidState)
}

func TestRejectOffer_InvalidDeclineReason(t *testing.T) {
	svc, offerRepo, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing))

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)
	offerRepo.On("GetDeclineReasonByID", ctx, 999).Return(nil, errors.New("not found"))

	req := &dto.RejectOfferRequest{DeclineReasonID: 999}

	_, err := svc.Reject(ctx, testOfferID, testSellerID, req)

	assert.Error(t, err)
}

// ---------- Cancel ----------

func TestCancelOffer_Success(t *testing.T) {
	svc, offerRepo, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing))

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)
	offerRepo.On("Update", ctx, mock.AnythingOfType("*models.Offer")).Return(nil)

	result, err := svc.Cancel(ctx, testOfferID, testBuyerID)

	require.NoError(t, err)
	assert.Equal(t, "cancelled", result.Status)
}

func TestCancelOffer_NotRequester(t *testing.T) {
	svc, offerRepo, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing))

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)

	// Seller tries to cancel -- only requester can cancel
	_, err := svc.Cancel(ctx, testOfferID, testSellerID)

	assert.ErrorIs(t, err, ErrForbidden)
}

func TestCancelOffer_NotPending(t *testing.T) {
	svc, offerRepo, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID)
	offer := testOffer(testOfferID, testBuyerID, strPtr(testListingID),
		withOfferListing(listing),
		withOfferStatus("accepted"),
	)

	offerRepo.On("GetByIDWithRelations", ctx, testOfferID).Return(offer, nil)

	_, err := svc.Cancel(ctx, testOfferID, testBuyerID)

	assert.ErrorIs(t, err, ErrInvalidState)
}

// ---------- List ----------

func TestListOffers_SellerDefaultsPending(t *testing.T) {
	svc, offerRepo, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	expectedFilter := repository.OfferFilter{
		UserID: testSellerID,
		Role:   "seller",
		Status: "pending", // default applied
		Offset: 0,
		Limit:  20,
	}

	offerRepo.On("List", ctx, expectedFilter).Return([]*models.Offer{}, 0, nil)

	offers, count, err := svc.List(ctx, testSellerID, "seller", "", "", "", "", 0, 20)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, offers)
	offerRepo.AssertCalled(t, "List", ctx, expectedFilter)
}

func TestListOffers_BuyerNoDefault(t *testing.T) {
	svc, offerRepo, _, _, _, _, _, _ := newOfferTestService()
	ctx := context.Background()

	expectedFilter := repository.OfferFilter{
		UserID: testBuyerID,
		Role:   "buyer",
		Status: "", // stays empty
		Offset: 0,
		Limit:  20,
	}

	offerRepo.On("List", ctx, expectedFilter).Return([]*models.Offer{}, 0, nil)

	offers, count, err := svc.List(ctx, testBuyerID, "buyer", "", "", "", "", 0, 20)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, offers)
	offerRepo.AssertCalled(t, "List", ctx, expectedFilter)
}

// ---------- isOfferParticipant ----------

func TestIsOfferParticipant(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newOfferTestService()

	listing := testListing(testListingID, testSellerID)
	service := testServiceModel(testServiceID, testProviderID)

	tests := []struct {
		name     string
		offer    *models.Offer
		userID   string
		expected bool
	}{
		{
			name:     "requester is participant",
			offer:    testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing)),
			userID:   testBuyerID,
			expected: true,
		},
		{
			name:     "seller is participant (item offer)",
			offer:    testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing)),
			userID:   testSellerID,
			expected: true,
		},
		{
			name:     "provider is participant (service offer)",
			offer:    testOffer(testOfferID, testClientID, nil, withOfferService(service)),
			userID:   testProviderID,
			expected: true,
		},
		{
			name:     "stranger is not participant",
			offer:    testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing)),
			userID:   "stranger-999",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := svc.isOfferParticipant(tc.offer, tc.userID)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ---------- isOfferOwner ----------

func TestIsOfferOwner(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newOfferTestService()

	listing := testListing(testListingID, testSellerID)
	service := testServiceModel(testServiceID, testProviderID)

	tests := []struct {
		name     string
		offer    *models.Offer
		userID   string
		expected bool
	}{
		{
			name:     "seller is owner (item offer)",
			offer:    testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing)),
			userID:   testSellerID,
			expected: true,
		},
		{
			name:     "requester is not owner (item offer)",
			offer:    testOffer(testOfferID, testBuyerID, strPtr(testListingID), withOfferListing(listing)),
			userID:   testBuyerID,
			expected: false,
		},
		{
			name:     "provider is owner (service offer)",
			offer:    testOffer(testOfferID, testClientID, nil, withOfferService(service)),
			userID:   testProviderID,
			expected: true,
		},
		{
			name:     "client is not owner (service offer)",
			offer:    testOffer(testOfferID, testClientID, nil, withOfferService(service)),
			userID:   testClientID,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := svc.isOfferOwner(tc.offer, tc.userID)
			assert.Equal(t, tc.expected, result)
		})
	}
}
