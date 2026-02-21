package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

const maxRecentServices = 20

// ServiceService handles service business logic
type ServiceService struct {
	repo           repository.ServiceRepository
	profileService *ProfileService
	redis          *cache.RedisClient
	invalidator    *cache.Invalidator
}

// NewServiceService creates a new service service
func NewServiceService(repo repository.ServiceRepository, profileService *ProfileService, redis *cache.RedisClient) *ServiceService {
	return &ServiceService{
		repo:           repo,
		profileService: profileService,
		redis:          redis,
		invalidator:    cache.NewInvalidator(redis),
	}
}

// Create creates a new service
func (s *ServiceService) Create(ctx context.Context, providerID string, req *dto.CreateServiceRequest) (*models.Service, error) {
	log := logger.FromContext(ctx)
	log.Info("creating new service",
		"provider_id", providerID,
		"service_type", req.ServiceType,
		"game", req.Game,
	)

	// Check uniqueness: one service per type per provider per game
	exists, err := s.repo.ExistsByProviderAndType(ctx, providerID, req.ServiceType, req.Game)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyExists
	}

	// Deduplicate platforms
	seen := make(map[string]bool)
	var uniquePlatforms []string
	for _, p := range req.Platforms {
		if !seen[p] {
			seen[p] = true
			uniquePlatforms = append(uniquePlatforms, p)
		}
	}

	service := &models.Service{
		ID:          uuid.New().String(),
		ProviderID:  providerID,
		ServiceType: req.ServiceType,
		Name:        req.Name,
		AskingFor:   req.AskingFor,
		Game:        req.Game,
		Ladder:      req.Ladder,
		Hardcore:    req.Hardcore,
		IsNonRotw:   req.IsNonRotw,
		Platforms:   uniquePlatforms,
		Region:      req.Region,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if req.Description != "" {
		service.Description = &req.Description
	}
	if req.AskingPrice != "" {
		service.AskingPrice = &req.AskingPrice
	}
	if req.Notes != "" {
		service.Notes = &req.Notes
	}

	if err := s.repo.Create(ctx, service); err != nil {
		log.Error("failed to create service", "error", err.Error())
		return nil, err
	}

	log.Info("service created successfully",
		"service_id", service.ID,
		"provider_id", providerID,
		"service_type", req.ServiceType,
	)
	fmt.Printf("[SERVICE] Created service: id=%s type=%s provider=%s\n", service.ID, req.ServiceType, providerID)

	// Invalidate provider cache
	_ = s.invalidator.InvalidateServiceProviders(ctx, service.Game)

	// Push to recent services cache
	profile, _ := s.profileService.GetByID(ctx, providerID)
	service.Provider = profile
	s.pushToRecentServices(ctx, service)

	return service, nil
}

// Update updates a service
func (s *ServiceService) Update(ctx context.Context, id string, userID string, req *dto.UpdateServiceRequest) (*models.Service, error) {
	service, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if service.ProviderID != userID {
		return nil, ErrForbidden
	}

	if req.Name != nil {
		service.Name = *req.Name
	}
	if req.Description != nil {
		service.Description = req.Description
	}
	if req.AskingPrice != nil {
		service.AskingPrice = req.AskingPrice
	}
	if req.AskingFor != nil {
		service.AskingFor = req.AskingFor
	}
	if req.Notes != nil {
		service.Notes = req.Notes
	}
	if len(req.Platforms) > 0 {
		service.Platforms = req.Platforms
	}
	if req.Region != nil {
		service.Region = *req.Region
	}
	service.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, service); err != nil {
		return nil, err
	}

	_ = s.invalidator.InvalidateService(ctx, id)
	_ = s.invalidator.InvalidateServiceProviders(ctx, service.Game)

	return service, nil
}

// Delete hard-deletes a service
func (s *ServiceService) Delete(ctx context.Context, id string, userID string) error {
	service, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if service.ProviderID != userID {
		return ErrForbidden
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	_ = s.invalidator.InvalidateService(ctx, id)
	_ = s.invalidator.InvalidateServiceProviders(ctx, service.Game)

	// Remove from recent services cache
	s.removeFromRecentServices(ctx, id)

	return nil
}

// Pause pauses an active service (hides from public search)
func (s *ServiceService) Pause(ctx context.Context, id string, userID string) error {
	service, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if service.ProviderID != userID {
		return ErrForbidden
	}

	if service.Status != "active" {
		return ErrInvalidState
	}

	service.Status = "paused"
	service.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, service); err != nil {
		return err
	}

	_ = s.invalidator.InvalidateService(ctx, id)
	_ = s.invalidator.InvalidateServiceProviders(ctx, service.Game)

	// Remove from recent services cache
	s.removeFromRecentServices(ctx, id)

	return nil
}

// Resume resumes a paused service (makes it visible in public search again)
func (s *ServiceService) Resume(ctx context.Context, id string, userID string) error {
	service, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if service.ProviderID != userID {
		return ErrForbidden
	}

	if service.Status != "paused" {
		return ErrInvalidState
	}

	service.Status = "active"
	service.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, service); err != nil {
		return err
	}

	_ = s.invalidator.InvalidateService(ctx, id)
	_ = s.invalidator.InvalidateServiceProviders(ctx, service.Game)

	// Push back to recent services cache
	profile, _ := s.profileService.GetByID(ctx, service.ProviderID)
	service.Provider = profile
	s.pushToRecentServices(ctx, service)

	return nil
}

// GetByID retrieves a service by ID
func (s *ServiceService) GetByID(ctx context.Context, id string) (*models.Service, error) {
	return s.repo.GetByIDWithProvider(ctx, id)
}

