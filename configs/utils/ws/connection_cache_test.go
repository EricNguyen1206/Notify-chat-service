package ws

import (
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const TEST_MSG = "test message"

// Mock WebSocket connection for testing
type mockConn struct {
	messages [][]byte
	closed   bool
	mu       sync.Mutex
}

func (m *mockConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return websocket.ErrCloseSent
	}
	m.messages = append(m.messages, data)
	return nil
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) ReadMessage() (messageType int, p []byte, err error) {
	return 0, nil, nil
}

func (m *mockConn) getMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]byte, len(m.messages))
	copy(result, m.messages)
	return result
}

// Helper function to create a test client with mock connection
func createTestClient(userID uint) *Client {
	return &Client{
		ID:       userID,
		Conn:     &mockConn{messages: make([][]byte, 0)},
		Channels: make(map[uint]bool),
	}
}

// Helper function to create a test hub
func createTestHub() *Hub {
	// Create a mock Redis client (we won't actually use Redis in these tests)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	return WsNewHub(redisClient)
}

func TestNewUserConnectionCache(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	if cache == nil {
		t.Fatal("NewUserConnectionCache returned nil")
	}

	if cache.hub != hub {
		t.Error("Cache hub reference not set correctly")
	}

	if cache.userConnections == nil {
		t.Error("userConnections map not initialized")
	}

	if cache.channelUsers == nil {
		t.Error("channelUsers map not initialized")
	}

	if cache.connectionMetadata == nil {
		t.Error("connectionMetadata map not initialized")
	}
}

func TestAddConnection(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Test adding a connection
	cache.AddConnection(client)

	// Verify connection was added
	if !cache.IsUserOnline(1) {
		t.Error("User should be online after adding connection")
	}

	// Verify metadata was created
	metadata, exists := cache.GetConnectionMetadata(1)
	if !exists {
		t.Error("Connection metadata should exist")
	}

	if metadata.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", metadata.UserID)
	}

	if metadata.ConnectedAt.IsZero() {
		t.Error("ConnectedAt should be set")
	}

	if metadata.LastActivity.IsZero() {
		t.Error("LastActivity should be set")
	}

	if metadata.Channels == nil {
		t.Error("Channels map should be initialized")
	}
}

func TestRemoveConnection(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection first
	cache.AddConnection(client)
	cache.AddUserToChannel(1, 100)

	// Verify user is in channel
	users := cache.GetOnlineUsersInChannel(100)
	if len(users) != 1 || users[0] != 1 {
		t.Error("User should be in channel before removal")
	}

	// Remove connection
	cache.RemoveConnection(1)

	// Verify connection was removed
	if cache.IsUserOnline(1) {
		t.Error("User should not be online after removing connection")
	}

	// Verify metadata was removed
	_, exists := cache.GetConnectionMetadata(1)
	if exists {
		t.Error("Connection metadata should not exist after removal")
	}

	// Verify user was removed from all channels
	users = cache.GetOnlineUsersInChannel(100)
	if len(users) != 0 {
		t.Error("User should be removed from all channels")
	}
}

func TestAddUserToChannel(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection first
	cache.AddConnection(client)

	// Add user to channel
	cache.AddUserToChannel(1, 100)

	// Verify user is in channel
	users := cache.GetOnlineUsersInChannel(100)
	if len(users) != 1 || users[0] != 1 {
		t.Error("User should be in channel")
	}

	// Verify metadata was updated
	metadata, _ := cache.GetConnectionMetadata(1)
	if !metadata.Channels[100] {
		t.Error("Channel should be in user's metadata")
	}
}

func TestRemoveUserFromChannel(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection and user to channel
	cache.AddConnection(client)
	cache.AddUserToChannel(1, 100)

	// Remove user from channel
	cache.RemoveUserFromChannel(1, 100)

	// Verify user is not in channel
	users := cache.GetOnlineUsersInChannel(100)
	if len(users) != 0 {
		t.Error("User should not be in channel after removal")
	}

	// Verify metadata was updated
	metadata, _ := cache.GetConnectionMetadata(1)
	if metadata.Channels[100] {
		t.Error("Channel should not be in user's metadata")
	}
}

