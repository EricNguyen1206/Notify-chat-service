package ws

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestAPIToWebSocketIntegration tests the complete flow from API endpoint to WebSocket clients
// This test simulates an API call that triggers a WebSocket broadcast
func TestAPIToWebSocketIntegration(t *testing.T) {
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

	// Subscribe clients to channels
	client1.WsAddChannel(100, hub)
	client2.WsAddChannel(100, hub)
	client3.WsAddChannel(200, hub)

	// Create a simple HTTP handler that broadcasts messages
	apiHandler := func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var requestBody struct {
			ChannelID uint   `json:"channelId"`
			UserID    uint   `json:"userId"`
			Text      string `json:"text"`
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&requestBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Create a mock chat message
		mockChat := &MockChat{
			ChannelID: requestBody.ChannelID,
			UserID:    requestBody.UserID,
			Text:      requestBody.Text,
			SentAt:    time.Now().Format(time.RFC3339),
		}

		// Broadcast the message
		hub.BroadcastMockMessage(mockChat)

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Message sent",
		})
	}

	// Create a test HTTP server with the handler
	server := httptest.NewServer(http.HandlerFunc(apiHandler))
	defer server.Close()

	// Test sending a message through the API
	t.Run("SendMessageThroughAPI", func(t *testing.T) {
		// Create request body
		requestBody := map[string]interface{}{
			"channelId": 100,
			"userId":    1,
			"text":      "Message from API test",
		}
		requestJSON, _ := json.Marshal(requestBody)

		// Send POST request to the test server
		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(requestJSON))
		if err != nil {
			t.Fatalf("Failed to send POST request: %v", err)
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		// Give time for message to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify clients in channel 100 received the message
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

		// Verify client in channel 200 did not receive the message
		mockConn3 := client3.Conn.(*mockConn)
		messages3 := mockConn3.getMessages()
		if len(messages3) != 0 {
			t.Errorf("Client 3 should not receive any messages, got %d", len(messages3))
		}

		// Verify message content
		var receivedMsg map[string]interface{}
		if err := json.Unmarshal(messages1[0], &receivedMsg); err != nil {
			t.Errorf("Failed to unmarshal received message: %v", err)
		} else {
			if receivedMsg["text"] != "Message from API test" {
				t.Errorf("Expected message text 'Message from API test', got '%v'", receivedMsg["text"])
			}
			if receivedMsg["channelId"] != float64(100) {
				t.Errorf("Expected channelId 100, got %v", receivedMsg["channelId"])
			}
		}
	})

	// Test sending multiple messages in quick succession
	t.Run("SendMultipleMessagesQuickly", func(t *testing.T) {
		// Clear previous messages
		for _, client := range []*Client{client1, client2, client3} {
			mockConn := client.Conn.(*mockConn)
			mockConn.mu.Lock()
			mockConn.messages = make([][]byte, 0)
			mockConn.mu.Unlock()
		}

		// Send 5 messages in quick succession
		for i := 0; i < 5; i++ {
			requestBody := map[string]interface{}{
				"channelId": 100,
				"userId":    1,
				"text":      "Quick message " + string(rune('A'+i)),
			}
			requestJSON, _ := json.Marshal(requestBody)

			resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(requestJSON))
			if err != nil {
				t.Fatalf("Failed to send POST request: %v", err)
			}
			resp.Body.Close()
		}

		// Give time for messages to be processed
		time.Sleep(100 * time.Millisecond)

		// Verify clients received all messages
		mockConn1 := client1.Conn.(*mockConn)
		messages1 := mockConn1.getMessages()
		if len(messages1) != 5 {
			t.Errorf("Client 1 should receive 5 messages, got %d", len(messages1))
		}

		mockConn2 := client2.Conn.(*mockConn)
		messages2 := mockConn2.getMessages()
		if len(messages2) != 5 {
			t.Errorf("Client 2 should receive 5 messages, got %d", len(messages2))
		}
	})
}

