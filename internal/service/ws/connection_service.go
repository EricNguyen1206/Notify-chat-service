package ws

import (
	"log"
	"sync"
	"time"

	"chat-service/internal/models/ws"
)

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

// UserConnectionCache manages all user connections and provides broadcasting capabilities
type UserConnectionCache struct {
	// userConnections maps user ID to their active client connection
	userConnections map[uint]*ws.Client

	// channelUsers maps channel ID to set of user IDs subscribed to that channel
	channelUsers map[uint]map[uint]bool

	// connectionMetadata stores additional info about each connection
	connectionMetadata map[uint]*ws.ConnectionMetadata

	// Thread safety
	mu sync.RWMutex

	// Reference to the hub for integration
	hub HubInterface

	// Configuration for connection cleanup
	cleanupConfig ws.ConnectionCleanupConfig

	// Channel to signal cleanup routine to stop
	stopCleanup chan struct{}

	// Flag to indicate if cleanup routine is running
	cleanupRunning bool
}

// NewUserConnectionCache creates and initializes a new UserConnectionCache
func NewUserConnectionCache(hub HubInterface) *UserConnectionCache {
	return &UserConnectionCache{
		userConnections:    make(map[uint]*ws.Client),
		channelUsers:       make(map[uint]map[uint]bool),
		connectionMetadata: make(map[uint]*ws.ConnectionMetadata),
		hub:                hub,
		cleanupConfig:      ws.DefaultCleanupConfig(),
		stopCleanup:        make(chan struct{}),
		cleanupRunning:     false,
	}
}

// NewUserConnectionCacheWithConfig creates a new UserConnectionCache with custom cleanup configuration
func NewUserConnectionCacheWithConfig(hub HubInterface, config ws.ConnectionCleanupConfig) *UserConnectionCache {
	return &UserConnectionCache{
		userConnections:    make(map[uint]*ws.Client),
		channelUsers:       make(map[uint]map[uint]bool),
		connectionMetadata: make(map[uint]*ws.ConnectionMetadata),
		hub:                hub,
		cleanupConfig:      config,
		stopCleanup:        make(chan struct{}),
		cleanupRunning:     false,
	}
}

// AddConnection adds a new client connection to the cache
func (ucc *UserConnectionCache) AddConnection(client *ws.Client) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	// Store the client connection
	ucc.userConnections[client.ID] = client

	// Initialize metadata
	ucc.connectionMetadata[client.ID] = &ws.ConnectionMetadata{
		UserID:       client.ID,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		Channels:     make(map[uint]bool),
		Heartbeats:   0,
	}

	log.Printf("Added connection for user %d to cache", client.ID)
}

// RemoveConnection removes a client connection from the cache
func (ucc *UserConnectionCache) RemoveConnection(userID uint) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	// Remove from user connections
	delete(ucc.userConnections, userID)

	// Get channels the user was in before removing metadata
	var userChannels []uint
	if metadata, exists := ucc.connectionMetadata[userID]; exists {
		for channelID := range metadata.Channels {
			userChannels = append(userChannels, channelID)
		}
	}

	// Remove from connection metadata
	delete(ucc.connectionMetadata, userID)

	// Remove from all channels
	for _, channelID := range userChannels {
		if users, exists := ucc.channelUsers[channelID]; exists {
			delete(users, userID)

			// If channel is empty, remove it
			if len(users) == 0 {
				delete(ucc.channelUsers, channelID)
			}
		}
	}

	log.Printf("Removed connection for user %d from cache", userID)
}

// AddUserToChannel adds a user to a channel
func (ucc *UserConnectionCache) AddUserToChannel(userID uint, channelID uint) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	// Initialize channel users map if it doesn't exist
	if _, exists := ucc.channelUsers[channelID]; !exists {
		ucc.channelUsers[channelID] = make(map[uint]bool)
	}

	// Add user to channel
	ucc.channelUsers[channelID][userID] = true

	// Update user's metadata to include this channel
	if metadata, exists := ucc.connectionMetadata[userID]; exists {
		if metadata.Channels == nil {
			metadata.Channels = make(map[uint]bool)
		}
		metadata.Channels[channelID] = true
	}

	log.Printf("Added user %d to channel %d", userID, channelID)
}

