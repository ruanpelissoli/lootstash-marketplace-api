package service

import (
	"context"
	"testing"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newChatTestService() (
	*ChatService,
	*mocks.MockChatRepository,
	*mocks.MockMessageRepository,
	*mocks.MockTradeRepository,
	*mocks.MockProfileRepository,
	*mocks.MockNotificationRepository,
) {
	chatRepo := new(mocks.MockChatRepository)
	messageRepo := new(mocks.MockMessageRepository)
	tradeRepo := new(mocks.MockTradeRepository)
	profileRepo := new(mocks.MockProfileRepository)
	notifRepo := new(mocks.MockNotificationRepository)

	profileService := NewProfileService(profileRepo, nil, nil)
	notifService := NewNotificationService(notifRepo, nil)

	svc := NewChatService(chatRepo, messageRepo, tradeRepo, profileService, notifService)

	return svc, chatRepo, messageRepo, tradeRepo, profileRepo, notifRepo
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestChatGetByID_Participant_Success(t *testing.T) {
	svc, chatRepo, _, _, _, _ := newChatTestService()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	chatRepo.On("GetByIDWithContext", ctx, testChatID).Return(chat, nil)

	result, err := svc.GetByID(ctx, testChatID, testSellerID)

	require.NoError(t, err)
	assert.Equal(t, testChatID, result.ID)
	chatRepo.AssertExpectations(t)
}

func TestChatGetByID_NonParticipant(t *testing.T) {
	svc, chatRepo, _, _, _, _ := newChatTestService()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	chatRepo.On("GetByIDWithContext", ctx, testChatID).Return(chat, nil)

	_, err := svc.GetByID(ctx, testChatID, "stranger-999")

	assert.ErrorIs(t, err, ErrForbidden)
	chatRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// SendMessage
// ---------------------------------------------------------------------------

func TestSendMessage_Success(t *testing.T) {
	svc, chatRepo, messageRepo, _, profileRepo, notifRepo := newChatTestService()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)
	senderProfile := testProfile(testSellerID)

	chatRepo.On("GetByIDWithContext", ctx, testChatID).Return(chat, nil)
	messageRepo.On("Create", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	profileRepo.On("GetByID", ctx, testSellerID).Return(senderProfile, nil)
	notifRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	msg, err := svc.SendMessage(ctx, testChatID, testSellerID, "Hello!")

	require.NoError(t, err)
	assert.Equal(t, testChatID, msg.ChatID)
	assert.Equal(t, testSellerID, msg.SenderID)
	assert.Equal(t, "Hello!", msg.Content)
	assert.Equal(t, "text", msg.MessageType)
	assert.NotEmpty(t, msg.ID)

	chatRepo.AssertExpectations(t)
	messageRepo.AssertExpectations(t)
	profileRepo.AssertExpectations(t)
	notifRepo.AssertExpectations(t)
}

func TestSendMessage_NonParticipant(t *testing.T) {
	svc, chatRepo, _, _, _, _ := newChatTestService()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	chatRepo.On("GetByIDWithContext", ctx, testChatID).Return(chat, nil)

	_, err := svc.SendMessage(ctx, testChatID, "stranger-999", "Hello!")

	assert.ErrorIs(t, err, ErrForbidden)
	chatRepo.AssertExpectations(t)
}

func TestSendMessage_InactiveChat(t *testing.T) {
	svc, chatRepo, _, _, _, _ := newChatTestService()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID, withTradeStatus("completed"))
	chat := testChatWithTrade(testChatID, trade)

	chatRepo.On("GetByIDWithContext", ctx, testChatID).Return(chat, nil)

	_, err := svc.SendMessage(ctx, testChatID, testSellerID, "Hello!")

	assert.ErrorIs(t, err, ErrInvalidState)
	chatRepo.AssertExpectations(t)
}

func TestSendMessage_DenormalizesParticipantIDs(t *testing.T) {
	svc, chatRepo, messageRepo, _, profileRepo, notifRepo := newChatTestService()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)
	senderProfile := testProfile(testBuyerID)

	chatRepo.On("GetByIDWithContext", ctx, testChatID).Return(chat, nil)
	messageRepo.On("Create", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	profileRepo.On("GetByID", ctx, testBuyerID).Return(senderProfile, nil)
	notifRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	msg, err := svc.SendMessage(ctx, testChatID, testBuyerID, "Trade me!")

	require.NoError(t, err)
	require.NotNil(t, msg.SellerID)
	require.NotNil(t, msg.BuyerID)
	assert.Equal(t, testSellerID, *msg.SellerID)
	assert.Equal(t, testBuyerID, *msg.BuyerID)

	messageRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetMessages
// ---------------------------------------------------------------------------

func TestGetMessages_Participant_Success(t *testing.T) {
	svc, chatRepo, messageRepo, _, _, _ := newChatTestService()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	expectedMessages := []*models.Message{
		{ID: "msg-1", ChatID: testChatID, SenderID: testSellerID, Content: "Hello"},
		{ID: "msg-2", ChatID: testChatID, SenderID: testBuyerID, Content: "Hi there"},
	}

	chatRepo.On("GetByIDWithContext", ctx, testChatID).Return(chat, nil)
	messageRepo.On("GetByChatID", ctx, testChatID, 0, 20).Return(expectedMessages, 2, nil)

	messages, total, err := svc.GetMessages(ctx, testChatID, testSellerID, 0, 20)

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, messages, 2)

	chatRepo.AssertExpectations(t)
	messageRepo.AssertExpectations(t)
}

func TestGetMessages_NonParticipant(t *testing.T) {
	svc, chatRepo, _, _, _, _ := newChatTestService()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	chatRepo.On("GetByIDWithContext", ctx, testChatID).Return(chat, nil)

	_, _, err := svc.GetMessages(ctx, testChatID, "stranger-999", 0, 20)

	assert.ErrorIs(t, err, ErrForbidden)
	chatRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// MarkMessagesAsRead
// ---------------------------------------------------------------------------

func TestMarkMessagesAsRead_SpecificIDs(t *testing.T) {
	svc, chatRepo, messageRepo, _, _, _ := newChatTestService()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	messageIDs := []string{"msg-1", "msg-2"}

	chatRepo.On("GetByIDWithContext", ctx, testChatID).Return(chat, nil)
	messageRepo.On("MarkAsRead", ctx, messageIDs, testBuyerID).Return(nil)

	err := svc.MarkMessagesAsRead(ctx, testChatID, testBuyerID, messageIDs)

	assert.NoError(t, err)
	messageRepo.AssertCalled(t, "MarkAsRead", ctx, messageIDs, testBuyerID)
	messageRepo.AssertNotCalled(t, "MarkAllAsReadInChat", mock.Anything, mock.Anything, mock.Anything)
	chatRepo.AssertExpectations(t)
	messageRepo.AssertExpectations(t)
}

func TestMarkMessagesAsRead_AllInChat(t *testing.T) {
	svc, chatRepo, messageRepo, _, _, _ := newChatTestService()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	chatRepo.On("GetByIDWithContext", ctx, testChatID).Return(chat, nil)
	messageRepo.On("MarkAllAsReadInChat", ctx, testChatID, testBuyerID).Return(nil)

	err := svc.MarkMessagesAsRead(ctx, testChatID, testBuyerID, []string{})

	assert.NoError(t, err)
	messageRepo.AssertCalled(t, "MarkAllAsReadInChat", ctx, testChatID, testBuyerID)
	messageRepo.AssertNotCalled(t, "MarkAsRead", mock.Anything, mock.Anything, mock.Anything)
	chatRepo.AssertExpectations(t)
	messageRepo.AssertExpectations(t)
}

func TestMarkMessagesAsRead_NonParticipant(t *testing.T) {
	svc, chatRepo, _, _, _, _ := newChatTestService()
	ctx := context.Background()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	chatRepo.On("GetByIDWithContext", ctx, testChatID).Return(chat, nil)

	err := svc.MarkMessagesAsRead(ctx, testChatID, "stranger-999", []string{"msg-1"})

	assert.ErrorIs(t, err, ErrForbidden)
	chatRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// isParticipant — Trade Chat
// ---------------------------------------------------------------------------

func TestIsParticipant_TradeChat_Seller(t *testing.T) {
	svc, _, _, _, _, _ := newChatTestService()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	assert.True(t, svc.isParticipant(chat, testSellerID))
}

func TestIsParticipant_TradeChat_Buyer(t *testing.T) {
	svc, _, _, _, _, _ := newChatTestService()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	assert.True(t, svc.isParticipant(chat, testBuyerID))
}

func TestIsParticipant_TradeChat_Stranger(t *testing.T) {
	svc, _, _, _, _, _ := newChatTestService()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	assert.False(t, svc.isParticipant(chat, "stranger-999"))
}

// ---------------------------------------------------------------------------
// isParticipant — ServiceRun Chat
// ---------------------------------------------------------------------------

func TestIsParticipant_ServiceRunChat_Provider(t *testing.T) {
	svc, _, _, _, _, _ := newChatTestService()

	sr := testServiceRun(testServiceRunID, testServiceID, testOfferID, testProviderID, testClientID)
	chat := testChatWithServiceRun(testChatID, sr)

	assert.True(t, svc.isParticipant(chat, testProviderID))
}

func TestIsParticipant_ServiceRunChat_Client(t *testing.T) {
	svc, _, _, _, _, _ := newChatTestService()

	sr := testServiceRun(testServiceRunID, testServiceID, testOfferID, testProviderID, testClientID)
	chat := testChatWithServiceRun(testChatID, sr)

	assert.True(t, svc.isParticipant(chat, testClientID))
}

// ---------------------------------------------------------------------------
// isChatActive
// ---------------------------------------------------------------------------

func TestIsChatActive_ActiveTrade(t *testing.T) {
	svc, _, _, _, _, _ := newChatTestService()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	chat := testChatWithTrade(testChatID, trade)

	assert.True(t, svc.isChatActive(chat))
}

func TestIsChatActive_CompletedTrade(t *testing.T) {
	svc, _, _, _, _, _ := newChatTestService()

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID, withTradeStatus("completed"))
	chat := testChatWithTrade(testChatID, trade)

	assert.False(t, svc.isChatActive(chat))
}

func TestIsChatActive_ActiveServiceRun(t *testing.T) {
	svc, _, _, _, _, _ := newChatTestService()

	sr := testServiceRun(testServiceRunID, testServiceID, testOfferID, testProviderID, testClientID)
	chat := testChatWithServiceRun(testChatID, sr)

	assert.True(t, svc.isChatActive(chat))
}

func TestIsChatActive_CompletedServiceRun(t *testing.T) {
	svc, _, _, _, _, _ := newChatTestService()

	sr := testServiceRun(testServiceRunID, testServiceID, testOfferID, testProviderID, testClientID, withServiceRunStatus("completed"))
	chat := testChatWithServiceRun(testChatID, sr)

	assert.False(t, svc.isChatActive(chat))
}
