package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

// TradeServiceNew handles trade business logic
type TradeServiceNew struct {
	db                  *database.BunDB
	repo                repository.TradeRepository
	listingRepo         repository.ListingRepository
	offerRepo           repository.OfferRepository
	transactionRepo     repository.TransactionRepository
	ratingRepo          repository.RatingRepository
	chatRepo            repository.ChatRepository
	notificationService *NotificationService
	profileService      *ProfileService
	listingService      *ListingService
	statsService        *StatsService
	redis               *cache.RedisClient
	invalidator         *cache.Invalidator
	supabaseURL         string
}

// NewTradeServiceNew creates a new trade service
func NewTradeServiceNew(
	db *database.BunDB,
	repo repository.TradeRepository,
	listingRepo repository.ListingRepository,
	offerRepo repository.OfferRepository,
	transactionRepo repository.TransactionRepository,
	ratingRepo repository.RatingRepository,
	chatRepo repository.ChatRepository,
	notificationService *NotificationService,
	profileService *ProfileService,
	listingService *ListingService,
	redis *cache.RedisClient,
	supabaseURL string,
) *TradeServiceNew {
	return &TradeServiceNew{
		db:                  db,
		repo:                repo,
		listingRepo:         listingRepo,
		offerRepo:           offerRepo,
		transactionRepo:     transactionRepo,
		ratingRepo:          ratingRepo,
		chatRepo:            chatRepo,
		notificationService: notificationService,
		profileService:      profileService,
		listingService:      listingService,
		redis:               redis,
		invalidator:         cache.NewInvalidator(redis),
		supabaseURL:         strings.TrimSuffix(supabaseURL, "/"),
	}
}

// SetStatsService sets the stats service for cache refresh on trade events
func (s *TradeServiceNew) SetStatsService(ss *StatsService) {
	s.statsService = ss
}

// offeredItemRaw represents the raw offered item from JSON
type offeredItemRaw struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	ImageURL string `json:"imageUrl,omitempty"`
	Quantity int    `json:"quantity"`
}

// transformOfferedItems converts raw offered items JSON to DTO with image URLs
func (s *TradeServiceNew) transformOfferedItems(rawItems json.RawMessage) []dto.OfferedItemResponse {
	if len(rawItems) == 0 {
		return nil
	}

	var items []offeredItemRaw
	if err := json.Unmarshal(rawItems, &items); err != nil {
		return nil
	}

	result := make([]dto.OfferedItemResponse, 0, len(items))
	for _, item := range items {
		resp := dto.OfferedItemResponse{
			ID:       item.ID,
			Name:     item.Name,
			Type:     item.Type,
			Quantity: item.Quantity,
		}

		// Use existing imageUrl if provided, otherwise generate it
		if item.ImageURL != "" {
			resp.ImageURL = item.ImageURL
		} else {
			resp.ImageURL = s.generateItemImageURL(item.Name, item.Type)
		}

		result = append(result, resp)
	}

	return result
}

// generateItemImageURL generates an image URL based on item type and name
func (s *TradeServiceNew) generateItemImageURL(name, itemType string) string {
	if s.supabaseURL == "" {
		return ""
	}

	// Normalize the name for URL: lowercase, replace spaces with hyphens, remove special chars
	normalized := strings.ToLower(name)
	normalized = strings.ReplaceAll(normalized, " ", "-")
	// Remove any characters that aren't alphanumeric or hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	normalized = reg.ReplaceAllString(normalized, "")
	// Remove consecutive hyphens
	for strings.Contains(normalized, "--") {
		normalized = strings.ReplaceAll(normalized, "--", "-")
	}
	normalized = strings.Trim(normalized, "-")

	// Determine the bucket path based on item type
	var path string
	switch strings.ToLower(itemType) {
	case "rune":
		path = fmt.Sprintf("runes/%s.png", normalized)
	case "gem":
		path = fmt.Sprintf("gems/%s.png", normalized)
	case "unique":
		path = fmt.Sprintf("uniques/%s.png", normalized)
	case "set":
		path = fmt.Sprintf("sets/%s.png", normalized)
	case "runeword":
		path = fmt.Sprintf("runewords/%s.png", normalized)
	case "base":
		path = fmt.Sprintf("bases/%s.png", normalized)
	default:
		path = fmt.Sprintf("items/%s.png", normalized)
	}

	return fmt.Sprintf("%s/storage/v1/object/public/d2-items/%s", s.supabaseURL, path)
}

