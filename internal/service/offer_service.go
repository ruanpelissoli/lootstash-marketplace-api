package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

// OfferService handles offer business logic
type OfferService struct {
	db                  *database.BunDB
	repo                repository.OfferRepository
	listingRepo         repository.ListingRepository
	tradeRepo           repository.TradeRepository
	chatRepo            repository.ChatRepository
	notificationService *NotificationService
	profileService      *ProfileService
	listingService      *ListingService
	redis               *cache.RedisClient
	invalidator         *cache.Invalidator
}

// NewOfferService creates a new offer service
func NewOfferService(
	db *database.BunDB,
	repo repository.OfferRepository,
	listingRepo repository.ListingRepository,
	tradeRepo repository.TradeRepository,
	chatRepo repository.ChatRepository,
	notificationService *NotificationService,
	profileService *ProfileService,
	listingService *ListingService,
	redis *cache.RedisClient,
) *OfferService {
	return &OfferService{
		db:                  db,
		repo:                repo,
		listingRepo:         listingRepo,
		tradeRepo:           tradeRepo,
		chatRepo:            chatRepo,
		notificationService: notificationService,
		profileService:      profileService,
		listingService:      listingService,
		redis:               redis,
		invalidator:         cache.NewInvalidator(redis),
	}
}

// Create creates a new offer
func (s *OfferService) Create(ctx context.Context, requesterID string, req *dto.CreateOfferRequest) (*models.Offer, error) {
	// Get the listing
	listing, err := s.listingRepo.GetByID(ctx, req.ListingID)
	if err != nil {
		return nil, err
	}

	// Can't request own listing
	if listing.SellerID == requesterID {
		return nil, ErrSelfAction
	}

	// Listing must be active
	if !listing.IsActive() {
		return nil, ErrInvalidState
	}

	// Check if listing has an active trade
	hasActive, err := s.tradeRepo.HasActiveTradeForListing(ctx, req.ListingID)
	if err != nil {
		return nil, err
	}
	if hasActive {
		return nil, ErrInvalidState
	}

	offer := &models.Offer{
		ID:           uuid.New().String(),
		ListingID:    req.ListingID,
		RequesterID:  requesterID,
		OfferedItems: req.OfferedItems,
		Status:       "pending",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if req.Message != "" {
		offer.Message = &req.Message
	}

	if err := s.repo.Create(ctx, offer); err != nil {
		return nil, err
	}

	// Notify the seller
	_ = s.notificationService.NotifyOfferReceived(ctx, listing.SellerID, offer.ID, listing.Name)

	return offer, nil
}

// GetByID retrieves an offer by ID
func (s *OfferService) GetByID(ctx context.Context, id string, userID string) (*models.Offer, error) {
	offer, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check user is a participant
	if offer.RequesterID != userID && offer.Listing.SellerID != userID {
		return nil, ErrForbidden
	}

	return offer, nil
}

// Accept accepts an offer and creates a Trade + Chat
func (s *OfferService) Accept(ctx context.Context, id string, userID string) (*models.Offer, *models.Trade, *models.Chat, error) {
	offer, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}

	// Only seller can accept
	if offer.Listing.SellerID != userID {
		return nil, nil, nil, ErrForbidden
	}

	// Must be pending
	if !offer.IsPending() {
		return nil, nil, nil, ErrInvalidState
	}

	// Check listing doesn't already have an active trade
	hasActive, err := s.tradeRepo.HasActiveTradeForListing(ctx, offer.ListingID)
	if err != nil {
		return nil, nil, nil, err
	}
	if hasActive {
		return nil, nil, nil, ErrInvalidState
	}

	now := time.Now()

	// Update offer
	offer.Status = "accepted"
	offer.AcceptedAt = &now
	offer.UpdatedAt = now

	if err := s.repo.Update(ctx, offer); err != nil {
		return nil, nil, nil, err
	}

	// Create trade
	trade := &models.Trade{
		ID:        uuid.New().String(),
		OfferID:   offer.ID,
		ListingID: offer.ListingID,
		SellerID:  offer.Listing.SellerID,
		BuyerID:   offer.RequesterID,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.tradeRepo.Create(ctx, trade); err != nil {
		return nil, nil, nil, err
	}

	// Create chat
	chat := &models.Chat{
		ID:        uuid.New().String(),
		TradeID:   trade.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.chatRepo.Create(ctx, chat); err != nil {
		return nil, nil, nil, err
	}

	// Notify the requester
	_ = s.notificationService.NotifyOfferAccepted(ctx, offer.RequesterID, offer.ID, offer.Listing.Name)

	return offer, trade, chat, nil
}

// Reject rejects an offer
func (s *OfferService) Reject(ctx context.Context, id string, userID string, req *dto.RejectOfferRequest) (*models.Offer, error) {
	offer, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	// Only seller can reject
	if offer.Listing.SellerID != userID {
		return nil, ErrForbidden
	}

	// Must be pending
	if !offer.IsPending() {
		return nil, ErrInvalidState
	}

	// Validate decline reason
	_, err = s.repo.GetDeclineReasonByID(ctx, req.DeclineReasonID)
	if err != nil {
		return nil, err
	}

	offer.Status = "rejected"
	offer.DeclineReasonID = &req.DeclineReasonID
	if req.DeclineNote != "" {
		offer.DeclineNote = &req.DeclineNote
	}
	offer.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, offer); err != nil {
		return nil, err
	}

	// Notify the requester
	_ = s.notificationService.NotifyOfferRejected(ctx, offer.RequesterID, offer.ID, offer.Listing.Name)

	return offer, nil
}

// Cancel cancels a pending offer (buyer only)
func (s *OfferService) Cancel(ctx context.Context, id string, userID string) (*models.Offer, error) {
	offer, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	// Only the requester (buyer) can cancel
	if offer.RequesterID != userID {
		return nil, ErrForbidden
	}

	// Can only cancel pending offers
	if !offer.IsPending() {
		return nil, ErrInvalidState
	}

	offer.Status = "cancelled"
	offer.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, offer); err != nil {
		return nil, err
	}

	return offer, nil
}

