package ws

import (
	"chat-service/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// WebSocketConnWrapper wraps gorilla websocket.Conn to implement WebSocketConnection interface
type WebSocketConnWrapper struct {
	*websocket.Conn
}

// Hub manages all WebSocket clients and message broadcasting
// Acts as a central coordinator for WebSocket connections and Redis pub/sub integration
// Hub manages all WebSocket clients and message broadcasting
type Hub struct {
	Clients         map[*Client]bool     // Registry of all active WebSocket clients
	Register        chan *Client         // Channel for registering new clients
	Unregister      chan *Client         // Channel for unregistering/disconnecting clients
	Broadcast       chan ChannelMessage  // Channel for broadcasting messages to Redis
	Redis           *redis.Client        // Redis client for pub/sub functionality
	ConnectionCache *UserConnectionCache // Connection cache for efficient user presence management
	ErrorHandler    ErrorHandler         // Error handler for connection and broadcast errors
	Metrics         *ConnectionMetrics   // Performance metrics tracker
	MonitoringHooks *MonitoringHooks     // Monitoring hooks for event callbacks
	mu              sync.RWMutex         // Read-write mutex for concurrent map access
}

// ChannelMessage represents a message to be broadcasted to a specific channel
// Used for internal communication between hub components
type ChannelMessage struct {
	ChannelID uint   `json:"channelId"` // Target channel identifier
	Data      []byte `json:"data"`      // Serialized message data (JSON)
}

// WsNewHub creates and initializes a new Hub instance
// Returns a configured hub ready to handle WebSocket connections
func WsNewHub(redisClient *redis.Client) *Hub {
	hub := &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan ChannelMessage),
		Redis:      redisClient,
	}

	// Initialize the connection cache with reference to the hub
	hub.ConnectionCache = NewUserConnectionCache(hub)

	// Initialize error handler
	hub.ErrorHandler = NewErrorHandler(hub)

	// Initialize metrics tracker (keep 1000 recent metrics)
	hub.Metrics = NewConnectionMetrics(1000)

	// Initialize monitoring hooks
	hub.MonitoringHooks = NewMonitoringHooks()

	return hub
}

// WsRun starts the hub's main event loop in a goroutine
// Handles client registration, unregistration, and message broadcasting
// Also starts the Redis listener for cross-instance communication
func (h *Hub) WsRun() {
	// Start Redis message listener for cross-instance communication
	go h.wsRedisListener()

	// Start connection cache cleanup routine
	h.ConnectionCache.StartCleanupRoutine()
	log.Printf("Started connection cache cleanup routine")

	for {
		select {
		case client := <-h.Register:
			// Register new client - add to active clients map
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()

			// Add client to connection cache
			h.ConnectionCache.AddConnection(client)
			log.Printf("Client registered: %d", client.ID)

			// Publish online status to Redis for distributed cache consistency
			h.publishUserPresenceUpdate(client.ID, 0, "online")

		case client := <-h.Unregister:
			// Get client channels before removing from cache
			var clientChannels []uint
			if metadata, exists := h.ConnectionCache.GetConnectionMetadata(client.ID); exists {
				for channelID := range metadata.Channels {
					clientChannels = append(clientChannels, channelID)
				}
			}

			// Unregister client - remove from active clients and close connection
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				client.Conn.Close()
				log.Printf("Client unregistered: %d", client.ID)
			}
			h.mu.Unlock()

			// Remove client from connection cache
			h.ConnectionCache.RemoveConnection(client.ID)

			// Publish offline status to Redis for distributed cache consistency
			h.publishUserPresenceUpdate(client.ID, 0, "offline")

			// Also publish leave events for each channel the client was in
			for _, channelID := range clientChannels {
				h.publishUserPresenceUpdate(client.ID, channelID, "leave")
			}

		case msg := <-h.Broadcast:
			// Broadcast message to Redis for cross-instance distribution
			ctx := context.Background()
			if err := h.Redis.Publish(ctx, "channel:"+strconv.Itoa(int(msg.ChannelID)), msg.Data).Err(); err != nil {
				log.Printf("Redis publish error: %v", err)
			} else {
				log.Printf("Message published to Redis channel: channel:%d", msg.ChannelID)
			}

			// Use connection cache for optimized local broadcasting
			h.broadcastToLocalClients(msg.ChannelID, msg.Data)
		}
	}
}

