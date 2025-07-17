package ws

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"
)

// TestCompleteMessageFlowFromAPIToClients tests the complete message flow from API to WebSocket clients
// This test simulates the entire flow from an API handler through the hub to connected clients
func TestCompleteMessageFlowFromAPIToClients(t *testing.T) {
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

	// Create a mock chat message (simulating what would come from the API)
	mockChat := &MockChat{
		ChannelID: 100,
		UserID:    1,
		Text:      "Hello from the API!",
		SentAt:    time.Now().Format(time.RFC3339),
	}

	// Broadcast the message using the BroadcastMockMessage method
	// This simulates what would happen when an API handler calls BroadcastMessage
	hub.BroadcastMockMessage(mockChat)

	// Give time for the message to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify clients in channel 100 received the message
	mockConn1 := client1.Conn.(*mockConn)
	messages1 := mockConn1.getMessages()
	if len(messages1) != 1 {
		t.Errorf("Client 1 should receive 1 message, got %d", len(messages1))
	}

	// Verify message content
	var receivedMsg map[string]interface{}
	if err := json.Unmarshal(messages1[0], &receivedMsg); err != nil {
		t.Errorf("Failed to unmarshal received message: %v", err)
	} else {
		if receivedMsg["text"] != "Hello from the API!" {
			t.Errorf("Expected message text 'Hello from the API!', got '%v'", receivedMsg["text"])
		}
		if receivedMsg["channelId"] != float64(100) {
			t.Errorf("Expected channelId 100, got %v", receivedMsg["channelId"])
		}
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

	// Test metrics were recorded
	metrics := hub.Metrics.GetMetricsByType(MetricBroadcast)
	if len(metrics) == 0 {
		t.Error("Expected broadcast metrics to be recorded")
	}
}

// TestMultiUserMultiChannelComplexScenario tests a complex scenario with multiple users across multiple channels
// This test simulates a more realistic scenario with users joining and leaving channels
func TestMultiUserMultiChannelComplexScenario(t *testing.T) {
	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Create 20 test clients
	numClients := 20
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = createTestClient(uint(i + 1))
		hub.Clients[clients[i]] = true
		hub.ConnectionCache.AddConnection(clients[i])
	}

	// Set up complex channel subscriptions:
	// - Channel 100: Users 1-10
	// - Channel 200: Users 6-15
	// - Channel 300: Users 11-20
	for i := 0; i < 10; i++ {
		clients[i].WsAddChannel(100, hub)
	}
	for i := 5; i < 15; i++ {
		clients[i].WsAddChannel(200, hub)
	}
	for i := 10; i < 20; i++ {
		clients[i].WsAddChannel(300, hub)
	}

	// Verify channel subscriptions
	users100 := hub.ConnectionCache.GetOnlineUsersInChannel(100)
	if len(users100) != 10 {
		t.Errorf("Expected 10 users in channel 100, got %d", len(users100))
	}
	users200 := hub.ConnectionCache.GetOnlineUsersInChannel(200)
	if len(users200) != 10 {
		t.Errorf("Expected 10 users in channel 200, got %d", len(users200))
	}
	users300 := hub.ConnectionCache.GetOnlineUsersInChannel(300)
	if len(users300) != 10 {
		t.Errorf("Expected 10 users in channel 300, got %d", len(users300))
	}

	// Broadcast messages to each channel
	mockChat100 := &MockChat{
		ChannelID: 100,
		UserID:    1,
		Text:      "Message to channel 100",
		SentAt:    time.Now().Format(time.RFC3339),
	}
	hub.BroadcastMockMessage(mockChat100)

	mockChat200 := &MockChat{
		ChannelID: 200,
		UserID:    6,
		Text:      "Message to channel 200",
		SentAt:    time.Now().Format(time.RFC3339),
	}
	hub.BroadcastMockMessage(mockChat200)

	mockChat300 := &MockChat{
		ChannelID: 300,
		UserID:    11,
		Text:      "Message to channel 300",
		SentAt:    time.Now().Format(time.RFC3339),
	}
	hub.BroadcastMockMessage(mockChat300)

	// Give time for messages to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify message receipt for each client
	// Expected message counts:
	// - Users 1-5: 1 message (channel 100)
	// - Users 6-10: 2 messages (channels 100, 200)
	// - Users 11-15: 2 messages (channels 200, 300)
	// - Users 16-20: 1 message (channel 300)
	expectedMessageCounts := []int{1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1}

	for i := 0; i < numClients; i++ {
		mockConn := clients[i].Conn.(*mockConn)
		messages := mockConn.getMessages()
		if len(messages) != expectedMessageCounts[i] {
			t.Errorf("Client %d should have received %d messages, got %d",
				i+1, expectedMessageCounts[i], len(messages))
		}
	}

	// Now simulate users leaving channels
	clients[5].WsRemoveChannel(100, hub)
	clients[10].WsRemoveChannel(200, hub)
	clients[15].WsRemoveChannel(300, hub)

	// Verify channel subscriptions after removals
	users100 = hub.ConnectionCache.GetOnlineUsersInChannel(100)
	if len(users100) != 9 {
		t.Errorf("Expected 9 users in channel 100 after removal, got %d", len(users100))
	}
	users200 = hub.ConnectionCache.GetOnlineUsersInChannel(200)
	if len(users200) != 9 {
		t.Errorf("Expected 9 users in channel 200 after removal, got %d", len(users200))
	}
	users300 = hub.ConnectionCache.GetOnlineUsersInChannel(300)
	if len(users300) != 9 {
		t.Errorf("Expected 9 users in channel 300 after removal, got %d", len(users300))
	}

	// Broadcast new messages to each channel
	mockChat100New := &MockChat{
		ChannelID: 100,
		UserID:    1,
		Text:      "New message to channel 100",
		SentAt:    time.Now().Format(time.RFC3339),
	}
	hub.BroadcastMockMessage(mockChat100New)

	mockChat200New := &MockChat{
		ChannelID: 200,
		UserID:    6,
		Text:      "New message to channel 200",
		SentAt:    time.Now().Format(time.RFC3339),
	}
	hub.BroadcastMockMessage(mockChat200New)

	mockChat300New := &MockChat{
		ChannelID: 300,
		UserID:    11,
		Text:      "New message to channel 300",
		SentAt:    time.Now().Format(time.RFC3339),
	}
	hub.BroadcastMockMessage(mockChat300New)

	// Give time for messages to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify users who left channels don't receive new messages
	mockConn6 := clients[5].Conn.(*mockConn)
	messages6 := mockConn6.getMessages()
	if len(messages6) != 3 { // 2 initial + 1 new from channel 200
		t.Errorf("Client 6 should have received 3 messages total, got %d", len(messages6))
	}

	mockConn11 := clients[10].Conn.(*mockConn)
	messages11 := mockConn11.getMessages()
	if len(messages11) != 3 { // 2 initial + 1 new from channel 300
		t.Errorf("Client 11 should have received 3 messages total, got %d", len(messages11))
	}

	mockConn16 := clients[15].Conn.(*mockConn)
	messages16 := mockConn16.getMessages()
	if len(messages16) != 1 { // 1 initial, no new messages
		t.Errorf("Client 16 should have received 1 message total, got %d", len(messages16))
	}
}

