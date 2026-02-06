package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

// RatingService handles rating business logic
type RatingService struct {
	repo                repository.RatingRepository
	transactionRepo     repository.TransactionRepository
	profileService      *ProfileService
	notificationService *NotificationService
}

// NewRatingService creates a new rating service
func NewRatingService(
	repo repository.RatingRepository,
	transactionRepo repository.TransactionRepository,
	profileService *ProfileService,
	notificationService *NotificationService,
) *RatingService {
	return &RatingService{
		repo:                repo,
		transactionRepo:     transactionRepo,
		profileService:      profileService,
		notificationService: notificationService,
	}
}

// Create creates a new rating for a transaction
func (s *RatingService) Create(ctx context.Context, raterID string, req *dto.CreateRatingRequest) (*models.Rating, error) {
	// Get the transaction
	transaction, err := s.transactionRepo.GetByID(ctx, req.TransactionID)
	if err != nil {
		return nil, err
	}

	// Verify rater is a participant
	if transaction.SellerID != raterID && transaction.BuyerID != raterID {
		return nil, ErrForbidden
	}

	// Check if already rated
	exists, err := s.repo.Exists(ctx, req.TransactionID, raterID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyExists
	}

	// Determine who is being rated
	var ratedID string
	if transaction.SellerID == raterID {
		ratedID = transaction.BuyerID
	} else {
		ratedID = transaction.SellerID
	}

	rating := &models.Rating{
		ID:            uuid.New().String(),
		TransactionID: req.TransactionID,
		RaterID:       raterID,
		RatedID:       ratedID,
		Stars:         req.Stars,
		CreatedAt:     time.Now(),
	}

	if req.Comment != "" {
		rating.Comment = &req.Comment
	}

	if err := s.repo.Create(ctx, rating); err != nil {
		return nil, err
	}

	// Notify the rated user
	_ = s.notificationService.NotifyRatingReceived(ctx, ratedID, req.TransactionID, req.Stars)

	return rating, nil
}

// GetByUserID retrieves ratings for a user
func (s *RatingService) GetByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.Rating, int, error) {
	return s.repo.GetByUserID(ctx, userID, offset, limit)
}

// GetByTransactionID retrieves ratings for a transaction
func (s *RatingService) GetByTransactionID(ctx context.Context, transactionID string) ([]*models.Rating, error) {
	return s.repo.GetByTransactionID(ctx, transactionID)
}

// ToResponse converts a rating model to a DTO response
func (s *RatingService) ToResponse(rating *models.Rating) *dto.RatingResponse {
	resp := &dto.RatingResponse{
		ID:            rating.ID,
		TransactionID: rating.TransactionID,
		RaterID:       rating.RaterID,
		RatedID:       rating.RatedID,
		Stars:         rating.Stars,
		Comment:       rating.GetComment(),
		CreatedAt:     rating.CreatedAt,
	}

	if rating.Rater != nil {
		resp.Rater = s.profileService.ToResponse(rating.Rater)
	}

	return resp
}