// UserPresenceUpdate represents a user presence status change for cross-instance synchronization
type UserPresenceUpdate struct {
	UserID     uint   `json:"userId"`     // User identifier
	ChannelID  uint   `json:"channelId"`  // Channel identifier
	Status     string `json:"status"`     // Status: "online", "offline", "join", "leave"
	Timestamp  int64  `json:"timestamp"`  // Unix timestamp of the update
	InstanceID string `json:"instanceId"` // Unique identifier for the hub instance
}

// wsRedisListener listens for messages from Redis pub/sub channels
// Distributes messages to all clients subscribed to the respective channels using connection cache
// Enables cross-instance communication when multiple hub instances are running
func (h *Hub) wsRedisListener() {
	// Generate a unique instance ID for this hub
	instanceID := generateInstanceID()
	log.Printf("Starting Redis listener for hub instance %s", instanceID)

	// Context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscribe to all channel messages using wildcard pattern
	// Also subscribe to presence updates for distributed cache consistency
	pubsub := h.Redis.Subscribe(ctx, "channel:*", "presence:update")
	defer pubsub.Close()

	// Handle Redis connection errors
	go func() {
		for {
			if err := pubsub.Ping(ctx); err != nil {
				log.Printf("Redis pubsub ping error: %v", err)
				// Try to resubscribe
				time.Sleep(5 * time.Second)
				newPubsub := h.Redis.Subscribe(ctx, "channel:*", "presence:update")
				pubsub = newPubsub
			}
			time.Sleep(30 * time.Second)
		}
	}()

	ch := pubsub.Channel()
	for msg := range ch {
		// Handle different Redis channel types
		if msg.Channel == "presence:update" {
			// Handle presence update for distributed cache consistency
			h.handlePresenceUpdate([]byte(msg.Payload), instanceID)
			continue
		}

		// Handle regular channel messages
		if len(msg.Channel) > 8 && msg.Channel[:8] == "channel:" {
			// Extract channelID from Redis channel name (e.g., "channel:123" -> "123")
			channelIDStr := msg.Channel[8:]
			log.Printf("Received message from Redis channel: %s", msg.Channel)

			// Parse channel ID
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
				onlineUsers := h.ConnectionCache.GetOnlineUsersInChannel(cid)
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
		if h.ConnectionCache.IsUserOnline(update.UserID) {
			h.ConnectionCache.AddUserToChannel(update.UserID, update.ChannelID)
			log.Printf("Updated local cache: User %d joined channel %d (from remote instance)",
				update.UserID, update.ChannelID)
		}
	case "leave":
		// If we have this user locally, update their channel subscription
		if h.ConnectionCache.IsUserOnline(update.UserID) {
			h.ConnectionCache.RemoveUserFromChannel(update.UserID, update.ChannelID)
			log.Printf("Updated local cache: User %d left channel %d (from remote instance)",
				update.UserID, update.ChannelID)
		}
	case "offline":
		// If we have this user locally, remove them from the cache
		// This is a safety measure in case a user connects to multiple instances
		// and disconnects from one but not the other
		if h.ConnectionCache.IsUserOnline(update.UserID) {
			// Only remove if the timestamp is newer than our last activity
			metadata, exists := h.ConnectionCache.GetConnectionMetadata(update.UserID)
			if exists {
				lastActivityUnix := metadata.LastActivity.Unix()
				if update.Timestamp > lastActivityUnix {
					h.ConnectionCache.RemoveConnection(update.UserID)
					log.Printf("Removed user %d from local cache due to remote offline status",
						update.UserID)
				}
			}
		}
	}
}

// publishUserPresenceUpdate publishes a user presence update to Redis
// This allows other hub instances to maintain consistent connection caches
func (h *Hub) publishUserPresenceUpdate(userID uint, channelID uint, status string) {
	// Generate a unique instance ID for this hub
	instanceID := generateInstanceID()

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
	if err := h.Redis.Publish(ctx, "presence:update", payload).Err(); err != nil {
		log.Printf("Failed to publish presence update: %v", err)
	} else {
		log.Printf("Published presence update: User %d in Channel %d is %s",
			userID, channelID, status)
	}
}

// generateInstanceID creates a unique identifier for this hub instance
func generateInstanceID() string {
	// Use a combination of hostname and timestamp for uniqueness
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano())
}

