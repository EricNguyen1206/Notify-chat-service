package handler

import (
	"net/http"
	"strconv"

	"chat-service/configs/middleware"
	"chat-service/internal/service"

	"github.com/gin-gonic/gin"
)

type ChannelHandler struct {
	channelService *service.ChannelService
}

func NewChannelHandler(channelService *service.ChannelService) *ChannelHandler {
	return &ChannelHandler{channelService: channelService}
}

// RegisterRoutes maps HTTP methods to handler functions
func (h *ChannelHandler) RegisterRoutes(r *gin.RouterGroup) {
	channels := r.Group("/channels")
	{
		channels.Use(middleware.Auth())
		channels.POST("/", h.CreateChannel)
		// Route for getting channels by server and user - must come before /:id routes
		channels.GET("/server/:serverId/user/:userId", h.GetChannelsByUserAndServer)
		// Individual channel routes with :id parameter
		channels.PUT("/:id", h.UpdateChannel)
		channels.DELETE("/:id", h.DeleteChannel)
		channels.GET("/:id", h.GetChannelByID)
		channels.POST("/:id/join", h.JoinChannel)
		channels.POST("/:id/leave", h.LeaveChannel)
		channels.DELETE("/:id/remove/:userId", h.RemoveUserFromChannel)
		channels.GET("/:id/messages", h.GetMessagesByChannelID)
	}
}

func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		OwnerID  uint   `json:"ownerId"`
		ServerID uint   `json:"serverId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	channel, err := h.channelService.CreateChannel(req.Name, req.OwnerID, req.ServerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel"})
		return
	}
	c.JSON(http.StatusOK, channel)
}

func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.channelService.UpdateChannel(uint(id), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Channel updated"})
}

func (h *ChannelHandler) DeleteChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.channelService.DeleteChannel(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Channel deleted"})
}

func (h *ChannelHandler) GetChannelByID(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	channel, err := h.channelService.GetChannelByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	c.JSON(http.StatusOK, channel)
}

func (h *ChannelHandler) GetChannelsByUserAndServer(c *gin.Context) {
	serverID, _ := strconv.ParseUint(c.Param("serverId"), 10, 64)
	userID, _ := strconv.ParseUint(c.Param("userId"), 10, 64)

	channels, err := h.channelService.GetChannelsByUserAndServer(uint(userID), uint(serverID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channels"})
		return
	}
	c.JSON(http.StatusOK, channels)
}

func (h *ChannelHandler) JoinChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		UserID uint `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.channelService.JoinChannel(uint(id), req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join channel"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Joined channel"})
}

func (h *ChannelHandler) LeaveChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		UserID uint `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.channelService.LeaveChannel(uint(id), req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave channel"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Left channel"})
}

func (h *ChannelHandler) RemoveUserFromChannel(c *gin.Context) {
	channelID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID, _ := strconv.ParseUint(c.Param("userId"), 10, 64)

	err := h.channelService.RemoveUserFromChannel(uint(channelID), uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User removed from channel"})
}

func (h *ChannelHandler) GetMessagesByChannelID(c *gin.Context) {
	channelID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	messages, err := h.channelService.GetChatMessagesByChannel(uint(channelID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load messages"})
		return
	}
	c.JSON(http.StatusOK, messages)
}
