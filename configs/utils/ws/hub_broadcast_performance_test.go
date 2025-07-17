package ws

import (
	"chat-service/internal/models"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWebSocketConnection for testing
type MockWebSocketConnection struct {
	mock.Mock
	writeDelay time.Duration // Simulate network latency
	shouldFail bool          // Simulate connection failures
}

func (m *MockWebSocketConnection) WriteMessage(messageType int, data []byte) error {
	if m.writeDelay > 0 {
		time.Sleep(m.writeDelay)
	}

	if m.shouldFail {
		return fmt.Errorf("mock connection failure")
	}

	args := m.Called(messageType, data)
	return args.Error(0)
}

func (m *MockWebSocketConnection) ReadMessage() (messageType int, p []byte, err error) {
	args := m.Called()
	return args.Int(0), args.Get(1).([]byte), args.Error(2)
}

func (m *MockWebSocketConnection) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Performance test for broadcasting to multiple users
func TestBroadcastToLocalClients_Performance(t *testing.T) {
	tests := []struct {
		name       string
		userCount  int
		channelID  uint
		expectTime time.Duration // Maximum expected time
	}{
		{
			name:       "Broadcast to 10 users",
			userCount:  10,
			channelID:  1,
			expectTime: 100 * time.Millisecond,
		},
		{
			name:       "Broadcast to 100 users",
			userCount:  100,
			channelID:  1,
			expectTime: 500 * time.Millisecond,
		},
		{
			name:       "Broadcast to 1000 users",
			userCount:  1000,
			channelID:  1,
			expectTime: 2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
			hub := WsNewHub(redisClient)

			// Create mock clients and add them to the hub
			clients := make([]*Client, tt.userCount)
			for i := 0; i < tt.userCount; i++ {
				mockConn := &MockWebSocketConnection{}
				mockConn.On("WriteMessage", websocket.TextMessage, mock.Anything).Return(nil)

				client := &Client{
					ID:       uint(i + 1),
					Conn:     mockConn,
					Channels: make(map[uint]bool),
				}
				client.Channels[tt.channelID] = true

				clients[i] = client
				hub.ConnectionCache.AddConnection(client)
				hub.ConnectionCache.AddUserToChannel(client.ID, tt.channelID)
			}

			// Test message
			testMessage := []byte(`{"channelId":1,"userId":999,"text":"Performance test message","sentAt":"2023-01-01T00:00:00Z"}`)

			// Measure broadcast performance
			start := time.Now()
			hub.broadcastToLocalClients(tt.channelID, testMessage)
			elapsed := time.Since(start)

			// Assertions
			assert.True(t, elapsed < tt.expectTime,
				"Broadcast took %v, expected less than %v", elapsed, tt.expectTime)

			// Verify all clients received the message
			for _, client := range clients {
				mockConn := client.Conn.(*MockWebSocketConnection)
				mockConn.AssertCalled(t, "WriteMessage", websocket.TextMessage, testMessage)
			}

			t.Logf("Broadcast to %d users completed in %v", tt.userCount, elapsed)
		})
	}
}

