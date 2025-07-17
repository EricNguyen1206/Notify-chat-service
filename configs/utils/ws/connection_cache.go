package ws

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ConnectionMetadata stores additional information about each connection
type ConnectionMetadata struct {
	UserID       uint          `json:"userId"`       // User identifier
	ConnectedAt  time.Time     `json:"connectedAt"`  // When the connection was established
	LastActivity time.Time     `json:"lastActivity"` // Last time the connection was active
	Channels     map[uint]bool `json:"channels"`     // Set of channels the user is subscribed to
	Heartbeats   int           `json:"heartbeats"`   // Count of successful heartbeats
}

// ConnectionCleanupConfig defines configuration for connection cleanup
type ConnectionCleanupConfig struct {
	// Maximum time a connection can be inactive before being considered stale
	InactivityTimeout time.Duration

	// How often to run the cleanup routine
	CleanupInterval time.Duration

	// How often to send heartbeats to clients
	HeartbeatInterval time.Duration

	// Maximum number of failed heartbeats before considering a connection stale
	MaxHeartbeatFailures int
}

// DefaultCleanupConfig returns the default configuration for connection cleanup
func DefaultCleanupConfig() ConnectionCleanupConfig {
	return ConnectionCleanupConfig{
		InactivityTimeout:    5 * time.Minute,
		CleanupInterval:      1 * time.Minute,
		HeartbeatInterval:    30 * time.Second,
		MaxHeartbeatFailures: 3,
	}
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

	// Configuration for connection cleanup
	cleanupConfig ConnectionCleanupConfig

	// Channel to signal cleanup routine to stop
	stopCleanup chan struct{}

	// Flag to indicate if cleanup routine is running
	cleanupRunning bool
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
		cleanupConfig:      DefaultCleanupConfig(),
		stopCleanup:        make(chan struct{}),
		cleanupRunning:     false,
	}
}

// NewUserConnectionCacheWithConfig creates a new UserConnectionCache with custom cleanup configuration
func NewUserConnectionCacheWithConfig(hub *Hub, config ConnectionCleanupConfig) *UserConnectionCache {
	return &UserConnectionCache{
		userConnections:    make(map[uint]*Client),
		channelUsers:       make(map[uint]map[uint]bool),
		connectionMetadata: make(map[uint]*ConnectionMetadata),
		hub:                hub,
		cleanupConfig:      config,
		stopCleanup:        make(chan struct{}),
		cleanupRunning:     false,
	}
}

// AddConnection registers a new user connection in the cache
func (ucc *UserConnectionCache) AddConnection(client *Client) {
	startTime := time.Now()
	success := true

	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	userID := client.ID
	ucc.userConnections[userID] = client

	// Initialize connection metadata
	now := time.Now()
	ucc.connectionMetadata[userID] = &ConnectionMetadata{
		UserID:       userID,
		ConnectedAt:  now,
		LastActivity: now,
		Channels:     make(map[uint]bool),
		Heartbeats:   0,
	}

	log.Printf("Added connection for user %d", userID)

	// Record operation metrics if metrics tracker is available
	if ucc.hub != nil && ucc.hub.Metrics != nil {
		duration := time.Since(startTime)
		ucc.hub.Metrics.RecordCacheOperationMetric("add_connection", duration, success)
	}

	// Trigger connection event if monitoring hooks are available
	if ucc.hub != nil && ucc.hub.MonitoringHooks != nil {
		event := ConnectionEvent{
			EventType:   "connect",
			UserID:      userID,
			Timestamp:   now,
			ConnectedAt: now,
		}
		ucc.hub.MonitoringHooks.TriggerConnectionHooks(event)
	}
}

// RemoveConnection removes a user connection from the cache
func (ucc *UserConnectionCache) RemoveConnection(userID uint) {
	startTime := time.Now()
	success := true

	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	// Get connection metadata before removal for event logging
	var connectedAt time.Time
	var channels []uint
	if metadata, exists := ucc.connectionMetadata[userID]; exists {
		connectedAt = metadata.ConnectedAt

		// Collect channels for logging
		channels = make([]uint, 0, len(metadata.Channels))
		for channelID := range metadata.Channels {
			channels = append(channels, channelID)
		}

		// Remove user from all channels they were subscribed to
		for channelID := range metadata.Channels {
			if users, channelExists := ucc.channelUsers[channelID]; channelExists {
				delete(users, userID)
				// Clean up empty channel entries
				if len(users) == 0 {
					delete(ucc.channelUsers, channelID)
				}
			}
		}
	} else {
		// If metadata doesn't exist, log an error
		log.Printf("Warning: Attempted to remove non-existent user %d from connection cache", userID)
		success = false

		// Log error event if error handler is available
		if ucc.hub != nil && ucc.hub.ErrorHandler != nil {
			ucc.hub.ErrorHandler.HandleCacheError("remove_connection",
				fmt.Errorf("user %d not found in connection cache", userID))
		}
	}

	// Remove user connection and metadata
	delete(ucc.userConnections, userID)
	delete(ucc.connectionMetadata, userID)

	// Record operation metrics if metrics tracker is available
	if ucc.hub != nil && ucc.hub.Metrics != nil {
		duration := time.Since(startTime)
		ucc.hub.Metrics.RecordCacheOperationMetric("remove_connection", duration, success)
	}

	// Trigger connection event if monitoring hooks are available
	if success && ucc.hub != nil && ucc.hub.MonitoringHooks != nil {
		event := ConnectionEvent{
			EventType:   "disconnect",
			UserID:      userID,
			Timestamp:   time.Now(),
			ConnectedAt: connectedAt,
		}
		ucc.hub.MonitoringHooks.TriggerConnectionHooks(event)
	}
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

// UpdateHeartbeat updates the heartbeat count for a user
// This is called when a heartbeat response is received from the client
func (ucc *UserConnectionCache) UpdateHeartbeat(userID uint) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	if metadata, exists := ucc.connectionMetadata[userID]; exists {
		metadata.Heartbeats++
		metadata.LastActivity = time.Now()
	}
}

