// package websocket

// import (
// 	"sync"

// 	"github.com/gorilla/websocket"
// )

// // Client represents a WebSocket connection for a user
// // Each client maintains its own connection and tracks which channels it's subscribed to
// type Client struct {
// 	ID       uint            // UserID - unique identifier for the user
// 	Conn     *websocket.Conn // WebSocket connection instance
// 	Channels map[uint]bool   // Set of channel IDs the client is subscribed to (using map for O(1) lookup)
// 	mu       sync.Mutex      // Mutex for thread-safe access to client data
// }

package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

type Client struct {
	id       string
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	channels map[string]bool // Set of channel IDs the client is subscribed to
	mu       sync.RWMutex
	logger   *logger.Logger
}

func NewClient(hub *Hub, conn *websocket.Conn, userID string, logger *logger.Logger) *Client {
	return &Client{
		id:       uuid.New().String(),
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		channels: make(map[string]bool),
		logger:   logger,
	}
}

func (c *Client) GetID() string {
	return c.id
}

func (c *Client) GetUserID() string {
	return c.userID
}

func (c *Client) GetChannels() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	channels := make([]string, 0, len(c.channels))
	for channelID := range c.channels {
		channels = append(channels, channelID)
	}
	return channels
}

func (c *Client) AddChannel(channelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channels[channelID] = true
}

func (c *Client) RemoveChannel(channelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.channels, channelID)
}

func (c *Client) IsInChannel(channelID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.channels[channelID]
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket error", "error", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			c.logger.Error("Failed to unmarshal message", "error", err)
			c.sendError("INVALID_MESSAGE", "Invalid message format")
			continue
		}

		// Set user ID and timestamp
		msg.UserID = c.userID
		msg.Timestamp = time.Now().Unix()
		if msg.ID == "" {
			msg.ID = uuid.New().String()
		}

		// Handle the message
		c.hub.handleMessage <- &ClientMessage{
			Client:  c,
			Message: &msg,
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) SendMessage(message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	default:
		close(c.send)
		return ErrClientDisconnected
	}
}

func (c *Client) sendError(code, message string) {
	errorMsg := &Message{
		ID:        uuid.New().String(),
		Type:      MessageTypeError,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	c.SendMessage(errorMsg)
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, userID string, logger *logger.Logger) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade WebSocket connection", "error", err)
		return
	}

	client := NewClient(hub, conn, userID, logger)
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in new goroutines
	go client.writePump()
	go client.readPump()
}
