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
	serviceRepo         repository.ServiceRepository
	tradeRepo           repository.TradeRepository
	chatRepo            repository.ChatRepository
	serviceRunRepo      repository.ServiceRunRepository
	notificationService *NotificationService
	profileService      *ProfileService
	listingService      *ListingService
	serviceService      *ServiceService
	statsService        *StatsService
	redis               *cache.RedisClient
	invalidator         *cache.Invalidator
}

// NewOfferService creates a new offer service
func NewOfferService(
	db *database.BunDB,
	repo repository.OfferRepository,
	listingRepo repository.ListingRepository,
	serviceRepo repository.ServiceRepository,
	tradeRepo repository.TradeRepository,
	chatRepo repository.ChatRepository,
	serviceRunRepo repository.ServiceRunRepository,
	notificationService *NotificationService,
	profileService *ProfileService,
	listingService *ListingService,
	serviceService *ServiceService,
	redis *cache.RedisClient,
) *OfferService {
	return &OfferService{
		db:                  db,
		repo:                repo,
		listingRepo:         listingRepo,
		serviceRepo:         serviceRepo,
		tradeRepo:           tradeRepo,
		chatRepo:            chatRepo,
		serviceRunRepo:      serviceRunRepo,
		notificationService: notificationService,
		profileService:      profileService,
		listingService:      listingService,
		serviceService:      serviceService,
		redis:               redis,
		invalidator:         cache.NewInvalidator(redis),
	}
}

// SetStatsService sets the stats service for cache refresh on offer events
func (s *OfferService) SetStatsService(ss *StatsService) {
	s.statsService = ss
}