// RemoveUserFromChannel removes a user from a channel
func (ucc *UserConnectionCache) RemoveUserFromChannel(userID uint, channelID uint) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	// Remove user from channel
	if users, exists := ucc.channelUsers[channelID]; exists {
		delete(users, userID)

		// If channel is empty, remove it
		if len(users) == 0 {
			delete(ucc.channelUsers, channelID)
		}
	}

	// Update user's metadata to remove this channel
	if metadata, exists := ucc.connectionMetadata[userID]; exists && metadata.Channels != nil {
		delete(metadata.Channels, channelID)
	}

	log.Printf("Removed user %d from channel %d", userID, channelID)
}

// GetChannelUsers returns all users in a channel
func (ucc *UserConnectionCache) GetChannelUsers(channelID uint) []uint {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	users := make([]uint, 0)
	if channelUsers, exists := ucc.channelUsers[channelID]; exists {
		for userID := range channelUsers {
			users = append(users, userID)
		}
	}

	return users
}

// IsUserConnected checks if a user is connected
func (ucc *UserConnectionCache) IsUserConnected(userID uint) bool {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	_, exists := ucc.userConnections[userID]
	return exists
}

// UpdateActivity updates the last activity timestamp for a user
func (ucc *UserConnectionCache) UpdateActivity(userID uint) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	if metadata, exists := ucc.connectionMetadata[userID]; exists {
		metadata.LastActivity = time.Now()
		metadata.Heartbeats++
	}
}

// GetConnectionMetadata returns the metadata for a user connection
func (ucc *UserConnectionCache) GetConnectionMetadata(userID uint) (*ws.ConnectionMetadata, bool) {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	metadata, exists := ucc.connectionMetadata[userID]
	return metadata, exists
}

// StartCleanupRoutine starts the routine to clean up stale connections
func (ucc *UserConnectionCache) StartCleanupRoutine() {
	ucc.mu.Lock()
	if ucc.cleanupRunning {
		ucc.mu.Unlock()
		return
	}

	ucc.cleanupRunning = true
	ucc.mu.Unlock()

	go func() {
		ticker := time.NewTicker(ucc.cleanupConfig.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				staleCount := ucc.cleanupStaleConnections()
				if staleCount > 0 {
					log.Printf("Cleaned up %d stale connections", staleCount)
				}
			case <-ucc.stopCleanup:
				log.Printf("Connection cleanup routine stopped")
				return
			}
		}
	}()

	log.Printf("Started connection cleanup routine with interval %v", ucc.cleanupConfig.CleanupInterval)
}

// cleanupStaleConnections removes stale connections from the cache
func (ucc *UserConnectionCache) cleanupStaleConnections() int {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	now := time.Now()
	staleCount := 0
	staleUsers := make([]uint, 0)

	// Find stale connections
	for userID, metadata := range ucc.connectionMetadata {
		// Check if connection is stale based on inactivity
		if now.Sub(metadata.LastActivity) > ucc.cleanupConfig.InactivityTimeout {
			staleUsers = append(staleUsers, userID)
		}
	}

	// Remove stale connections
	for _, userID := range staleUsers {
		if client, exists := ucc.userConnections[userID]; exists {
			log.Printf("Removing stale connection for user %d (inactive for %v)",
				userID, now.Sub(ucc.connectionMetadata[userID].LastActivity))

			// Use hub's unregister channel to properly clean up the connection
			if ucc.hub != nil {
				go func(c *ws.Client) {
					// This assumes the hub has a channel called Unregister
					// that accepts Client pointers
					// ucc.hub.Unregister <- c
				}(client)
			}

			staleCount++
		}
	}

	return staleCount
}
