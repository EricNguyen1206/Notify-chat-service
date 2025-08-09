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
// @Summary WebSocket connection for real-time messaging
// @Description Establish a WebSocket connection for real-time messaging with typed message support.
// @Description
// @Description ## Message Types
// @Description The WebSocket API uses typed messages with the following enum values:
// @Description
// @Description ### Connection Events
// @Description - `connection.connect` - Connection established (server -> client)
// @Description - `connection.disconnect` - Connection closed (server -> client)
// @Description - `connection.ping` - Ping message (client -> server)
// @Description - `connection.pong` - Pong response (server -> client)
// @Description
// @Description ### Channel Events
// @Description - `channel.join` - Join a channel (client -> server)
// @Description - `channel.leave` - Leave a channel (client -> server)
// @Description - `channel.message` - Send/receive channel message (bidirectional)
// @Description - `channel.typing` - Typing indicator (client -> server)
// @Description - `channel.stop_typing` - Stop typing indicator (client -> server)
// @Description
// @Description ### Channel Member Events
// @Description - `channel.member.join` - Member joined channel (server -> client)
// @Description - `channel.member.leave` - Member left channel (server -> client)
// @Description
// @Description ### User Events
// @Description - `user.status` - User status update (server -> client)
// @Description - `user.notification` - User notification (server -> client)
// @Description
// @Description ### Error Events
// @Description - `error` - Error message (server -> client)
// @Description
// @Description ## Message Format
// @Description All messages follow this JSON structure:
// @Description ```json
// @Description {
// @Description   "id": "unique-message-id",
// @Description   "type": "message-type-enum",
// @Description   "data": { /* type-specific data */ },
// @Description   "timestamp": 1234567890,
// @Description   "user_id": "user-id"
// @Description }
// @Description ```
// @Description
// @Description ## Example Messages
// @Description
// @Description ### Join Channel
// @Description ```json
// @Description {
// @Description   "id": "msg-123",
// @Description   "type": "channel.join",
// @Description   "data": { "channel_id": "channel-123" },
// @Description   "timestamp": 1234567890,
// @Description   "user_id": "user-456"
// @Description }
// @Description ```
// @Description
// @Description ### Send Message
// @Description ```json
// @Description {
// @Description   "id": "msg-456",
// @Description   "type": "channel.message",
// @Description   "data": {
// @Description     "channel_id": "channel-123",
// @Description     "text": "Hello world!",
// @Description     "url": null,
// @Description     "fileName": null
// @Description   },
// @Description   "timestamp": 1234567890,
// @Description   "user_id": "user-456"
// @Description }
// @Description ```
// @Description
// @Description ### Error Response
// @Description ```json
// @Description {
// @Description   "id": "error-789",
// @Description   "type": "error",
// @Description   "data": {
// @Description     "code": "INVALID_MESSAGE",
// @Description     "message": "Invalid message format"
// @Description   },
// @Description   "timestamp": 1234567890,
// @Description   "user_id": "user-456"
// @Description }
// @Description ```
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