// Create creates a new offer (item or service)
func (s *OfferService) Create(ctx context.Context, requesterID string, req *dto.CreateOfferRequest) (*models.Offer, error) {
	offer := &models.Offer{
		ID:           uuid.New().String(),
		Type:         req.Type,
		RequesterID:  requesterID,
		OfferedItems: req.OfferedItems,
		Status:       "pending",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if req.Message != "" {
		offer.Message = &req.Message
	}

	if req.Type == "service" {
		// Service offer
		if req.ServiceID == nil || *req.ServiceID == "" {
			return nil, ErrInvalidState
		}

		service, err := s.serviceRepo.GetByID(ctx, *req.ServiceID)
		if err != nil {
			return nil, err
		}

		if service.ProviderID == requesterID {
			return nil, ErrSelfAction
		}

		if !service.IsActive() {
			return nil, ErrInvalidState
		}

		offer.ServiceID = req.ServiceID
	} else {
		// Item offer
		if req.ListingID == nil || *req.ListingID == "" {
			return nil, ErrInvalidState
		}

		listing, err := s.listingRepo.GetByID(ctx, *req.ListingID)
		if err != nil {
			return nil, err
		}

		if listing.SellerID == requesterID {
			return nil, ErrSelfAction
		}

		if !listing.IsActive() {
			return nil, ErrInvalidState
		}

		// Check if listing has an active trade
		hasActive, err := s.tradeRepo.HasActiveTradeForListing(ctx, *req.ListingID)
		if err != nil {
			return nil, err
		}
		if hasActive {
			return nil, ErrInvalidState
		}

		offer.ListingID = req.ListingID
	}

	if err := s.repo.Create(ctx, offer); err != nil {
		return nil, err
	}

	// Notify the seller/provider
	if req.Type == "service" {
		service, _ := s.serviceRepo.GetByID(ctx, *req.ServiceID)
		if service != nil {
			_ = s.notificationService.NotifyOfferReceived(ctx, service.ProviderID, offer.ID, service.Name)
		}
	} else {
		listing, _ := s.listingRepo.GetByID(ctx, *req.ListingID)
		if listing != nil {
			_ = s.notificationService.NotifyOfferReceived(ctx, listing.SellerID, offer.ID, listing.Name)
		}
	}

	return offer, nil
}

// GetByID retrieves an offer by ID
func (s *OfferService) GetByID(ctx context.Context, id string, userID string) (*models.Offer, error) {
	offer, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check user is a participant
	if !s.isOfferParticipant(offer, userID) {
		return nil, ErrForbidden
	}

	return offer, nil
}

// Accept accepts an offer and creates a Trade+Chat (item) or ServiceRun+Chat (service)
func (s *OfferService) Accept(ctx context.Context, id string, userID string) (*models.Offer, *models.Trade, *models.ServiceRun, *models.Chat, error) {
	offer, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Only seller/provider can accept
	if !s.isOfferOwner(offer, userID) {
		return nil, nil, nil, nil, ErrForbidden
	}

	if !offer.IsPending() {
		return nil, nil, nil, nil, ErrInvalidState
	}

	now := time.Now()
	offer.Status = "accepted"
	offer.AcceptedAt = &now
	offer.UpdatedAt = now

	if err := s.repo.Update(ctx, offer); err != nil {
		return nil, nil, nil, nil, err
	}

	if offer.IsServiceOffer() {
		// Create ServiceRun + Chat
		serviceRun := &models.ServiceRun{
			ID:         uuid.New().String(),
			ServiceID:  *offer.ServiceID,
			OfferID:    offer.ID,
			ProviderID: offer.Service.ProviderID,
			ClientID:   offer.RequesterID,
			Status:     "active",
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := s.serviceRunRepo.Create(ctx, serviceRun); err != nil {
			return nil, nil, nil, nil, err
		}

		serviceRunID := serviceRun.ID
		chat := &models.Chat{
			ID:           uuid.New().String(),
			ServiceRunID: &serviceRunID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := s.chatRepo.Create(ctx, chat); err != nil {
			return nil, nil, nil, nil, err
		}

		_ = s.notificationService.NotifyOfferAccepted(ctx, offer.RequesterID, offer.ID, offer.Service.Name)
		_ = s.notificationService.NotifyServiceRunCreated(ctx, offer.RequesterID, serviceRun.ID, offer.Service.Name)

		if s.statsService != nil {
			go s.statsService.RefreshHomeStats(context.Background())
		}

		return offer, nil, serviceRun, chat, nil
	}

	// Item offer: create Trade + Chat
	hasActive, err := s.tradeRepo.HasActiveTradeForListing(ctx, *offer.ListingID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if hasActive {
		return nil, nil, nil, nil, ErrInvalidState
	}

	trade := &models.Trade{
		ID:        uuid.New().String(),
		OfferID:   offer.ID,
		ListingID: *offer.ListingID,
		SellerID:  offer.Listing.SellerID,
		BuyerID:   offer.RequesterID,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.tradeRepo.Create(ctx, trade); err != nil {
		return nil, nil, nil, nil, err
	}

	tradeID := trade.ID
	chat := &models.Chat{
		ID:        uuid.New().String(),
		TradeID:   &tradeID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.chatRepo.Create(ctx, chat); err != nil {
		return nil, nil, nil, nil, err
	}

	_ = s.notificationService.NotifyOfferAccepted(ctx, offer.RequesterID, offer.ID, offer.Listing.Name)

	if s.statsService != nil {
		go s.statsService.RefreshHomeStats(context.Background())
	}

	return offer, trade, nil, chat, nil
}

// Reject rejects an offer
func (s *OfferService) Reject(ctx context.Context, id string, userID string, req *dto.RejectOfferRequest) (*models.Offer, error) {
	offer, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	if !s.isOfferOwner(offer, userID) {
		return nil, ErrForbidden
	}

	if !offer.IsPending() {
		return nil, ErrInvalidState
	}

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

	itemName := s.getOfferItemName(offer)
	_ = s.notificationService.NotifyOfferRejected(ctx, offer.RequesterID, offer.ID, itemName)

	return offer, nil
}

// Cancel cancels a pending offer (buyer only)
func (s *OfferService) Cancel(ctx context.Context, id string, userID string) (*models.Offer, error) {
	offer, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	if offer.RequesterID != userID {
		return nil, ErrForbidden
	}

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
func (s *OfferService) List(ctx context.Context, userID string, role string, status string, offerType string, listingID string, serviceID string, offset, limit int) ([]*models.Offer, int, error) {
	if role == "seller" && status == "" {
		status = "pending"
	}

	filter := repository.OfferFilter{
		UserID:    userID,
		Role:      role,
		Status:    status,
		Type:      offerType,
		ListingID: listingID,
		ServiceID: serviceID,
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
		Type:         offer.Type,
		ListingID:    offer.GetListingID(),
		ServiceID:    offer.GetServiceID(),
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

	if offer.Service != nil && s.serviceService != nil {
		resp.Service = s.serviceService.ToServiceResponse(offer.Service)
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

	if offer.ServiceRun != nil {
		resp.ServiceRunID = &offer.ServiceRun.ID
	}

	return resp
}

// ToDetailResponse converts an offer model to a detailed DTO response
func (s *OfferService) ToDetailResponse(offer *models.Offer, userID string) *dto.OfferDetailResponse {
	return &dto.OfferDetailResponse{
		OfferResponse: *s.ToResponse(offer),
	}
}

// isOfferParticipant checks if the user is a participant in the offer
func (s *OfferService) isOfferParticipant(offer *models.Offer, userID string) bool {
	if offer.RequesterID == userID {
		return true
	}
	if offer.IsItemOffer() && offer.Listing != nil && offer.Listing.SellerID == userID {
		return true
	}
	if offer.IsServiceOffer() && offer.Service != nil && offer.Service.ProviderID == userID {
		return true
	}
	return false
}

// isOfferOwner checks if the user is the seller/provider for this offer
func (s *OfferService) isOfferOwner(offer *models.Offer, userID string) bool {
	if offer.IsItemOffer() && offer.Listing != nil {
		return offer.Listing.SellerID == userID
	}
	if offer.IsServiceOffer() && offer.Service != nil {
		return offer.Service.ProviderID == userID
	}
	return false
}

// getOfferItemName returns the name of the item/service for this offer
func (s *OfferService) getOfferItemName(offer *models.Offer) string {
	if offer.IsItemOffer() && offer.Listing != nil {
		return offer.Listing.Name
	}
	if offer.IsServiceOffer() && offer.Service != nil {
		return offer.Service.Name
	}
	return "item"
}