// ListMyServices lists services for a provider
func (s *ServiceService) ListMyServices(ctx context.Context, providerID string, offset, limit int) ([]*models.Service, int, error) {
	return s.repo.ListByProviderID(ctx, providerID, offset, limit)
}

// ListProviders lists provider cards with their services
func (s *ServiceService) ListProviders(ctx context.Context, filter repository.ServiceProviderFilter) ([]dto.ProviderCardResponse, int, error) {
	providers, count, err := s.repo.ListProviders(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	results := make([]dto.ProviderCardResponse, 0, len(providers))
	for _, pw := range providers {
		results = append(results, s.ToProviderCardResponse(pw.Provider, pw.Services))
	}

	return results, count, nil
}

// GetProviderDetail returns a single provider card. Accepts UUID or username.
func (s *ServiceService) GetProviderDetail(ctx context.Context, identifier string) (*dto.ProviderCardResponse, error) {
	profile, err := s.profileService.GetByIdentifier(ctx, identifier)
	if err != nil {
		return nil, err
	}

	services, err := s.repo.GetProviderServices(ctx, profile.ID)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, ErrNotFound
	}

	card := s.ToProviderCardResponse(profile, services)
	return &card, nil
}

// ToServiceResponse converts a service model to a DTO
func (s *ServiceService) ToServiceResponse(service *models.Service) *dto.ServiceResponse {
	return &dto.ServiceResponse{
		ID:          service.ID,
		ServiceType: service.ServiceType,
		Name:        service.Name,
		Description: service.GetDescription(),
		AskingPrice: service.GetAskingPrice(),
		AskingFor:   service.AskingFor,
		Game:        service.Game,
		Ladder:      service.Ladder,
		Hardcore:    service.Hardcore,
		IsNonRotw:   service.IsNonRotw,
		Platforms:   service.Platforms,
		Region:      service.Region,
		Notes:       service.GetNotes(),
		Status:      service.Status,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}
}

// ToProviderCardResponse converts a provider and services to a provider card DTO
func (s *ServiceService) ToProviderCardResponse(provider *models.Profile, services []*models.Service) dto.ProviderCardResponse {
	serviceResponses := make([]dto.ServiceResponse, 0, len(services))
	for _, svc := range services {
		serviceResponses = append(serviceResponses, *s.ToServiceResponse(svc))
	}

	return dto.ProviderCardResponse{
		Provider: s.profileService.ToResponse(provider),
		Services: serviceResponses,
	}
}

// pushToRecentServices adds/updates a provider's service in the home:recent:services cache
func (s *ServiceService) pushToRecentServices(ctx context.Context, service *models.Service) {
	cards := s.readRecentServicesCards(ctx)

	serviceResp := s.ToServiceResponse(service)
	providerResp := s.profileService.ToResponse(service.Provider)

	// Find existing provider card
	found := false
	for i, card := range cards {
		if card.Provider != nil && card.Provider.ID == service.ProviderID {
			cards[i].Services = append(cards[i].Services, *serviceResp)
			found = true
			break
		}
	}

	if !found {
		newCard := dto.ProviderCardResponse{
			Provider: providerResp,
			Services: []dto.ServiceResponse{*serviceResp},
		}
		cards = append([]dto.ProviderCardResponse{newCard}, cards...)
	}

	if len(cards) > maxRecentServices {
		cards = cards[:maxRecentServices]
	}

	s.writeRecentServicesCards(ctx, cards)
}

// removeFromRecentServices removes a service from the home:recent:services cache by ID
func (s *ServiceService) removeFromRecentServices(ctx context.Context, id string) {
	cards := s.readRecentServicesCards(ctx)

	for i, card := range cards {
		for j, svc := range card.Services {
			if svc.ID == id {
				cards[i].Services = append(card.Services[:j], card.Services[j+1:]...)
				if len(cards[i].Services) == 0 {
					cards = append(cards[:i], cards[i+1:]...)
				}
				s.writeRecentServicesCards(ctx, cards)
				return
			}
		}
	}
}

// GetRecentServices returns recent services from the home:recent:services cache in provider card format
func (s *ServiceService) GetRecentServices(ctx context.Context) ([]dto.ProviderCardResponse, error) {
	cards := s.readRecentServicesCards(ctx)
	if cards == nil {
		return nil, nil
	}
	return cards, nil
}

// readRecentServicesCards reads the provider cards from Redis
func (s *ServiceService) readRecentServicesCards(ctx context.Context) []dto.ProviderCardResponse {
	data, err := s.redis.Get(ctx, cache.HomeRecentServicesKey())
	if err != nil || data == "" {
		return nil
	}
	var cards []dto.ProviderCardResponse
	if json.Unmarshal([]byte(data), &cards) != nil {
		return nil
	}
	return cards
}

// writeRecentServicesCards writes the provider cards to Redis
func (s *ServiceService) writeRecentServicesCards(ctx context.Context, cards []dto.ProviderCardResponse) {
	data, err := json.Marshal(cards)
	if err != nil {
		return
	}
	_ = s.redis.Set(ctx, cache.HomeRecentServicesKey(), string(data), 0)
}

// WarmRecentServices populates the home:recent:services cache on startup
func (s *ServiceService) WarmRecentServices(ctx context.Context) {
	filter := repository.ServiceProviderFilter{
		Limit: maxRecentServices,
	}

	providers, _, err := s.repo.ListProviders(ctx, filter)
	if err != nil {
		logger.Log.Warn("failed to warm recent services", "error", err.Error())
		return
	}

	if len(providers) == 0 {
		return
	}

	// Convert to provider card responses (same format as search endpoint)
	cards := make([]dto.ProviderCardResponse, 0, len(providers))
	for _, pw := range providers {
		cards = append(cards, s.ToProviderCardResponse(pw.Provider, pw.Services))
	}

	s.writeRecentServicesCards(ctx, cards)
}
