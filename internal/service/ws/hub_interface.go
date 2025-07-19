package ws

// HubInterface defines the interface for the WebSocket hub
// This allows us to avoid circular dependencies between packages
type HubInterface interface {
	// Run starts the WebSocket hub
	Run()

	// BroadcastMessage broadcasts a message to all clients in a channel
	BroadcastMessage(msg interface{})

	// Methods for client registration/unregistration are handled through channels
}
