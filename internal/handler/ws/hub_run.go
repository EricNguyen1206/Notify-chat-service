package ws

import (
	"log"
)

// Run starts the WebSocket hub
// Also starts the Redis listener for cross-instance communication
func (h *Hub) Run() {
	// Start Redis message listener for cross-instance communication
	go h.redisListener()

	// Start connection cache cleanup routine
	h.ConnectionCache.StartCleanupRoutine()
	log.Printf("Started connection cache cleanup routine")

	for {
		select {
		case client := <-h.Register:
			// Register new client - add to active clients map
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()

			// Add client to connection cache
			h.ConnectionCache.AddConnection(client)
			log.Printf("Client registered: %d", client.ID)

			// Publish online status to Redis for distributed cache consistency
			h.publishUserPresenceUpdate(client.ID, 0, "online")

		case client := <-h.Unregister:
			// Get client channels before removing from cache
			var clientChannels []uint
			if metadata, exists := h.ConnectionCache.GetConnectionMetadata(client.ID); exists {
				for channelID := range metadata.Channels {
					clientChannels = append(clientChannels, channelID)
				}
			}

			// Unregister client - remove from active clients and close connection
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				client.Conn.Close()
				log.Printf("Client unregistered: %d", client.ID)
			}
			h.mu.Unlock()

			// Remove client from connection cache
			h.ConnectionCache.RemoveConnection(client.ID)

			// Publish offline status to Redis for distributed cache consistency
			h.publishUserPresenceUpdate(client.ID, 0, "offline")

		case message := <-h.Broadcast:
			// Broadcast message to all clients in the specified channel
			h.broadcastToLocalClients(message.ChannelID, message.Data)
		}
	}
}

// Note: The actual implementations of redisListener and publishUserPresenceUpdate
// are in redis_pubsub.go
