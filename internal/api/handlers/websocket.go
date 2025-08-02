package handlers

import (
	"chat-service/internal/websocket"
	"log"

	"github.com/gin-gonic/gin"
)

type WSHandler struct {
	hub *websocket.Hub
}

func NewWSHandler(hub *websocket.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// HandleWebSocket godoc
// @Summary WebSocket connection
// @Description Establish a WebSocket connection for real-time messaging
// @Tags websocket
// @Accept json
// @Produce json
// @Param userId query string true "User ID for WebSocket connection"
// @Success 101 "Switching Protocols - WebSocket connection established"
// @Failure 400 {object} map[string]interface{} "Bad request - missing or invalid userId parameter"
// @Router /ws [get]
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	// Get userId from query parameters: /api/v1/ws?userId=1
	userID := c.Query("userId")
	if userID == "" {
		log.Printf("🔴 WebSocket connection failed: missing userId parameter")
		c.JSON(400, gin.H{"error": "userId parameter is required"})
		return
	}

	log.Printf("🟢 New WebSocket connection request from User ID: %s", userID)

	// Use the ServeWS function from websocket package for proper client creation and registration
	websocket.ServeWS(h.hub, c.Writer, c.Request, userID)
}
