package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"chat-service/internal/models"
	"chat-service/internal/repositories/postgres"
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

	// Chat repository for message storage
	chatRepo *postgres.ChatRepository

	// Redis PubSub connection
	pubsub *redis.PubSub

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Mutex for thread safety
	mu sync.RWMutex
}

func NewHub(redisService *services.RedisService, chatRepo *postgres.ChatRepository) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	hub := &Hub{
		clients:        make(map[*Client]bool),
		userClients:    make(map[string]map[*Client]bool),
		channelClients: make(map[string]map[*Client]bool),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		handleMessage:  make(chan *ClientMessage),
		redisService:   redisService,
		chatRepo:       chatRepo,
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

	// Set user online in Redis
	if err := h.redisService.SetUserOnline(h.ctx, client.userID); err != nil {
		slog.Error("Failed to set user online", "userID", client.userID, "error", err)
	}

	// Send connection success message
	client.SendMessage(&Message{
		ID:        fmt.Sprintf("conn_%d", time.Now().UnixNano()),
		Type:      MessageTypeConnect,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"client_id": client.id,
			"status":    "connected",
		},
	})
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		// Remove from clients map
		delete(h.clients, client)

		// Remove from user clients map
		if userClients, exists := h.userClients[client.userID]; exists {
			delete(userClients, client)
			if len(userClients) == 0 {
				delete(h.userClients, client.userID)
				// Set user offline if no more clients
				if err := h.redisService.SetUserOffline(h.ctx, client.userID); err != nil {
					slog.Error("Failed to set user offline", "userID", client.userID, "error", err)
				}
			}
		}

		// Remove from all channel subscriptions
		for channelID := range client.channels {
			h.removeClientFromChannel(client, channelID)
		}

		close(client.send)
	}
}

func (h *Hub) handleClientMessage(clientMsg *ClientMessage) {
	client := clientMsg.Client
	message := clientMsg.Message

	slog.Debug("Handling client message", "type", message.Type, "userID", client.userID)

	switch message.Type {
	case MessageTypeJoinChannel:
		h.handleJoinChannel(client, message)
	case MessageTypeLeaveChannel:
		h.handleLeaveChannel(client, message)
	case MessageTypeChannelMessage:
		h.handleChannelMessage(client, message)
	case MessageTypeTyping:
		h.handleTyping(client, message)
	case MessageTypePing:
		h.handlePing(client, message)
	default:
		slog.Warn("Unknown message type", "type", message.Type)
		client.sendError("UNKNOWN_MESSAGE_TYPE", "Unknown message type")
	}
}

func (h *Hub) handleJoinChannel(client *Client, message *Message) {
	var data JoinChannelData
	if err := h.mapToStruct(message.Data, &data); err != nil {
		client.sendError("INVALID_DATA", "Invalid join channel data")
		return
	}

	// Check if user can join channel (implement permission checks here)
	canJoin, err := h.canUserJoinChannel(client.userID, data.ChannelID)
	if err != nil {
		slog.Error("Failed to check channel permissions", "error", err)
		client.sendError("PERMISSION_ERROR", "Failed to check permissions")
		return
	}

	if !canJoin {
		client.sendError("PERMISSION_DENIED", "Permission denied to join channel")
		return
	}

	// Add client to channel
	h.addClientToChannel(client, data.ChannelID)

	// Update Redis
	if err := h.redisService.JoinChannel(h.ctx, client.userID, data.ChannelID); err != nil {
		slog.Error("Failed to join channel in Redis", "error", err)
		client.sendError("JOIN_FAILED", "Failed to join channel")
		return
	}

	// Send success response
	client.SendMessage(&Message{
		ID:        fmt.Sprintf("join_%d", time.Now().UnixNano()),
		Type:      MessageTypeJoinChannel,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"channel_id": data.ChannelID,
			"status":     "joined",
		},
	})
}

