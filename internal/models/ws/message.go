package ws

// ChannelMessage represents a message to be broadcasted to a specific channel
// Used for internal communication between hub components
type ChannelMessage struct {
	ChannelID uint   `json:"channelId"` // Target channel identifier
	Data      []byte `json:"data"`      // Serialized message data (JSON)
}