// broadcastRedisMessageToLocalClients handles Redis message distribution using connection cache
// Implements concurrent message delivery for Redis messages
func (h *Hub) broadcastRedisMessageToLocalClients(channelID uint, message []byte) {
	// Get online users in the channel from connection cache
	onlineUsers := h.ConnectionCache.GetOnlineUsersInChannel(channelID)

	if len(onlineUsers) == 0 {
		log.Printf("No online users in channel %d for Redis message", channelID)
		return
	}

	log.Printf("Distributing Redis message to %d online users in channel %d", len(onlineUsers), channelID)

	// Use goroutines for concurrent message delivery
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	failureCount := 0

	// Set a reasonable timeout for the entire broadcast operation
	const broadcastTimeout = 5 * time.Second
	done := make(chan struct{})

	go func() {
		// Wait for all goroutines to complete
		wg.Wait()
		close(done)
	}()

	// Track broadcast start time for performance metrics
	startTime := time.Now()

	for _, userID := range onlineUsers {
		wg.Add(1)
		go func(uid uint) {
			defer wg.Done()

			// Get the client connection from cache
			client, exists := h.ConnectionCache.GetConnection(uid)
			if !exists {
				log.Printf("Client connection not found for user %d (Redis message)", uid)
				return
			}

			// Send Redis message with connection failure handling
			client.mu.Lock()
			err := client.Conn.WriteMessage(websocket.TextMessage, message)
			client.mu.Unlock()

			if err != nil {
				log.Printf("Redis message write error for user %d: %v", uid, err)

				// Use error handler if available
				if h.ErrorHandler != nil {
					h.ErrorHandler.HandleBroadcastError(channelID, uid, err)
				}

				// Handle connection failure by unregistering the client (non-blocking)
				select {
				case h.Unregister <- client:
				default:
					// If unregister channel is full, just log and continue
					log.Printf("Failed to unregister client %d: channel full", uid)
					// Force remove from connection cache as fallback
					h.ConnectionCache.RemoveConnection(uid)
				}
				mu.Lock()
				failureCount++
				mu.Unlock()
			} else {
				log.Printf("Redis message sent to user %d in channel %d", uid, channelID)
				mu.Lock()
				successCount++
				mu.Unlock()
				// Update last activity in connection cache
				h.ConnectionCache.UpdateLastActivity(uid)
			}
		}(userID)
	}

	// Wait for completion or timeout
	select {
	case <-done:
		// All goroutines completed normally
	case <-time.After(broadcastTimeout):
		log.Printf("Warning: Redis broadcast to channel %d timed out after %v", channelID, broadcastTimeout)

		// Log timeout event if error handler is available
		if h.ErrorHandler != nil {
			h.ErrorHandler.LogEvent(RedisError, SeverityWarning,
				fmt.Sprintf("Redis broadcast to channel %d timed out after %v", channelID, broadcastTimeout), nil)
		}
	}

	// Calculate duration and record metrics
	duration := time.Since(startTime)
	log.Printf("Redis message distribution completed in %v: %d successful, %d failed in channel %d",
		duration, successCount, failureCount, channelID)

	// Record metrics if available
	if h.Metrics != nil {
		h.Metrics.RecordMetric(PerformanceMetric{
			Type:         MetricRedis,
			Operation:    "redis_broadcast",
			Duration:     duration,
			SuccessCount: successCount,
			FailureCount: failureCount,
			ChannelID:    channelID,
			UserCount:    len(onlineUsers),
			MessageSize:  len(message),
			Timestamp:    time.Now(),
		})
	}
}