func (h *Hub) handleLeaveChannel(client *Client, message *Message) {
	var data LeaveChannelData
	if err := h.mapToStruct(message.Data, &data); err != nil {
		client.sendError("INVALID_DATA", "Invalid leave channel data")
		return
	}

	// Remove client from channel
	h.removeClientFromChannel(client, data.ChannelID)

	// Update Redis
	if err := h.redisService.LeaveChannel(h.ctx, client.userID, data.ChannelID); err != nil {
		slog.Error("Failed to leave channel in Redis", "error", err)
	}

	// Send success response
	client.SendMessage(&Message{
		ID:        fmt.Sprintf("leave_%d", time.Now().UnixNano()),
		Type:      MessageTypeLeaveChannel,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"channel_id": data.ChannelID,
			"status":     "left",
		},
	})
}

func (h *Hub) handleChannelMessage(client *Client, message *Message) {
	var data models.ChatRequest
	if err := h.mapToStruct(message.Data, &data); err != nil {
		client.sendError("INVALID_DATA", "Invalid message data")
		return
	}
	slog.Info("Received channel message", "channelID", data.ChannelID, "userID", client.userID)

	// Check if client is in channel
	channelIDStr := strconv.FormatUint(uint64(data.ChannelID), 10)
	if !client.IsInChannel(channelIDStr) {
		client.sendError("NOT_IN_CHANNEL", "You are not in this channel")
		return
	}

	// Check rate limit
	rateLimitKey := fmt.Sprintf("rate_limit:message:%s:%s", client.userID, channelIDStr)
	allowed, err := h.redisService.CheckRateLimit(h.ctx, rateLimitKey, 10, time.Minute)
	if err != nil {
		slog.Error("Failed to check rate limit", "error", err)
		client.sendError("RATE_LIMIT_ERROR", "Failed to check rate limit")
		return
	}

	if !allowed {
		client.sendError("RATE_LIMITED", "Rate limit exceeded")
		return
	}

	// Convert client.userID (string) to uint
	senderIDUint, err := strconv.ParseUint(client.userID, 10, 64)
	if err != nil {
		slog.Error("Failed to convert userID to uint", "userID", client.userID, "error", err)
		client.sendError("INVALID_USER_ID", "Invalid user ID")
		return
	}

	// Save message to database
	chat := &models.Chat{
		SenderID:  uint(senderIDUint),
		ChannelID: data.ChannelID,
		Text:      data.Text,
		URL:       data.URL,
		FileName:  data.FileName,
	}
	slog.Debug("❤️ Creating chat message", "chat", chat)
	if err := h.chatRepo.Create(chat); err != nil {
		slog.Error("Failed to save chat message to DB", "userID", client.userID, "error", err)
		client.sendError("ERROR", "Failed to create chat message")
		return
	}

	// Preload sender data
	chat, err = h.chatRepo.FindByID(chat.ID)
	if err != nil {
		slog.Error("Failed to preload chat sender data", "chatID", chat.ID, "error", err)
		client.sendError("ERROR", "Failed to preload chat sender data")
		return
	}

	// Prepare message for broadcast
	chatData, err := h.structToMap(chat)
	if err != nil {
		slog.Error("Failed to convert chat to map", "error", err)
		client.sendError("ERROR", "Failed to create chat message")
		return
	}
	broadcastMessage := &Message{
		ID:        message.ID,
		Type:      MessageTypeChannelMessage,
		UserID:    client.userID,
		Timestamp: time.Now().Unix(),
		Data:      chatData,
	}

	// Publish to Redis for other server instances
	if err := h.redisService.PublishChannelMessage(h.ctx, channelIDStr, broadcastMessage); err != nil {
		slog.Error("Failed to publish message to Redis", "error", err)
		client.sendError("PUBLISH_FAILED", "Failed to send message")
		return
	}
}

func (h *Hub) handleTyping(client *Client, message *Message) {
	var data TypingData
	if err := h.mapToStruct(message.Data, &data); err != nil {
		client.sendError("INVALID_DATA", "Invalid typing data")
		return
	}

	// Check if client is in channel
	if !client.IsInChannel(data.ChannelID) {
		return
	}

	// Create typing message
	typingMessage := &Message{
		ID:        fmt.Sprintf("typing_%d", time.Now().UnixNano()),
		Type:      MessageTypeTyping,
		UserID:    client.userID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"channel_id": data.ChannelID,
			"is_typing":  data.IsTyping,
		},
	}

	// Broadcast to channel (excluding sender)
	h.broadcastToChannelExcept(data.ChannelID, typingMessage, client)
}

