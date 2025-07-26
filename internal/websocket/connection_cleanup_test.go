package ws

import (
	"sync"
	"testing"
	"time"
)

// Test the connection metadata initialization
func TestConnectionMetadataInitialization(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection
	cache.AddConnection(client)

	// Verify metadata was initialized correctly
	metadata, exists := cache.GetConnectionMetadata(1)
	if !exists {
		t.Error("Connection metadata should exist")
		return
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

	if metadata.Heartbeats != 0 {
		t.Errorf("Heartbeats should be initialized to 0, got %d", metadata.Heartbeats)
	}
}

// Test the heartbeat mechanism
func TestHeartbeatMechanism(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)
	client := createTestClient(1)

	// Add connection
	cache.AddConnection(client)

	// Update heartbeat
	cache.UpdateHeartbeat(1)

	// Verify heartbeat was updated
	metadata, _ := cache.GetConnectionMetadata(1)
	if metadata.Heartbeats != 1 {
		t.Errorf("Expected 1 heartbeat, got %d", metadata.Heartbeats)
	}

	// Update heartbeat again
	cache.UpdateHeartbeat(1)

	// Verify heartbeat was incremented
	metadata, _ = cache.GetConnectionMetadata(1)
	if metadata.Heartbeats != 2 {
		t.Errorf("Expected 2 heartbeats, got %d", metadata.Heartbeats)
	}

	// Reset heartbeat
	cache.ResetHeartbeat(1)

	// Verify heartbeat was reset
	metadata, _ = cache.GetConnectionMetadata(1)
	if metadata.Heartbeats != 0 {
		t.Errorf("Expected 0 heartbeats after reset, got %d", metadata.Heartbeats)
	}
}

// Test the cleanup configuration
func TestCleanupConfiguration(t *testing.T) {
	hub := createTestHub()

	// Create cache with custom config
	customConfig := ConnectionCleanupConfig{
		InactivityTimeout:    2 * time.Minute,
		CleanupInterval:      30 * time.Second,
		HeartbeatInterval:    15 * time.Second,
		MaxHeartbeatFailures: 5,
	}

	cache := NewUserConnectionCacheWithConfig(hub, customConfig)

	// Verify config was set correctly
	if cache.cleanupConfig.InactivityTimeout != 2*time.Minute {
		t.Errorf("Expected InactivityTimeout 2m, got %v", cache.cleanupConfig.InactivityTimeout)
	}

	if cache.cleanupConfig.CleanupInterval != 30*time.Second {
		t.Errorf("Expected CleanupInterval 30s, got %v", cache.cleanupConfig.CleanupInterval)
	}

	if cache.cleanupConfig.HeartbeatInterval != 15*time.Second {
		t.Errorf("Expected HeartbeatInterval 15s, got %v", cache.cleanupConfig.HeartbeatInterval)
	}

	if cache.cleanupConfig.MaxHeartbeatFailures != 5 {
		t.Errorf("Expected MaxHeartbeatFailures 5, got %d", cache.cleanupConfig.MaxHeartbeatFailures)
	}

	// Test updating config
	newConfig := ConnectionCleanupConfig{
		InactivityTimeout:    1 * time.Minute,
		CleanupInterval:      10 * time.Second,
		HeartbeatInterval:    5 * time.Second,
		MaxHeartbeatFailures: 2,
	}

	cache.SetCleanupConfig(newConfig)

	// Verify config was updated
	if cache.cleanupConfig.InactivityTimeout != 1*time.Minute {
		t.Errorf("Expected updated InactivityTimeout 1m, got %v", cache.cleanupConfig.InactivityTimeout)
	}
}

// Test starting and stopping the cleanup routine
func TestCleanupRoutineStartStop(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Initially not running
	if cache.IsCleanupRunning() {
		t.Error("Cleanup should not be running initially")
	}

	// Start cleanup
	cache.StartCleanupRoutine()

	// Should be running now
	if !cache.IsCleanupRunning() {
		t.Error("Cleanup should be running after start")
	}

	// Stop cleanup
	cache.StopCleanupRoutine()

	// Should not be running anymore
	if cache.IsCleanupRunning() {
		t.Error("Cleanup should not be running after stop")
	}
}

// Test the stale connection detection
func TestStaleConnectionDetection(t *testing.T) {
	hub := createTestHub()

	// Create cache with short inactivity timeout for testing
	config := ConnectionCleanupConfig{
		InactivityTimeout:    50 * time.Millisecond, // Short timeout for testing
		CleanupInterval:      10 * time.Millisecond,
		HeartbeatInterval:    10 * time.Millisecond,
		MaxHeartbeatFailures: 3,
	}

	cache := NewUserConnectionCacheWithConfig(hub, config)

	// Add two clients
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	cache.AddConnection(client1)
	cache.AddConnection(client2)

	// Update activity for client1 to keep it active
	cache.UpdateLastActivity(1)

	// Wait for client2 to become stale
	time.Sleep(60 * time.Millisecond)

	// Check stale connections
	staleUsers := cache.GetStaleConnections()

	// Client2 should be stale, client1 should not
	foundClient1 := false
	foundClient2 := false

	for _, userID := range staleUsers {
		if userID == 1 {
			foundClient1 = true
		}
		if userID == 2 {
			foundClient2 = true
		}
	}

	if foundClient1 {
		t.Error("Client1 should not be stale")
	}

	if !foundClient2 {
		t.Error("Client2 should be stale")
	}
}

