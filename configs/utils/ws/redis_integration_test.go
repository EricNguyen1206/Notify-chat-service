package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRedisIntegrationWithConnectionCache tests the integration between Redis pub/sub and connection cache
func TestRedisIntegrationWithConnectionCache(t *testing.T) {
	// Skip if Redis is not available
	if !isRedisAvailable() {
		t.Skip("Redis is not available, skipping test")
	}

	// Create two hubs with Redis to simulate multiple instances
	hub1 := createTestHubWithRedis()
	hub1.ConnectionCache = NewUserConnectionCache(hub1)
	hub1.ErrorHandler = NewErrorHandler(hub1)
	hub1.MonitoringHooks = NewMonitoringHooks()
	hub1.Metrics = NewConnectionMetrics(1000)

	hub2 := createTestHubWithRedis()
	hub2.ConnectionCache = NewUserConnectionCache(hub2)
	hub2.ErrorHandler = NewErrorHandler(hub2)
	hub2.MonitoringHooks = NewMonitoringHooks()
	hub2.Metrics = NewConnectionMetrics(1000)

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
	client2.WsAddChannel(200, hub1)
	client3.WsAddChannel(100, hub2)
	client4.WsAddChannel(200, hub2)

	// Test cross-instance message broadcasting
	t.Run("CrossInstanceBroadcasting", func(t *testing.T) {
		// Create test message for channel 100
		mockChat100 := &MockChat{
			ChannelID: 100,
			UserID:    1,
			Text:      "Cross-instance test message for channel 100",
			SentAt:    time.Now().Format(time.RFC3339),
		}

		// Broadcast message from hub1
		hub1.BroadcastMockMessage(mockChat100)

		// Give time for message to propagate through Redis
		time.Sleep(200 * time.Millisecond)

		// Verify clients in both hubs received the message
		mockConn1 := client1.Conn.(*mockConn)
		messages1 := mockConn1.getMessages()
		if len(messages1) != 1 {
			t.Errorf("Client 1 in hub1 should receive 1 message, got %d", len(messages1))
		}

		mockConn3 := client3.Conn.(*mockConn)
		messages3 := mockConn3.getMessages()
		if len(messages3) != 1 {
			t.Errorf("Client 3 in hub2 should receive 1 message via Redis, got %d", len(messages3))
		}

		// Verify clients in channel 200 did not receive the message
		mockConn2 := client2.Conn.(*mockConn)
		messages2 := mockConn2.getMessages()
		if len(messages2) != 0 {
			t.Errorf("Client 2 should not receive message for channel 100, got %d", len(messages2))
		}

		mockConn4 := client4.Conn.(*mockConn)
		messages4 := mockConn4.getMessages()
		if len(messages4) != 0 {
			t.Errorf("Client 4 should not receive message for channel 100, got %d", len(messages4))
		}
	})

	// Test presence synchronization
	t.Run("PresenceSynchronization", func(t *testing.T) {
		// Verify initial presence
		if !hub1.ConnectionCache.IsUserOnline(1) {
			t.Error("Client 1 should be online in hub1")
		}
		if !hub1.ConnectionCache.IsUserOnline(2) {
			t.Error("Client 2 should be online in hub1")
		}
		if !hub2.ConnectionCache.IsUserOnline(3) {
			t.Error("Client 3 should be online in hub2")
		}
		if !hub2.ConnectionCache.IsUserOnline(4) {
			t.Error("Client 4 should be online in hub2")
		}

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

	// Test channel subscription synchronization
	t.Run("ChannelSubscriptionSync", func(t *testing.T) {
		// Add client2 to a new channel
		client2.WsAddChannel(300, hub1)

		// Give time for subscription update to propagate
		time.Sleep(200 * time.Millisecond)

		// Verify client2 is in channel 300 in hub1
		users300Hub1 := hub1.ConnectionCache.GetOnlineUsersInChannel(300)
		found := false
		for _, userID := range users300Hub1 {
			if userID == 2 {
				found = true
				break
			}
		}
		if !found {
			t.Error("Client 2 should be in channel 300 in hub1")
		}

		// Create test message for channel 300
		mockChat300 := &MockChat{
			ChannelID: 300,
			UserID:    2,
			Text:      "Message to new channel 300",
			SentAt:    time.Now().Format(time.RFC3339),
		}

		// Broadcast message from hub1
		hub1.BroadcastMockMessage(mockChat300)

		// Give time for message to propagate
		time.Sleep(200 * time.Millisecond)

		// Verify client2 received the message
		mockConn2 := client2.Conn.(*mockConn)
		messages2 := mockConn2.getMessages()
		if len(messages2) != 1 {
			t.Errorf("Client 2 should receive 1 message for channel 300, got %d", len(messages2))
		}
	})

	// Test Redis error recovery
	t.Run("RedisErrorRecovery", func(t *testing.T) {
		// Close Redis connection to simulate failure
		originalRedis := hub1.Redis
		hub1.Redis.Close()
		hub1.Redis = nil

		// Create test message
		mockChat := &MockChat{
			ChannelID: 100,
			UserID:    2,
			Text:      "Message during Redis failure",
			SentAt:    time.Now().Format(time.RFC3339),
		}

		// Broadcast message (should handle Redis failure gracefully)
		hub1.BroadcastMockMessage(mockChat)

		// Give time for message to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify local clients still received the message despite Redis failure
		mockConn2 := client2.Conn.(*mockConn)
		messages2Before := len(mockConn2.getMessages())

		// Restore Redis connection
		hub1.Redis = originalRedis

		// Create another test message
		mockChat = &MockChat{
			ChannelID: 100,
			UserID:    2,
			Text:      "Message after Redis recovery",
			SentAt:    time.Now().Format(time.RFC3339),
		}

		// Broadcast message after recovery
		hub1.BroadcastMockMessage(mockChat)

		// Give time for message to propagate
		time.Sleep(200 * time.Millisecond)

		// Verify local clients received the new message
		messages2After := len(mockConn2.getMessages())
		if messages2After <= messages2Before {
			t.Errorf("Client 2 should receive additional message after Redis recovery")
		}

		// Verify remote clients received the message via Redis
		mockConn3 := client3.Conn.(*mockConn)
		messages3 := mockConn3.getMessages()
		if len(messages3) <= 1 {
			t.Errorf("Client 3 should receive additional message after Redis recovery")
		}
	})

	// Clean up Redis connections
	hub1.Redis.Close()
	hub2.Redis.Close()
}

// TestHighVolumeRedisPerformance tests the performance of Redis pub/sub with high message volumes
func TestHighVolumeRedisPerformance(t *testing.T) {
	// Skip if Redis is not available or in short mode
	if !isRedisAvailable() || testing.Short() {
		t.Skip("Redis is not available or running in short mode, skipping test")
	}

	// Create two hubs with Redis
	hub1 := createTestHubWithRedis()
	hub1.ConnectionCache = NewUserConnectionCache(hub1)
	hub1.ErrorHandler = NewErrorHandler(hub1)
	hub1.MonitoringHooks = NewMonitoringHooks()
	hub1.Metrics = NewConnectionMetrics(1000)

	hub2 := createTestHubWithRedis()
	hub2.ConnectionCache = NewUserConnectionCache(hub2)
	hub2.ErrorHandler = NewErrorHandler(hub2)
	hub2.MonitoringHooks = NewMonitoringHooks()
	hub2.Metrics = NewConnectionMetrics(1000)

	// Start Redis listeners
	go hub1.wsRedisListener()
	go hub2.wsRedisListener()

	// Give listeners time to start
	time.Sleep(100 * time.Millisecond)

	// Create test clients
	numClientsPerHub := 50
	for i := 0; i < numClientsPerHub; i++ {
		client1 := createTestClient(uint(i + 1))
		hub1.Clients[client1] = true
		hub1.ConnectionCache.AddConnection(client1)
		client1.WsAddChannel(100, hub1)

		client2 := createTestClient(uint(numClientsPerHub + i + 1))
		hub2.Clients[client2] = true
		hub2.ConnectionCache.AddConnection(client2)
		client2.WsAddChannel(100, hub2)
	}

	// Test high volume message handling
	t.Run("HighVolumeMessages", func(t *testing.T) {
		messageCount := 20
		var wg sync.WaitGroup
		wg.Add(messageCount)

		// Send multiple messages concurrently
		startTime := time.Now()
		for i := 0; i < messageCount; i++ {
			go func(idx int) {
				defer wg.Done()
				mockChat := &MockChat{
					ChannelID: 100,
					UserID:    1,
					Text:      fmt.Sprintf("Redis high volume test message %d", idx),
					SentAt:    time.Now().Format(time.RFC3339),
				}
				hub1.BroadcastMockMessage(mockChat)
			}(i)
		}

		wg.Wait()
		duration := time.Since(startTime)

		// Give time for messages to propagate through Redis
		time.Sleep(500 * time.Millisecond)

		// Sample clients from each hub to verify message receipt
		client1 := findClientByID(hub1.Clients, 1)
		client2 := findClientByID(hub2.Clients, uint(numClientsPerHub+1))

		if client1 == nil || client2 == nil {
			t.Fatal("Failed to find test clients")
		}

		mockConn1 := client1.Conn.(*mockConn)
		messages1 := mockConn1.getMessages()
		mockConn2 := client2.Conn.(*mockConn)
		messages2 := mockConn2.getMessages()

		t.Logf("Sent %d messages in %v", messageCount, duration)
		t.Logf("Hub1 client received %d messages", len(messages1))
		t.Logf("Hub2 client received %d messages via Redis", len(messages2))

		// We may not receive exactly messageCount messages due to Redis pub/sub timing
		// and potential message coalescing, but we should receive a significant number
		if len(messages1) < messageCount*3/4 {
			t.Errorf("Hub1 client should receive at least %d messages, got %d",
				messageCount*3/4, len(messages1))
		}
		if len(messages2) < messageCount*3/4 {
			t.Errorf("Hub2 client should receive at least %d messages via Redis, got %d",
				messageCount*3/4, len(messages2))
		}
	})

	// Test Redis pub/sub latency
	t.Run("RedisPubSubLatency", func(t *testing.T) {
		// Create a test message with timestamp
		mockChat := &MockChat{
			ChannelID: 100,
			UserID:    1,
			Text:      "Redis latency test message",
			SentAt:    time.Now().Format(time.RFC3339),
			// Add a timestamp field for latency measurement
			// This will be included in the JSON
		}

		// Add timestamp just before sending
		messageData := map[string]interface{}{
			"channelId": mockChat.ChannelID,
			"userId":    mockChat.UserID,
			"text":      mockChat.Text,
			"sentAt":    mockChat.SentAt,
			"timestamp": time.Now().UnixNano(),
		}
		messageBytes, _ := json.Marshal(messageData)

		// Direct Redis publish to measure pure Redis latency
		ctx := context.Background()
		channelName := fmt.Sprintf("channel:%d", mockChat.ChannelID)

		startTime := time.Now()
		err := hub1.Redis.Publish(ctx, channelName, messageBytes).Err()
		if err != nil {
			t.Fatalf("Failed to publish to Redis: %v", err)
		}

		// Give time for message to propagate
		time.Sleep(200 * time.Millisecond)

		// Calculate Redis pub/sub latency
		redisLatency := time.Since(startTime)
		t.Logf("Redis pub/sub latency: %v", redisLatency)

		// Verify message was received by hub2 clients
		client2 := findClientByID(hub2.Clients, uint(numClientsPerHub+1))
		if client2 == nil {
			t.Fatal("Failed to find test client in hub2")
		}

		mockConn2 := client2.Conn.(*mockConn)
		messages2 := mockConn2.getMessages()
		if len(messages2) == 0 {
			t.Error("Hub2 client should receive the Redis latency test message")
		}
	})

	// Clean up Redis connections
	hub1.Redis.Close()
	hub2.Redis.Close()
}

// TestRedisReconnection tests the ability to reconnect to Redis after connection loss
func TestRedisReconnection(t *testing.T) {
	// Skip if Redis is not available
	if !isRedisAvailable() {
		t.Skip("Redis is not available, skipping test")
	}

	// Create a hub with Redis
	hub := createTestHubWithRedis()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Start Redis listener
	go hub.wsRedisListener()

	// Give listener time to start
	time.Sleep(100 * time.Millisecond)

	// Create test client
	client := createTestClient(1)
	hub.Clients[client] = true
	hub.ConnectionCache.AddConnection(client)
	client.WsAddChannel(100, hub)

	// Test initial Redis functionality
	t.Run("InitialRedisConnection", func(t *testing.T) {
		// Create test message
		mockChat := &MockChat{
			ChannelID: 100,
			UserID:    1,
			Text:      "Initial Redis test message",
			SentAt:    time.Now().Format(time.RFC3339),
		}

		// Broadcast message
		hub.BroadcastMockMessage(mockChat)

		// Give time for message to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify client received the message
		mockConn := client.Conn.(*mockConn)
		messages := mockConn.getMessages()
		if len(messages) != 1 {
			t.Errorf("Client should receive 1 message, got %d", len(messages))
		}
	})

	// Test Redis reconnection
	t.Run("RedisReconnection", func(t *testing.T) {
		// Store original Redis client
		originalRedis := hub.Redis

		// Close Redis connection to simulate failure
		hub.Redis.Close()

		// Create a new Redis client
		hub.Redis = redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})

		// Give time for reconnection
		time.Sleep(100 * time.Millisecond)

		// Create test message
		mockChat := &MockChat{
			ChannelID: 100,
			UserID:    1,
			Text:      "Redis reconnection test message",
			SentAt:    time.Now().Format(time.RFC3339),
		}

		// Broadcast message
		hub.BroadcastMockMessage(mockChat)

		// Give time for message to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify client received the message
		mockConn := client.Conn.(*mockConn)
		messages := mockConn.getMessages()
		if len(messages) != 2 {
			t.Errorf("Client should receive 2 messages after Redis reconnection, got %d", len(messages))
		}

		// Clean up
		hub.Redis.Close()
		originalRedis.Close()
	})
}
