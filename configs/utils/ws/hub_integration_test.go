package ws

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Helper function to get mock connection from client for testing
func getMockConn(client *Client) *mockConn {
	if mockConn, ok := client.Conn.(*mockConn); ok {
		return mockConn
	}
	return nil
}

func TestHubConnectionCacheIntegration(t *testing.T) {
	t.Run("Client registration updates connection cache", func(t *testing.T) {
		hub := createTestHub()
		client := createTestClient(123)

		// Start hub in goroutine
		go hub.WsRun()
		defer func() {
			// Clean shutdown would require more complex setup
		}()

		// Give hub time to start
		time.Sleep(10 * time.Millisecond)

		// Register client
		hub.Register <- client

		// Give time for registration to process
		time.Sleep(10 * time.Millisecond)

		// Verify client is in hub
		hub.mu.RLock()
		_, exists := hub.Clients[client]
		hub.mu.RUnlock()
		assert.True(t, exists, "Client should be registered in hub")

		// Verify client is in connection cache
		assert.True(t, hub.ConnectionCache.IsUserOnline(123), "User should be online in cache")

		// Verify connection metadata exists
		metadata, exists := hub.ConnectionCache.GetConnectionMetadata(123)
		assert.True(t, exists, "Connection metadata should exist")
		assert.Equal(t, uint(123), metadata.UserID, "Metadata should have correct user ID")
		assert.NotZero(t, metadata.ConnectedAt, "Connected timestamp should be set")
	})

	t.Run("Client unregistration cleans up connection cache", func(t *testing.T) {
		hub := createTestHub()
		client := createTestClient(456)

		// Start hub in goroutine
		go hub.WsRun()

		// Give hub time to start
		time.Sleep(10 * time.Millisecond)

		// Register client first
		hub.Register <- client
		time.Sleep(10 * time.Millisecond)

		// Add client to a channel
		client.WsAddChannel(1, hub)

		// Verify client is online and in channel
		assert.True(t, hub.ConnectionCache.IsUserOnline(456), "User should be online")
		users := hub.ConnectionCache.GetOnlineUsersInChannel(1)
		assert.Contains(t, users, uint(456), "User should be in channel")

		// Unregister client
		hub.Unregister <- client
		time.Sleep(10 * time.Millisecond)

		// Verify client is removed from hub
		hub.mu.RLock()
		_, exists := hub.Clients[client]
		hub.mu.RUnlock()
		assert.False(t, exists, "Client should be unregistered from hub")

		// Verify client is removed from connection cache
		assert.False(t, hub.ConnectionCache.IsUserOnline(456), "User should be offline in cache")

		// Verify client is removed from all channels
		users = hub.ConnectionCache.GetOnlineUsersInChannel(1)
		assert.NotContains(t, users, uint(456), "User should be removed from channel")

		// Verify metadata is cleaned up
		_, exists = hub.ConnectionCache.GetConnectionMetadata(456)
		assert.False(t, exists, "Connection metadata should be removed")
	})

	t.Run("Channel subscription updates connection cache", func(t *testing.T) {
		hub := createTestHub()
		client := createTestClient(789)

		// Start hub in goroutine
		go hub.WsRun()

		// Give hub time to start
		time.Sleep(10 * time.Millisecond)

		// Register client
		hub.Register <- client
		time.Sleep(10 * time.Millisecond)

		// Subscribe to channel
		client.WsAddChannel(2, hub)

		// Verify client is in channel in connection cache
		users := hub.ConnectionCache.GetOnlineUsersInChannel(2)
		assert.Contains(t, users, uint(789), "User should be in channel")

		// Verify metadata is updated
		metadata, exists := hub.ConnectionCache.GetConnectionMetadata(789)
		assert.True(t, exists, "Connection metadata should exist")
		assert.True(t, metadata.Channels[2], "Channel should be in metadata")

		// Unsubscribe from channel
		client.WsRemoveChannel(2, hub)

		// Verify client is removed from channel in connection cache
		users = hub.ConnectionCache.GetOnlineUsersInChannel(2)
		assert.NotContains(t, users, uint(789), "User should be removed from channel")

		// Verify metadata is updated
		metadata, exists = hub.ConnectionCache.GetConnectionMetadata(789)
		assert.True(t, exists, "Connection metadata should still exist")
		assert.False(t, metadata.Channels[2], "Channel should be removed from metadata")
	})

	t.Run("Multiple clients and channels integration", func(t *testing.T) {
		hub := createTestHub()

		// Create multiple clients
		client1 := createTestClient(100)
		client2 := createTestClient(200)
		client3 := createTestClient(300)

		// Start hub in goroutine
		go hub.WsRun()

		// Give hub time to start
		time.Sleep(10 * time.Millisecond)

		// Register all clients
		hub.Register <- client1
		hub.Register <- client2
		hub.Register <- client3
		time.Sleep(10 * time.Millisecond)

		// Subscribe clients to different channels
		client1.WsAddChannel(1, hub) // Client 1 -> Channel 1
		client1.WsAddChannel(2, hub) // Client 1 -> Channel 2
		client2.WsAddChannel(1, hub) // Client 2 -> Channel 1
		client3.WsAddChannel(2, hub) // Client 3 -> Channel 2

		// Verify all users are online
		onlineUsers := hub.ConnectionCache.GetOnlineUsers()
		assert.Len(t, onlineUsers, 3, "Should have 3 online users")
		assert.Contains(t, onlineUsers, uint(100), "User 100 should be online")
		assert.Contains(t, onlineUsers, uint(200), "User 200 should be online")
		assert.Contains(t, onlineUsers, uint(300), "User 300 should be online")

		// Verify channel subscriptions
		channel1Users := hub.ConnectionCache.GetOnlineUsersInChannel(1)
		assert.Len(t, channel1Users, 2, "Channel 1 should have 2 users")
		assert.Contains(t, channel1Users, uint(100), "User 100 should be in channel 1")
		assert.Contains(t, channel1Users, uint(200), "User 200 should be in channel 1")

		channel2Users := hub.ConnectionCache.GetOnlineUsersInChannel(2)
		assert.Len(t, channel2Users, 2, "Channel 2 should have 2 users")
		assert.Contains(t, channel2Users, uint(100), "User 100 should be in channel 2")
		assert.Contains(t, channel2Users, uint(300), "User 300 should be in channel 2")

		// Unregister one client
		hub.Unregister <- client2
		time.Sleep(10 * time.Millisecond)

		// Verify client 2 is removed from all channels
		channel1Users = hub.ConnectionCache.GetOnlineUsersInChannel(1)
		assert.Len(t, channel1Users, 1, "Channel 1 should have 1 user after unregistration")
		assert.Contains(t, channel1Users, uint(100), "User 100 should still be in channel 1")
		assert.NotContains(t, channel1Users, uint(200), "User 200 should be removed from channel 1")

		// Verify online users count
		onlineUsers = hub.ConnectionCache.GetOnlineUsers()
		assert.Len(t, onlineUsers, 2, "Should have 2 online users after unregistration")
		assert.NotContains(t, onlineUsers, uint(200), "User 200 should be offline")
	})

	t.Run("Connection cache consistency during concurrent operations", func(t *testing.T) {
		hub := createTestHub()

		// Start hub in goroutine
		go hub.WsRun()
		time.Sleep(10 * time.Millisecond)

		const numClients = 10
		const numChannels = 3

		var wg sync.WaitGroup
		clients := make([]*Client, numClients)

		// Create and register clients concurrently
		for i := 0; i < numClients; i++ {
			wg.Add(1)
			go func(clientID int) {
				defer wg.Done()

				client := createTestClient(uint(clientID + 1))
				clients[clientID] = client

				// Register client
				hub.Register <- client
				time.Sleep(5 * time.Millisecond)

				// Subscribe to random channels
				for j := 0; j < numChannels; j++ {
					if (clientID+j)%2 == 0 { // Subscribe to some channels based on pattern
						client.WsAddChannel(uint(j+1), hub)
					}
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(50 * time.Millisecond) // Allow all operations to complete

		// Verify all clients are registered
		onlineUsers := hub.ConnectionCache.GetOnlineUsers()
		assert.Len(t, onlineUsers, numClients, "All clients should be online")

		// Verify channel subscriptions are consistent
		for channelID := uint(1); channelID <= numChannels; channelID++ {
			users := hub.ConnectionCache.GetOnlineUsersInChannel(channelID)

			// Verify each user in the channel is actually online
			for _, userID := range users {
				assert.True(t, hub.ConnectionCache.IsUserOnline(userID),
					"User %d in channel %d should be online", userID, channelID)

				// Verify metadata consistency
				metadata, exists := hub.ConnectionCache.GetConnectionMetadata(userID)
				assert.True(t, exists, "Metadata should exist for user %d", userID)
				assert.True(t, metadata.Channels[channelID],
					"Channel %d should be in metadata for user %d", channelID, userID)
			}
		}

		// Unregister half the clients concurrently
		for i := 0; i < numClients/2; i++ {
			wg.Add(1)
			go func(clientID int) {
				defer wg.Done()
				hub.Unregister <- clients[clientID]
			}(i)
		}

		wg.Wait()
		time.Sleep(50 * time.Millisecond) // Allow all operations to complete

		// Verify remaining clients
		onlineUsers = hub.ConnectionCache.GetOnlineUsers()
		assert.Len(t, onlineUsers, numClients/2, "Half the clients should remain online")

		// Verify channel consistency after partial unregistration
		for channelID := uint(1); channelID <= numChannels; channelID++ {
			users := hub.ConnectionCache.GetOnlineUsersInChannel(channelID)

			// All users in channels should still be online
			for _, userID := range users {
				assert.True(t, hub.ConnectionCache.IsUserOnline(userID),
					"User %d in channel %d should still be online", userID, channelID)
			}
		}
	})
}
