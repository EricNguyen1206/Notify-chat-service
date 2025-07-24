package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"chat-service/internal/services"

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
}

func NewHub(redisService *services.RedisService) *Hub {
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
			slog.Info("WebSocket hub shutting down")
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

	slog.Info("Client registered", "clientID", client.id, "userID", client.userID)

	// Set user online in Redis
	if err := h.redisService.SetUserOnline(h.ctx, client.userID); err != nil {
		slog.Error("Failed to set user online", "userID", client.userID, "error", err)
	}
}

func (h *Hub) subscribeToRedis() {
	// Subscribe to all channel message patterns (wildcard)
	h.pubsub = h.redisService.PSubscribe(h.ctx, "chat:channel:*")
	go func() {
		for {
			msg, err := h.pubsub.ReceiveMessage(h.ctx)
			if err != nil {
				if h.ctx.Err() != nil {
					return // context cancelled, exit goroutine
				}
				slog.Error("Redis pubsub receive error", "error", err)
				continue
			}
			// Extract channel ID from topic (e.g., chat:channel:<id>)
			var channelID string
			_, err = fmt.Sscanf(msg.Channel, "chat:channel:%s", &channelID)
			if err != nil {
				slog.Error("Failed to parse channel ID from topic", "topic", msg.Channel, "error", err)
				continue
			}
			// Broadcast to all clients in this channel
			h.mu.RLock()
			clients := h.channelClients[channelID]
			h.mu.RUnlock()
			for client := range clients {
				client.SendMessage(&Message{
					Type:      MessageTypeChannelMessage,
					Data:      map[string]interface{}{"raw": msg.Payload},
					Timestamp: 0, // Could parse from payload if needed
				})
			}
		}
	}()
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client)
	// Remove from userClients
	if userMap, ok := h.userClients[client.userID]; ok {
		delete(userMap, client)
		if len(userMap) == 0 {
			delete(h.userClients, client.userID)
			// Set user offline in Redis
			if err := h.redisService.SetUserOffline(h.ctx, client.userID); err != nil {
				slog.Error("Failed to set user offline", "userID", client.userID, "error", err)
			}
		}
	}
	// Remove from all channelClients
	for channelID := range client.channels {
		if chMap, ok := h.channelClients[channelID]; ok {
			delete(chMap, client)
			if len(chMap) == 0 {
				delete(h.channelClients, channelID)
			}
		}
	}
	slog.Info("Client unregistered", "clientID", client.id, "userID", client.userID)
}

func (h *Hub) handleClientMessage(clientMsg *ClientMessage) {
	client := clientMsg.Client
	msg := clientMsg.Message
	switch msg.Type {
	case MessageTypeJoinChannel:
		// Join channel
		if data, ok := msg.Data["channel_id"].(string); ok {
			h.mu.Lock()
			if h.channelClients[data] == nil {
				h.channelClients[data] = make(map[*Client]bool)
			}
			h.channelClients[data][client] = true
			h.mu.Unlock()
			client.AddChannel(data)
			// Optionally: update Redis membership
			if err := h.redisService.JoinChannel(h.ctx, client.userID, data); err != nil {
				slog.Error("Failed to join channel in Redis", "userID", client.userID, "channelID", data, "error", err)
			}
		}
	case MessageTypeLeaveChannel:
		if data, ok := msg.Data["channel_id"].(string); ok {
			h.mu.Lock()
			if chMap, ok := h.channelClients[data]; ok {
				delete(chMap, client)
				if len(chMap) == 0 {
					delete(h.channelClients, data)
				}
			}
			h.mu.Unlock()
			client.RemoveChannel(data)
			if err := h.redisService.LeaveChannel(h.ctx, client.userID, data); err != nil {
				slog.Error("Failed to leave channel in Redis", "userID", client.userID, "channelID", data, "error", err)
			}
		}
	case MessageTypeChannelMessage:
		if data, ok := msg.Data["channel_id"].(string); ok {
			// Publish to Redis so all subscribers get it
			if err := h.redisService.PublishChannelMessage(h.ctx, data, msg); err != nil {
				slog.Error("Failed to publish channel message", "channelID", data, "error", err)
			}
		}
	default:
		slog.Warn("Unknown message type", "type", msg.Type)
	}
}

func (h *Hub) RedisService() *services.RedisService {
	return h.redisService
}

func (h *Hub) Context() context.Context {
	return h.ctx
}
