package ws

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Hub struct {
	clients     map[uint]map[*Client]bool // userID -> client connections (support multiple tabs)
	register    chan *Client
	unregister  chan *Client
	directQueue chan DirectMessage	// directQueue is for sending direct messages between users
	channelQueue chan ChannelMessage // channelQueue is for future use, e.g., broadcasting messages to channels

	mu sync.RWMutex
}

var ChatHub = newHub()

func newHub() *Hub {
	h := &Hub{
		clients:     make(map[uint]map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		directQueue: make(chan DirectMessage),
	}
	go h.Run()
	return h
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)

		case client := <-h.unregister:
			h.removeClient(client)

		case msg := <-h.directQueue:
			h.sendDirectMessage(msg)

		case msg := <-h.channelQueue:
			h.broadcastChannelMessage(msg)
		}
	}
}

// Public method to regist new client
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// Public method to unregister client
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// Public method send message 1-1
func (h *Hub) SendDirectMessage(msg DirectMessage) {
	h.directQueue <- msg
}

// Public method to send message to a channel
// This method is used to send messages to all clients in a specific channel
// It can be used for future features like group chats or channels
func (h *Hub) SendChannelMessage(msg ChannelMessage) {
	h.channelQueue <- msg
}

// Public method to join a channel
// This method allows a client to join a specific channel by its ID
// It maintains a map of channel clients to track which clients are in which channels
// This is useful for broadcasting messages to specific channels
func (h *Hub) JoinChannel(client *Client, channelID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.channelClients[channelID] == nil {
		h.channelClients[channelID] = make(map[*Client]bool)
	}
	h.channelClients[channelID][client] = true
}

// Public method to leave a channel
func (h *Hub) LeaveChannel(client *Client, channelID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.channelClients[channelID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.channelClients, channelID)
		}
	}
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]bool)
	}
	h.clients[client.UserID][client] = true
	fmt.Println("User", client.UserID, "connected")
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.UserID]; ok {
		delete(h.clients[client.UserID], client)
		if len(h.clients[client.UserID]) == 0 {
			delete(h.clients, client.UserID)
		}
	}
	fmt.Println("User", client.UserID, "disconnected")
}

func (h *Hub) sendDirectMessage(msg DirectMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	payload := struct {
		From      uint      `json:"from"`
		To        uint      `json:"to"`
		Content   string    `json:"content"`
		Timestamp time.Time `json:"timestamp"`
	}{
		From:      msg.FromUserID,
		To:        msg.ToUserID,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("❌ Failed to marshal message:", err)
		return
	}

	if receivers, ok := h.clients[msg.ToUserID]; ok {
		for client := range receivers {
			select {
			case client.Send <- data:
			default:
				// If channel is full, remove client
				go h.removeClient(client)
			}
		}
	}
}

func (h *Hub) broadcastChannelMessage(msg ChannelMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	payload := struct {
		From      uint      `json:"from"`
		ChannelID uint      `json:"channelId"`
		Content   string    `json:"content"`
		Timestamp time.Time `json:"timestamp"`
	}{
		From:      msg.FromUserID,
		ChannelID: msg.ChannelID,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("❌ Failed to marshal channel message:", err)
		return
	}

	if clients, ok := h.channelClients[msg.ChannelID]; ok {
		for client := range clients {
			select {
			case client.Send <- data:
			default:
				go h.removeClient(client)
			}
		}
	}
}