func TestGetOnlineUsersInChannel(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Add multiple users
	for i := uint(1); i <= 3; i++ {
		client := createTestClient(i)
		cache.AddConnection(client)
		cache.AddUserToChannel(i, 100)
	}

	// Add one user to different channel
	cache.AddUserToChannel(2, 200)

	// Test channel 100
	users := cache.GetOnlineUsersInChannel(100)
	if len(users) != 3 {
		t.Errorf("Expected 3 users in channel 100, got %d", len(users))
	}

	// Test channel 200
	users = cache.GetOnlineUsersInChannel(200)
	if len(users) != 1 || users[0] != 2 {
		t.Error("Expected user 2 in channel 200")
	}

	// Test non-existent channel
	users = cache.GetOnlineUsersInChannel(999)
	if len(users) != 0 {
		t.Error("Non-existent channel should return empty slice")
	}
}

func TestGetOnlineUsers(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Add multiple users
	expectedUsers := []uint{1, 2, 3}
	for _, userID := range expectedUsers {
		client := createTestClient(userID)
		cache.AddConnection(client)
	}

	// Get all online users
	users := cache.GetOnlineUsers()
	if len(users) != len(expectedUsers) {
		t.Errorf("Expected %d users, got %d", len(expectedUsers), len(users))
	}

	// Verify all expected users are present
	userMap := make(map[uint]bool)
	for _, userID := range users {
		userMap[userID] = true
	}

	for _, expectedUserID := range expectedUsers {
		if !userMap[expectedUserID] {
			t.Errorf("Expected user %d to be online", expectedUserID)
		}
	}
}

func TestIsUserOnline(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// User should not be online initially
	if cache.IsUserOnline(1) {
		t.Error("User should not be online initially")
	}

	// Add connection
	cache.AddConnection(client)

	// User should be online now
	if !cache.IsUserOnline(1) {
		t.Error("User should be online after adding connection")
	}

	// Remove connection
	cache.RemoveConnection(1)

	// User should not be online anymore
	if cache.IsUserOnline(1) {
		t.Error("User should not be online after removing connection")
	}
}

func TestConcurrentAccess(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Number of goroutines and operations
	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Run concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				userID := uint(goroutineID*numOperations + j)
				channelID := uint(goroutineID % 5) // Use 5 different channels

				client := createTestClient(userID)

				// Add connection
				cache.AddConnection(client)

				// Add to channel
				cache.AddUserToChannel(userID, channelID)

				// Check if online
				cache.IsUserOnline(userID)

				// Get online users in channel
				cache.GetOnlineUsersInChannel(channelID)

				// Remove from channel
				cache.RemoveUserFromChannel(userID, channelID)

				// Remove connection
				cache.RemoveConnection(userID)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify cache is empty after all operations
	onlineUsers := cache.GetOnlineUsers()
	if len(onlineUsers) != 0 {
		t.Errorf("Expected 0 online users after concurrent test, got %d", len(onlineUsers))
	}
}

func TestUpdateLastActivity(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection
	cache.AddConnection(client)

	// Get initial metadata
	metadata1, _ := cache.GetConnectionMetadata(1)
	initialActivity := metadata1.LastActivity

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update last activity
	cache.UpdateLastActivity(1)

	// Get updated metadata
	metadata2, _ := cache.GetConnectionMetadata(1)
	updatedActivity := metadata2.LastActivity

	// Verify last activity was updated
	if !updatedActivity.After(initialActivity) {
		t.Error("LastActivity should be updated to a later time")
	}
}

func TestBroadcastToChannel(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Create test clients with mock connections
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		clients[i] = createTestClient(uint(i + 1))
		cache.AddConnection(clients[i])
		cache.AddUserToChannel(uint(i+1), 100)
	}

	// Broadcast message to channel
	message := []byte(TEST_MSG)
	err := cache.BroadcastToChannel(100, message)

	if err != nil {
		t.Errorf("BroadcastToChannel should not return error: %v", err)
	}

	// Verify all clients received the message
	for i, client := range clients {
		mockConn, ok := client.Conn.(*mockConn)
		if !ok {
			t.Errorf("Client %d connection is not a mockConn", i+1)
			continue
		}
		messages := mockConn.getMessages()

		if len(messages) != 1 {
			t.Errorf("Client %d should receive 1 message, got %d", i+1, len(messages))
			continue
		}

		if string(messages[0]) != string(message) {
			t.Errorf("Client %d received wrong message: expected %s, got %s",
				i+1, string(message), string(messages[0]))
		}
	}
}

