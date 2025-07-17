package ws

import (
	"sync"
	"time"
)

// ConnectionMetadata stores additional information about each connection
type ConnectionMetadata struct {
	UserID       uint          `json:"userId"`       // User identifier
	ConnectedAt  time.Time     `json:"connectedAt"`  // When the connection was established
	LastActivity time.Time     `json:"lastActivity"` // Last time the connection was active
	Channels     map[uint]bool `json:"channels"`     // Set of channels the user is subscribed to
}

// UserConnectionCache manages all user connections and provides broadcasting capabilities
type UserConnectionCache struct {
	// userConnections maps user ID to their active client connection
	userConnections map[uint]*Client

	// channelUsers maps channel ID to set of user IDs subscribed to that channel
	channelUsers map[uint]map[uint]bool

	// connectionMetadata stores additional info about each connection
	connectionMetadata map[uint]*ConnectionMetadata

	// Thread safety
	mu sync.RWMutex

	// Reference to the hub for integration
	hub *Hub
}

// Broadcaster interface defines methods for targeted message delivery
type Broadcaster interface {
	// BroadcastToChannel sends a message to all online users in a specific channel
	BroadcastToChannel(channelID uint, message []byte) error

	// BroadcastToUser sends a message to a specific online user
	BroadcastToUser(userID uint, message []byte) error

	// BroadcastToUsers sends a message to multiple specific online users
	BroadcastToUsers(userIDs []uint, message []byte) error

	// GetOnlineUsersInChannel returns all online users in a specific channel
	GetOnlineUsersInChannel(channelID uint) []uint

	// GetOnlineUsers returns all currently online users
	GetOnlineUsers() []uint

	// IsUserOnline checks if a specific user is currently online
	IsUserOnline(userID uint) bool
}

// NewUserConnectionCache creates and initializes a new UserConnectionCache
func NewUserConnectionCache(hub *Hub) *UserConnectionCache {
	return &UserConnectionCache{
		userConnections:    make(map[uint]*Client),
		channelUsers:       make(map[uint]map[uint]bool),
		connectionMetadata: make(map[uint]*ConnectionMetadata),
		hub:                hub,
	}
}

// AddConnection registers a new user connection in the cache
func (ucc *UserConnectionCache) AddConnection(client *Client) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	userID := client.ID
	ucc.userConnections[userID] = client

	// Initialize connection metadata
	ucc.connectionMetadata[userID] = &ConnectionMetadata{
		UserID:       userID,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		Channels:     make(map[uint]bool),
	}
}

// RemoveConnection removes a user connection from the cache
func (ucc *UserConnectionCache) RemoveConnection(userID uint) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	// Remove user from all channels they were subscribed to
	if metadata, exists := ucc.connectionMetadata[userID]; exists {
		for channelID := range metadata.Channels {
			if users, channelExists := ucc.channelUsers[channelID]; channelExists {
				delete(users, userID)
				// Clean up empty channel entries
				if len(users) == 0 {
					delete(ucc.channelUsers, channelID)
				}
			}
		}
	}

	// Remove user connection and metadata
	delete(ucc.userConnections, userID)
	delete(ucc.connectionMetadata, userID)
}

// AddUserToChannel adds a user to a channel's subscription list
func (ucc *UserConnectionCache) AddUserToChannel(userID uint, channelID uint) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	// Initialize channel users map if it doesn't exist
	if ucc.channelUsers[channelID] == nil {
		ucc.channelUsers[channelID] = make(map[uint]bool)
	}

	// Add user to channel
	ucc.channelUsers[channelID][userID] = true

	// Update user's metadata
	if metadata, exists := ucc.connectionMetadata[userID]; exists {
		metadata.Channels[channelID] = true
		metadata.LastActivity = time.Now()
	}
}