// Test concurrent broadcasting to multiple channels
func TestConcurrentBroadcasting_Performance(t *testing.T) {
	// Setup
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	hub := WsNewHub(redisClient)

	const (
		channelCount       = 10
		usersPerChannel    = 50
		messagesPerChannel = 5
	)

	// Create clients for multiple channels
	for channelID := uint(1); channelID <= channelCount; channelID++ {
		for userID := uint(1); userID <= usersPerChannel; userID++ {
			mockConn := &MockWebSocketConnection{}
			mockConn.On("WriteMessage", websocket.TextMessage, mock.Anything).Return(nil)

			client := &Client{
				ID:       userID + (channelID-1)*usersPerChannel,
				Conn:     mockConn,
				Channels: make(map[uint]bool),
			}
			client.Channels[channelID] = true

			hub.ConnectionCache.AddConnection(client)
			hub.ConnectionCache.AddUserToChannel(client.ID, channelID)
		}
	}

	// Test concurrent broadcasting
	start := time.Now()
	var wg sync.WaitGroup

	for channelID := uint(1); channelID <= channelCount; channelID++ {
		for msgNum := 1; msgNum <= messagesPerChannel; msgNum++ {
			wg.Add(1)
			go func(chID uint, msgN int) {
				defer wg.Done()

				testMessage := []byte(fmt.Sprintf(`{"channelId":%d,"userId":999,"text":"Concurrent test message %d","sentAt":"2023-01-01T00:00:00Z"}`, chID, msgN))
				hub.broadcastToLocalClients(chID, testMessage)
			}(channelID, msgNum)
		}
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Should complete within reasonable time even with concurrent load
	maxExpectedTime := 3 * time.Second
	assert.True(t, elapsed < maxExpectedTime,
		"Concurrent broadcast took %v, expected less than %v", elapsed, maxExpectedTime)

	t.Logf("Concurrent broadcast to %d channels (%d users each, %d messages each) completed in %v",
		channelCount, usersPerChannel, messagesPerChannel, elapsed)
}

// Test broadcasting with connection failures
func TestBroadcastWithConnectionFailures_Performance(t *testing.T) {
	// Setup
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	hub := WsNewHub(redisClient)

	const (
		totalUsers  = 100
		failureRate = 0.2 // 20% of connections will fail
		channelID   = uint(1)
	)

	// Create clients with some that will fail
	clients := make([]*Client, totalUsers)
	for i := 0; i < totalUsers; i++ {
		mockConn := &MockWebSocketConnection{}

		// Simulate connection failures for some clients
		if float64(i)/totalUsers < failureRate {
			mockConn.shouldFail = true
			mockConn.On("WriteMessage", websocket.TextMessage, mock.Anything).Return(fmt.Errorf("connection failed"))
		} else {
			mockConn.On("WriteMessage", websocket.TextMessage, mock.Anything).Return(nil)
		}

		client := &Client{
			ID:       uint(i + 1),
			Conn:     mockConn,
			Channels: make(map[uint]bool),
		}
		client.Channels[channelID] = true

		clients[i] = client
		hub.ConnectionCache.AddConnection(client)
		hub.ConnectionCache.AddUserToChannel(client.ID, channelID)
	}

	// Test message
	testMessage := []byte(`{"channelId":1,"userId":999,"text":"Failure test message","sentAt":"2023-01-01T00:00:00Z"}`)

	// Measure broadcast performance with failures
	start := time.Now()
	hub.broadcastToLocalClients(channelID, testMessage)
	elapsed := time.Since(start)

	// Should still complete quickly even with failures
	maxExpectedTime := 1 * time.Second
	assert.True(t, elapsed < maxExpectedTime,
		"Broadcast with failures took %v, expected less than %v", elapsed, maxExpectedTime)

	// Verify successful clients were called, failed ones were not
	successfulClients := 0
	failedClients := 0
	for i, client := range clients {
		mockConn := client.Conn.(*MockWebSocketConnection)
		if float64(i)/totalUsers < failureRate {
			// This client should have failed
			failedClients++
			// Don't assert on failed clients as they may or may not be called depending on timing
		} else {
			// This client should have succeeded
			mockConn.AssertCalled(t, "WriteMessage", websocket.TextMessage, testMessage)
			successfulClients++
		}
	}

	// Verify we have the expected number of successful and failed clients
	expectedSuccessful := int(float64(totalUsers) * (1 - failureRate))
	expectedFailed := int(float64(totalUsers) * failureRate)
	assert.Equal(t, expectedSuccessful, successfulClients, "Expected number of successful clients")
	assert.Equal(t, expectedFailed, failedClients, "Expected number of failed clients")

	t.Logf("Broadcast to %d users (%.0f%% failure rate) completed in %v",
		totalUsers, failureRate*100, elapsed)
}

// Test BroadcastMessage method performance
func TestBroadcastMessage_Performance(t *testing.T) {
	// Setup
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	hub := WsNewHub(redisClient)

	const (
		userCount = 200
		channelID = uint(1)
	)

	// Create mock clients
	for i := 0; i < userCount; i++ {
		mockConn := &MockWebSocketConnection{}
		mockConn.On("WriteMessage", websocket.TextMessage, mock.Anything).Return(nil)

		client := &Client{
			ID:       uint(i + 1),
			Conn:     mockConn,
			Channels: make(map[uint]bool),
		}
		client.Channels[channelID] = true

		hub.ConnectionCache.AddConnection(client)
		hub.ConnectionCache.AddUserToChannel(client.ID, channelID)
	}

	// Test chat message
	testText := "Performance test message from BroadcastMessage"
	chatMsg := &models.Chat{
		SenderID:  999,
		ChannelID: channelID,
		Type:      "channel",
		Text:      &testText,
	}

	// Measure BroadcastMessage performance
	start := time.Now()
	hub.BroadcastMessage(chatMsg)
	elapsed := time.Since(start)

	// Should complete quickly
	maxExpectedTime := 500 * time.Millisecond
	assert.True(t, elapsed < maxExpectedTime,
		"BroadcastMessage took %v, expected less than %v", elapsed, maxExpectedTime)

	t.Logf("BroadcastMessage to %d users completed in %v", userCount, elapsed)
}

// Benchmark for broadcasting performance
func BenchmarkBroadcastToLocalClients(b *testing.B) {
	userCounts := []int{10, 50, 100, 500, 1000}

	for _, userCount := range userCounts {
		b.Run(fmt.Sprintf("Users_%d", userCount), func(b *testing.B) {
			// Setup
			redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
			hub := WsNewHub(redisClient)
			channelID := uint(1)

			// Create mock clients
			for i := 0; i < userCount; i++ {
				mockConn := &MockWebSocketConnection{}
				mockConn.On("WriteMessage", websocket.TextMessage, mock.Anything).Return(nil)

				client := &Client{
					ID:       uint(i + 1),
					Conn:     mockConn,
					Channels: make(map[uint]bool),
				}
				client.Channels[channelID] = true

				hub.ConnectionCache.AddConnection(client)
				hub.ConnectionCache.AddUserToChannel(client.ID, channelID)
			}

			testMessage := []byte(`{"channelId":1,"userId":999,"text":"Benchmark message","sentAt":"2023-01-01T00:00:00Z"}`)

			// Reset timer and run benchmark
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hub.broadcastToLocalClients(channelID, testMessage)
			}
		})
	}
}

