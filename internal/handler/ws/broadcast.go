package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"chat-service/internal/models"
	modelws "chat-service/internal/models/ws"
)

// BroadcastMessage optimized method using connection cache for targeted delivery
func (h *Hub) BroadcastMessage(message interface{}) {
	// Type assertion to get the concrete type
	msg, ok := message.(*models.Chat)
	if !ok {
		log.Printf("Failed to broadcast message: invalid message type")
		return
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal chat message: %v", err)
		return
	}

	// Start a goroutine for local broadcasting to avoid blocking
	go func() {
		// Use connection cache for immediate local broadcasting
		startTime := time.Now()
		successCount, failureCount := h.broadcastToLocalClients(msg.ChannelID, msgBytes)
		duration := time.Since(startTime)

		// Log performance metrics
		log.Printf("Local broadcast to channel %d completed in %v: %d successful, %d failed",
			msg.ChannelID, duration, successCount, failureCount)
	}()

	// Also send to Redis for cross-instance distribution (non-blocking)
	go func() {
		// Try direct Redis publish first for lower latency
		ctx := context.Background()
		channelKey := "channel:" + string(msg.ChannelID)
		err := h.Redis.Publish(ctx, channelKey, msgBytes).Err()
		if err != nil {
			log.Printf("Failed to publish message to Redis channel %s: %v", channelKey, err)
		}
	}()
}

// broadcastToLocalClients sends a message to all clients subscribed to a specific channel
func (h *Hub) broadcastToLocalClients(channelID uint, message []byte) (int, int) {
	successCount := 0
	failureCount := 0

	// Get all online users in the channel
	onlineUsers := h.ConnectionCache.GetChannelUsers(channelID)

	// If no online users, return early
	if len(onlineUsers) == 0 {
		return 0, 0
	}

	// Send message to each client
	for _, userID := range onlineUsers {
		err := h.sendMessageToUser(userID, message)
		if err != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	return successCount, failureCount
}

// sendMessageToUser sends a message to a specific user
func (h *Hub) sendMessageToUser(userID uint, message []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Find the client for this user
	var targetClient *modelws.Client
	for client := range h.Clients {
		if client.ID == userID {
			targetClient = client
			break
		}
	}

	if targetClient == nil {
		return fmt.Errorf("user %d not found in active clients", userID)
	}

	// Send the message
	err := targetClient.Conn.WriteMessage(TextMessage, message)
	if err != nil {
		log.Printf("Failed to send message to user %d: %v", userID, err)
		return err
	}

	return nil
}