// WsAddChannel subscribes a client to a specific channel
// Thread-safe operation that adds the channel to the client's subscription list
// Updates connection cache to maintain consistent channel subscription mapping
func (c *Client) WsAddChannel(channelID uint, hub *Hub) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Initialize channels map if not already done
	if c.Channels == nil {
		c.Channels = make(map[uint]bool)
	}

	// Skip if client is already subscribed to this channel
	if _, alreadySubscribed := c.Channels[channelID]; alreadySubscribed {
		log.Printf("Client %d already subscribed to channel %d, skipping", c.ID, channelID)
		return
	}

	// Add channel to client's subscription list
	c.Channels[channelID] = true

	// Update connection cache
	if hub != nil && hub.ConnectionCache != nil {
		// Ensure user is registered in connection cache before adding to channel
		if !hub.ConnectionCache.IsUserOnline(c.ID) {
			log.Printf("Client %d not registered in connection cache, registering now", c.ID)
			hub.ConnectionCache.AddConnection(c)
		}

		// Add user to channel in connection cache
		hub.ConnectionCache.AddUserToChannel(c.ID, channelID)

		// Update last activity timestamp
		hub.ConnectionCache.UpdateLastActivity(c.ID)

		// Publish join event to Redis for distributed cache consistency
		hub.publishUserPresenceUpdate(c.ID, channelID, "join")

		// Log channel users after addition for debugging
		onlineUsers := hub.ConnectionCache.GetOnlineUsersInChannel(channelID)
		log.Printf("Channel %d now has %d users after client %d subscribed",
			channelID, len(onlineUsers), c.ID)
	}

	log.Printf("Client %d subscribed to channel %d", c.ID, channelID)
}

// WsRemoveChannel unsubscribes a client from a specific channel
// Thread-safe operation that removes the channel from the client's subscription list
// Updates connection cache to maintain consistent channel subscription mapping
func (c *Client) WsRemoveChannel(channelID uint, hub *Hub) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Skip if client is not subscribed to this channel
	if _, subscribed := c.Channels[channelID]; !subscribed {
		log.Printf("Client %d not subscribed to channel %d, skipping", c.ID, channelID)
		return
	}

	// Remove channel from client's subscription list
	delete(c.Channels, channelID)

	// Update connection cache
	if hub != nil && hub.ConnectionCache != nil {
		// Remove user from channel in connection cache
		hub.ConnectionCache.RemoveUserFromChannel(c.ID, channelID)

		// Update last activity timestamp
		hub.ConnectionCache.UpdateLastActivity(c.ID)

		// Publish leave event to Redis for distributed cache consistency
		hub.publishUserPresenceUpdate(c.ID, channelID, "leave")

		// Log channel users after removal for debugging
		onlineUsers := hub.ConnectionCache.GetOnlineUsersInChannel(channelID)
		log.Printf("Channel %d now has %d users after client %d unsubscribed",
			channelID, len(onlineUsers), c.ID)
	}

	log.Printf("Client %d unsubscribed from channel %d", c.ID, channelID)
}

