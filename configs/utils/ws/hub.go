package ws

import (
	"chat-service/internal/models"
	"context"
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// Hub manages all WebSocket clients and message broadcasting
// Acts as a central coordinator for WebSocket connections and Redis pub/sub integration
type Hub struct {
	Clients    map[*Client]bool    // Registry of all active WebSocket clients
	Register   chan *Client        // Channel for registering new clients
	Unregister chan *Client        // Channel for unregistering/disconnecting clients
	Broadcast  chan ChannelMessage // Channel for broadcasting messages to Redis
	Redis      *redis.Client       // Redis client for pub/sub functionality
	mu         sync.RWMutex        // Read-write mutex for concurrent map access
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
	return &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan ChannelMessage),
		Redis:      redisClient,
	}
}

// WsRun starts the hub's main event loop in a goroutine
// Handles client registration, unregistration, and message broadcasting
// Also starts the Redis listener for cross-instance communication
func (h *Hub) WsRun() {
	// Start Redis message listener for cross-instance communication
	go h.wsRedisListener()

	for {
		select {
		case client := <-h.Register:
			// Register new client - add to active clients map
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered: %d", client.ID)

		case client := <-h.Unregister:
			// Unregister client - remove from active clients and close connection
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				client.Conn.Close()
				log.Printf("Client unregistered: %d", client.ID)
			}
			h.mu.Unlock()

		case msg := <-h.Broadcast:
			// Broadcast message to Redis for cross-instance distribution
			ctx := context.Background()
			if err := h.Redis.Publish(ctx, "channel:"+strconv.Itoa(int(msg.ChannelID)), msg.Data).Err(); err != nil {
				log.Printf("Redis publish error: %v", err)
			} else {
				log.Printf("Message published to Redis channel: channel:%s", msg.ChannelID)
			}

			// Also broadcast directly to local clients for immediate delivery
			h.mu.RLock()
			clientCount := 0
			for client := range h.Clients {
				client.mu.Lock()
				// Check if client is subscribed to this channel
				if _, ok := client.Channels[msg.ChannelID]; ok {
					// Send message to client via WebSocket
					if err := client.Conn.WriteMessage(websocket.TextMessage, msg.Data); err != nil {
						log.Printf("Write error: %v", err)
						// Handle write error by unregistering the client
						h.Unregister <- client
					} else {
						clientCount++
						log.Printf("Message sent to client %d in channel %s", client.ID, msg.ChannelID)
					}
				}
				client.mu.Unlock()
			}
			h.mu.RUnlock()
			log.Printf("Message broadcasted to %d local clients in channel %s", clientCount, msg.ChannelID)
		}
	}
}

// wsRedisListener listens for messages from Redis pub/sub channels
// Distributes messages to all clients subscribed to the respective channels
// Enables cross-instance communication when multiple hub instances are running
func (h *Hub) wsRedisListener() {
	// Subscribe to all channel messages using wildcard pattern
	pubsub := h.Redis.Subscribe(context.Background(), "channel:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		// Extract channelID from Redis channel name (e.g., "channel:123" -> "123")
		channelID := msg.Channel[6:]
		log.Printf("Received message from Redis channel: %s", msg.Channel)

		// Iterate through all active clients and send message to subscribed ones
		h.mu.RLock()
		clientCount := 0
		for client := range h.Clients {
			client.mu.Lock()
			// Check if client is subscribed to this channel
			channelIDUint, err := strconv.ParseUint(channelID, 10, 64)
			if err != nil {
				log.Printf("Failed to parse channel ID from Redis message: %v", err)
				continue
			}
			if _, ok := client.Channels[uint(channelIDUint)]; ok {
				// Send message to client via WebSocket
				if err := client.Conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
					log.Printf("Write error: %v", err)
					// Handle write error by unregistering the client
					h.Unregister <- client
				} else {
					clientCount++
					log.Printf("Redis message sent to client %d in channel %s", client.ID, channelID)
				}
			}
			client.mu.Unlock()
		}
		h.mu.RUnlock()
		log.Printf("Redis message distributed to %d local clients in channel %s", clientCount, channelID)
	}
}

// WsAddChannel subscribes a client to a specific channel
// Thread-safe operation that adds the channel to the client's subscription list
func (c *Client) WsAddChannel(channelID uint) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Initialize channels map if not already done
	if c.Channels == nil {
		c.Channels = make(map[uint]bool)
	}
	c.Channels[channelID] = true
	log.Printf("Client %d subscribed to channel %s", c.ID, channelID)
}

// WsRemoveChannel unsubscribes a client from a specific channel
// Thread-safe operation that removes the channel from the client's subscription list
func (c *Client) WsRemoveChannel(channelID uint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Channels, channelID)
	log.Printf("Client %d unsubscribed from channel %s", c.ID, channelID)
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
			Action    string `json:"action"`    // Message action: "join", "leave", or "message"
			ChannelID uint   `json:"channelId"` // Target channel identifier
			Text      string `json:"text"`      // Message text (for "message" action)
		}

		if err := json.Unmarshal(message, &msgData); err != nil {
			log.Printf("🔴 Client %d: JSON decode error: %v", c.ID, err)
			log.Printf("🔴 Client %d: Raw message that failed to parse: %s", c.ID, string(message))
			continue
		}

		log.Printf("✅ Client %d: JSON decoded successfully - Action: %s, ChannelID: %s, Text: %s",
			c.ID, msgData.Action, msgData.ChannelID, msgData.Text)

		// Handle different message actions
		switch msgData.Action {
		case "join":
			// Subscribe client to the specified channel
			log.Printf("🟢 Client %d: Attempting to join channel %s", c.ID, msgData.ChannelID)
			c.WsAddChannel(msgData.ChannelID)
			log.Printf("✅ Client %d: Successfully joined channel %s", c.ID, msgData.ChannelID)

		case "leave":
			// Unsubscribe client from the specified channel
			log.Printf("🟡 Client %d: Attempting to leave channel %s", c.ID, msgData.ChannelID)
			c.WsRemoveChannel(msgData.ChannelID)
			log.Printf("✅ Client %d: Successfully left channel %s", c.ID, msgData.ChannelID)

		case "message":
			// Create a complete message structure with metadata
			log.Printf("💬 Client %d: Sending message to channel %s: %s", c.ID, msgData.ChannelID, msgData.Text)

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
			log.Printf("📤 Client %d: Broadcasting message to channel %s: %s", c.ID, msgData.ChannelID, string(msgBytes))

			hub.Broadcast <- ChannelMessage{
				ChannelID: uint(msgData.ChannelID),
				Data:      msgBytes,
			}
			log.Printf("✅ Client %d: Message queued for broadcasting to channel %s", c.ID, msgData.ChannelID)

		default:
			log.Printf("⚠️ Client %d: Unknown action '%s' received", c.ID, msgData.Action)
		}
	}

	log.Printf("🔴 Client %d: Message handler stopped", c.ID)
}

func (h *Hub) BroadcastMessage(msg *models.Chat) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal chat message: %v", err)
		return
	}
	h.Broadcast <- ChannelMessage{
		ChannelID: msg.ChannelID,
		Data:      msgBytes,
	}
}
