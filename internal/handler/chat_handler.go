package handler

import (
	"net/http"
	"strconv"

	"chat-service/configs/middleware"
	"chat-service/configs/utils/ws"
	"chat-service/internal/models"
	"chat-service/internal/repository"
	"chat-service/internal/service"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	channelService *service.ChannelService
	chatRepo       *repository.ChatRepository
	hub            *ws.Hub
}

func NewChatHandler(channelService *service.ChannelService, chatRepo *repository.ChatRepository, hub *ws.Hub) *ChatHandler {
	return &ChatHandler{channelService: channelService, chatRepo: chatRepo, hub: hub}
}

// RegisterRoutes maps HTTP methods to handler functions
func (h *ChatHandler) RegisterRoutes(r *gin.RouterGroup) {
	chats := r.Group("/chats")
	{
		chats.Use(middleware.Auth())
		chats.GET("/channel/:id", h.GetChannelMessages)
		chats.POST("/", h.CreateChatMessage)
	}
}

// GetChannelMessages godoc
// @Summary Get chat messages in a channel
// @Description Get all chat messages for a specific channel
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Success 200 {array} models.ChatResponse "List of chat messages"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 404 {object} map[string]interface{} "Channel not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /chats/channel/{id} [get]
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}
	responses := make([]models.ChatResponse, 0, len(messages))
	for _, m := range messages {
		responses = append(responses, models.ChatResponse{
			ID:         m.ID,
			Type:       m.Type,
			SenderID:   m.SenderID,
			SenderName: m.Sender.Username,
			Text:       m.Text,
			URL:        m.URL,
			FileName:   m.FileName,
			CreatedAt:  m.CreatedAt,
			ChannelID:  &m.ChannelID,
		})
	}
	c.JSON(http.StatusOK, responses)
}

// CreateChatMessage godoc
// @Summary Create a new chat message
// @Description Create a new chat message (channel or direct)
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ChatRequest true "Chat message data"
// @Success 201 {object} models.ChatResponse "Chat message created"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid input data"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /chats/ [post]
func (h *ChatHandler) CreateChatMessage(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var req models.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	chat := &models.Chat{
		SenderID:  userID,
		Type:      req.Type,
		ChannelID: *req.ChannelID,
		Text:      req.Text,
		URL:       req.URL,
		FileName:  req.FileName,
	}
	if err := h.chatRepo.Create(chat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat message"})
		return
	}

	// Send message to WebSocket clients
	h.hub.BroadcastMessage(chat)

	// Optionally preload sender for response
	response := models.ChatResponse{
		ID:         chat.ID,
		Type:       chat.Type,
		SenderID:   chat.SenderID,
		SenderName: "", // You may want to fetch sender name if needed
		Text:       chat.Text,
		URL:        chat.URL,
		FileName:   chat.FileName,
		CreatedAt:  chat.CreatedAt,
		ChannelID:  &chat.ChannelID,
	}
	c.JSON(http.StatusCreated, response)
}