func TestBroadcastToUser(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection
	cache.AddConnection(client)

	// Broadcast message to user
	message := []byte(TEST_MSG)
	err := cache.BroadcastToUser(1, message)

	if err != nil {
		t.Errorf("BroadcastToUser should not return error: %v", err)
	}

	// Verify client received the message
	mockConn, ok := client.Conn.(*mockConn)
	if !ok {
		t.Error("Client connection is not a mockConn")
		return
	}
	messages := mockConn.getMessages()

	if len(messages) != 1 {
		t.Errorf("Client should receive 1 message, got %d", len(messages))
	} else if string(messages[0]) != string(message) {
		t.Errorf("Client received wrong message: expected %s, got %s",
			string(message), string(messages[0]))
	}
}

func TestBroadcastToUsers(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Create test clients
	userIDs := []uint{1, 2, 3}
	clients := make([]*Client, len(userIDs))

	for i, userID := range userIDs {
		clients[i] = createTestClient(userID)
		cache.AddConnection(clients[i])
	}

	// Broadcast message to specific users
	message := []byte(TEST_MSG)
	err := cache.BroadcastToUsers(userIDs, message)

	if err != nil {
		t.Errorf("BroadcastToUsers should not return error: %v", err)
	}

	// Verify all specified clients received the message
	for i, client := range clients {
		mockConn, ok := client.Conn.(*mockConn)
		if !ok {
			t.Errorf("Client %d connection is not a mockConn", userIDs[i])
			continue
		}
		messages := mockConn.getMessages()

		if len(messages) != 1 {
			t.Errorf("Client %d should receive 1 message, got %d", userIDs[i], len(messages))
			continue
		}

		if string(messages[0]) != string(message) {
			t.Errorf("Client %d received wrong message: expected %s, got %s",
				userIDs[i], string(message), string(messages[0]))
		}
	}
}

// Test broadcasting to empty channel
func TestBroadcastToEmptyChannel(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Broadcast to non-existent channel
	message := []byte(TEST_MSG)
	err := cache.BroadcastToChannel(999, message)

	if err != nil {
		t.Errorf("BroadcastToChannel to empty channel should not return error: %v", err)
	}
}

// Test broadcasting to offline user
func TestBroadcastToOfflineUser(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Broadcast to non-existent user
	message := []byte(TEST_MSG)
	err := cache.BroadcastToUser(999, message)

	if err != nil {
		t.Errorf("BroadcastToUser to offline user should not return error: %v", err)
	}
}

// Test broadcasting with failed connections
func TestBroadcastWithFailedConnections(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Create clients with one having a closed connection
	client1 := createTestClient(1)
	client2 := createTestClient(2)

	// Close client2's connection to simulate failure
	mockConn2, ok := client2.Conn.(*mockConn)
	if !ok {
		t.Fatal("Client connection is not a mockConn")
	}
	mockConn2.Close()

	cache.AddConnection(client1)
	cache.AddConnection(client2)
	cache.AddUserToChannel(1, 100)
	cache.AddUserToChannel(2, 100)

	// Broadcast message to channel
	message := []byte(TEST_MSG)
	err := cache.BroadcastToChannel(100, message)

	// Should return error from failed connection
	if err == nil {
		t.Error("BroadcastToChannel should return error when connection fails")
	}

	// Verify successful client still received message
	mockConn1, ok := client1.Conn.(*mockConn)
	if !ok {
		t.Fatal("Client connection is not a mockConn")
	}
	messages := mockConn1.getMessages()

	if len(messages) != 1 {
		t.Errorf("Successful client should receive 1 message, got %d", len(messages))
	} else if string(messages[0]) != string(message) {
		t.Errorf("Client received wrong message: expected %s, got %s",
			string(message), string(messages[0]))
	}
}

// Test broadcasting to users with mixed online/offline status
func TestBroadcastToMixedUsers(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Add only some users
	client1 := createTestClient(1)
	client3 := createTestClient(3)
	cache.AddConnection(client1)
	cache.AddConnection(client3)

	// Try to broadcast to mix of online and offline users
	userIDs := []uint{1, 2, 3} // User 2 is offline
	message := []byte(TEST_MSG)
	err := cache.BroadcastToUsers(userIDs, message)

	if err != nil {
		t.Errorf("BroadcastToUsers with mixed users should not return error: %v", err)
	}

	// Verify only online users received message
	mockConn1, ok := client1.Conn.(*mockConn)
	if !ok {
		t.Fatal("Client 1 connection is not a mockConn")
	}
	messages1 := mockConn1.getMessages()

	if len(messages1) != 1 {
		t.Errorf("Client 1 should receive 1 message, got %d", len(messages1))
	}

	mockConn3, ok := client3.Conn.(*mockConn)
	if !ok {
		t.Fatal("Client 3 connection is not a mockConn")
	}
	messages3 := mockConn3.getMessages()

	if len(messages3) != 1 {
		t.Errorf("Client 3 should receive 1 message, got %d", len(messages3))
	}
}

