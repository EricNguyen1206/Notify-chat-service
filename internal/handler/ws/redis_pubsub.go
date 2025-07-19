package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// UserPresenceUpdate represents a user presence update message
type UserPresenceUpdate struct {
	UserID     uint   `json:"userId"`
	ChannelID  uint   `json:"channelId"`
	Status     string `json:"status"` // "online", "offline", "join", "leave"
	Timestamp  int64  `json:"timestamp"`
	InstanceID string `json:"instanceId"`
}

// redisListener listens for messages from Redis pub/sub
func (h *Hub) redisListener() {
	ctx := context.Background()

	// Generate a unique instance ID to avoid processing our own messages
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano())

	// Subscribe to presence updates channel
	presencePubSub := h.Redis.Subscribe(ctx, "presence_updates")
	defer presencePubSub.Close()

	// Start a goroutine to handle presence updates
	go func() {
		presenceChannel := presencePubSub.Channel()
		for msg := range presenceChannel {
			h.handlePresenceUpdate([]byte(msg.Payload), instanceID)
		}
	}()

	// Subscribe to all channel messages using pattern matching
	channelPubSub := h.Redis.PSubscribe(ctx, "channel:*")
	defer channelPubSub.Close()

	// Process messages from Redis
	channel := channelPubSub.Channel()
	for msg := range channel {
		// Extract channel ID from Redis channel name
		channelName := msg.Channel
		if len(channelName) > 8 && channelName[:8] == "channel:" {
			channelIDStr := channelName[8:]
			channelIDUint, err := strconv.ParseUint(channelIDStr, 10, 64)
			if err != nil {
				log.Printf("Failed to parse channel ID from Redis message: %v", err)
				continue
			}
			channelID := uint(channelIDUint)

			// Use connection cache for efficient user lookup and concurrent broadcasting
			// Process Redis messages in a separate goroutine to avoid blocking the Redis listener
			go func(cid uint, payload []byte) {
				// Check if there are any online users in this channel before broadcasting
				onlineUsers := h.ConnectionCache.GetChannelUsers(cid)
				if len(onlineUsers) == 0 {
					log.Printf("Skipping Redis message for channel %d: no online users", cid)
					return
				}

				h.broadcastRedisMessageToLocalClients(cid, payload)
			}(channelID, []byte(msg.Payload))
		} else {
			log.Printf("Received message from unknown Redis channel: %s", msg.Channel)
		}
	}
}

// handlePresenceUpdate processes presence updates from other hub instances
// Updates the local connection cache to maintain consistency across instances
func (h *Hub) handlePresenceUpdate(payload []byte, instanceID string) {
	var update UserPresenceUpdate
	if err := json.Unmarshal(payload, &update); err != nil {
		log.Printf("Failed to parse presence update: %v", err)
		return
	}

	// Skip updates from this instance to avoid loops
	if update.InstanceID == instanceID {
		return
	}

	log.Printf("Received presence update: User %d in Channel %d is %s",
		update.UserID, update.ChannelID, update.Status)

	// Update local connection cache based on presence update
	switch update.Status {
	case "join":
		// If we have this user locally, update their channel subscription
		if h.ConnectionCache.IsUserConnected(update.UserID) {
			h.ConnectionCache.AddUserToChannel(update.UserID, update.ChannelID)
			log.Printf("Updated local cache: User %d joined channel %d (from remote instance)",
				update.UserID, update.ChannelID)
		}
	case "leave":
		// If we have this user locally, update their channel subscription
		if h.ConnectionCache.IsUserConnected(update.UserID) {
			h.ConnectionCache.RemoveUserFromChannel(update.UserID, update.ChannelID)
			log.Printf("Updated local cache: User %d left channel %d (from remote instance)",
				update.UserID, update.ChannelID)
		}
	}
}

// publishUserPresenceUpdate publishes user presence updates to Redis
func (h *Hub) publishUserPresenceUpdate(userID uint, channelID uint, status string) {
	// Generate a unique instance ID to avoid processing our own messages
	instanceID, _ := os.Hostname()

	update := UserPresenceUpdate{
		UserID:     userID,
		ChannelID:  channelID,
		Status:     status,
		Timestamp:  time.Now().Unix(),
		InstanceID: instanceID,
	}

	payload, err := json.Marshal(update)
	if err != nil {
		log.Printf("Failed to marshal presence update: %v", err)
		return
	}

	ctx := context.Background()
	err = h.Redis.Publish(ctx, "presence_updates", payload).Err()
	if err != nil {
		log.Printf("Failed to publish presence update: %v", err)
	}
}

// broadcastRedisMessageToLocalClients broadcasts a message received from Redis to local clients
func (h *Hub) broadcastRedisMessageToLocalClients(channelID uint, message []byte) {
	// Get all online users in the channel
	onlineUsers := h.ConnectionCache.GetChannelUsers(channelID)

	// If no online users, return early
	if len(onlineUsers) == 0 {
		return
	}

	// Send message to each client
	for _, userID := range onlineUsers {
		err := h.sendMessageToUser(userID, message)
		if err != nil {
			log.Printf("Failed to broadcast Redis message to user %d: %v", userID, err)
		}
	}
}
