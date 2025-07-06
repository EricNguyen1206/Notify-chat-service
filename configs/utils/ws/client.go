package ws

// import (
// 	"github.com/gorilla/websocket"
// )

// // Client represents a connected user
// type Client struct {
// 	UserID uint
// 	Conn   *websocket.Conn // The WebSocket connection
// 	Send   chan []byte     // Outgoing message queue
// 	Hub    *Hub            // Pointer to the main Hub
// }

// // ReadPump listens for incoming messages from the WebSocket
// func (c *Client) ReadPump() {
// 	defer func() {
// 		c.Hub.Unregister <- c
// 		c.Conn.Close()
// 	}()

// 	for {
// 		_, message, err := c.Conn.ReadMessage()
// 		if err != nil {
// 			break
// 		}
// 		// Pass raw message to Hub to decode and route
// 		c.Hub.Broadcast <- Envelope{
// 			Client:  c,
// 			Message: message,
// 		}
// 	}
// }

// // WritePump writes messages from the Send channel to the WebSocket
// func (c *Client) WritePump() {
// 	defer c.Conn.Close()
// 	for msg := range c.Send {
// 		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
// 		if err != nil {
// 			break
// 		}
// 	}
// }

// // type DirectMessage struct {
// // 	FromUserID uint
// // 	ToUserID   uint
// // 	Content    string
// // 	Timestamp  time.Time
// // }

// // type ChannelMessage struct {
// // 	FromUserID uint
// // 	ChannelID  uint
// // 	Content    string
// // 	Timestamp  time.Time
// // }
