package ws

import (
	"testing"
	"time"
)

// Test the stale connection detection with direct time setting
func TestStaleConnectionDetectionDirect(t *testing.T) {
	hub := createTestHub()

	// Create cache with longer timeout for reliability
	config := ConnectionCleanupConfig{
		InactivityTimeout:    1 * time.Minute, // Use a longer timeout for reliability
		CleanupInterval:      10 * time.Second,
		HeartbeatInterval:    10 * time.Second,
		MaxHeartbeatFailures: 3,
	}

	cache := NewUserConnectionCacheWithConfig(hub, config)

	// Add two clients
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	cache.AddConnection(client1)
	cache.AddConnection(client2)

	now := time.Now()

	// Set client1 to be active (just now)
	cache.SetLastActivityTime(1, now)

	// Set client2 to be stale (2 minutes ago)
	cache.SetLastActivityTime(2, now.Add(-2*time.Minute))

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

// Test the cleanup of stale connections with direct time setting
func TestStaleConnectionCleanupDirect(t *testing.T) {
	hub := createTestHub()

	// Create cache with longer timeout for reliability
	config := ConnectionCleanupConfig{
		InactivityTimeout:    1 * time.Minute, // Use a longer timeout for reliability
		CleanupInterval:      10 * time.Second,
		HeartbeatInterval:    10 * time.Second,
		MaxHeartbeatFailures: 3,
	}

	cache := NewUserConnectionCacheWithConfig(hub, config)

	// Add two clients
	client1 := createTestClient(1)
	client2 := createTestClient(2)
	cache.AddConnection(client1)
	cache.AddConnection(client2)

	now := time.Now()

	// Set client1 to be active (just now)
	cache.SetLastActivityTime(1, now)

	// Set client2 to be stale (2 minutes ago)
	cache.SetLastActivityTime(2, now.Add(-2*time.Minute))

	// Verify client2 is stale before cleanup
	staleUsers := cache.GetStaleConnections()
	if len(staleUsers) != 1 || staleUsers[0] != 2 {
		t.Fatalf("Expected only client2 to be stale, got %v", staleUsers)
	}

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

// Test connection lifecycle with activity updates with direct time setting
func TestConnectionLifecycleWithActivityDirect(t *testing.T) {
	hub := createTestHub()

	// Create cache with longer timeout for reliability
	config := ConnectionCleanupConfig{
		InactivityTimeout:    1 * time.Minute, // Use a longer timeout for reliability
		CleanupInterval:      10 * time.Second,
		HeartbeatInterval:    10 * time.Second,
		MaxHeartbeatFailures: 3,
	}

	cache := NewUserConnectionCacheWithConfig(hub, config)

	// Add a client
	client := createTestClient(1)
	cache.AddConnection(client)

	now := time.Now()

	// Set initial activity time to now
	cache.SetLastActivityTime(1, now)

	// Initially should not be stale
	staleUsers := cache.GetStaleConnections()
	if len(staleUsers) > 0 {
		t.Error("No users should be stale initially")
	}

	// Set activity time to 30 seconds ago (half the timeout)
	cache.SetLastActivityTime(1, now.Add(-30*time.Second))

	// Should still not be stale
	staleUsers = cache.GetStaleConnections()
	if len(staleUsers) > 0 {
		t.Error("User should not be stale after half the timeout")
	}

	// Set activity time to 2 minutes ago (beyond the timeout)
	cache.SetLastActivityTime(1, now.Add(-2*time.Minute))

	// Now should be stale
	staleUsers = cache.GetStaleConnections()
	if len(staleUsers) != 1 || staleUsers[0] != 1 {
		t.Error("User should be stale after inactivity timeout")
	}
}