// TestHighVolumePerformance tests the performance of the connection cache with high message volumes
func TestHighVolumePerformance(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping high volume performance test in short mode")
	}

	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Create a large number of test clients
	numClients := 500 // Reduced from 1000 for faster tests
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = createTestClient(uint(i + 1))
		hub.Clients[clients[i]] = true
		hub.ConnectionCache.AddConnection(clients[i])
		clients[i].WsAddChannel(100, hub)
	}

	// Measure memory before test
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	// Send a high volume of messages
	numMessages := 20
	startTime := time.Now()

	for i := 0; i < numMessages; i++ {
		mockChat := &MockChat{
			ChannelID: 100,
			UserID:    1,
			Text:      fmt.Sprintf("High volume test message %d", i),
			SentAt:    time.Now().Format(time.RFC3339),
		}
		hub.BroadcastMockMessage(mockChat)
		time.Sleep(10 * time.Millisecond) // Small delay between messages
	}

	// Wait for all messages to be processed
	time.Sleep(200 * time.Millisecond)

	// Calculate total duration
	duration := time.Since(startTime)

	// Measure memory after test
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)

	// Calculate memory usage
	memoryUsed := memStatsAfter.TotalAlloc - memStatsBefore.TotalAlloc

	// Log performance metrics
	t.Logf("Sent %d messages to %d clients in %v", numMessages, numClients, duration)
	t.Logf("Total broadcasts: %d", numMessages*numClients)
	t.Logf("Average time per message: %v", duration/time.Duration(numMessages))
	t.Logf("Memory used: %d bytes", memoryUsed)
	t.Logf("Memory per client: %d bytes", memoryUsed/uint64(numClients))

	// Verify clients received messages
	// Check a sample of clients
	sampleIndices := []int{0, numClients / 4, numClients / 2, 3 * numClients / 4, numClients - 1}
	for _, idx := range sampleIndices {
		mockConn := clients[idx].Conn.(*mockConn)
		messages := mockConn.getMessages()
		if len(messages) != numMessages {
			t.Errorf("Client %d should have received %d messages, got %d",
				idx+1, numMessages, len(messages))
		}
	}

	// Check metrics were recorded
	metrics := hub.Metrics.GetMetricsByType(MetricBroadcast)
	if len(metrics) < numMessages {
		t.Errorf("Expected at least %d broadcast metrics, got %d", numMessages, len(metrics))
	}
}