func (h *Hub) handlePing(client *Client, message *Message) {
	pongMessage := &Message{
		ID:        fmt.Sprintf("pong_%d", time.Now().UnixNano()),
		Type:      MessageTypePong,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"ping_id": message.ID,
		},
	}
	client.SendMessage(pongMessage)
}

func (h *Hub) addClientToChannel(client *Client, channelID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.channelClients[channelID] == nil {
		h.channelClients[channelID] = make(map[*Client]bool)
	}

	h.channelClients[channelID][client] = true
	client.AddChannel(channelID)

	slog.Debug("Client added to channel", "clientID", client.id, "channelID", channelID)
}

func (h *Hub) removeClientFromChannel(client *Client, channelID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if channelClients, exists := h.channelClients[channelID]; exists {
		delete(channelClients, client)
		if len(channelClients) == 0 {
			delete(h.channelClients, channelID)
		}
	}

	client.RemoveChannel(channelID)

	slog.Debug("Client removed from channel", "clientID", client.id, "channelID", channelID)
}

func (h *Hub) broadcastToChannel(channelID string, message *Message) {
	h.mu.RLock()
	clients := h.channelClients[channelID]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- h.messageToBytes(message):
		default:
			h.unregisterClient(client)
		}
	}
}

func (h *Hub) broadcastToChannelExcept(channelID string, message *Message, exceptClient *Client) {
	h.mu.RLock()
	clients := h.channelClients[channelID]
	h.mu.RUnlock()

	for client := range clients {
		if client != exceptClient {
			select {
			case client.send <- h.messageToBytes(message):
			default:
				h.unregisterClient(client)
			}
		}
	}
}

func (h *Hub) broadcastToUser(userID string, message *Message) {
	h.mu.RLock()
	clients := h.userClients[userID]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- h.messageToBytes(message):
		default:
			h.unregisterClient(client)
		}
	}
}

// =============================================================================
// STEP 7: Redis PubSub Integration
// =============================================================================

func (h *Hub) subscribeToRedis() {
	// Subscribe to all channel patterns
	h.pubsub = h.redisService.PSubscribe(h.ctx, "chat:channel:*", "channel:*:events", "user:*:notifications")

	go h.handleRedisMessages()
}

func (h *Hub) handleRedisMessages() {
	ch := h.pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			h.processRedisMessage(msg)
		case <-h.ctx.Done():
			return
		}
	}
}

func (h *Hub) processRedisMessage(msg *redis.Message) {
	var message Message
	if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
		slog.Error("Failed to unmarshal Redis message", "error", err)
		return
	}

	// Determine routing based on channel pattern
	switch {
	case len(msg.Channel) > 13 && msg.Channel[:13] == "chat:channel:":
		// Channel message: chat:channel:{channel_id}
		channelID := msg.Channel[13:]
		h.broadcastToChannel(channelID, &message)

	case len(msg.Channel) > 8 && msg.Channel[:8] == "channel:":
		// Channel events: channel:{channel_id}:events
		if len(msg.Channel) > 15 && msg.Channel[len(msg.Channel)-7:] == ":events" {
			channelID := msg.Channel[8 : len(msg.Channel)-7]
			h.broadcastToChannel(channelID, &message)
		}

	case len(msg.Channel) > 5 && msg.Channel[:5] == "user:":
		// User notifications: user:{user_id}:notifications
		if len(msg.Channel) > 18 && msg.Channel[len(msg.Channel)-13:] == ":notifications" {
			userID := msg.Channel[5 : len(msg.Channel)-14]
			h.broadcastToUser(userID, &message)
		}
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func (h *Hub) messageToBytes(message *Message) []byte {
	data, err := json.Marshal(message)
	if err != nil {
		slog.Error("Failed to marshal message", "error", err)
		return nil
	}
	return data
}

func (h *Hub) mapToStruct(data map[string]interface{}, dest interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, dest)
}

// structToMap converts a struct to map[string]interface{}
func (h *Hub) structToMap(obj interface{}) (map[string]interface{}, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *Hub) canUserJoinChannel(userID, channelID string) (bool, error) {
	// Implement your channel permission logic here
	// For now, allow all users to join all channels
	return true, nil
}
