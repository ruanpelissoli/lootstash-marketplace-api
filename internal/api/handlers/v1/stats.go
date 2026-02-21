package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// StatsHandler handles marketplace statistics endpoints
type StatsHandler struct {
	service        *service.StatsService
	listingService *service.ListingService
	serviceService *service.ServiceService
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(service *service.StatsService, listingService *service.ListingService, serviceService *service.ServiceService) *StatsHandler {
	return &StatsHandler{
		service:        service,
		listingService: listingService,
		serviceService: serviceService,
	}
}

// GetMarketplaceStats returns marketplace statistics
// GET /api/v1/marketplace/stats
func (h *StatsHandler) GetMarketplaceStats(c *fiber.Ctx) error {
	stats, err := h.service.GetMarketplaceStats(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve marketplace statistics",
			Code:    500,
		})
	}

	return c.JSON(stats)
}

// GetRecentListings returns recently created listings from cache
// GET /api/v1/marketplace/recent
func (h *StatsHandler) GetRecentListings(c *fiber.Ctx) error {
	listings, err := h.listingService.GetRecentListings(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve recent listings",
			Code:    500,
		})
	}

	if listings == nil {
		listings = []dto.ListingCardResponse{}
	}

	return c.JSON(listings)
}

// GetRecentServices returns recently created services from cache
// GET /api/v1/marketplace/recent-services
func (h *StatsHandler) GetRecentServices(c *fiber.Ctx) error {
	services, err := h.serviceService.GetRecentServices(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve recent services",
			Code:    500,
		})
	}

	if services == nil {
		services = []dto.ProviderCardResponse{}
	}

	return c.JSON(services)
}