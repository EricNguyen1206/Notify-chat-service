package ws

import (
	"encoding/json"
	"testing"
	"time"
)

// TestChannelSubscriptionManagement tests the channel subscription management functionality
// of the connection cache
func TestChannelSubscriptionManagement(t *testing.T) {
	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Create test clients
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	client3 := createTestClient(3)

	// Register clients with the hub
	hub.Clients[client1] = true
	hub.Clients[client2] = true
	hub.Clients[client3] = true

	// Add clients to connection cache
	hub.ConnectionCache.AddConnection(client1)
	hub.ConnectionCache.AddConnection(client2)
	hub.ConnectionCache.AddConnection(client3)

	// Test adding clients to channels
	t.Run("AddUserToChannel", func(t *testing.T) {
		// Subscribe clients to channels
		client1.WsAddChannel(100, hub)
		client1.WsAddChannel(200, hub)
		client2.WsAddChannel(100, hub)
		client3.WsAddChannel(300, hub)

		// Verify channel subscriptions in connection cache
		users100 := hub.ConnectionCache.GetOnlineUsersInChannel(100)
		if len(users100) != 2 {
			t.Errorf("Expected 2 users in channel 100, got %d", len(users100))
		}

		users200 := hub.ConnectionCache.GetOnlineUsersInChannel(200)
		if len(users200) != 1 {
			t.Errorf("Expected 1 user in channel 200, got %d", len(users200))
		}

		users300 := hub.ConnectionCache.GetOnlineUsersInChannel(300)
		if len(users300) != 1 {
			t.Errorf("Expected 1 user in channel 300, got %d", len(users300))
		}

		// Verify client channel subscriptions
		if !client1.Channels[100] {
			t.Error("Client 1 should be subscribed to channel 100")
		}
		if !client1.Channels[200] {
			t.Error("Client 1 should be subscribed to channel 200")
		}
		if !client2.Channels[100] {
			t.Error("Client 2 should be subscribed to channel 100")
		}
		if !client3.Channels[300] {
			t.Error("Client 3 should be subscribed to channel 300")
		}

		// Verify connection metadata
		metadata1, exists := hub.ConnectionCache.GetConnectionMetadata(1)
		if !exists {
			t.Fatal("Connection metadata for client 1 should exist")
		}
		if !metadata1.Channels[100] || !metadata1.Channels[200] {
			t.Error("Client 1 metadata should include channels 100 and 200")
		}
	})

	// Test removing clients from channels
	t.Run("RemoveUserFromChannel", func(t *testing.T) {
		// Remove client1 from channel 100
		client1.WsRemoveChannel(100, hub)

		// Verify channel subscriptions in connection cache
		users100 := hub.ConnectionCache.GetOnlineUsersInChannel(100)
		if len(users100) != 1 {
			t.Errorf("Expected 1 user in channel 100 after removal, got %d", len(users100))
		}

		// Verify client channel subscriptions
		if client1.Channels[100] {
			t.Error("Client 1 should not be subscribed to channel 100")
		}
		if !client1.Channels[200] {
			t.Error("Client 1 should still be subscribed to channel 200")
		}

		// Verify connection metadata
		metadata1, exists := hub.ConnectionCache.GetConnectionMetadata(1)
		if !exists {
			t.Fatal("Connection metadata for client 1 should exist")
		}
		if metadata1.Channels[100] {
			t.Error("Client 1 metadata should not include channel 100")
		}
		if !metadata1.Channels[200] {
			t.Error("Client 1 metadata should still include channel 200")
		}
	})

	// Test message broadcasting to specific channels
	t.Run("ChannelSpecificBroadcasting", func(t *testing.T) {
		// Create test messages for each channel
		message100 := []byte(`{"channelId":100,"userId":2,"text":"Message to channel 100","sentAt":"2023-01-01T12:00:00Z"}`)
		message200 := []byte(`{"channelId":200,"userId":1,"text":"Message to channel 200","sentAt":"2023-01-01T12:00:00Z"}`)
		message300 := []byte(`{"channelId":300,"userId":3,"text":"Message to channel 300","sentAt":"2023-01-01T12:00:00Z"}`)

		// Broadcast messages to each channel
		hub.ConnectionCache.BroadcastToChannel(100, message100)
		hub.ConnectionCache.BroadcastToChannel(200, message200)
		hub.ConnectionCache.BroadcastToChannel(300, message300)

		// Give time for messages to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify client1 received only message for channel 200
		mockConn1 := client1.Conn.(*mockConn)
		messages1 := mockConn1.getMessages()
		if len(messages1) != 1 {
			t.Errorf("Client 1 should receive 1 message, got %d", len(messages1))
		}
		if len(messages1) > 0 {
			var msg map[string]interface{}
			if err := json.Unmarshal(messages1[0], &msg); err != nil {
				t.Errorf("Failed to unmarshal message: %v", err)
			} else if msg["channelId"] != float64(200) {
				t.Errorf("Client 1 should receive message for channel 200, got %v", msg["channelId"])
			}
		}

		// Verify client2 received only message for channel 100
		mockConn2 := client2.Conn.(*mockConn)
		messages2 := mockConn2.getMessages()
		if len(messages2) != 1 {
			t.Errorf("Client 2 should receive 1 message, got %d", len(messages2))
		}
		if len(messages2) > 0 {
			var msg map[string]interface{}
			if err := json.Unmarshal(messages2[0], &msg); err != nil {
				t.Errorf("Failed to unmarshal message: %v", err)
			} else if msg["channelId"] != float64(100) {
				t.Errorf("Client 2 should receive message for channel 100, got %v", msg["channelId"])
			}
		}

		// Verify client3 received only message for channel 300
		mockConn3 := client3.Conn.(*mockConn)
		messages3 := mockConn3.getMessages()
		if len(messages3) != 1 {
			t.Errorf("Client 3 should receive 1 message, got %d", len(messages3))
		}
		if len(messages3) > 0 {
			var msg map[string]interface{}
			if err := json.Unmarshal(messages3[0], &msg); err != nil {
				t.Errorf("Failed to unmarshal message: %v", err)
			} else if msg["channelId"] != float64(300) {
				t.Errorf("Client 3 should receive message for channel 300, got %v", msg["channelId"])
			}
		}
	})

	// Test client unregistration and channel cleanup
	t.Run("UnregistrationChannelCleanup", func(t *testing.T) {
		// Unregister client2
		hub.Unregister <- client2
		processUnregistration(hub, client2)

		// Verify client2 is removed from all channels
		users100 := hub.ConnectionCache.GetOnlineUsersInChannel(100)
		for _, userID := range users100 {
			if userID == 2 {
				t.Error("Client 2 should be removed from channel 100")
			}
		}

		// Verify client2 is removed from connection cache
		if hub.ConnectionCache.IsUserOnline(2) {
			t.Error("Client 2 should be removed from connection cache")
		}

		// Broadcast another message to channel 100
		message100 := []byte(`{"channelId":100,"userId":1,"text":"Another message to channel 100","sentAt":"2023-01-01T12:00:00Z"}`)
		hub.ConnectionCache.BroadcastToChannel(100, message100)

		// Give time for message to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify client2 did not receive the message
		mockConn2 := client2.Conn.(*mockConn)
		messages2 := mockConn2.getMessages()
		initialCount := len(messages2)

		// Broadcast one more message
		message100New := []byte(`{"channelId":100,"userId":1,"text":"Final message to channel 100","sentAt":"2023-01-01T12:00:00Z"}`)
		hub.ConnectionCache.BroadcastToChannel(100, message100New)

		// Give time for message to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify client2 still has the same number of messages
		messages2New := mockConn2.getMessages()
		if len(messages2New) != initialCount {
			t.Errorf("Client 2 should not receive new messages after unregistration, got %d vs initial %d",
				len(messages2New), initialCount)
		}
	})

	// Test multiple channel subscriptions and message routing
	t.Run("MultiChannelSubscription", func(t *testing.T) {
		// Clear previous messages
		for _, client := range []*Client{client1, client3} {
			mockConn := client.Conn.(*mockConn)
			mockConn.mu.Lock()
			mockConn.messages = make([][]byte, 0)
			mockConn.mu.Unlock()
		}

		// Subscribe client3 to multiple channels
		client3.WsAddChannel(200, hub)
		client3.WsAddChannel(400, hub)

		// Verify channel subscriptions
		users200 := hub.ConnectionCache.GetOnlineUsersInChannel(200)
		if len(users200) != 2 {
			t.Errorf("Expected 2 users in channel 200, got %d", len(users200))
		}

		users300 := hub.ConnectionCache.GetOnlineUsersInChannel(300)
		if len(users300) != 1 {
			t.Errorf("Expected 1 user in channel 300, got %d", len(users300))
		}

		users400 := hub.ConnectionCache.GetOnlineUsersInChannel(400)
		if len(users400) != 1 {
			t.Errorf("Expected 1 user in channel 400, got %d", len(users400))
		}

		// Broadcast messages to each channel
		message200 := []byte(`{"channelId":200,"userId":1,"text":"New message to channel 200","sentAt":"2023-01-01T12:00:00Z"}`)
		message300 := []byte(`{"channelId":300,"userId":3,"text":"New message to channel 300","sentAt":"2023-01-01T12:00:00Z"}`)
		message400 := []byte(`{"channelId":400,"userId":3,"text":"New message to channel 400","sentAt":"2023-01-01T12:00:00Z"}`)

		hub.ConnectionCache.BroadcastToChannel(200, message200)
		hub.ConnectionCache.BroadcastToChannel(300, message300)
		hub.ConnectionCache.BroadcastToChannel(400, message400)

		// Give time for messages to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify client1 received only message for channel 200
		mockConn1 := client1.Conn.(*mockConn)
		messages1 := mockConn1.getMessages()
		if len(messages1) != 1 {
			t.Errorf("Client 1 should receive 1 message, got %d", len(messages1))
		}

		// Verify client3 received messages for channels 200, 300, and 400
		mockConn3 := client3.Conn.(*mockConn)
		messages3 := mockConn3.getMessages()
		if len(messages3) != 3 {
			t.Errorf("Client 3 should receive 3 messages, got %d", len(messages3))
		}

		// Count messages by channel for client3
		channelCounts := make(map[float64]int)
		for _, msgBytes := range messages3 {
			var msg map[string]interface{}
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				t.Errorf("Failed to unmarshal message: %v", err)
				continue
			}
			channelID, ok := msg["channelId"].(float64)
			if !ok {
				t.Errorf("Expected channelId to be a number, got %T", msg["channelId"])
				continue
			}
			channelCounts[channelID]++
		}

		if channelCounts[200] != 1 {
			t.Errorf("Client 3 should receive 1 message for channel 200, got %d", channelCounts[200])
		}
		if channelCounts[300] != 1 {
			t.Errorf("Client 3 should receive 1 message for channel 300, got %d", channelCounts[300])
		}
		if channelCounts[400] != 1 {
			t.Errorf("Client 3 should receive 1 message for channel 400, got %d", channelCounts[400])
		}
	})
}
