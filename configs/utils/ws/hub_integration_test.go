package ws

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestHubIntegrationWithConnectionCache tests the integration between Hub and ConnectionCache
func TestHubIntegrationWithConnectionCache(t *testing.T) {
	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()

	// Create test clients
	client1 := createTestClient(1)
	client2 := createTestClient(2)

	// Test client registration
	t.Run("ClientRegistration", func(t *testing.T) {
		// Register clients through the hub's register channel
		hub.Register <- client1
		hub.Register <- client2

		// Process the registration (simulating the hub's run loop)
		processRegistration(hub, client1)
		processRegistration(hub, client2)

		// Verify clients are in the hub's clients map
		if _, ok := hub.Clients[client1]; !ok {
			t.Error("Client 1 should be in hub's clients map")
		}
		if _, ok := hub.Clients[client2]; !ok {
			t.Error("Client 2 should be in hub's clients map")
		}

		// Verify clients are in the connection cache
		if !hub.ConnectionCache.IsUserOnline(1) {
			t.Error("Client 1 should be in connection cache")
		}
		if !hub.ConnectionCache.IsUserOnline(2) {
			t.Error("Client 2 should be in connection cache")
		}
	})

	// Test channel subscription
	t.Run("ChannelSubscription", func(t *testing.T) {
		// Subscribe clients to channels
		client1.WsAddChannel(100, hub)
		client2.WsAddChannel(100, hub)
		client2.WsAddChannel(200, hub)

		// Verify channel subscriptions in clients
		if _, ok := client1.Channels[100]; !ok {
			t.Error("Client 1 should be subscribed to channel 100")
		}
		if _, ok := client2.Channels[100]; !ok {
			t.Error("Client 2 should be subscribed to channel 100")
		}
		if _, ok := client2.Channels[200]; !ok {
			t.Error("Client 2 should be subscribed to channel 200")
		}

		// Verify channel subscriptions in connection cache
		users100 := hub.ConnectionCache.GetOnlineUsersInChannel(100)
		if len(users100) != 2 {
			t.Errorf("Expected 2 users in channel 100, got %d", len(users100))
		}

		users200 := hub.ConnectionCache.GetOnlineUsersInChannel(200)
		if len(users200) != 1 {
			t.Errorf("Expected 1 user in channel 200, got %d", len(users200))
		}
	})

	// Test message broadcasting
	t.Run("MessageBroadcasting", func(t *testing.T) {
		// Create test message
		chatMsg := &MockChat{
			ChannelID: 100,
			UserID:    1,
			Text:      "Test message",
			SentAt:    time.Now().Format(time.RFC3339),
		}

		// Broadcast message
		hub.BroadcastMockMessage(chatMsg)

		// Give time for message to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify both clients received the message
		mockConn1 := client1.Conn.(*mockConn)
		messages1 := mockConn1.getMessages()
		if len(messages1) != 1 {
			t.Errorf("Client 1 should receive 1 message, got %d", len(messages1))
		}

		mockConn2 := client2.Conn.(*mockConn)
		messages2 := mockConn2.getMessages()
		if len(messages2) != 1 {
			t.Errorf("Client 2 should receive 1 message, got %d", len(messages2))
		}
	})

	// Test client unregistration
	t.Run("ClientUnregistration", func(t *testing.T) {
		// Unregister client1
		hub.Unregister <- client1

		// Process the unregistration (simulating the hub's run loop)
		processUnregistration(hub, client1)

		// Verify client1 is removed from the hub's clients map
		if _, ok := hub.Clients[client1]; ok {
			t.Error("Client 1 should be removed from hub's clients map")
		}

		// Verify client1 is removed from the connection cache
		if hub.ConnectionCache.IsUserOnline(1) {
			t.Error("Client 1 should be removed from connection cache")
		}

		// Verify client1 is removed from channel subscriptions
		users100 := hub.ConnectionCache.GetOnlineUsersInChannel(100)
		for _, userID := range users100 {
			if userID == 1 {
				t.Error("Client 1 should be removed from channel 100 subscriptions")
			}
		}
	})
}

