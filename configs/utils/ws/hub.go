package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// Client đại diện cho người dùng kết nối qua WebSocket
type Client struct {
	ID       uint            // UUID hoặc UserID
	Conn     *websocket.Conn // Kết nối WebSocket
	Channels map[string]bool // Các channel đang tham gia (set)
	mu       sync.Mutex      // Khóa cho concurrent access
}

// Hub quản lý tất cả clients và xử lý message
type Hub struct {
	Clients    map[*Client]bool    // Tập hợp clients đang hoạt động
	Register   chan *Client        // Kênh đăng ký client mới
	Unregister chan *Client        // Kênh hủy đăng ký client
	Broadcast  chan ChannelMessage // Kênh broadcast message
	Redis      *redis.Client       // Redis client cho pub/sub
	mu         sync.RWMutex        // Khóa cho concurrent map access
}

// ChannelMessage cấu trúc tin nhắn nhóm
type ChannelMessage struct {
	ChannelID string `json:"channelId"`
	Data      []byte `json:"data"`
}

// NewHub tạo hub mới
func NewHub(redisClient *redis.Client) *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan ChannelMessage),
		Redis:      redisClient,
	}
}

// Run khởi chạy hub trong goroutine
func (h *Hub) Run() {
	// Khởi chạy Redis message listener
	go h.redisListener()

	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered: %d", client.ID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				client.Conn.Close()
				log.Printf("Client unregistered: %d", client.ID)
			}
			h.mu.Unlock()

		case msg := <-h.Broadcast:
			// Publish message tới Redis channel
			ctx := context.Background()
			if err := h.Redis.Publish(ctx, "channel:"+msg.ChannelID, msg.Data).Err(); err != nil {
				log.Printf("Redis publish error: %v", err)
			}
		}
	}
}

// redisListener lắng nghe message từ Redis
func (h *Hub) redisListener() {
	pubsub := h.Redis.Subscribe(context.Background(), "channel:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		// Extract channelID từ channel name (channel:123 -> 123)
		channelID := msg.Channel[6:]

		h.mu.RLock()
		for client := range h.Clients {
			client.mu.Lock()
			// Kiểm tra client có trong channel không
			if _, ok := client.Channels[channelID]; ok {
				// Gửi message tới client
				if err := client.Conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
					log.Printf("Write error: %v", err)
					// Xử lý lỗi bằng cách đóng kết nối
					h.Unregister <- client
				}
			}
			client.mu.Unlock()
		}
		h.mu.RUnlock()
	}
}

// AddChannel thêm client vào channel
func (c *Client) AddChannel(channelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Channels == nil {
		c.Channels = make(map[string]bool)
	}
	c.Channels[channelID] = true
}

// RemoveChannel xóa client khỏi channel
func (c *Client) RemoveChannel(channelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Channels, channelID)
}

// HandleIncomingMessages xử lý message từ client
func (c *Client) HandleIncomingMessages(hub *Hub) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Read error: %v", err)
			}
			break
		}

		// Phân tích message JSON
		var msgData struct {
			Action    string `json:"action"`
			ChannelID string `json:"channelId"`
			Text      string `json:"text"`
		}

		if err := json.Unmarshal(message, &msgData); err != nil {
			log.Printf("JSON decode error: %v", err)
			continue
		}

		switch msgData.Action {
		case "join":
			c.AddChannel(msgData.ChannelID)
			log.Printf("Client %d joined channel %s", c.ID, msgData.ChannelID)

		case "leave":
			c.RemoveChannel(msgData.ChannelID)
			log.Printf("Client %d left channel %s", c.ID, msgData.ChannelID)

		case "message":
			// Tạo message để broadcast
			fullMsg := struct {
				ChannelID string `json:"channelId"`
				UserID    uint   `json:"userId"`
				Text      string `json:"text"`
				SentAt    string `json:"sentAt"`
			}{
				ChannelID: msgData.ChannelID,
				UserID:    c.ID,
				Text:      msgData.Text,
				SentAt:    time.Now().Format(time.RFC3339),
			}

			msgBytes, _ := json.Marshal(fullMsg)
			hub.Broadcast <- ChannelMessage{
				ChannelID: msgData.ChannelID,
				Data:      msgBytes,
			}
		}
	}
}
