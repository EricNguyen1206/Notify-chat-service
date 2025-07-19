package ws

import (
	"log"
	"sync"

	modelws "chat-service/internal/models/ws"
)

// ClientWrapper wraps a Client to add methods that can't be defined on the original type
type ClientWrapper struct {
	*modelws.Client
	mu sync.Mutex
}

// NewClientWrapper creates a new ClientWrapper
func NewClientWrapper(client *modelws.Client) *ClientWrapper {
	return &ClientWrapper{
		Client: client,
	}
}

// WsHandleIncomingMessages handles incoming WebSocket messages from a client
// Runs in a separate goroutine for each client connection
func (c *ClientWrapper) WsHandleIncomingMessages(hub *Hub) {
	// Ensure client is unregistered and connection is closed when function exits
	defer func() {
		hub.Unregister <- c.Client
		c.Client.Conn.Close()
	}()

	log.Printf("🟢 Client %d: Started message handler", c.ID)

	for {
		// Read message from WebSocket connection
		_, message, err := c.Client.Conn.ReadMessage()
		if err != nil {
			// Log unexpected close errors but handle normal disconnections gracefully
			log.Printf("🟡 Client %d: Connection closed: %v", c.ID, err)
			break
		}

		// Log raw message received
		log.Printf("📥 Client %d: Received raw message: %s", c.ID, string(message))

		// Process the message (implementation details omitted for brevity)
		// This would typically involve parsing the message and taking appropriate action
		log.Printf("📌 Client %d: Processing message", c.ID)
	}
}

// WsAddChannel adds a client to a channel
func (c *ClientWrapper) WsAddChannel(channelID uint, hub *Hub) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Initialize channels map if not already done
	if c.Client.Channels == nil {
		c.Client.Channels = make(map[uint]bool)
	}

	// Skip if client is already subscribed to this channel
	if _, alreadySubscribed := c.Client.Channels[channelID]; alreadySubscribed {
		log.Printf("Client %d already subscribed to channel %d, skipping", c.ID, channelID)
		return
	}

	// Add channel to client's subscription list
	c.Client.Channels[channelID] = true

	// Update connection cache
	if hub != nil && hub.ConnectionCache != nil {
		// Ensure user is registered in connection cache before adding to channel
		if !hub.ConnectionCache.IsUserConnected(c.ID) {
			log.Printf("Client %d not registered in connection cache, registering now", c.ID)
			hub.ConnectionCache.AddConnection(c.Client)
		}

		// Add user to channel in connection cache
		hub.ConnectionCache.AddUserToChannel(c.ID, channelID)

		// Update last activity timestamp
		hub.ConnectionCache.UpdateActivity(c.ID)

		// Log channel users after addition for debugging
		onlineUsers := hub.ConnectionCache.GetChannelUsers(channelID)
		log.Printf("Channel %d now has %d users after client %d subscribed",
			channelID, len(onlineUsers), c.ID)
	}

	log.Printf("Client %d subscribed to channel %d", c.ID, channelID)
}

// WsRemoveChannel removes a client from a channel
func (c *ClientWrapper) WsRemoveChannel(channelID uint, hub *Hub) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Skip if client is not subscribed to this channel
	if c.Client.Channels == nil || !c.Client.Channels[channelID] {
		log.Printf("Client %d not subscribed to channel %d, skipping", c.ID, channelID)
		return
	}

	// Remove channel from client's subscription list
	delete(c.Client.Channels, channelID)

	// Update connection cache
	if hub != nil && hub.ConnectionCache != nil {
		// Remove user from channel in connection cache
		hub.ConnectionCache.RemoveUserFromChannel(c.ID, channelID)

		// Update last activity timestamp
		hub.ConnectionCache.UpdateActivity(c.ID)

		// Log channel users after removal for debugging
		onlineUsers := hub.ConnectionCache.GetChannelUsers(channelID)
		log.Printf("Channel %d now has %d users after client %d unsubscribed",
			channelID, len(onlineUsers), c.ID)
	}

	log.Printf("Client %d unsubscribed from channel %d", c.ID, channelID)
}
