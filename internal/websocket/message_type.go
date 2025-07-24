package websocket

// WebSocket message types
const (
	// Connection events
	MessageTypeConnect    = "connection.connect"
	MessageTypeDisconnect = "connection.disconnect"
	MessageTypePing       = "connection.ping"
	MessageTypePong       = "connection.pong"

	// Channel events
	MessageTypeJoinChannel    = "channel.join"
	MessageTypeLeaveChannel   = "channel.leave"
	MessageTypeChannelMessage = "channel.message"
	MessageTypeTyping         = "channel.typing"
	MessageTypeStopTyping     = "channel.stop_typing"

	// Channel member events
	MessageTypeMemberJoin  = "channel.member.join"
	MessageTypeMemberLeave = "channel.member.leave"

	// User events
	MessageTypeUserStatus   = "user.status"
	MessageTypeNotification = "user.notification"

	// Error events
	MessageTypeError = "error"
)

// Base message structure
type Message struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
	UserID    string                 `json:"user_id,omitempty"`
}

// Specific message data structures
type JoinChannelData struct {
	ChannelID string `json:"channel_id" binding:"required"`
}

type LeaveChannelData struct {
	ChannelID string `json:"channel_id" binding:"required"`
}

type ChannelMessageData struct {
	ChannelID string `json:"channel_id" binding:"required"`
	Content   string `json:"content" binding:"required"`
	ReplyToID string `json:"reply_to_id,omitempty"`
}

type TypingData struct {
	ChannelID string `json:"channel_id" binding:"required"`
	IsTyping  bool   `json:"is_typing"`
}

type UserStatusData struct {
	Status   string `json:"status"`
	LastSeen int64  `json:"last_seen"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
