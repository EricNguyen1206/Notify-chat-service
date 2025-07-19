package ws

import (
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