// TestRedisIntegrationWithHub tests the integration between Redis and Hub with connection cache
func TestRedisIntegrationWithHub(t *testing.T) {
	// Skip if Redis is not available
	if !isRedisAvailable() {
		t.Skip("Redis is not available, skipping test")
	}

	// Create two hubs with Redis to simulate multiple instances
	hub1 := createTestHubWithRedis()
	hub1.ConnectionCache = NewUserConnectionCache(hub1)
	hub1.ErrorHandler = NewErrorHandler(hub1)

	hub2 := createTestHubWithRedis()
	hub2.ConnectionCache = NewUserConnectionCache(hub2)
	hub2.ErrorHandler = NewErrorHandler(hub2)

	// Start Redis listeners
	go hub1.wsRedisListener()
	go hub2.wsRedisListener()

	// Give listeners time to start
	time.Sleep(100 * time.Millisecond)

	// Create test clients
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	client3 := createTestClient(3)
	client4 := createTestClient(4)

	// Register clients with hubs
	hub1.Register <- client1
	hub1.Register <- client2
	hub2.Register <- client3
	hub2.Register <- client4

	// Process registrations
	processRegistration(hub1, client1)
	processRegistration(hub1, client2)
	processRegistration(hub2, client3)
	processRegistration(hub2, client4)

	// Subscribe clients to channels
	client1.WsAddChannel(100, hub1)
	client2.WsAddChannel(100, hub1)
	client3.WsAddChannel(100, hub2)
	client4.WsAddChannel(200, hub2)

	// Test cross-instance message broadcasting
	t.Run("CrossInstanceBroadcasting", func(t *testing.T) {
		// Create test message for channel 100
		chatMsg := &MockChat{
			ChannelID: 100,
			UserID:    1,
			Text:      "Cross-instance test message",
			SentAt:    time.Now().Format(time.RFC3339),
		}

		// Broadcast message from hub1
		hub1.BroadcastMockMessage(chatMsg)

		// Give time for message to propagate through Redis
		time.Sleep(200 * time.Millisecond)

		// Verify clients in hub1 received the message
		mockConn1 := client1.Conn.(*mockConn)
		messages1 := mockConn1.getMessages()
		if len(messages1) != 1 {
			t.Errorf("Client 1 should receive 1 message, got %d", len(messages1))
		}

		mockConn2 := client2.Conn.(*mockConn)
		messages2 := mockConn2.getMessages()
		if len(messages2) != 1 {
			t.Errorf("Client 2 should receive 1 message, got %d", len(messages2))
		}

		// Verify client3 in hub2 received the message via Redis
		mockConn3 := client3.Conn.(*mockConn)
		messages3 := mockConn3.getMessages()
		if len(messages3) != 1 {
			t.Errorf("Client 3 should receive 1 message via Redis, got %d", len(messages3))
		}

		// Verify client4 in hub2 did not receive the message (different channel)
		mockConn4 := client4.Conn.(*mockConn)
		messages4 := mockConn4.getMessages()
		if len(messages4) != 0 {
			t.Errorf("Client 4 should not receive any messages, got %d", len(messages4))
		}
	})

	// Test presence synchronization
	t.Run("PresenceSynchronization", func(t *testing.T) {
		// Unregister client1 from hub1
		hub1.Unregister <- client1
		processUnregistration(hub1, client1)

		// Give time for presence update to propagate through Redis
		time.Sleep(200 * time.Millisecond)

		// Verify client1 is removed from hub1's connection cache
		if hub1.ConnectionCache.IsUserOnline(1) {
			t.Error("Client 1 should be removed from hub1's connection cache")
		}

		// In a real distributed system, hub2 would also remove client1 from its cache
		// if it had it, but in our test setup they're separate caches
	})

	// Clean up Redis connections
	hub1.Redis.Close()
	hub2.Redis.Close()
}

