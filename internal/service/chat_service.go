package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

// ChatService handles chat business logic
type ChatService struct {
	chatRepo            repository.ChatRepository
	messageRepo         repository.MessageRepository
	tradeRepo           repository.TradeRepository
	profileService      *ProfileService
	notificationService *NotificationService
}

// NewChatService creates a new chat service
func NewChatService(
	chatRepo repository.ChatRepository,
	messageRepo repository.MessageRepository,
	tradeRepo repository.TradeRepository,
	profileService *ProfileService,
	notificationService *NotificationService,
) *ChatService {
	return &ChatService{
		chatRepo:            chatRepo,
		messageRepo:         messageRepo,
		tradeRepo:           tradeRepo,
		profileService:      profileService,
		notificationService: notificationService,
	}
}

// getParticipants returns the two participant IDs for a chat
func (s *ChatService) getParticipants(chat *models.Chat) (string, string) {
	if chat.IsTradeChat() && chat.Trade != nil {
		return chat.Trade.SellerID, chat.Trade.BuyerID
	}
	if chat.IsServiceRunChat() && chat.ServiceRun != nil {
		return chat.ServiceRun.ProviderID, chat.ServiceRun.ClientID
	}
	return "", ""
}

// isParticipant checks if a user is a participant of the chat
func (s *ChatService) isParticipant(chat *models.Chat, userID string) bool {
	a, b := s.getParticipants(chat)
	return a == userID || b == userID
}

// isChatActive checks if the chat's parent entity is still active
func (s *ChatService) isChatActive(chat *models.Chat) bool {
	if chat.IsTradeChat() && chat.Trade != nil {
		return chat.Trade.IsActive()
	}
	if chat.IsServiceRunChat() && chat.ServiceRun != nil {
		return chat.ServiceRun.IsActive()
	}
	return false
}

// GetByID retrieves a chat by ID
func (s *ChatService) GetByID(ctx context.Context, id string, userID string) (*models.Chat, error) {
	chat, err := s.chatRepo.GetByIDWithContext(ctx, id)
	if err != nil {
		return nil, err
	}

	if !s.isParticipant(chat, userID) {
		return nil, ErrForbidden
	}

	return chat, nil
}

// SendMessage sends a message in a chat
func (s *ChatService) SendMessage(ctx context.Context, chatID string, senderID string, content string) (*models.Message, error) {
	chat, err := s.chatRepo.GetByIDWithContext(ctx, chatID)
	if err != nil {
		return nil, err
	}

	if !s.isParticipant(chat, senderID) {
		return nil, ErrForbidden
	}

	if !s.isChatActive(chat) {
		return nil, ErrInvalidState
	}

	participantA, participantB := s.getParticipants(chat)

	message := &models.Message{
		ID:          uuid.New().String(),
		ChatID:      chatID,
		SenderID:    senderID,
		Content:     content,
		MessageType: "text",
		CreatedAt:   time.Now(),
		// Denormalized for Realtime RLS
		SellerID: &participantA,
		BuyerID:  &participantB,
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, err
	}

	// Notify the other party
	var recipientID string
	if participantA == senderID {
		recipientID = participantB
	} else {
		recipientID = participantA
	}

	// Get sender profile for notification
	sender, _ := s.profileService.GetByID(ctx, senderID)
	senderName := "User"
	if sender != nil {
		senderName = sender.GetDisplayName()
	}

	_ = s.notificationService.NotifyNewMessage(ctx, recipientID, chatID, senderName)

	return message, nil
}

// GetMessages retrieves messages for a chat
func (s *ChatService) GetMessages(ctx context.Context, chatID string, userID string, offset, limit int) ([]*models.Message, int, error) {
	chat, err := s.chatRepo.GetByIDWithContext(ctx, chatID)
	if err != nil {
		return nil, 0, err
	}

	if !s.isParticipant(chat, userID) {
		return nil, 0, ErrForbidden
	}

	return s.messageRepo.GetByChatID(ctx, chatID, offset, limit)
}

// MarkMessagesAsRead marks messages as read
// If messageIDs is empty, marks all unread messages in the chat as read
func (s *ChatService) MarkMessagesAsRead(ctx context.Context, chatID string, userID string, messageIDs []string) error {
	chat, err := s.chatRepo.GetByIDWithContext(ctx, chatID)
	if err != nil {
		return err
	}

	if !s.isParticipant(chat, userID) {
		return ErrForbidden
	}

	// If no specific messageIDs provided, mark all unread messages in chat
	if len(messageIDs) == 0 {
		return s.messageRepo.MarkAllAsReadInChat(ctx, chatID, userID)
	}

	return s.messageRepo.MarkAsRead(ctx, messageIDs, userID)
}

// ToChatResponse converts a chat model to a DTO response
func (s *ChatService) ToChatResponse(chat *models.Chat) *dto.ChatResponse {
	resp := &dto.ChatResponse{
		ID:           chat.ID,
		TradeID:      chat.GetTradeID(),
		ServiceRunID: chat.GetServiceRunID(),
		CreatedAt:    chat.CreatedAt,
		UpdatedAt:    chat.UpdatedAt,
	}

	return resp
}

// ToMessageResponse converts a message model to a DTO response
func (s *ChatService) ToMessageResponse(message *models.Message) *dto.MessageResponse {
	resp := &dto.MessageResponse{
		ID:          message.ID,
		ChatID:      message.ChatID,
		SenderID:    message.SenderID,
		Content:     message.Content,
		MessageType: message.MessageType,
		ReadAt:      message.ReadAt,
		CreatedAt:   message.CreatedAt,
	}

	if message.Sender != nil {
		resp.Sender = s.profileService.ToResponse(message.Sender)
	}

	return resp
}
