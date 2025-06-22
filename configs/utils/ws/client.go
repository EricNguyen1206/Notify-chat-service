package ws

import (
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a connected user
type Client struct {
	UserID uint
	Conn   *websocket.Conn
	// ServerId string
	Send chan []byte
}

type DirectMessage struct {
	FromUserID uint
	ToUserID   uint
	Content    string
	Timestamp  time.Time
}
