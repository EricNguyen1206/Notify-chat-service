package ws

import (
	"testing"
	"time"
)

// TestConnectionMetadataTracking tests the tracking of connection metadata
func TestConnectionMetadataTracking(t *testing.T) {
	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Create test client
	client := createTestClient(1)

	// Test metadata initialization
	t.Run("MetadataInitialization", func(t *testing.T) {
		// Register client with the hub
		hub.Clients[client] = true
		hub.ConnectionCache.AddConnection(client)

		// Verify metadata was created
		metadata, exists := hub.ConnectionCache.GetConnectionMetadata(1)
		if !exists {
			t.Fatal("Connection metadata should exist")
		}

		// Check metadata fields
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
		if len(metadata.Channels) != 0 {
			t.Errorf("Channels map should be empty, got %d entries", len(metadata.Channels))
		}
		if metadata.Heartbeats != 0 {
			t.Errorf("Heartbeats should be 0, got %d", metadata.Heartbeats)
		}
	})

	// Test channel subscription tracking
	t.Run("ChannelSubscriptionTracking", func(t *testing.T) {
		// Subscribe client to channels
		client.WsAddChannel(100, hub)
		client.WsAddChannel(200, hub)

		// Verify metadata was updated
		metadata, exists := hub.ConnectionCache.GetConnectionMetadata(1)
		if !exists {
			t.Fatal("Connection metadata should exist")
		}

		// Check channel subscriptions
		if !metadata.Channels[100] {
			t.Error("Metadata should include channel 100")
		}
		if !metadata.Channels[200] {
			t.Error("Metadata should include channel 200")
		}

		// Unsubscribe from a channel
		client.WsRemoveChannel(100, hub)

		// Verify metadata was updated
		metadata, exists = hub.ConnectionCache.GetConnectionMetadata(1)
		if !exists {
			t.Fatal("Connection metadata should exist")
		}

		// Check channel subscriptions
		if metadata.Channels[100] {
			t.Error("Metadata should not include channel 100")
		}
		if !metadata.Channels[200] {
			t.Error("Metadata should still include channel 200")
		}
	})

	// Test activity tracking
	t.Run("ActivityTracking", func(t *testing.T) {
		// Get initial last activity time
		initialMetadata, _ := hub.ConnectionCache.GetConnectionMetadata(1)
		initialActivity := initialMetadata.LastActivity

		// Wait a moment
		time.Sleep(10 * time.Millisecond)

		// Update last activity
		hub.ConnectionCache.UpdateLastActivity(1)

		// Verify last activity was updated
		updatedMetadata, _ := hub.ConnectionCache.GetConnectionMetadata(1)
		updatedActivity := updatedMetadata.LastActivity

		if !updatedActivity.After(initialActivity) {
			t.Error("LastActivity should be updated to a later time")
		}
	})

	// Test heartbeat tracking
	t.Run("HeartbeatTracking", func(t *testing.T) {
		// Get initial heartbeat count
		initialMetadata, _ := hub.ConnectionCache.GetConnectionMetadata(1)
		initialHeartbeats := initialMetadata.Heartbeats

		// Update heartbeat
		hub.ConnectionCache.UpdateHeartbeat(1)

		// Verify heartbeat count was incremented
		updatedMetadata, _ := hub.ConnectionCache.GetConnectionMetadata(1)
		updatedHeartbeats := updatedMetadata.Heartbeats

		if updatedHeartbeats != initialHeartbeats+1 {
			t.Errorf("Heartbeats should be incremented by 1, got %d vs initial %d",
				updatedHeartbeats, initialHeartbeats)
		}

		// Reset heartbeat
		hub.ConnectionCache.ResetHeartbeat(1)

		// Verify heartbeat count was reset
		resetMetadata, _ := hub.ConnectionCache.GetConnectionMetadata(1)
		resetHeartbeats := resetMetadata.Heartbeats

		if resetHeartbeats != 0 {
			t.Errorf("Heartbeats should be reset to 0, got %d", resetHeartbeats)
		}
	})
}