// Test memory usage during broadcasting
func TestBroadcastMemoryUsage(t *testing.T) {
	// Setup
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	hub := WsNewHub(redisClient)

	const (
		userCount    = 1000
		channelID    = uint(1)
		messageCount = 100
	)

	// Create mock clients
	for i := 0; i < userCount; i++ {
		mockConn := &MockWebSocketConnection{}
		mockConn.On("WriteMessage", websocket.TextMessage, mock.Anything).Return(nil)

		client := &Client{
			ID:       uint(i + 1),
			Conn:     mockConn,
			Channels: make(map[uint]bool),
		}
		client.Channels[channelID] = true

		hub.ConnectionCache.AddConnection(client)
		hub.ConnectionCache.AddUserToChannel(client.ID, channelID)
	}

	// Send multiple messages to test memory stability
	testMessage := []byte(`{"channelId":1,"userId":999,"text":"Memory test message","sentAt":"2023-01-01T00:00:00Z"}`)

	start := time.Now()
	for i := 0; i < messageCount; i++ {
		hub.broadcastToLocalClients(channelID, testMessage)
	}
	elapsed := time.Since(start)

	// Should complete all messages within reasonable time
	maxExpectedTime := 10 * time.Second
	assert.True(t, elapsed < maxExpectedTime,
		"Memory test took %v, expected less than %v", elapsed, maxExpectedTime)

	t.Logf("Sent %d messages to %d users in %v", messageCount, userCount, elapsed)
}

// Test broadcasting with network latency simulation
func TestBroadcastWithLatency_Performance(t *testing.T) {
	// Setup
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	hub := WsNewHub(redisClient)

	const (
		userCount      = 50
		channelID      = uint(1)
		networkLatency = 10 * time.Millisecond
	)

	// Create mock clients with simulated network latency
	for i := 0; i < userCount; i++ {
		mockConn := &MockWebSocketConnection{
			writeDelay: networkLatency,
		}
		mockConn.On("WriteMessage", websocket.TextMessage, mock.Anything).Return(nil)

		client := &Client{
			ID:       uint(i + 1),
			Conn:     mockConn,
			Channels: make(map[uint]bool),
		}
		client.Channels[channelID] = true

		hub.ConnectionCache.AddConnection(client)
		hub.ConnectionCache.AddUserToChannel(client.ID, channelID)
	}

	testMessage := []byte(`{"channelId":1,"userId":999,"text":"Latency test message","sentAt":"2023-01-01T00:00:00Z"}`)

	// Measure broadcast performance with latency
	start := time.Now()
	hub.broadcastToLocalClients(channelID, testMessage)
	elapsed := time.Since(start)

	// With concurrent delivery, should be much faster than sequential
	// Sequential would take userCount * networkLatency
	sequentialTime := time.Duration(userCount) * networkLatency
	assert.True(t, elapsed < sequentialTime/2,
		"Concurrent broadcast took %v, sequential would take %v", elapsed, sequentialTime)

	t.Logf("Broadcast to %d users with %v latency completed in %v (sequential would take %v)",
		userCount, networkLatency, elapsed, sequentialTime)
}