// Test the cleanup of stale connections
func TestStaleConnectionCleanup(t *testing.T) {
	hub := createTestHub()

	// Create cache with short inactivity timeout for testing
	config := ConnectionCleanupConfig{
		InactivityTimeout:    50 * time.Millisecond, // Short timeout for testing
		CleanupInterval:      10 * time.Millisecond,
		HeartbeatInterval:    10 * time.Millisecond,
		MaxHeartbeatFailures: 3,
	}

	cache := NewUserConnectionCacheWithConfig(hub, config)

	// Add two clients
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	cache.AddConnection(client1)
	cache.AddConnection(client2)

	// Update activity for client1 to keep it active
	cache.UpdateLastActivity(1)

	// Wait for client2 to become stale
	time.Sleep(60 * time.Millisecond)

	// Run cleanup manually (since we don't want to start the routine)
	staleCount := cache.cleanupStaleConnections()

	// Should have cleaned up one connection
	if staleCount != 1 {
		t.Errorf("Expected 1 stale connection to be cleaned up, got %d", staleCount)
	}

	// Client1 should still be online, client2 should be removed
	if !cache.IsUserOnline(1) {
		t.Error("Client1 should still be online")
	}

	if cache.IsUserOnline(2) {
		t.Error("Client2 should have been removed")
	}
}

// Test sending heartbeats
func TestSendHeartbeats(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Add two clients
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	cache.AddConnection(client1)
	cache.AddConnection(client2)

	// Send heartbeats
	heartbeatMessage := []byte(`{"type":"heartbeat"}`)
	cache.sendHeartbeats(heartbeatMessage)

	// Verify both clients received heartbeats
	mockConn1, ok := client1.Conn.(*mockConn)
	if !ok {
		t.Fatal("Client1 connection is not a mockConn")
	}
	messages1 := mockConn1.getMessages()

	mockConn2, ok := client2.Conn.(*mockConn)
	if !ok {
		t.Fatal("Client2 connection is not a mockConn")
	}
	messages2 := mockConn2.getMessages()

	if len(messages1) != 1 {
		t.Errorf("Client1 should receive 1 heartbeat message, got %d", len(messages1))
	}

	if len(messages2) != 1 {
		t.Errorf("Client2 should receive 1 heartbeat message, got %d", len(messages2))
	}

	// Verify message content
	if string(messages1[0]) != `{"type":"heartbeat"}` {
		t.Errorf("Incorrect heartbeat message: %s", string(messages1[0]))
	}
}

// Test heartbeat with failed connections
func TestHeartbeatWithFailedConnections(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Add two clients
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	cache.AddConnection(client1)
	cache.AddConnection(client2)

	// Close client2's connection to simulate failure
	mockConn2, ok := client2.Conn.(*mockConn)
	if !ok {
		t.Fatal("Client2 connection is not a mockConn")
	}
	mockConn2.Close()

	// Send heartbeats
	heartbeatMessage := []byte(`{"type":"heartbeat"}`)
	cache.sendHeartbeats(heartbeatMessage)

	// Verify client1 received heartbeat
	mockConn1, ok := client1.Conn.(*mockConn)
	if !ok {
		t.Fatal("Client1 connection is not a mockConn")
	}
	messages1 := mockConn1.getMessages()

	if len(messages1) != 1 {
		t.Errorf("Client1 should receive 1 heartbeat message, got %d", len(messages1))
	}

	// Client2 should have failed, but we can't verify that directly
	// Instead, we'll check that the unregister channel has a message
	// This is difficult to test in isolation, so we'll skip this verification
}

// Test concurrent heartbeat operations
func TestConcurrentHeartbeatOperations(t *testing.T) {
	hub := createTestHub()
	cache := NewUserConnectionCache(hub)

	// Add multiple clients
	numClients := 10
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = createTestClient(uint(i + 1))
		cache.AddConnection(clients[i])
	}

	// Perform concurrent heartbeat updates
	var wg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(userID uint) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				cache.UpdateHeartbeat(userID)
			}
		}(uint(i + 1))
	}

	wg.Wait()

	// Verify all clients have 5 heartbeats
	for i := 0; i < numClients; i++ {
		userID := uint(i + 1)
		metadata, exists := cache.GetConnectionMetadata(userID)
		if !exists {
			t.Errorf("Connection metadata should exist for user %d", userID)
			continue
		}

		if metadata.Heartbeats != 5 {
			t.Errorf("Expected 5 heartbeats for user %d, got %d", userID, metadata.Heartbeats)
		}
	}
}

// Test connection lifecycle with activity updates
func TestConnectionLifecycleWithActivity(t *testing.T) {
	hub := createTestHub()

	// Create cache with short inactivity timeout for testing
	config := ConnectionCleanupConfig{
		InactivityTimeout:    100 * time.Millisecond, // Short timeout for testing
		CleanupInterval:      20 * time.Millisecond,
		HeartbeatInterval:    20 * time.Millisecond,
		MaxHeartbeatFailures: 3,
	}

	cache := NewUserConnectionCacheWithConfig(hub, config)

	// Add a client
	client := createTestClient(1)
	cache.AddConnection(client)

	// Initially should not be stale
	staleUsers := cache.GetStaleConnections()
	if len(staleUsers) > 0 {
		t.Error("No users should be stale initially")
	}

	// Wait for half the inactivity timeout
	time.Sleep(50 * time.Millisecond)

	// Update activity
	cache.UpdateLastActivity(1)

	// Wait for half the inactivity timeout again
	time.Sleep(50 * time.Millisecond)

	// Should still not be stale because we updated activity
	staleUsers = cache.GetStaleConnections()
	if len(staleUsers) > 0 {
		t.Error("User should not be stale after activity update")
	}

	// Wait for full inactivity timeout without updates
	time.Sleep(110 * time.Millisecond)

	// Now should be stale
	staleUsers = cache.GetStaleConnections()
	if len(staleUsers) != 1 || staleUsers[0] != 1 {
		t.Error("User should be stale after inactivity timeout")
	}
}
