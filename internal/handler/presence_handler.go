package handler

import (
	"chat-service/configs/utils/ws"
	"chat-service/internal/service"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PresenceHandler struct {
	presenceService *service.PresenceService
	friendService   *service.FriendService
	hub             *ws.Hub
}

func NewPresenceHandler(
	presenceService *service.PresenceService,
	friendService *service.FriendService,
	hub *ws.Hub,
) *PresenceHandler {
	return &PresenceHandler{
		presenceService: presenceService,
		friendService:   friendService,
		hub:             hub,
	}
}

// HandlePresenceConnection - Handle WebSocket connection
func (h *PresenceHandler) HandlePresenceConnection(c *gin.Context) {
	// Get user ID from query parameter
	userIDStr := c.Query("userId")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid user ID: %v", err)})
		return
	}
	ws, err := ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	// Register connection in hub for broadcasting
	h.hub.Register(uint(userID), ws)

	// Mark user as online
	if err := h.presenceService.SetOnline(uint(userID)); err != nil {
		log.Printf("Error marking user online: %v", err)
		return
	}

	// Subscribe to status updates
	statusUpdates, err := h.presenceService.SubscribeToStatusUpdates(c.Request.Context())
	if err != nil {
		log.Printf("Error subscribing to status updates: %v", err)
		return
	}

	// Heartbeat ticker
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case update := <-statusUpdates:
			h.hub.BroadcastUserStatus(update.UserID, update.Status)

		case <-heartbeat.C:
			if err := h.presenceService.SetOnline(uint(userID)); err != nil {
				log.Printf("Heartbeat update error: %v", err)
				return
			}

		case <-c.Request.Context().Done():
			h.hub.Unregister(uint(userID))
			h.presenceService.SetOffline(uint(userID))
			return
		}
	}
}

// Register routes
func (h *PresenceHandler) RegisterRoutes(r *gin.RouterGroup) {
	presence := r.Group("/presence")
	{
		presence.GET("", h.HandlePresenceConnection)
	}
}