// GetByID retrieves a trade by ID
func (s *TradeServiceNew) GetByID(ctx context.Context, id string, userID string) (*models.Trade, error) {
	trade, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check user is a participant
	if trade.SellerID != userID && trade.BuyerID != userID {
		return nil, ErrForbidden
	}

	return trade, nil
}

// Complete marks a trade as completed and creates a transaction
func (s *TradeServiceNew) Complete(ctx context.Context, id string, userID string) (*models.Trade, *models.Transaction, error) {
	trade, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	// Either party can complete
	if trade.SellerID != userID && trade.BuyerID != userID {
		return nil, nil, ErrForbidden
	}

	// If already completed, return existing trade and transaction (idempotent)
	if trade.IsCompleted() {
		transaction, err := s.transactionRepo.GetByTradeID(ctx, trade.ID)
		if err != nil {
			return nil, nil, err
		}
		return trade, transaction, nil
	}

	// Must be active (not cancelled)
	if !trade.IsActive() {
		return nil, nil, ErrInvalidState
	}

	now := time.Now()
	trade.Status = "completed"
	trade.CompletedAt = &now
	trade.UpdatedAt = now

	if err := s.repo.Update(ctx, trade); err != nil {
		return nil, nil, err
	}

	// Sync offer status to completed
	if trade.Offer != nil {
		trade.Offer.Status = "completed"
		trade.Offer.UpdatedAt = now
		_ = s.offerRepo.Update(ctx, trade.Offer)
	}

	// Update listing status to completed
	listing, err := s.listingRepo.GetByID(ctx, trade.ListingID)
	if err != nil {
		return nil, nil, err
	}
	listing.Status = "completed"
	_ = s.listingRepo.Update(ctx, listing)

	// Create transaction
	tradeID := trade.ID
	listingID := trade.ListingID
	transaction := &models.Transaction{
		ID:           uuid.New().String(),
		TradeID:      &tradeID,
		ListingID:    &listingID,
		SellerID:     trade.SellerID,
		BuyerID:      trade.BuyerID,
		ItemName:     listing.Name,
		ItemDetails:  listing.Stats,
		OfferedItems: trade.Offer.OfferedItems,
		CreatedAt:    now,
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, nil, err
	}

	// Notify the other party that trade is completed
	var recipientID string
	if trade.SellerID == userID {
		recipientID = trade.BuyerID
	} else {
		recipientID = trade.SellerID
	}
	_ = s.notificationService.NotifyTradeCompleted(ctx, recipientID, trade.ID, listing.Name)

	// Invalidate listing DTO cache (status changed to completed)
	_ = s.invalidator.InvalidateListingDTO(ctx, trade.ListingID)

	// Remove from the appropriate recent cache
	s.listingService.RemoveFromRecentByListing(ctx, listing)

	// Refresh home stats (tradesToday + activeListings changed)
	if s.statsService != nil {
		go s.statsService.RefreshHomeStats(context.Background())
	}

	return trade, transaction, nil
}

// Cancel cancels an active trade (either party)
func (s *TradeServiceNew) Cancel(ctx context.Context, id string, userID string, reason string) (*models.Trade, error) {
	trade, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	// Either party can cancel
	if trade.SellerID != userID && trade.BuyerID != userID {
		return nil, ErrForbidden
	}

	// Can only cancel active trades
	if !trade.IsActive() {
		return nil, ErrInvalidState
	}

	now := time.Now()
	trade.Status = "cancelled"
	trade.CancelledAt = &now
	trade.CancelledBy = &userID
	if reason != "" {
		trade.CancelReason = &reason
	}
	trade.UpdatedAt = now

	if err := s.repo.Update(ctx, trade); err != nil {
		return nil, err
	}

	// Sync offer status to cancelled
	if trade.Offer != nil {
		trade.Offer.Status = "cancelled"
		trade.Offer.UpdatedAt = now
		_ = s.offerRepo.Update(ctx, trade.Offer)
	}

	// Listing becomes visible again - ensure it's active
	listing, err := s.listingRepo.GetByID(ctx, trade.ListingID)
	if err == nil && listing.Status != "completed" && listing.Status != "cancelled" {
		listing.Status = "active"
		_ = s.listingRepo.Update(ctx, listing)
	}

	// Notify the other party
	var recipientID string
	if trade.SellerID == userID {
		recipientID = trade.BuyerID
	} else {
		recipientID = trade.SellerID
	}
	_ = s.notificationService.NotifyTradeCancelled(ctx, recipientID, trade.ID, listing.Name)

	// Invalidate listing DTO cache (status may have changed back to active)
	_ = s.invalidator.InvalidateListingDTO(ctx, trade.ListingID)

	// Refresh home stats (activeListings may have changed)
	if s.statsService != nil {
		go s.statsService.RefreshHomeStats(context.Background())
	}

	return trade, nil
}