// RemoveUserFromChannel removes a user from a channel's subscription list
func (ucc *UserConnectionCache) RemoveUserFromChannel(userID uint, channelID uint) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	// Remove user from channel
	if users, exists := ucc.channelUsers[channelID]; exists {
		delete(users, userID)
		// Clean up empty channel entries
		if len(users) == 0 {
			delete(ucc.channelUsers, channelID)
		}
	}

	// Update user's metadata
	if metadata, exists := ucc.connectionMetadata[userID]; exists {
		delete(metadata.Channels, channelID)
		metadata.LastActivity = time.Now()
	}
}

// GetOnlineUsersInChannel returns all online users in a specific channel
func (ucc *UserConnectionCache) GetOnlineUsersInChannel(channelID uint) []uint {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	users, exists := ucc.channelUsers[channelID]
	if !exists {
		return []uint{}
	}

	result := make([]uint, 0, len(users))
	for userID := range users {
		// Double-check that the user is still connected
		if _, connected := ucc.userConnections[userID]; connected {
			result = append(result, userID)
		}
	}

	return result
}

// GetOnlineUsers returns all currently online users
func (ucc *UserConnectionCache) GetOnlineUsers() []uint {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	result := make([]uint, 0, len(ucc.userConnections))
	for userID := range ucc.userConnections {
		result = append(result, userID)
	}

	return result
}

// IsUserOnline checks if a specific user is currently online
func (ucc *UserConnectionCache) IsUserOnline(userID uint) bool {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	_, exists := ucc.userConnections[userID]
	return exists
}

// GetConnection returns the client connection for a specific user
func (ucc *UserConnectionCache) GetConnection(userID uint) (*Client, bool) {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	client, exists := ucc.userConnections[userID]
	return client, exists
}

// GetConnectionMetadata returns metadata for a specific user connection
func (ucc *UserConnectionCache) GetConnectionMetadata(userID uint) (*ConnectionMetadata, bool) {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	metadata, exists := ucc.connectionMetadata[userID]
	return metadata, exists
}

// UpdateLastActivity updates the last activity timestamp for a user
func (ucc *UserConnectionCache) UpdateLastActivity(userID uint) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	if metadata, exists := ucc.connectionMetadata[userID]; exists {
		metadata.LastActivity = time.Now()
	}
}

// BroadcastToChannel sends a message to all online users in a specific channel
func (ucc *UserConnectionCache) BroadcastToChannel(channelID uint, message []byte) error {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	users, exists := ucc.channelUsers[channelID]
	if !exists {
		return nil // No users in channel
	}

	var lastError error
	successCount := 0

	for userID := range users {
		if client, connected := ucc.userConnections[userID]; connected {
			client.mu.Lock()
			err := client.Conn.WriteMessage(1, message) // 1 = TextMessage
			client.mu.Unlock()

			if err != nil {
				lastError = err
				// Remove failed connection (will be handled by hub)
				go func(c *Client) {
					ucc.hub.Unregister <- c
				}(client)
			} else {
				successCount++
			}
		}
	}

	return lastError
}

// BroadcastToUser sends a message to a specific online user
func (ucc *UserConnectionCache) BroadcastToUser(userID uint, message []byte) error {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	client, exists := ucc.userConnections[userID]
	if !exists {
		return nil // User not online
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	err := client.Conn.WriteMessage(1, message) // 1 = TextMessage
	if err != nil {
		// Remove failed connection (will be handled by hub)
		go func(c *Client) {
			ucc.hub.Unregister <- c
		}(client)
	}

	return err
}

// BroadcastToUsers sends a message to multiple specific online users
func (ucc *UserConnectionCache) BroadcastToUsers(userIDs []uint, message []byte) error {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	var lastError error
	successCount := 0

	for _, userID := range userIDs {
		if client, connected := ucc.userConnections[userID]; connected {
			client.mu.Lock()
			err := client.Conn.WriteMessage(1, message) // 1 = TextMessage
			client.mu.Unlock()

			if err != nil {
				lastError = err
				// Remove failed connection (will be handled by hub)
				go func(c *Client) {
					ucc.hub.Unregister <- c
				}(client)
			} else {
				successCount++
			}
		}
	}

	return lastError
}
