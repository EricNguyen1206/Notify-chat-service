package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"chat-service/configs/utils"
	"chat-service/configs/utils/ws"
	"chat-service/internal/models"
	"chat-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatHandler struct {
	chatService service.ChatService
	hub         *ws.Hub
}

func NewChatHandler(chatService service.ChatService) *ChatHandler {
	hub := ws.ChatHub
	go hub.Run()
	return &ChatHandler{
		chatService: chatService,
		hub:         hub,
	}
}

func (h *ChatHandler) CreateChat(c *gin.Context) {
	userID, err := utils.StringToUint(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chat, err := h.chatService.CreateChat(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast the message
	message := &models.ChatResponse{
		Type:       chat.Type,
		SenderID:   chat.SenderID,
		ReceiverID: chat.ReceiverID,
		ServerID:   chat.ServerID,
		ChannelID:  chat.ChannelID,
		Text:       chat.Text,
		URL:        chat.URL,
		FileName:   chat.FileName,
	}
	h.chatService.BroadcastMessage(h.hub, message)

	c.JSON(http.StatusCreated, chat)
}

func (h *ChatHandler) GetChat(c *gin.Context) {
	id, _ := utils.StringToUint(c.Param("id"))
	chat, err := h.chatService.GetChat(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chat)
}

func (h *ChatHandler) GetUserChats(c *gin.Context) {
	userID, err := utils.StringToUint(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	chats, err := h.chatService.GetUserChats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chats)
}

func (h *ChatHandler) GetChannelChats(c *gin.Context) {
	channelID, _ := utils.StringToUint(c.Param("channelId"))
	chats, err := h.chatService.GetChannelChats(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chats)
}

func (h *ChatHandler) GetFriendChats(c *gin.Context) {
	friendID, _ := utils.StringToUint(c.Param("friendId"))
	chats, err := h.chatService.GetFriendChats(c.Request.Context(), friendID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chats)
}

func (h *ChatHandler) DeleteChat(c *gin.Context) {
	userID, err := utils.StringToUint(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, _ := utils.StringToUint(c.Param("id"))
	if err := h.chatService.DeleteChat(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ChatHandler) WebSocket(c *gin.Context) {
	// Get user ID from query parameter
	userIDStr := c.Query("userId")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// serverID := c.Query("serverId")

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &ws.Client{
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: uint(userID),
	}

	ws.ChatHub.RegisterClient(client)

	go writePump(client)
	go readPump(client)
}

func readPump(client *ws.Client) {
	defer func() {
		ws.ChatHub.UnregisterClient(client)
		client.Conn.Close()
	}()

	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}

		var parsed struct {
			To      uint   `json:"to"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(msg, &parsed); err != nil {
			continue
		}

		ws.ChatHub.SendDirectMessage(ws.DirectMessage{
			FromUserID: client.UserID,
			ToUserID:   parsed.To,
			Content:    parsed.Content,
			Timestamp:  time.Now().UTC(),
		})
	}
}

func writePump(client *ws.Client) {
	defer client.Conn.Close()

	for {
		select {
		case msg, ok := <-client.Send:
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			client.Conn.WriteMessage(websocket.TextMessage, msg)
		}
	}
}

func (h *ChatHandler) RegisterRoutes(r *gin.RouterGroup) {
	chats := r.Group("/chats")
	{
		chats.POST("", h.CreateChat)
		chats.GET("/:id", h.GetChat)
		chats.GET("/user", h.GetUserChats)
		chats.GET("/channel/:channelId", h.GetChannelChats)
		chats.GET("/friend/:friendId", h.GetFriendChats)
		chats.DELETE("/:id", h.DeleteChat)
		chats.GET("/ws", h.WebSocket)
	}
}
