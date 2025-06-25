package ws

import (
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a connected user
type Client struct {
	UserID uint
	Conn   *websocket.Conn
	Send chan []byte
}

type DirectMessage struct {
	FromUserID uint
	ToUserID   uint
	Content    string
	Timestamp  time.Time
}

type ChannelMessage struct {
	FromUserID uint
	ChannelID  uint
	Content    string
	Timestamp  time.Time
}