// WsHandleIncomingMessages processes incoming WebSocket messages from a client
// Handles different message types: join, leave, and message actions
// Runs in a separate goroutine for each client connection
func (c *Client) WsHandleIncomingMessages(hub *Hub) {
	// Ensure client is unregistered and connection is closed when function exits
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	log.Printf("🟢 Client %d: Started message handler", c.ID)

	for {
		// Read message from WebSocket connection
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			// Log unexpected close errors but handle normal disconnections gracefully
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("🔴 Client %d: Read error: %v", c.ID, err)
			} else {
				log.Printf("🟡 Client %d: Connection closed normally", c.ID)
			}
			break
		}

		// Log raw message received
		log.Printf("📥 Client %d: Received raw message: %s", c.ID, string(message))

		// Parse incoming JSON message
		var msgData struct {
			Action    string `json:"action"`    // Message action: "join", "leave", "message", or "heartbeat"
			Type      string `json:"type"`      // Message type (for system messages like heartbeat)
			ChannelID uint   `json:"channelId"` // Target channel identifier
			Text      string `json:"text"`      // Message text (for "message" action)
		}

		if err := json.Unmarshal(message, &msgData); err != nil {
			log.Printf("🔴 Client %d: JSON decode error: %v", c.ID, err)
			log.Printf("🔴 Client %d: Raw message that failed to parse: %s", c.ID, string(message))
			continue
		}

		// Update last activity timestamp for any message received
		if hub.ConnectionCache != nil {
			hub.ConnectionCache.UpdateLastActivity(c.ID)
		}

		// Handle heartbeat responses separately
		if msgData.Type == "heartbeat-response" {
			// Update heartbeat count in connection cache
			if hub.ConnectionCache != nil {
				hub.ConnectionCache.UpdateHeartbeat(c.ID)
			}
			continue
		}

		log.Printf("✅ Client %d: JSON decoded successfully - Action: %s, ChannelID: %d, Text: %s",
			c.ID, msgData.Action, msgData.ChannelID, msgData.Text)

		// Handle different message actions
		switch msgData.Action {
		case "join":
			// Subscribe client to the specified channel
			log.Printf("🟢 Client %d: Attempting to join channel %d", c.ID, msgData.ChannelID)
			c.WsAddChannel(msgData.ChannelID, hub)
			log.Printf("✅ Client %d: Successfully joined channel %d", c.ID, msgData.ChannelID)

		case "leave":
			// Unsubscribe client from the specified channel
			log.Printf("🟡 Client %d: Attempting to leave channel %d", c.ID, msgData.ChannelID)
			c.WsRemoveChannel(msgData.ChannelID, hub)
			log.Printf("✅ Client %d: Successfully left channel %d", c.ID, msgData.ChannelID)

		case "message":
			// Create a complete message structure with metadata
			log.Printf("💬 Client %d: Sending message to channel %d: %s", c.ID, msgData.ChannelID, msgData.Text)

			fullMsg := struct {
				ChannelID uint   `json:"channelId"` // Target channel
				UserID    uint   `json:"userId"`    // Sender's user ID
				Text      string `json:"text"`      // Message content
				SentAt    string `json:"sentAt"`    // Timestamp in RFC3339 format
			}{
				ChannelID: uint(msgData.ChannelID),
				UserID:    c.ID,
				Text:      msgData.Text,
				SentAt:    time.Now().Format(time.RFC3339),
			}

			// Serialize and broadcast the message
			msgBytes, _ := json.Marshal(fullMsg)
			log.Printf("📤 Client %d: Broadcasting message to channel %d: %s", c.ID, msgData.ChannelID, string(msgBytes))

			hub.Broadcast <- ChannelMessage{
				ChannelID: uint(msgData.ChannelID),
				Data:      msgBytes,
			}
			log.Printf("✅ Client %d: Message queued for broadcasting to channel %d", c.ID, msgData.ChannelID)

			// Reset heartbeat count since user is active
			if hub.ConnectionCache != nil {
				hub.ConnectionCache.ResetHeartbeat(c.ID)
			}

		case "heartbeat":
			// Client sent a heartbeat, respond with a heartbeat response
			heartbeatResponse := []byte(`{"type":"heartbeat-response"}`)
			c.mu.Lock()
			err := c.Conn.WriteMessage(websocket.TextMessage, heartbeatResponse)
			c.mu.Unlock()

			if err != nil {
				log.Printf("🔴 Client %d: Failed to send heartbeat response: %v", c.ID, err)
			} else {
				log.Printf("💓 Client %d: Heartbeat response sent", c.ID)
			}

			// Update last activity
			if hub.ConnectionCache != nil {
				hub.ConnectionCache.UpdateLastActivity(c.ID)
			}

		default:
			log.Printf("⚠️ Client %d: Unknown action '%s' received", c.ID, msgData.Action)
		}
	}

	log.Printf("🔴 Client %d: Message handler stopped", c.ID)
}