// TestBackwardCompatibility tests that the connection cache implementation maintains
// backward compatibility with existing WebSocket functionality
func TestBackwardCompatibility(t *testing.T) {
	// Create a hub WITHOUT connection cache (simulating old implementation)
	oldHub := createTestHub()

	// Create a hub WITH connection cache (new implementation)
	newHub := createTestHub()
	newHub.ConnectionCache = NewUserConnectionCache(newHub)
	newHub.ErrorHandler = NewErrorHandler(newHub)
	newHub.MonitoringHooks = NewMonitoringHooks()
	newHub.Metrics = NewConnectionMetrics(1000)

	// Create test clients
	oldClient := createTestClient(1)
	newClient := createTestClient(2)

	// Register clients with their respective hubs
	oldHub.Clients[oldClient] = true
	newHub.Clients[newClient] = true
	newHub.ConnectionCache.AddConnection(newClient)

	// Subscribe clients to the same channel
	oldClient.Channels[100] = true
	newClient.WsAddChannel(100, newHub)

	// Test message broadcasting
	t.Run("BroadcastCompatibility", func(t *testing.T) {
		// Create test messages
		oldMessage := []byte(`{"channelId":100,"userId":1,"text":"Old hub message","sentAt":"2023-01-01T12:00:00Z"}`)
		newMessage := []byte(`{"channelId":100,"userId":2,"text":"New hub message","sentAt":"2023-01-01T12:00:00Z"}`)

		// Broadcast using old hub's method (direct client iteration)
		for client := range oldHub.Clients {
			if _, ok := client.Channels[100]; ok {
				client.mu.Lock()
				client.Conn.WriteMessage(1, oldMessage)
				client.mu.Unlock()
			}
		}

		// Broadcast using new hub's method (connection cache)
		newHub.ConnectionCache.BroadcastToChannel(100, newMessage)

		// Give time for messages to be processed
		time.Sleep(50 * time.Millisecond)

		// Verify old client received message
		oldMockConn := oldClient.Conn.(*mockConn)
		oldMessages := oldMockConn.getMessages()
		if len(oldMessages) != 1 {
			t.Errorf("Old client should receive 1 message, got %d", len(oldMessages))
		}

		// Verify new client received message
		newMockConn := newClient.Conn.(*mockConn)
		newMessages := newMockConn.getMessages()
		if len(newMessages) != 1 {
			t.Errorf("New client should receive 1 message, got %d", len(newMessages))
		}
	})

	// Test client registration/unregistration
	t.Run("ClientLifecycleCompatibility", func(t *testing.T) {
		// Create new test clients
		oldClient2 := createTestClient(3)
		newClient2 := createTestClient(4)

		// Register with old hub
		oldHub.Clients[oldClient2] = true
		oldClient2.Channels[100] = true

		// Register with new hub
		newHub.Register <- newClient2
		processRegistration(newHub, newClient2)
		newClient2.WsAddChannel(100, newHub)

		// Verify registration
		if _, ok := oldHub.Clients[oldClient2]; !ok {
			t.Error("Old client should be registered with old hub")
		}
		if _, ok := newHub.Clients[newClient2]; !ok {
			t.Error("New client should be registered with new hub")
		}
		if !newHub.ConnectionCache.IsUserOnline(4) {
			t.Error("New client should be in connection cache")
		}

		// Unregister from old hub
		delete(oldHub.Clients, oldClient2)
		oldClient2.Conn.Close()

		// Unregister from new hub
		newHub.Unregister <- newClient2
		processUnregistration(newHub, newClient2)

		// Verify unregistration
		if _, ok := oldHub.Clients[oldClient2]; ok {
			t.Error("Old client should be unregistered from old hub")
		}
		if _, ok := newHub.Clients[newClient2]; ok {
			t.Error("New client should be unregistered from new hub")
		}
		if newHub.ConnectionCache.IsUserOnline(4) {
			t.Error("New client should be removed from connection cache")
		}
	})
}
