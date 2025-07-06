package handler

import (
	"chat-service/configs"
	"chat-service/configs/utils"
	"chat-service/configs/utils/ws"
	"log"

	"github.com/gin-gonic/gin"
)

type WSHandler struct {
	hub *ws.Hub
}

func NewWSHandler(hub *ws.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// RegisterRoutes maps HTTP methods to handler functions
func (h *WSHandler) RegisterRoutes(r *gin.RouterGroup) {
	wsRoutes := r.Group("/ws")
	{
		wsRoutes.GET("", h.HandleWebSocket)
	}
}

func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	// Get userId from query parameters: /api/ws?userId=1
	userIDStr := c.Query("userId")
	if userIDStr == "" {
		log.Printf("🔴 WebSocket connection failed: missing userId parameter")
		c.JSON(400, gin.H{"error": "userId parameter is required"})
		return
	}

	userID, err := utils.StringToUint(userIDStr)
	if err != nil {
		log.Printf("🔴 WebSocket connection failed: invalid userId '%s': %v", userIDStr, err)
		c.JSON(400, gin.H{"error": "invalid userId parameter"})
		return
	}

	log.Printf("🟢 New WebSocket connection request from User ID: %d", userID)

	// Upgrade the connection to websocket protocol
	conn, err := configs.ConfigInstance.WSUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("🔴 WebSocket upgrade failed for User %d: %v", userID, err)
		return
	}
	log.Printf("✅ WebSocket upgrade success for User %d", userID)

	// Create new Client
	client := &ws.Client{
		ID:   userID,
		Conn: conn,
	}

	log.Printf("📝 Registering client %d to hub", userID)
	// Regist client to hub
	h.hub.Register <- client

	log.Printf("🚀 Starting message handler for client %d", userID)
	// Start handle message
	go client.WsHandleIncomingMessages(h.hub)
}
