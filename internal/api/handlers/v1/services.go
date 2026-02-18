package v1

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/middleware"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// ServiceHandler handles service-related requests
type ServiceHandler struct {
	service   *service.ServiceService
	validator *validator.Validate
}

// NewServiceHandler creates a new service handler
func NewServiceHandler(service *service.ServiceService) *ServiceHandler {
	return &ServiceHandler{
		service:   service,
		validator: validator.New(),
	}
}

// ListProviders handles GET /api/v1/services
func (h *ServiceHandler) ListProviders(c *fiber.Ctx) error {
	var filter dto.ServiceProvidersFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	var serviceTypes []string
	if filter.ServiceType != "" {
		serviceTypes = strings.Split(filter.ServiceType, ",")
	}

	repoFilter := repository.ServiceProviderFilter{
		ServiceType: serviceTypes,
		Game:        filter.Game,
		Ladder:      filter.Ladder,
		Hardcore:    filter.Hardcore,
		IsNonRotw:   filter.IsNonRotw,
		Platforms:   parsePlatformsFromString(filter.Platforms),
		Region:      filter.Region,
		Offset:      filter.GetOffset(),
		Limit:       filter.GetLimit(),
	}

	providers, count, err := h.service.ListProviders(c.Context(), repoFilter)
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to list service providers",
			"error", err.Error(),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list service providers",
			Code:    500,
		})
	}

	return c.JSON(dto.NewPaginatedResponse(providers, filter.Page, filter.GetLimit(), count))
}

// GetProviderDetail handles GET /api/v1/services/providers/:id
func (h *ServiceHandler) GetProviderDetail(c *fiber.Ctx) error {
	providerID := c.Params("id")
	if providerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Provider ID is required",
			Code:    400,
		})
	}

	card, err := h.service.GetProviderDetail(c.Context(), providerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, service.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Provider not found",
				Code:    404,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get provider detail",
			"error", err.Error(),
			"provider_id", providerID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get provider detail",
			Code:    500,
		})
	}

	return c.JSON(card)
}

// Create handles POST /api/v1/services
func (h *ServiceHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.CreateServiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	if err := h.validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
			Code:    400,
		})
	}

	svc, err := h.service.Create(c.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{
				Error:   "already_exists",
				Message: "You already have a service of this type for this game",
				Code:    409,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to create service",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create service",
			Code:    500,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(h.service.ToServiceResponse(svc))
}

// Update handles PATCH /api/v1/services/:id
func (h *ServiceHandler) Update(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	var req dto.UpdateServiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	if err := h.validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
			Code:    400,
		})
	}

	svc, err := h.service.Update(c.Context(), id, userID, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Service not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You can only update your own services",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to update service",
			"error", err.Error(),
			"service_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update service",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToServiceResponse(svc))
}

// Delete handles DELETE /api/v1/services/:id
func (h *ServiceHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	err := h.service.Delete(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Service not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You can only delete your own services",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to delete service",
			"error", err.Error(),
			"service_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete service",
			Code:    500,
		})
	}

	return c.JSON(dto.SuccessResponse{Success: true, Message: "Service cancelled"})
}

// ListMy handles GET /api/v1/my/services
func (h *ServiceHandler) ListMy(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var filter dto.Pagination
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	services, count, err := h.service.ListMyServices(c.Context(), userID, filter.GetOffset(), filter.GetLimit())
	if err != nil {
		logger.FromContext(c.UserContext()).Error("failed to list my services",
			"error", err.Error(),
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list services",
			Code:    500,
		})
	}

	items := make([]dto.ServiceResponse, 0, len(services))
	for _, svc := range services {
		items = append(items, *h.service.ToServiceResponse(svc))
	}

	return c.JSON(dto.NewPaginatedResponse(items, filter.Page, filter.GetLimit(), count))
}