// TestRedisErrorRecovery tests recovery from Redis connection failures
func TestRedisErrorRecovery(t *testing.T) {
	// Skip if Redis is not available
	if !isRedisAvailable() {
		t.Skip("Redis is not available, skipping test")
	}

	// Create a hub with Redis
	hub := createTestHubWithRedis()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)

	// Create test client
	client := createTestClient(1)
	hub.Register <- client
	processRegistration(hub, client)
	client.WsAddChannel(100, hub)

	// Test Redis error handling
	t.Run("RedisErrorHandling", func(t *testing.T) {
		// Create test message
		chatMsg := &MockChat{
			ChannelID: 100,
			UserID:    1,
			Text:      "Test message during Redis failure",
			SentAt:    time.Now().Format(time.RFC3339),
		}

		// Close Redis connection to simulate failure
		hub.Redis.Close()

		// Broadcast message (should handle Redis failure gracefully)
		hub.BroadcastMockMessage(chatMsg)

		// Give time for message to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify client still received the message locally despite Redis failure
		mockConn := client.Conn.(*mockConn)
		messages := mockConn.getMessages()
		if len(messages) != 1 {
			t.Errorf("Client should receive 1 message despite Redis failure, got %d", len(messages))
		}

		// Reconnect to Redis
		hub.Redis = createTestHubWithRedis().Redis

		// Broadcast another message
		chatMsg = &MockChat{
			ChannelID: 100,
			UserID:    1,
			Text:      "Test message after Redis recovery",
			SentAt:    time.Now().Format(time.RFC3339),
		}
		hub.BroadcastMockMessage(chatMsg)

		// Give time for message to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify client received the second message
		messages = mockConn.getMessages()
		if len(messages) != 2 {
			t.Errorf("Client should receive 2 messages after Redis recovery, got %d", len(messages))
		}
	})

	// Clean up
	hub.Redis.Close()
}

// TestHighVolumeRedisMessages tests handling high volume of Redis messages
func TestHighVolumeRedisMessages(t *testing.T) {
	// Skip if Redis is not available
	if !isRedisAvailable() {
		t.Skip("Redis is not available, skipping test")
	}

	// Create two hubs with Redis
	hub1 := createTestHubWithRedis()
	hub1.ConnectionCache = NewUserConnectionCache(hub1)
	hub1.ErrorHandler = NewErrorHandler(hub1)

	hub2 := createTestHubWithRedis()
	hub2.ConnectionCache = NewUserConnectionCache(hub2)
	hub2.ErrorHandler = NewErrorHandler(hub2)

	// Start Redis listeners
	go hub1.wsRedisListener()
	go hub2.wsRedisListener()

	// Give listeners time to start
	time.Sleep(100 * time.Millisecond)

	// Create test clients
	client1 := createTestClient(1)
	hub1.Register <- client1
	processRegistration(hub1, client1)
	client1.WsAddChannel(100, hub1)

	client2 := createTestClient(2)
	hub2.Register <- client2
	processRegistration(hub2, client2)
	client2.WsAddChannel(100, hub2)

	// Test high volume message handling
	t.Run("HighVolumeMessages", func(t *testing.T) {
		messageCount := 50
		var wg sync.WaitGroup
		wg.Add(messageCount)

		// Send multiple messages concurrently
		startTime := time.Now()
		for i := 0; i < messageCount; i++ {
			go func(idx int) {
				defer wg.Done()
				chatMsg := &MockChat{
					ChannelID: 100,
					UserID:    1,
					Text:      fmt.Sprintf("Message %d", idx),
					SentAt:    time.Now().Format(time.RFC3339),
				}
				hub1.BroadcastMockMessage(chatMsg)
			}(i)
		}

		wg.Wait()
		duration := time.Since(startTime)

		// Give time for messages to propagate through Redis
		time.Sleep(500 * time.Millisecond)

		// Verify clients received messages
		mockConn1 := client1.Conn.(*mockConn)
		messages1 := mockConn1.getMessages()
		mockConn2 := client2.Conn.(*mockConn)
		messages2 := mockConn2.getMessages()

		t.Logf("Sent %d messages in %v", messageCount, duration)
		t.Logf("Client 1 received %d messages", len(messages1))
		t.Logf("Client 2 received %d messages via Redis", len(messages2))

		// We may not receive exactly messageCount messages due to Redis pub/sub timing
		// and potential message coalescing, but we should receive a significant number
		if len(messages1) < messageCount*3/4 {
			t.Errorf("Client 1 should receive at least %d messages, got %d",
				messageCount*3/4, len(messages1))
		}
		if len(messages2) < messageCount*3/4 {
			t.Errorf("Client 2 should receive at least %d messages via Redis, got %d",
				messageCount*3/4, len(messages2))
		}
	})

	// Clean up
	hub1.Redis.Close()
	hub2.Redis.Close()
}

// Helper functions

// processRegistration simulates the hub's registration process
func processRegistration(hub *Hub, client *Client) {
	hub.Clients[client] = true
	hub.ConnectionCache.AddConnection(client)
}

// processUnregistration simulates the hub's unregistration process
func processUnregistration(hub *Hub, client *Client) {
	delete(hub.Clients, client)
	client.Conn.Close()
	hub.ConnectionCache.RemoveConnection(client.ID)
}
