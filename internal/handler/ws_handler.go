package handler

import (
	"chat-service/configs"
	"chat-service/configs/utils"
	"chat-service/configs/utils/ws"
	"log"

	"github.com/gin-gonic/gin"
)

type WSHandler struct {
	hub *ws.Hub
}

func NewWSHandler(hub *ws.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	userID, _ := utils.StringToUint(c.GetString("userID"))

	// Nâng cấp kết nối lên WebSocket
	conn, err := configs.ConfigInstance.WSUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("🔴 WebSocket upgrade failed: %v", err)
		return
	}
	log.Printf("✅ WebSocket upgrade success")

	// Tạo client mới
	client := &ws.Client{
		ID:   userID,
		Conn: conn,
	}

	// Regist client to hub
	h.hub.Register <- client

	// Start handle message
	go client.HandleIncomingMessages(h.hub)
}