// List retrieves trades for a user
func (s *TradeServiceNew) List(ctx context.Context, userID string, status string, offset, limit int) ([]*models.Trade, int, error) {
	filter := repository.TradeFilter{
		UserID: userID,
		Status: status,
		Offset: offset,
		Limit:  limit,
	}
	return s.repo.List(ctx, filter)
}

// ToResponse converts a trade model to a DTO response
// For completed trades, it fetches the transaction and checks rating status
func (s *TradeServiceNew) ToResponse(trade *models.Trade) *dto.TradeResponse {
	return s.toResponseInternal(trade, nil, "")
}

// ToResponseWithUser converts a trade model to a DTO response with user-specific rating info
// For completed trades, it fetches the transaction and checks if the user has already rated
func (s *TradeServiceNew) ToResponseWithUser(ctx context.Context, trade *models.Trade, userID string) *dto.TradeResponse {
	return s.toResponseInternal(trade, &ctx, userID)
}

// toResponseInternal is the internal implementation that handles both cases
func (s *TradeServiceNew) toResponseInternal(trade *models.Trade, ctx *context.Context, userID string) *dto.TradeResponse {
	resp := &dto.TradeResponse{
		ID:           trade.ID,
		OfferID:      trade.OfferID,
		ListingID:    trade.ListingID,
		SellerID:     trade.SellerID,
		BuyerID:      trade.BuyerID,
		Status:       trade.Status,
		CancelReason: trade.GetCancelReason(),
		CancelledBy:  trade.GetCancelledBy(),
		CreatedAt:    trade.CreatedAt,
		UpdatedAt:    trade.UpdatedAt,
		CompletedAt:  trade.CompletedAt,
		CancelledAt:  trade.CancelledAt,
		CanRate:      false,
	}

	if trade.Listing != nil {
		resp.Listing = s.listingService.ToResponse(trade.Listing)
	}

	if trade.Seller != nil {
		resp.Seller = s.profileService.ToResponse(trade.Seller)
	}

	if trade.Buyer != nil {
		resp.Buyer = s.profileService.ToResponse(trade.Buyer)
	}

	if trade.Chat != nil {
		resp.ChatID = &trade.Chat.ID
	}

	if trade.Offer != nil {
		resp.OfferedItems = s.transformOfferedItems(trade.Offer.OfferedItems)
	}

	// For completed trades, fetch transaction and check rating eligibility
	if trade.IsCompleted() && ctx != nil && userID != "" {
		transaction, err := s.transactionRepo.GetByTradeID(*ctx, trade.ID)
		if err == nil && transaction != nil {
			resp.TransactionID = &transaction.ID

			// Check if user can rate (has not already rated)
			hasRated, err := s.ratingRepo.Exists(*ctx, transaction.ID, userID)
			if err == nil {
				resp.CanRate = !hasRated
			}
		}
	}

	return resp
}

// ToDetailResponse converts a trade model to a detailed DTO response
func (s *TradeServiceNew) ToDetailResponse(ctx context.Context, trade *models.Trade, userID string) *dto.TradeDetailResponse {
	return &dto.TradeDetailResponse{
		TradeResponse: *s.ToResponseWithUser(ctx, trade, userID),
		CanComplete:   trade.IsActive() && (trade.SellerID == userID || trade.BuyerID == userID),
		CanCancel:     trade.IsActive(),
		CanMessage:    trade.IsActive(),
	}
}
