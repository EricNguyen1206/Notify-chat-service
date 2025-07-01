package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origin
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Add these constants for WebSocket message types
const (
	// TextMessage denotes a text data message
	TextMessage = websocket.TextMessage
	// BinaryMessage denotes a binary data message
	BinaryMessage = websocket.BinaryMessage
	// CloseMessage denotes a close control message
	CloseMessage = websocket.CloseMessage
	// PingMessage denotes a ping control message
	PingMessage = websocket.PingMessage
	// PongMessage denotes a pong control message
	PongMessage = websocket.PongMessage
)
