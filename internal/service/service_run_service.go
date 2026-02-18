package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

// ServiceRunService handles service run business logic
type ServiceRunService struct {
	repo                repository.ServiceRunRepository
	transactionRepo     repository.TransactionRepository
	ratingRepo          repository.RatingRepository
	chatRepo            repository.ChatRepository
	notificationService *NotificationService
	profileService      *ProfileService
	serviceService      *ServiceService
	redis               *cache.RedisClient
	invalidator         *cache.Invalidator
}

// NewServiceRunService creates a new service run service
func NewServiceRunService(
	repo repository.ServiceRunRepository,
	transactionRepo repository.TransactionRepository,
	ratingRepo repository.RatingRepository,
	chatRepo repository.ChatRepository,
	notificationService *NotificationService,
	profileService *ProfileService,
	serviceService *ServiceService,
	redis *cache.RedisClient,
) *ServiceRunService {
	return &ServiceRunService{
		repo:                repo,
		transactionRepo:     transactionRepo,
		ratingRepo:          ratingRepo,
		chatRepo:            chatRepo,
		notificationService: notificationService,
		profileService:      profileService,
		serviceService:      serviceService,
		redis:               redis,
		invalidator:         cache.NewInvalidator(redis),
	}
}

// GetByID retrieves a service run by ID with participant check
func (s *ServiceRunService) GetByID(ctx context.Context, id string, userID string) (*models.ServiceRun, error) {
	run, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	if run.ProviderID != userID && run.ClientID != userID {
		return nil, ErrForbidden
	}

	return run, nil
}

// Complete marks a service run as completed and creates a transaction
func (s *ServiceRunService) Complete(ctx context.Context, id string, userID string) (*models.ServiceRun, *models.Transaction, error) {
	run, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	// Either party can complete
	if run.ProviderID != userID && run.ClientID != userID {
		return nil, nil, ErrForbidden
	}

	// If already completed, return existing run and transaction (idempotent)
	if run.IsCompleted() {
		transaction, err := s.transactionRepo.GetByServiceRunID(ctx, run.ID)
		if err != nil {
			return nil, nil, err
		}
		return run, transaction, nil
	}

	if !run.IsActive() {
		return nil, nil, ErrInvalidState
	}

	now := time.Now()
	run.Status = "completed"
	run.CompletedAt = &now
	run.UpdatedAt = now

	if err := s.repo.Update(ctx, run); err != nil {
		return nil, nil, err
	}

	// Create transaction for rating eligibility
	serviceRunID := run.ID
	itemDetails, _ := json.Marshal(map[string]string{
		"serviceType": run.Service.ServiceType,
		"serviceName": run.Service.Name,
	})
	transaction := &models.Transaction{
		ID:           uuid.New().String(),
		ServiceRunID: &serviceRunID,
		SellerID:     run.ProviderID,
		BuyerID:      run.ClientID,
		ItemName:     run.Service.Name,
		ItemDetails:  itemDetails,
		OfferedItems: run.Offer.OfferedItems,
		CreatedAt:    now,
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, nil, err
	}

	// Notify the other party - service stays active
	var recipientID string
	if run.ProviderID == userID {
		recipientID = run.ClientID
	} else {
		recipientID = run.ProviderID
	}
	_ = s.notificationService.NotifyServiceRunCompleted(ctx, recipientID, run.ID, run.Service.Name)

	return run, transaction, nil
}

// Cancel cancels an active service run
func (s *ServiceRunService) Cancel(ctx context.Context, id string, userID string, reason string) (*models.ServiceRun, error) {
	run, err := s.repo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	if run.ProviderID != userID && run.ClientID != userID {
		return nil, ErrForbidden
	}

	if !run.IsActive() {
		return nil, ErrInvalidState
	}

	now := time.Now()
	run.Status = "cancelled"
	run.CancelledAt = &now
	run.CancelledBy = &userID
	if reason != "" {
		run.CancelReason = &reason
	}
	run.UpdatedAt = now

	if err := s.repo.Update(ctx, run); err != nil {
		return nil, err
	}

	// Notify the other party - service stays active
	var recipientID string
	if run.ProviderID == userID {
		recipientID = run.ClientID
	} else {
		recipientID = run.ProviderID
	}
	_ = s.notificationService.NotifyServiceRunCancelled(ctx, recipientID, run.ID, run.Service.Name)

	return run, nil
}

// List retrieves service runs for a user
func (s *ServiceRunService) List(ctx context.Context, userID string, role string, status string, offset, limit int) ([]*models.ServiceRun, int, error) {
	filter := repository.ServiceRunFilter{
		UserID: userID,
		Role:   role,
		Status: status,
		Offset: offset,
		Limit:  limit,
	}
	return s.repo.List(ctx, filter)
}

// ToResponse converts a service run model to a DTO response
func (s *ServiceRunService) ToResponse(run *models.ServiceRun) *dto.ServiceRunResponse {
	return s.toResponseInternal(run, nil, "")
}

// ToResponseWithUser converts a service run with user-specific rating info
func (s *ServiceRunService) ToResponseWithUser(ctx context.Context, run *models.ServiceRun, userID string) *dto.ServiceRunResponse {
	return s.toResponseInternal(run, &ctx, userID)
}

func (s *ServiceRunService) toResponseInternal(run *models.ServiceRun, ctx *context.Context, userID string) *dto.ServiceRunResponse {
	resp := &dto.ServiceRunResponse{
		ID:           run.ID,
		ServiceID:    run.ServiceID,
		OfferID:      run.OfferID,
		ProviderID:   run.ProviderID,
		ClientID:     run.ClientID,
		Status:       run.Status,
		CancelReason: run.GetCancelReason(),
		CancelledBy:  run.GetCancelledBy(),
		CanRate:      false,
		CreatedAt:    run.CreatedAt,
		UpdatedAt:    run.UpdatedAt,
		CompletedAt:  run.CompletedAt,
		CancelledAt:  run.CancelledAt,
	}

	if run.Service != nil {
		resp.Service = s.serviceService.ToServiceResponse(run.Service)
	}

	if run.Provider != nil {
		resp.Provider = s.profileService.ToResponse(run.Provider)
	}

	if run.Client != nil {
		resp.Client = s.profileService.ToResponse(run.Client)
	}

	if run.Chat != nil {
		resp.ChatID = &run.Chat.ID
	}

	if run.Offer != nil {
		resp.OfferedItems = run.Offer.OfferedItems
	}

	// For completed runs, fetch transaction and check rating eligibility
	if run.IsCompleted() && ctx != nil && userID != "" {
		transaction, err := s.transactionRepo.GetByServiceRunID(*ctx, run.ID)
		if err == nil && transaction != nil {
			resp.TransactionID = &transaction.ID

			hasRated, err := s.ratingRepo.Exists(*ctx, transaction.ID, userID)
			if err == nil {
				resp.CanRate = !hasRated
			}
		}
	}

	return resp
}

// ToDetailResponse converts a service run to a detailed DTO response
func (s *ServiceRunService) ToDetailResponse(ctx context.Context, run *models.ServiceRun, userID string) *dto.ServiceRunDetailResponse {
	return &dto.ServiceRunDetailResponse{
		ServiceRunResponse: *s.ToResponseWithUser(ctx, run, userID),
		CanComplete:        run.IsActive() && (run.ProviderID == userID || run.ClientID == userID),
		CanCancel:          run.IsActive(),
		CanMessage:         run.IsActive(),
	}
}
