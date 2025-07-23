package websocket

import (
	"context"
	"fmt"
	"sync"

	"chat-service/internal/services"
	"pkg/logger"

	"github.com/redis/go-redis/v9"
)

var (
	ErrClientDisconnected = fmt.Errorf("client disconnected")
	ErrChannelNotFound    = fmt.Errorf("channel not found")
)

type ClientMessage struct {
	Client  *Client
	Message *Message
}

type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Client lookup by user ID
	userClients map[string]map[*Client]bool

	// Channel subscriptions
	channelClients map[string]map[*Client]bool

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Handle messages from clients
	handleMessage chan *ClientMessage

	// Redis service for PubSub
	redisService *services.RedisService

	// Redis PubSub connection
	pubsub *redis.PubSub

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Mutex for thread safety
	mu sync.RWMutex

	logger *logger.Logger
}

func NewHub(redisService *services.RedisService, logger *logger.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	hub := &Hub{
		clients:        make(map[*Client]bool),
		userClients:    make(map[string]map[*Client]bool),
		channelClients: make(map[string]map[*Client]bool),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		handleMessage:  make(chan *ClientMessage),
		redisService:   redisService,
		ctx:            ctx,
		cancel:         cancel,
		logger:         logger,
	}

	return hub
}

func (h *Hub) Run() {
	// Subscribe to Redis channels
	h.subscribeToRedis()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case clientMsg := <-h.handleMessage:
			h.handleClientMessage(clientMsg)

		case <-h.ctx.Done():
			h.logger.Info("WebSocket hub shutting down")
			return
		}
	}
}

func (h *Hub) Stop() {
	h.cancel()
	if h.pubsub != nil {
		h.pubsub.Close()
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true

	// Add to user clients map
	if h.userClients[client.userID] == nil {
		h.userClients[client.userID] = make(map[*Client]bool)
	}
	h.userClients[client.userID][client] = true

	h.logger.Info("Client registered", "clientID", client.id, "userID", client.userID)

	// Set user online in Redis
	if err := h.redisService.SetUserOnline(h.ctx, client.userID); err != nil {
		h.logger.Error("Failed to set user online", "userID", client.userID, "error", err)
	}
}
