package v1

import (
	"database/sql"
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/middleware"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
)

// ChatHandler handles chat-related requests
type ChatHandler struct {
	service   *service.ChatService
	validator *validator.Validate
}

// NewChatHandler creates a new chat handler
func NewChatHandler(service *service.ChatService) *ChatHandler {
	return &ChatHandler{
		service:   service,
		validator: validator.New(),
	}
}

// GetByID handles GET /api/v1/chats/:id
func (h *ChatHandler) GetByID(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	id := c.Params("id")

	chat, err := h.service.GetByID(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Chat not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have access to this chat",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get chat",
			"error", err.Error(),
			"chat_id", id,
			"user_id", userID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get chat",
			Code:    500,
		})
	}

	return c.JSON(h.service.ToChatResponse(chat))
}

// GetMessages handles GET /api/v1/chats/:id/messages
func (h *ChatHandler) GetMessages(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	chatID := c.Params("id")

	var filter dto.MessagesFilterRequest
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
			Code:    400,
		})
	}

	messages, count, err := h.service.GetMessages(c.Context(), chatID, userID, filter.GetOffset(), filter.GetLimit())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Chat not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You are not a participant in this chat",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to get messages",
			"error", err.Error(),
			"user_id", userID,
			"chat_id", chatID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get messages",
			Code:    500,
		})
	}

	// Convert to response
	items := make([]dto.MessageResponse, 0, len(messages))
	for _, message := range messages {
		items = append(items, *h.service.ToMessageResponse(message))
	}

	return c.JSON(dto.NewPaginatedResponse(items, filter.Page, filter.GetLimit(), count))
}

// SendMessage handles POST /api/v1/chats/:id/messages
func (h *ChatHandler) SendMessage(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	chatID := c.Params("id")

	var req dto.SendChatMessageRequest
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

	message, err := h.service.SendMessage(c.Context(), chatID, userID, req.Content)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Chat not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You are not a participant in this chat",
				Code:    403,
			})
		}
		if errors.Is(err, service.ErrInvalidState) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "bad_request",
				Message: "Messaging is only available for active trades",
				Code:    400,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to send message",
			"error", err.Error(),
			"user_id", userID,
			"chat_id", chatID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to send message",
			Code:    500,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(h.service.ToMessageResponse(message))
}

// MarkRead handles POST /api/v1/chats/:id/read
// If no messageIds provided in body, marks all unread messages in the chat as read
func (h *ChatHandler) MarkRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	chatID := c.Params("id")

	var req dto.MarkChatMessagesReadRequest
	// Body is optional - if empty, we'll mark all messages as read
	_ = c.BodyParser(&req)

	// messageIDs can be empty - service will handle marking all as read
	if err := h.service.MarkMessagesAsRead(c.Context(), chatID, userID, req.MessageIDs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "not_found",
				Message: "Chat not found",
				Code:    404,
			})
		}
		if errors.Is(err, service.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "You are not a participant in this chat",
				Code:    403,
			})
		}
		logger.FromContext(c.UserContext()).Error("failed to mark messages as read",
			"error", err.Error(),
			"user_id", userID,
			"chat_id", chatID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to mark messages as read",
			Code:    500,
		})
	}

	return c.JSON(dto.SuccessResponse{Success: true})
}