// TestConnectionCleanup tests the automatic cleanup of stale connections
func TestConnectionCleanup(t *testing.T) {
	// Create a hub with connection cache with short cleanup intervals for testing
	hub := createTestHub()

	// Create a custom cleanup config with short intervals
	cleanupConfig := ConnectionCleanupConfig{
		InactivityTimeout:    200 * time.Millisecond, // Short timeout for testing
		CleanupInterval:      100 * time.Millisecond, // Run cleanup frequently
		HeartbeatInterval:    50 * time.Millisecond,  // Send heartbeats frequently
		MaxHeartbeatFailures: 2,
	}

	hub.ConnectionCache = NewUserConnectionCacheWithConfig(hub, cleanupConfig)
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
	client3.WsAddChannel(100, hub)

	// Start cleanup routine
	hub.ConnectionCache.StartCleanupRoutine()

	// Test automatic cleanup of stale connections
	t.Run("StaleConnectionCleanup", func(t *testing.T) {
		// Verify all clients are initially online
		if !hub.ConnectionCache.IsUserOnline(1) {
			t.Error("Client 1 should be online")
		}
		if !hub.ConnectionCache.IsUserOnline(2) {
			t.Error("Client 2 should be online")
		}
		if !hub.ConnectionCache.IsUserOnline(3) {
			t.Error("Client 3 should be online")
		}

		// Set client2's last activity to a time in the past
		staleTime := time.Now().Add(-500 * time.Millisecond) // Older than inactivity timeout
		hub.ConnectionCache.SetLastActivityTime(2, staleTime)

		// Keep client1 and client3 active
		hub.ConnectionCache.UpdateLastActivity(1)
		hub.ConnectionCache.UpdateLastActivity(3)

		// Wait for cleanup routine to run
		time.Sleep(300 * time.Millisecond)

		// Verify client2 was removed due to inactivity
		if hub.ConnectionCache.IsUserOnline(2) {
			t.Error("Client 2 should be removed due to inactivity")
		}

		// Verify client1 and client3 are still online
		if !hub.ConnectionCache.IsUserOnline(1) {
			t.Error("Client 1 should still be online")
		}
		if !hub.ConnectionCache.IsUserOnline(3) {
			t.Error("Client 3 should still be online")
		}

		// Verify client2 was removed from channel subscriptions
		users100 := hub.ConnectionCache.GetOnlineUsersInChannel(100)
		for _, userID := range users100 {
			if userID == 2 {
				t.Error("Client 2 should be removed from channel 100")
			}
		}
	})

	// Test heartbeat mechanism
	t.Run("HeartbeatMechanism", func(t *testing.T) {
		// Wait for heartbeats to be sent
		time.Sleep(100 * time.Millisecond)

		// Check if clients received heartbeat messages
		mockConn1 := client1.Conn.(*mockConn)
		messages1 := mockConn1.getMessages()

		// Find heartbeat messages
		heartbeatCount := 0
		for _, msg := range messages1 {
			if string(msg) == `{"type":"heartbeat"}` {
				heartbeatCount++
			}
		}

		if heartbeatCount == 0 {
			t.Error("Client 1 should receive heartbeat messages")
		}
	})

	// Stop cleanup routine
	hub.ConnectionCache.StopCleanupRoutine()
}

// TestGetStaleConnections tests the GetStaleConnections method
func TestGetStaleConnections(t *testing.T) {
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

	// Set client2's last activity to a time in the past
	staleTime := time.Now().Add(-10 * time.Minute) // Much older than default inactivity timeout
	hub.ConnectionCache.SetLastActivityTime(2, staleTime)

	// Get stale connections
	staleConnections := hub.ConnectionCache.GetStaleConnections()

	// Verify client2 is identified as stale
	found := false
	for _, userID := range staleConnections {
		if userID == 2 {
			found = true
			break
		}
	}

	if !found {
		t.Error("Client 2 should be identified as stale")
	}

	// Verify client1 and client3 are not identified as stale
	for _, userID := range staleConnections {
		if userID == 1 || userID == 3 {
			t.Errorf("Client %d should not be identified as stale", userID)
		}
	}
}
