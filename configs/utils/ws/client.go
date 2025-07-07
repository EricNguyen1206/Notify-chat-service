package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket connection for a user
// Each client maintains its own connection and tracks which channels it's subscribed to
type Client struct {
	ID       uint            // UserID - unique identifier for the user
	Conn     *websocket.Conn // WebSocket connection instance
	Channels map[string]bool // Set of channel IDs the client is subscribed to (using map for O(1) lookup)
	mu       sync.Mutex      // Mutex for thread-safe access to client data
}