// TestRedisIntegrationWithMultipleHubs tests Redis pub/sub integration with connection cache across multiple hubs
func TestRedisIntegrationWithMultipleHubs(t *testing.T) {
	// Skip if Redis is not available
	if !isRedisAvailable() {
		t.Skip("Redis is not available, skipping test")
	}

	// Create three hubs to simulate multiple instances
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

	hub3 := createTestHubWithRedis()
	hub3.ConnectionCache = NewUserConnectionCache(hub3)
	hub3.ErrorHandler = NewErrorHandler(hub3)
	hub3.MonitoringHooks = NewMonitoringHooks()
	hub3.Metrics = NewConnectionMetrics(1000)

	// Start Redis listeners for all hubs
	go hub1.wsRedisListener()
	go hub2.wsRedisListener()
	go hub3.wsRedisListener()

	// Give listeners time to start
	time.Sleep(100 * time.Millisecond)

	// Create test clients for each hub
	// Hub 1: Clients 1-5 in channel 100
	// Hub 2: Clients 6-10 in channel 100, Clients 11-15 in channel 200
	// Hub 3: Clients 16-20 in channel 200, Clients 21-25 in channel 300
	for i := 1; i <= 5; i++ {
		client := createTestClient(uint(i))
		hub1.Clients[client] = true
		hub1.ConnectionCache.AddConnection(client)
		client.WsAddChannel(100, hub1)
	}

	for i := 6; i <= 10; i++ {
		client := createTestClient(uint(i))
		hub2.Clients[client] = true
		hub2.ConnectionCache.AddConnection(client)
		client.WsAddChannel(100, hub2)
	}

	for i := 11; i <= 15; i++ {
		client := createTestClient(uint(i))
		hub2.Clients[client] = true
		hub2.ConnectionCache.AddConnection(client)
		client.WsAddChannel(200, hub2)
	}

	for i := 16; i <= 20; i++ {
		client := createTestClient(uint(i))
		hub3.Clients[client] = true
		hub3.ConnectionCache.AddConnection(client)
		client.WsAddChannel(200, hub3)
	}

	for i := 21; i <= 25; i++ {
		client := createTestClient(uint(i))
		hub3.Clients[client] = true
		hub3.ConnectionCache.AddConnection(client)
		client.WsAddChannel(300, hub3)
	}

	// Verify channel subscriptions in each hub
	users100Hub1 := hub1.ConnectionCache.GetOnlineUsersInChannel(100)
	if len(users100Hub1) != 5 {
		t.Errorf("Expected 5 users in channel 100 in hub1, got %d", len(users100Hub1))
	}

	users100Hub2 := hub2.ConnectionCache.GetOnlineUsersInChannel(100)
	if len(users100Hub2) != 5 {
		t.Errorf("Expected 5 users in channel 100 in hub2, got %d", len(users100Hub2))
	}

	users200Hub2 := hub2.ConnectionCache.GetOnlineUsersInChannel(200)
	if len(users200Hub2) != 5 {
		t.Errorf("Expected 5 users in channel 200 in hub2, got %d", len(users200Hub2))
	}

	users200Hub3 := hub3.ConnectionCache.GetOnlineUsersInChannel(200)
	if len(users200Hub3) != 5 {
		t.Errorf("Expected 5 users in channel 200 in hub3, got %d", len(users200Hub3))
	}

	users300Hub3 := hub3.ConnectionCache.GetOnlineUsersInChannel(300)
	if len(users300Hub3) != 5 {
		t.Errorf("Expected 5 users in channel 300 in hub3, got %d", len(users300Hub3))
	}

	// Test cross-instance message broadcasting for channel 100
	// Broadcast from hub1 to channel 100
	mockChat100 := &MockChat{
		ChannelID: 100,
		UserID:    1,
		Text:      "Message to channel 100 from hub1",
		SentAt:    time.Now().Format(time.RFC3339),
	}
	hub1.BroadcastMockMessage(mockChat100)

	// Give time for message to propagate through Redis
	time.Sleep(200 * time.Millisecond)

	// Verify clients in hub1 and hub2 received the message for channel 100
	// Check a sample client from each hub
	client1 := findClientByID(hub1.Clients, 1)
	if client1 == nil {
		t.Fatal("Client 1 not found in hub1")
	}
	mockConn1 := client1.Conn.(*mockConn)
	messages1 := mockConn1.getMessages()
	if len(messages1) != 1 {
		t.Errorf("Client 1 in hub1 should receive 1 message, got %d", len(messages1))
	}

	client6 := findClientByID(hub2.Clients, 6)
	if client6 == nil {
		t.Fatal("Client 6 not found in hub2")
	}
	mockConn6 := client6.Conn.(*mockConn)
	messages6 := mockConn6.getMessages()
	if len(messages6) != 1 {
		t.Errorf("Client 6 in hub2 should receive 1 message via Redis, got %d", len(messages6))
	}

	// Test cross-instance message broadcasting for channel 200
	// Broadcast from hub2 to channel 200
	mockChat200 := &MockChat{
		ChannelID: 200,
		UserID:    11,
		Text:      "Message to channel 200 from hub2",
		SentAt:    time.Now().Format(time.RFC3339),
	}
	hub2.BroadcastMockMessage(mockChat200)

	// Give time for message to propagate through Redis
	time.Sleep(200 * time.Millisecond)

	// Verify clients in hub2 and hub3 received the message for channel 200
	client11 := findClientByID(hub2.Clients, 11)
	if client11 == nil {
		t.Fatal("Client 11 not found in hub2")
	}
	mockConn11 := client11.Conn.(*mockConn)
	messages11 := mockConn11.getMessages()
	if len(messages11) != 1 {
		t.Errorf("Client 11 in hub2 should receive 1 message, got %d", len(messages11))
	}

	client16 := findClientByID(hub3.Clients, 16)
	if client16 == nil {
		t.Fatal("Client 16 not found in hub3")
	}
	mockConn16 := client16.Conn.(*mockConn)
	messages16 := mockConn16.getMessages()
	if len(messages16) != 1 {
		t.Errorf("Client 16 in hub3 should receive 1 message via Redis, got %d", len(messages16))
	}

	// Test presence synchronization
	// Unregister client1 from hub1
	hub1.Unregister <- client1
	processUnregistration(hub1, client1)

	// Give time for presence update to propagate through Redis
	time.Sleep(200 * time.Millisecond)

	// Verify client1 is removed from hub1's connection cache
	if hub1.ConnectionCache.IsUserOnline(1) {
		t.Error("Client 1 should be removed from hub1's connection cache")
	}

	// Clean up Redis connections
	hub1.Redis.Close()
	hub2.Redis.Close()
	hub3.Redis.Close()
}

// Helper function to find a client by ID in a hub's clients map
func findClientByID(clients map[*Client]bool, id uint) *Client {
	for client := range clients {
		if client.ID == id {
			return client
		}
	}
	return nil
}
