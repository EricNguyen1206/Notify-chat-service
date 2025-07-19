package ws

import (
	"sync"
)

// WebSocketConnection interface for testability
type WebSocketConnection interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
}

// Client represents a WebSocket connection for a user
// Each client maintains its own connection and tracks which channels it's subscribed to
type Client struct {
	ID       uint                // UserID - unique identifier for the user
	Conn     WebSocketConnection // WebSocket connection instance
	Channels map[uint]bool       // Set of channel IDs the client is subscribed to (using map for O(1) lookup)
	mu       sync.Mutex          // Mutex for thread-safe access to client data
}