// Test GetOnlineUsersInChannel with disconnected users
func TestGetOnlineUsersInChannelWithDisconnectedUsers(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Add users to channel
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	cache.AddConnection(client1)
	cache.AddConnection(client2)
	cache.AddUserToChannel(1, 100)
	cache.AddUserToChannel(2, 100)

	// Remove one user's connection but leave them in channel mapping
	cache.RemoveConnection(2)

	// GetOnlineUsersInChannel should only return connected users
	users := cache.GetOnlineUsersInChannel(100)
	if len(users) != 1 || users[0] != 1 {
		t.Errorf("Expected only user 1 to be online in channel, got %v", users)
	}
}

// Test concurrent broadcasting scenarios
func TestConcurrentBroadcasting(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Create multiple clients
	numClients := 10
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = createTestClient(uint(i + 1))
		cache.AddConnection(clients[i])
		cache.AddUserToChannel(uint(i+1), 100)
	}

	// Perform concurrent broadcasts
	numBroadcasts := 5
	var wg sync.WaitGroup
	wg.Add(numBroadcasts)

	for i := 0; i < numBroadcasts; i++ {
		go func(broadcastID int) {
			defer wg.Done()
			message := []byte("broadcast " + string(rune(broadcastID+'0')))
			err := cache.BroadcastToChannel(100, message)
			if err != nil {
				t.Errorf("Concurrent broadcast %d failed: %v", broadcastID, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all clients received all messages
	for i, client := range clients {
		mockConn, ok := client.Conn.(*mockConn)
		if !ok {
			t.Errorf("Client %d connection is not a mockConn", i+1)
			continue
		}
		messages := mockConn.getMessages()

		if len(messages) != numBroadcasts {
			t.Errorf("Client %d should receive %d messages, got %d",
				i+1, numBroadcasts, len(messages))
		}
	}
}

// Test error handling during user presence checking
func TestIsUserOnlineErrorScenarios(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Test with zero user ID
	if cache.IsUserOnline(0) {
		t.Error("User ID 0 should not be considered online")
	}

	// Test with very large user ID
	if cache.IsUserOnline(^uint(0)) { // Max uint value
		t.Error("Max uint user ID should not be considered online")
	}

	// Add and remove user quickly
	client := createTestClient(1)
	cache.AddConnection(client)

	if !cache.IsUserOnline(1) {
		t.Error("User should be online after adding")
	}

	cache.RemoveConnection(1)

	if cache.IsUserOnline(1) {
		t.Error("User should not be online after removing")
	}
}

// Test GetOnlineUsersInChannel error scenarios
func TestGetOnlineUsersInChannelErrorScenarios(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Test with zero channel ID
	users := cache.GetOnlineUsersInChannel(0)
	if len(users) != 0 {
		t.Error("Channel ID 0 should return empty user list")
	}

	// Test with very large channel ID
	users = cache.GetOnlineUsersInChannel(^uint(0)) // Max uint value
	if len(users) != 0 {
		t.Error("Max uint channel ID should return empty user list")
	}

	// Test after adding and removing all users from channel
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	cache.AddConnection(client1)
	cache.AddConnection(client2)
	cache.AddUserToChannel(1, 100)
	cache.AddUserToChannel(2, 100)

	// Verify users are in channel
	users = cache.GetOnlineUsersInChannel(100)
	if len(users) != 2 {
		t.Errorf("Expected 2 users in channel, got %d", len(users))
	}

	// Remove all users
	cache.RemoveUserFromChannel(1, 100)
	cache.RemoveUserFromChannel(2, 100)

	// Channel should be cleaned up and return empty list
	users = cache.GetOnlineUsersInChannel(100)
	if len(users) != 0 {
		t.Error("Channel should return empty list after all users removed")
	}
}

// Test broadcasting with nil message
func TestBroadcastWithNilMessage(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	cache.AddConnection(client)
	cache.AddUserToChannel(1, 100)

	// Test broadcasting nil message
	err := cache.BroadcastToChannel(100, nil)
	if err != nil {
		t.Errorf("Broadcasting nil message should not return error: %v", err)
	}

	// Verify client received nil message
	mockConn, ok := client.Conn.(*mockConn)
	if !ok {
		t.Fatal("Client connection is not a mockConn")
	}
	messages := mockConn.getMessages()

	if len(messages) != 1 {
		t.Errorf("Client should receive 1 message, got %d", len(messages))
	} else if messages[0] != nil {
		t.Error("Client should receive nil message")
	}
}

// Test broadcasting with empty message
func TestBroadcastWithEmptyMessage(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	cache.AddConnection(client)
	cache.AddUserToChannel(1, 100)

	// Test broadcasting empty message
	emptyMessage := []byte{}
	err := cache.BroadcastToChannel(100, emptyMessage)
	if err != nil {
		t.Errorf("Broadcasting empty message should not return error: %v", err)
	}

	// Verify client received empty message
	mockConn, ok := client.Conn.(*mockConn)
	if !ok {
		t.Fatal("Client connection is not a mockConn")
	}
	messages := mockConn.getMessages()

	if len(messages) != 1 {
		t.Errorf("Client should receive 1 message, got %d", len(messages))
	} else if len(messages[0]) != 0 {
		t.Error("Client should receive empty message")
	}
}

// Test channel subscription cache updates through WsAddChannel and WsRemoveChannel
func TestChannelSubscriptionCacheUpdates(t *testing.T) {
	hub := createTestHub()
	client := createTestClient(1)

	// Add connection to cache first
	hub.ConnectionCache.AddConnection(client)

	// Test WsAddChannel updates cache
	client.WsAddChannel(100, hub)

	// Verify user is in channel cache
	users := hub.ConnectionCache.GetOnlineUsersInChannel(100)
	if len(users) != 1 || users[0] != 1 {
		t.Error("WsAddChannel should update connection cache")
	}

	// Verify metadata is updated
	metadata, exists := hub.ConnectionCache.GetConnectionMetadata(1)
	if !exists {
		t.Error("Connection metadata should exist")
	}
	if !metadata.Channels[100] {
		t.Error("Channel should be in user's metadata after WsAddChannel")
	}

	// Test WsRemoveChannel updates cache
	client.WsRemoveChannel(100, hub)

	// Verify user is removed from channel cache
	users = hub.ConnectionCache.GetOnlineUsersInChannel(100)
	if len(users) != 0 {
		t.Error("WsRemoveChannel should update connection cache")
	}

	// Verify metadata is updated
	metadata, exists = hub.ConnectionCache.GetConnectionMetadata(1)
	if !exists {
		t.Error("Connection metadata should still exist")
	}
	if metadata.Channels[100] {
		t.Error("Channel should be removed from user's metadata after WsRemoveChannel")
	}
}

// Test cache consistency when clients join/leave channels
func TestCacheConsistencyOnChannelOperations(t *testing.T) {
	hub := createTestHub()

	// Create multiple clients
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		clients[i] = createTestClient(uint(i + 1))
		hub.ConnectionCache.AddConnection(clients[i])
	}

	// Test multiple users joining same channel
	for i, client := range clients {
		client.WsAddChannel(200, hub)

		// Verify incremental addition
		users := hub.ConnectionCache.GetOnlineUsersInChannel(200)
		if len(users) != i+1 {
			t.Errorf("Expected %d users in channel after adding client %d, got %d", i+1, i+1, len(users))
		}

		// Verify user is in the list
		found := false
		for _, userID := range users {
			if userID == uint(i+1) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("User %d should be in channel after WsAddChannel", i+1)
		}
	}

	// Test users joining multiple channels
	clients[0].WsAddChannel(201, hub)
	clients[0].WsAddChannel(202, hub)

	// Verify user is in multiple channels
	metadata, _ := hub.ConnectionCache.GetConnectionMetadata(1)
	if !metadata.Channels[200] || !metadata.Channels[201] || !metadata.Channels[202] {
		t.Error("User should be in all subscribed channels")
	}

	// Test partial channel leaving
	clients[1].WsRemoveChannel(200, hub)

	// Verify user 2 is removed but others remain
	users := hub.ConnectionCache.GetOnlineUsersInChannel(200)
	if len(users) != 2 {
		t.Errorf("Expected 2 users in channel after removing one, got %d", len(users))
	}

	// Verify user 2 is not in the list
	for _, userID := range users {
		if userID == 2 {
			t.Error("User 2 should not be in channel after WsRemoveChannel")
		}
	}

	// Test all users leaving channel
	clients[0].WsRemoveChannel(200, hub)
	clients[2].WsRemoveChannel(200, hub)

	// Verify channel is cleaned up
	users = hub.ConnectionCache.GetOnlineUsersInChannel(200)
	if len(users) != 0 {
		t.Error("Channel should be empty after all users leave")
	}
}

// Test concurrent channel subscription operations
func TestConcurrentChannelSubscriptions(t *testing.T) {
	hub := createTestHub()

	const numClients = 10
	const numChannels = 5

	// Create clients
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = createTestClient(uint(i + 1))
		hub.ConnectionCache.AddConnection(clients[i])
	}

	var wg sync.WaitGroup

	// Concurrent channel subscriptions
	for i, client := range clients {
		wg.Add(1)
		go func(clientIndex int, c *Client) {
			defer wg.Done()

			// Subscribe to multiple channels
			for j := 0; j < numChannels; j++ {
				if (clientIndex+j)%2 == 0 { // Subscribe to some channels based on pattern
					c.WsAddChannel(uint(j+1), hub)
				}
			}
		}(i, client)
	}

	wg.Wait()

	// Verify all subscriptions are consistent
	for channelID := uint(1); channelID <= numChannels; channelID++ {
		users := hub.ConnectionCache.GetOnlineUsersInChannel(channelID)

		// Verify each user in channel has the channel in their metadata
		for _, userID := range users {
			metadata, exists := hub.ConnectionCache.GetConnectionMetadata(userID)
			if !exists {
				t.Errorf("Metadata should exist for user %d", userID)
				continue
			}
			if !metadata.Channels[channelID] {
				t.Errorf("Channel %d should be in metadata for user %d", channelID, userID)
			}
		}
	}

	// Concurrent channel unsubscriptions
	for i, client := range clients {
		wg.Add(1)
		go func(clientIndex int, c *Client) {
			defer wg.Done()

			// Unsubscribe from some channels
			for j := 0; j < numChannels; j++ {
				if (clientIndex+j)%3 == 0 { // Unsubscribe from some channels
					c.WsRemoveChannel(uint(j+1), hub)
				}
			}
		}(i, client)
	}

	wg.Wait()

	// Verify consistency after unsubscriptions
	for channelID := uint(1); channelID <= numChannels; channelID++ {
		users := hub.ConnectionCache.GetOnlineUsersInChannel(channelID)

		// Verify each user in channel still has the channel in their metadata
		for _, userID := range users {
			metadata, exists := hub.ConnectionCache.GetConnectionMetadata(userID)
			if !exists {
				t.Errorf("Metadata should exist for user %d", userID)
				continue
			}
			if !metadata.Channels[channelID] {
				t.Errorf("Channel %d should still be in metadata for user %d", channelID, userID)
			}
		}
	}
}

// Test channel subscription with nil hub (error case)
func TestChannelSubscriptionWithNilHub(t *testing.T) {
	client := createTestClient(1)

	// Test WsAddChannel with nil hub (should not panic)
	client.WsAddChannel(100, nil)

	// Verify channel is added to client but not to cache
	if !client.Channels[100] {
		t.Error("Channel should be added to client even with nil hub")
	}

	// Test WsRemoveChannel with nil hub (should not panic)
	client.WsRemoveChannel(100, nil)

	// Verify channel is removed from client
	if client.Channels[100] {
		t.Error("Channel should be removed from client even with nil hub")
	}
}

// Test channel subscription with nil connection cache
func TestChannelSubscriptionWithNilCache(t *testing.T) {
	hub := createTestHub()
	hub.ConnectionCache = nil // Set cache to nil
	client := createTestClient(1)

	// Test WsAddChannel with nil cache (should not panic)
	client.WsAddChannel(100, hub)

	// Verify channel is added to client
	if !client.Channels[100] {
		t.Error("Channel should be added to client even with nil cache")
	}

	// Test WsRemoveChannel with nil cache (should not panic)
	client.WsRemoveChannel(100, hub)

	// Verify channel is removed from client
	if client.Channels[100] {
		t.Error("Channel should be removed from client even with nil cache")
	}
}
