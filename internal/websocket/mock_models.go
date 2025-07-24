package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MockChat represents a chat message for testing purposes
type MockChat struct {
	ChannelID uint   `json:"channelId"`
	UserID    uint   `json:"userId"`
	Text      string `json:"text"`
	SentAt    string `json:"sentAt"`
}

// BroadcastMessage is a method to broadcast a mock chat message
// This allows us to test the BroadcastMessage functionality without depending on the actual models.Chat
func (h *Hub) BroadcastMockMessage(msg *MockChat) {
	// Convert the mock chat to JSON
	msgBytes, _ := json.Marshal(msg)

	// Start a goroutine for local broadcasting to avoid blocking
	go func() {
		// Use connection cache for immediate local broadcasting
		startTime := time.Now()
		successCount, failureCount := h.broadcastToLocalClients(msg.ChannelID, msgBytes)
		duration := time.Since(startTime)

		// Log performance metrics
		if h.Metrics != nil {
			h.Metrics.RecordBroadcastMetric(
				msg.ChannelID,
				duration,
				successCount,
				failureCount,
				len(msgBytes),
			)
		}
	}()

	// Also send to Redis for cross-instance distribution (non-blocking) if Redis is available
	if h.Redis != nil {
		go func() {
			// Try direct Redis publish
			ctx := context.Background()
			channelName := "channel:" + fmt.Sprintf("%d", msg.ChannelID)
			err := h.Redis.Publish(ctx, channelName, msgBytes).Err()
			if err != nil {
				// Handle Redis error
				if h.ErrorHandler != nil {
					h.ErrorHandler.HandleRedisError("publish", err)
				}
			}
		}()
	}
}
