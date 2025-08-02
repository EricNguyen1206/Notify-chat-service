package handlers

import (
	"net/http"
	"strconv"

	"chat-service/internal/models"
	"chat-service/internal/repositories/postgres"
	"chat-service/internal/services"
	"chat-service/internal/websocket"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	channelService *services.ChannelService
	userService    services.UserService
	chatRepo       *postgres.ChatRepository
	hub            *websocket.Hub
}

func NewChatHandler(channelService *services.ChannelService, chatRepo *postgres.ChatRepository, hub *websocket.Hub) *ChatHandler {
	return &ChatHandler{channelService: channelService, chatRepo: chatRepo, hub: hub}
}

// GetChannelMessages godoc
// @Summary Get chat messages in a channel
// @Description Get all chat messages for a specific channel (paginated)
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Param limit query int false "Page size"
// @Param before query int false "Cursor for infinite scroll (timestamp)"
// @Success 200 {object} models.PaginatedChatResponse "Paginated chat messages"
// @Failure 401 {object} models.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} models.ErrorResponse "Channel not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @OperationId getChannelMessages
// @Router /messages/channel/{id} [get]
func (h *ChatHandler) GetChannelMessages(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	// Parse pagination params
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	var before *int64
	if b := c.Query("before"); b != "" {
		if parsed, err := strconv.ParseInt(b, 10, 64); err == nil {
			before = &parsed
		}
	}

	messages, err := h.channelService.GetChatMessagesByChannelWithPagination(uint(channelID), limit, before)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get messages",
			Details: err.Error(),
		})
		return
	}
	responses := make([]models.ChatResponse, 0, len(messages))
	var nextCursor *int64
	for _, m := range messages {
		responses = append(responses, models.ChatResponse{
			ID:         m.ID,
			SenderID:   m.SenderID,
			SenderName: m.Sender.Username,
			Text:       m.Text,
			URL:        m.URL,
			FileName:   m.FileName,
			CreatedAt:  m.CreatedAt,
			ChannelID:  &m.ChannelID,
		})
		unixTime := m.CreatedAt.Unix()
		nextCursor = &unixTime // last message timestamp for infinite scroll
	}
	paginated := models.PaginatedChatResponse{
		Items:      responses,
		Total:      len(responses),
		NextCursor: nextCursor,
	}
	c.JSON(http.StatusOK, paginated)
}

// SendMessage godoc
// @Summary Create a new chat message
// @Description Create a new chat message (channel or direct)
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ChatRequest true "Chat message data"
// @Success 201 {object} models.ChatResponse "Chat message created"
// @Failure 400 {object} models.ErrorResponse "Bad request - invalid input data"
// @Failure 401 {object} models.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @OperationId sendChatMessage
// @Router /messages/ [post]
func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var req models.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid input data",
			Details: err.Error(),
		})
		return
	}
	chat := &models.Chat{
		SenderID:  userID,
		ChannelID: *req.ChannelID,
		Text:      req.Text,
		URL:       req.URL,
		FileName:  req.FileName,
	}
	if err := h.chatRepo.Create(chat); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create chat message",
			Details: err.Error(),
		})
		return
	}

	// TODO: Implement WebSocket broadcasting for real-time messaging
	// The hub will handle Redis publishing internally when WebSocket clients are connected

	// Optionally preload sender for response
	// Preload sender to get name and avatar
	sender, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to fetch sender info",
			Details: err.Error(),
		})
		return
	}
	response := models.ChatResponse{
		ID:           chat.ID,
		SenderID:     chat.SenderID,
		SenderName:   sender.Username,
		SenderAvatar: sender.Avatar, // assuming Avatar field exists
		Text:         chat.Text,
		URL:          chat.URL,
		FileName:     chat.FileName,
		CreatedAt:    chat.CreatedAt,
		ChannelID:    &chat.ChannelID,
	}
	c.JSON(http.StatusCreated, response)
}
