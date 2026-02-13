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
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(service *service.StatsService, listingService *service.ListingService) *StatsHandler {
	return &StatsHandler{
		service:        service,
		listingService: listingService,
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
