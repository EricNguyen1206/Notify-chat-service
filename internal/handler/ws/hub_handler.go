package ws

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	modelws "chat-service/internal/models/ws"
	servicews "chat-service/internal/service/ws"
)

// WebSocketConnWrapper wraps gorilla websocket.Conn to implement WebSocketConnection interface
type WebSocketConnWrapper struct {
	*websocket.Conn
}

// Hub manages all WebSocket clients and message broadcasting
// Acts as a central coordinator for WebSocket connections and Redis pub/sub integration
type Hub struct {
	Clients         map[*modelws.Client]bool       // Registry of all active WebSocket clients
	Register        chan *modelws.Client           // Channel for registering new clients
	Unregister      chan *modelws.Client           // Channel for unregistering/disconnecting clients
	Broadcast       chan modelws.ChannelMessage    // Channel for broadcasting messages to Redis
	Redis           *redis.Client                  // Redis client for pub/sub functionality
	ConnectionCache *servicews.UserConnectionCache // Connection cache for efficient user presence management
	ErrorHandler    servicews.ErrorHandler         // Error handler for connection and broadcast errors
	Metrics         *servicews.ConnectionMetrics   // Performance metrics tracker
	MonitoringHooks *servicews.MonitoringHooks     // Monitoring hooks for event callbacks
	mu              sync.RWMutex                   // Read-write mutex for concurrent map access
}

// NewHub creates and initializes a new Hub instance
// Returns a configured hub ready to handle WebSocket connections
func NewHub(redisClient *redis.Client) *Hub {
	hub := &Hub{
		Clients:    make(map[*modelws.Client]bool),
		Register:   make(chan *modelws.Client),
		Unregister: make(chan *modelws.Client),
		Broadcast:  make(chan modelws.ChannelMessage),
		Redis:      redisClient,
	}

	// Initialize the connection cache with reference to the hub
	hub.ConnectionCache = servicews.NewUserConnectionCache(hub)

	// Initialize error handler
	hub.ErrorHandler = servicews.NewErrorHandler(hub)

	// Initialize metrics tracker (keep 1000 recent metrics)
	hub.Metrics = servicews.NewConnectionMetrics(1000)

	// Initialize monitoring hooks
	hub.MonitoringHooks = servicews.NewMonitoringHooks()

	return hub
}
