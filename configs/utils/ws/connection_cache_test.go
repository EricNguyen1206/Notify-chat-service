package ws

import (
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockWebSocketConn is a mock implementation of websocket.Conn for testing
type MockWebSocketConn struct {
	messages [][]byte
	closed   bool
	mu       sync.Mutex
}

func (m *MockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return websocket.ErrCloseSent
	}

	m.messages = append(m.messages, data)
	return nil
}

func (m *MockWebSocketConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return nil
}

func (m *MockWebSocketConn) GetMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([][]byte, len(m.messages))
	copy(result, m.messages)
	return result
}

func (m *MockWebSocketConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.closed
}

// Helper function to create a test client with mock connection
func createTestClient(userID uint) *Client {
	return &Client{
		ID:       userID,
		Conn:     &MockWebSocketConn{},
		Channels: make(map[uint]bool),
	}
}

func TestNewUserConnectionCache(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)

	assert.NotNil(t, cache)
	assert.Equal(t, hub, cache.hub)
	assert.NotNil(t, cache.userConnections)
	assert.NotNil(t, cache.channelUsers)
	assert.NotNil(t, cache.connectionMetadata)
}

func TestAddConnection(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Test adding a connection
	cache.AddConnection(1, client)

	// Verify connection was added
	assert.True(t, cache.IsUserOnline(1))

	// Verify metadata was created
	metadata, exists := cache.GetConnectionMetadata(1)
	require.True(t, exists)
	assert.Equal(t, uint(1), metadata.UserID)
	assert.NotZero(t, metadata.ConnectedAt)
	assert.NotZero(t, metadata.LastActivity)
	assert.NotNil(t, metadata.Channels)
}

func TestRemoveConnection(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection first
	cache.AddConnection(1, client)
	cache.AddUserToChannel(1, 100)

	// Verify connection exists
	assert.True(t, cache.IsUserOnline(1))
	assert.Contains(t, cache.GetOnlineUsersInChannel(100), uint(1))

	// Remove connection
	cache.RemoveConnection(1)

	// Verify connection was removed
	assert.False(t, cache.IsUserOnline(1))
	assert.NotContains(t, cache.GetOnlineUsersInChannel(100), uint(1))

	// Verify metadata was removed
	_, exists := cache.GetConnectionMetadata(1)
	assert.False(t, exists)
}

func TestAddUserToChannel(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection first
	cache.AddConnection(1, client)

	// Add user to channel
	cache.AddUserToChannel(1, 100)

	// Verify user is in channel
	users := cache.GetOnlineUsersInChannel(100)
	assert.Contains(t, users, uint(1))

	// Verify metadata was updated
	metadata, exists := cache.GetConnectionMetadata(1)
	require.True(t, exists)
	assert.True(t, metadata.Channels[100])
}

func TestRemoveUserFromChannel(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection and channel subscription
	cache.AddConnection(1, client)
	cache.AddUserToChannel(1, 100)

	// Verify user is in channel
	assert.Contains(t, cache.GetOnlineUsersInChannel(100), uint(1))

	// Remove user from channel
	cache.RemoveUserFromChannel(1, 100)

	// Verify user is no longer in channel
	assert.NotContains(t, cache.GetOnlineUsersInChannel(100), uint(1))

	// Verify metadata was updated
	metadata, exists := cache.GetConnectionMetadata(1)
	require.True(t, exists)
	assert.False(t, metadata.Channels[100])
}

func TestGetOnlineUsers(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)

	// Initially no users online
	users := cache.GetOnlineUsers()
	assert.Empty(t, users)

	// Add multiple users
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	client3 := createTestClient(3)

	cache.AddConnection(1, client1)
	cache.AddConnection(2, client2)
	cache.AddConnection(3, client3)

	// Verify all users are returned
	users = cache.GetOnlineUsers()
	assert.Len(t, users, 3)
	assert.Contains(t, users, uint(1))
	assert.Contains(t, users, uint(2))
	assert.Contains(t, users, uint(3))
}

func TestGetOnlineUsersInChannel(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)

	// Add multiple users
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	client3 := createTestClient(3)

	cache.AddConnection(1, client1)
	cache.AddConnection(2, client2)
	cache.AddConnection(3, client3)

	// Add users to different channels
	cache.AddUserToChannel(1, 100)
	cache.AddUserToChannel(2, 100)
	cache.AddUserToChannel(3, 200)

	// Test channel 100
	users100 := cache.GetOnlineUsersInChannel(100)
	assert.Len(t, users100, 2)
	assert.Contains(t, users100, uint(1))
	assert.Contains(t, users100, uint(2))
	assert.NotContains(t, users100, uint(3))

	// Test channel 200
	users200 := cache.GetOnlineUsersInChannel(200)
	assert.Len(t, users200, 1)
	assert.Contains(t, users200, uint(3))

	// Test non-existent channel
	users300 := cache.GetOnlineUsersInChannel(300)
	assert.Empty(t, users300)
}