// broadcastToLocalClients uses connection cache for optimized local message broadcasting
// Implements concurrent message delivery using goroutines for better performance
// Returns counts of successful and failed message deliveries
func (h *Hub) broadcastToLocalClients(channelID uint, message []byte) (int, int) {
	// Get online users in the channel from connection cache
	onlineUsers := h.ConnectionCache.GetOnlineUsersInChannel(channelID)

	if len(onlineUsers) == 0 {
		log.Printf("No online users in channel %d", channelID)
		return 0, 0
	}

	log.Printf("Broadcasting to %d online users in channel %d", len(onlineUsers), channelID)

	// Use goroutines for concurrent message delivery
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	failureCount := 0

	// Set a reasonable timeout for the entire broadcast operation
	const broadcastTimeout = 5 * time.Second
	done := make(chan struct{})

	go func() {
		// Wait for all goroutines to complete
		wg.Wait()
		close(done)
	}()

	// Track broadcast start time for performance metrics
	startTime := time.Now()

	// Launch a goroutine for each user
	for _, userID := range onlineUsers {
		wg.Add(1)
		go func(uid uint) {
			defer wg.Done()

			// Get the client connection from cache
			client, exists := h.ConnectionCache.GetConnection(uid)
			if !exists {
				log.Printf("Client connection not found for user %d", uid)
				return
			}

			// Send message with connection failure handling
			client.mu.Lock()
			err := client.Conn.WriteMessage(websocket.TextMessage, message)
			client.mu.Unlock()

			if err != nil {
				log.Printf("Write error for user %d: %v", uid, err)

				// Use error handler if available
				if h.ErrorHandler != nil {
					h.ErrorHandler.HandleBroadcastError(channelID, uid, err)
				}

				// Handle connection failure by unregistering the client (non-blocking)
				select {
				case h.Unregister <- client:
				default:
					// If unregister channel is full, just log and continue
					log.Printf("Failed to unregister client %d: channel full", uid)
					// Force remove from connection cache as fallback
					h.ConnectionCache.RemoveConnection(uid)
				}
				mu.Lock()
				failureCount++
				mu.Unlock()
			} else {
				log.Printf("Message sent to user %d in channel %d", uid, channelID)
				mu.Lock()
				successCount++
				mu.Unlock()
				// Update last activity in connection cache
				h.ConnectionCache.UpdateLastActivity(uid)
			}
		}(userID)
	}

	// Wait for completion or timeout
	select {
	case <-done:
		// All goroutines completed normally
	case <-time.After(broadcastTimeout):
		log.Printf("Warning: Broadcast to channel %d timed out after %v", channelID, broadcastTimeout)

		// Log timeout event if error handler is available
		if h.ErrorHandler != nil {
			h.ErrorHandler.LogEvent(BroadcastError, SeverityWarning,
				fmt.Sprintf("Broadcast to channel %d timed out after %v", channelID, broadcastTimeout), nil)
		}
	}

	// Calculate duration and record metrics
	duration := time.Since(startTime)
	log.Printf("Message broadcast completed: %d successful, %d failed in channel %d (duration: %v)",
		successCount, failureCount, channelID, duration)

	// Record metrics if available
	if h.Metrics != nil {
		h.Metrics.RecordBroadcastMetric(
			channelID,
			duration,
			successCount,
			failureCount,
			len(message),
		)
	}

	return successCount, failureCount
}

// BroadcastMessage optimized method using connection cache for targeted delivery
func (h *Hub) BroadcastMessage(msg *models.Chat) {
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
		err := h.Redis.Publish(ctx, "channel:"+strconv.Itoa(int(msg.ChannelID)), msgBytes).Err()
		if err != nil {
			// Handle Redis error
			log.Printf("Direct Redis publish failed: %v, falling back to channel", err)

			// Create error handler if needed
			errorHandler := NewRedisErrorHandler(h)
			errorHandler.HandlePublishError(msg.ChannelID, err)

			// Fall back to channel method
			select {
			case h.Broadcast <- ChannelMessage{
				ChannelID: msg.ChannelID,
				Data:      msgBytes,
			}:
				log.Printf("Message queued for Redis distribution to channel %d", msg.ChannelID)
			default:
				// If channel is full, log warning but don't block
				log.Printf("Warning: Redis broadcast channel full, skipping cross-instance distribution for channel %d", msg.ChannelID)
			}
		} else {
			log.Printf("Direct Redis publish successful for channel %d", msg.ChannelID)
		}
	}()
}