// List retrieves offers for a user
func (s *OfferService) List(ctx context.Context, userID string, role string, status string, listingID string, offset, limit int) ([]*models.Offer, int, error) {
	// For sellers, default to pending offers only (actionable items)
	if role == "seller" && status == "" {
		status = "pending"
	}

	filter := repository.OfferFilter{
		UserID:    userID,
		Role:      role,
		Status:    status,
		ListingID: listingID,
		Offset:    offset,
		Limit:     limit,
	}
	return s.repo.List(ctx, filter)
}

// GetDeclineReasons retrieves all active decline reasons
func (s *OfferService) GetDeclineReasons(ctx context.Context) ([]*models.DeclineReason, error) {
	return s.repo.GetDeclineReasons(ctx)
}

// ToResponse converts an offer model to a DTO response
func (s *OfferService) ToResponse(offer *models.Offer) *dto.OfferResponse {
	resp := &dto.OfferResponse{
		ID:           offer.ID,
		ListingID:    offer.ListingID,
		RequesterID:  offer.RequesterID,
		OfferedItems: offer.OfferedItems,
		Message:      offer.GetMessage(),
		Status:       offer.Status,
		DeclineNote:  offer.GetDeclineNote(),
		CreatedAt:    offer.CreatedAt,
		UpdatedAt:    offer.UpdatedAt,
		AcceptedAt:   offer.AcceptedAt,
	}

	if offer.Listing != nil {
		resp.Listing = s.listingService.ToResponse(offer.Listing)
	}

	if offer.Requester != nil {
		resp.Requester = s.profileService.ToResponse(offer.Requester)
	}

	if offer.DeclineReason != nil {
		resp.DeclineReason = &dto.DeclineReasonResponse{
			ID:      offer.DeclineReason.ID,
			Code:    offer.DeclineReason.Code,
			Message: offer.DeclineReason.Message,
		}
	}

	if offer.Trade != nil {
		resp.TradeID = &offer.Trade.ID
	}

	return resp
}

// ToDetailResponse converts an offer model to a detailed DTO response
func (s *OfferService) ToDetailResponse(offer *models.Offer, userID string) *dto.OfferDetailResponse {
	return &dto.OfferDetailResponse{
		OfferResponse: *s.ToResponse(offer),
	}
}