func TestIsUserOnline(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// User not online initially
	assert.False(t, cache.IsUserOnline(1))

	// Add connection
	cache.AddConnection(1, client)
	assert.True(t, cache.IsUserOnline(1))

	// Remove connection
	cache.RemoveConnection(1)
	assert.False(t, cache.IsUserOnline(1))
}

func TestBroadcastToUser(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection
	cache.AddConnection(1, client)

	// Broadcast message
	message := []byte("test message")
	err := cache.BroadcastToUser(1, message)
	assert.NoError(t, err)

	// Verify message was sent
	mockConn := client.Conn.(*MockWebSocketConn)
	messages := mockConn.GetMessages()
	assert.Len(t, messages, 1)
	assert.Equal(t, message, messages[0])

	// Test broadcasting to offline user
	err = cache.BroadcastToUser(999, message)
	assert.NoError(t, err) // Should not error, just skip
}

func TestBroadcastToChannel(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)

	// Add multiple users
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	client3 := createTestClient(3)

	cache.AddConnection(1, client1)
	cache.AddConnection(2, client2)
	cache.AddConnection(3, client3)

	// Add users to channel
	cache.AddUserToChannel(1, 100)
	cache.AddUserToChannel(2, 100)
	// User 3 not in channel 100

	// Broadcast message to channel
	message := []byte("channel message")
	err := cache.BroadcastToChannel(100, message)
	assert.NoError(t, err)

	// Verify message was sent to users in channel
	mockConn1 := client1.Conn.(*MockWebSocketConn)
	mockConn2 := client2.Conn.(*MockWebSocketConn)
	mockConn3 := client3.Conn.(*MockWebSocketConn)

	messages1 := mockConn1.GetMessages()
	messages2 := mockConn2.GetMessages()
	messages3 := mockConn3.GetMessages()

	assert.Len(t, messages1, 1)
	assert.Len(t, messages2, 1)
	assert.Len(t, messages3, 0) // User 3 not in channel

	assert.Equal(t, message, messages1[0])
	assert.Equal(t, message, messages2[0])
}

func TestBroadcastToUsers(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)

	// Add multiple users
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	client3 := createTestClient(3)

	cache.AddConnection(1, client1)
	cache.AddConnection(2, client2)
	cache.AddConnection(3, client3)

	// Broadcast to specific users
	message := []byte("targeted message")
	userIDs := []uint{1, 3, 999} // Include non-existent user
	err := cache.BroadcastToUsers(userIDs, message)
	assert.NoError(t, err)

	// Verify message was sent to specified users
	mockConn1 := client1.Conn.(*MockWebSocketConn)
	mockConn2 := client2.Conn.(*MockWebSocketConn)
	mockConn3 := client3.Conn.(*MockWebSocketConn)

	messages1 := mockConn1.GetMessages()
	messages2 := mockConn2.GetMessages()
	messages3 := mockConn3.GetMessages()

	assert.Len(t, messages1, 1)
	assert.Len(t, messages2, 0) // User 2 not targeted
	assert.Len(t, messages3, 1)

	assert.Equal(t, message, messages1[0])
	assert.Equal(t, message, messages3[0])
}

func TestConcurrentAccess(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)

	// Test concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Concurrent add/remove operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				userID := uint(id*numOperations + j)
				client := createTestClient(userID)

				// Add connection
				cache.AddConnection(userID, client)

				// Add to channel
				cache.AddUserToChannel(userID, 100)

				// Check if online
				cache.IsUserOnline(userID)

				// Get online users
				cache.GetOnlineUsers()
				cache.GetOnlineUsersInChannel(100)

				// Remove from channel
				cache.RemoveUserFromChannel(userID, 100)

				// Remove connection
				cache.RemoveConnection(userID)
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is empty after all operations
	assert.Empty(t, cache.GetOnlineUsers())
	assert.Empty(t, cache.GetOnlineUsersInChannel(100))
}

func TestUpdateLastActivity(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection
	cache.AddConnection(1, client)

	// Get initial metadata
	metadata1, exists := cache.GetConnectionMetadata(1)
	require.True(t, exists)
	initialActivity := metadata1.LastActivity

	// Wait a bit and update activity
	time.Sleep(10 * time.Millisecond)
	cache.UpdateLastActivity(1)

	// Verify activity was updated
	metadata2, exists := cache.GetConnectionMetadata(1)
	require.True(t, exists)
	assert.True(t, metadata2.LastActivity.After(initialActivity))
}

func TestChannelCleanup(t *testing.T) {
	hub := &Hub{}
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection and channel subscription
	cache.AddConnection(1, client)
	cache.AddUserToChannel(1, 100)

	// Verify channel exists
	users := cache.GetOnlineUsersInChannel(100)
	assert.Len(t, users, 1)

	// Remove user from channel (should clean up empty channel)
	cache.RemoveUserFromChannel(1, 100)

	// Verify channel is cleaned up
	users = cache.GetOnlineUsersInChannel(100)
	assert.Empty(t, users)

	// Verify internal channel map is cleaned up
	cache.mu.RLock()
	_, exists := cache.channelUsers[100]
	cache.mu.RUnlock()
	assert.False(t, exists)
}
