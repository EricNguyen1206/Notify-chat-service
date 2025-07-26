package ws

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// mockConn implements the WebSocketConnection interface for testing
type mockConn struct {
	mu       sync.Mutex
	messages [][]byte
	closed   bool
}

func (m *mockConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosedConnection
	}
	m.messages = append(m.messages, data)
	return nil
}

func (m *mockConn) ReadMessage() (messageType int, p []byte, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, nil, ErrClosedConnection
	}
	return 1, []byte(`{"type":"heartbeat-response"}`), nil
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) getMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]byte, len(m.messages))
	copy(result, m.messages)
	return result
}

// ErrClosedConnection is returned when attempting to use a closed connection
var ErrClosedConnection = &mockError{"connection closed"}

type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}

// Helper functions for tests
func createTestHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan ChannelMessage),
		Metrics:    NewConnectionMetrics(100),
	}
}

func createTestClient(userID uint) *Client {
	return &Client{
		ID:       userID,
		Conn:     &mockConn{messages: make([][]byte, 0)},
		Channels: make(map[uint]bool),
	}
}

// isRedisAvailable checks if Redis is available for testing
func isRedisAvailable() bool {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Default Redis address
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	return err == nil
}

// createTestHubWithRedis creates a test hub with a real Redis client
func createTestHubWithRedis() *Hub {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Default Redis address
	})

	hub := &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan ChannelMessage),
		Redis:      client,
	}

	return hub
}