// ResetHeartbeat resets the heartbeat count for a user
// This is useful when a user performs an action, indicating they are active
func (ucc *UserConnectionCache) ResetHeartbeat(userID uint) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	if metadata, exists := ucc.connectionMetadata[userID]; exists {
		metadata.Heartbeats = 0
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

// StartCleanupRoutine starts the periodic cleanup routine for stale connections
// This should be called after initializing the connection cache
func (ucc *UserConnectionCache) StartCleanupRoutine() {
	ucc.mu.Lock()
	if ucc.cleanupRunning {
		ucc.mu.Unlock()
		return
	}
	ucc.cleanupRunning = true
	ucc.mu.Unlock()

	log.Printf("Starting connection cache cleanup routine (interval: %v, inactivity timeout: %v)",
		ucc.cleanupConfig.CleanupInterval, ucc.cleanupConfig.InactivityTimeout)

	// Start heartbeat routine
	go ucc.heartbeatRoutine()

	// Start cleanup routine
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
				log.Printf("Stopping connection cache cleanup routine")
				return
			}
		}
	}()
}

// StopCleanupRoutine stops the periodic cleanup routine
func (ucc *UserConnectionCache) StopCleanupRoutine() {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	if !ucc.cleanupRunning {
		return
	}

	close(ucc.stopCleanup)
	ucc.cleanupRunning = false
	ucc.stopCleanup = make(chan struct{})
}

// cleanupStaleConnections removes stale connections based on inactivity timeout
// Returns the number of connections that were cleaned up
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
			go func(c *Client) {
				ucc.hub.Unregister <- c
			}(client)

			staleCount++
		}
	}

	return staleCount
}

// heartbeatRoutine sends periodic heartbeats to all connected clients
// This helps detect inactive connections that haven't properly closed
func (ucc *UserConnectionCache) heartbeatRoutine() {
	ticker := time.NewTicker(ucc.cleanupConfig.HeartbeatInterval)
	defer ticker.Stop()

	heartbeatMessage := []byte(`{"type":"heartbeat"}`)

	for {
		select {
		case <-ticker.C:
			ucc.sendHeartbeats(heartbeatMessage)
		case <-ucc.stopCleanup:
			return
		}
	}
}

// sendHeartbeats sends a heartbeat message to all connected clients
// and updates their activity status
func (ucc *UserConnectionCache) sendHeartbeats(message []byte) {
	ucc.mu.RLock()
	// Create a copy of user IDs to avoid holding the lock during sends
	userIDs := make([]uint, 0, len(ucc.userConnections))
	for userID := range ucc.userConnections {
		userIDs = append(userIDs, userID)
	}
	ucc.mu.RUnlock()

	successCount := 0
	failCount := 0

	// Send heartbeats to all users
	for _, userID := range userIDs {
		ucc.mu.RLock()
		client, exists := ucc.userConnections[userID]
		ucc.mu.RUnlock()

		if !exists {
			continue
		}

		// Send heartbeat
		client.mu.Lock()
		err := client.Conn.WriteMessage(1, message) // 1 = TextMessage
		client.mu.Unlock()

		if err != nil {
			failCount++
			// Connection failed, unregister client
			go func(c *Client) {
				ucc.hub.Unregister <- c
			}(client)
		} else {
			successCount++
			// Update last activity
			ucc.UpdateLastActivity(userID)
		}
	}

	if successCount > 0 || failCount > 0 {
		log.Printf("Heartbeat sent to %d clients (%d successful, %d failed)",
			successCount+failCount, successCount, failCount)
	}
}

// SetCleanupConfig updates the cleanup configuration
func (ucc *UserConnectionCache) SetCleanupConfig(config ConnectionCleanupConfig) {
	ucc.mu.Lock()

	// Store the new configuration
	ucc.cleanupConfig = config

	// Check if cleanup is running
	wasRunning := ucc.cleanupRunning

	// If cleanup is running, stop it first
	if wasRunning {
		// Signal to stop the current routines
		close(ucc.stopCleanup)

		// Create a new stop channel
		ucc.stopCleanup = make(chan struct{})

		// Set flag to false
		ucc.cleanupRunning = false
	}

	ucc.mu.Unlock()

	// If it was running, restart it with the new configuration
	if wasRunning {
		// Start new routines with updated configuration
		ucc.StartCleanupRoutine()
	}
}

// GetStaleConnections returns a list of user IDs with stale connections
// This is useful for monitoring and debugging
func (ucc *UserConnectionCache) GetStaleConnections() []uint {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()

	now := time.Now()
	staleUsers := make([]uint, 0)

	for userID, metadata := range ucc.connectionMetadata {
		if now.Sub(metadata.LastActivity) > ucc.cleanupConfig.InactivityTimeout {
			staleUsers = append(staleUsers, userID)
		}
	}

	return staleUsers
}

// IsCleanupRunning returns whether the cleanup routine is currently running
func (ucc *UserConnectionCache) IsCleanupRunning() bool {
	ucc.mu.RLock()
	defer ucc.mu.RUnlock()
	return ucc.cleanupRunning
}

// SetLastActivityTime sets the last activity time for a user (for testing purposes)
func (ucc *UserConnectionCache) SetLastActivityTime(userID uint, t time.Time) {
	ucc.mu.Lock()
	defer ucc.mu.Unlock()

	if metadata, exists := ucc.connectionMetadata[userID]; exists {
		metadata.LastActivity = t
	}
}
