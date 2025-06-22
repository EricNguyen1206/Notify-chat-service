package handler

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"strconv"
// 	"time"

// 	"chat-service/configs/utils/ws"

// 	"github.com/gin-gonic/gin"
// 	"github.com/gorilla/websocket"
// )

// var upgrader = websocket.Upgrader{
// 	CheckOrigin: func(r *http.Request) bool { return true },
// }

// func HandleWebSocket(c *gin.Context) {
// 	// Get user ID from query parameter
// 	userIDStr := c.Query("userId")
// 	userID, err := strconv.ParseUint(userIDStr, 10, 32)
// 	if err != nil {
// 		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid user ID: %v", err)})
// 		return
// 	}
// 	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
// 	if err != nil {
// 		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade to WebSocket"})
// 		return
// 	}

// 	client := &ws.Client{
// 		UserID: uint(userID),
// 		Conn:   conn,
// 		Send:   make(chan []byte, 256),
// 	}

// 	ws.ChatHub.RegisterClient(client)

// 	go writePump(client)
// 	readPump(client)
// }

// func readPump(client *ws.Client) {
// 	defer func() {
// 		ws.ChatHub.UnregisterClient(client)
// 		client.Conn.Close()
// 	}()

// 	for {
// 		_, msg, err := client.Conn.ReadMessage()
// 		if err != nil {
// 			break
// 		}

// 		var parsed struct {
// 			To      uint   `json:"to"`
// 			Content string `json:"content"`
// 		}
// 		if err := json.Unmarshal(msg, &parsed); err != nil {
// 			continue
// 		}

// 		ws.ChatHub.SendDirectMessage(ws.DirectMessage{
// 			FromUserID: client.UserID,
// 			ToUserID:   parsed.To,
// 			Content:    parsed.Content,
// 			Timestamp:  time.Now().UTC(),
// 		})
// 	}
// }

// func writePump(client *ws.Client) {
// 	defer client.Conn.Close()

// 	for {
// 		select {
// 		case msg, ok := <-client.Send:
// 			if !ok {
// 				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
// 				return
// 			}
// 			client.Conn.WriteMessage(websocket.TextMessage, msg)
// 		}
// 	}
// }
