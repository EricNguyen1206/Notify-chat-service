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
		channels.GET("/", h.GetUserChannels)
		channels.POST("/", h.CreateChannel)
		// Individual channel routes with :id parameter
		channels.GET("/:id", h.GetChannelByID)
		channels.PUT("/:id", h.UpdateChannel)
		channels.DELETE("/:id", h.DeleteChannel)
		// user-channel relation logic
		channels.POST("/:id/user", h.AddUserToChannel)
		channels.PUT("/:id/user", h.LeaveChannel)
		channels.DELETE("/:id/user", h.RemoveUserFromChannel)
	}
}

func (h *ChannelHandler) GetUserChannels(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	channels, err := h.channelService.GetUserChannels(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get channel"})
		return
	}
	c.JSON(http.StatusOK, channels)
}

func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	channel, err := h.channelService.CreateChannel(req.Name, userID)
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
	userID := c.MustGet("user_id").(uint)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.channelService.DeleteChannel(userID, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

func (h *ChannelHandler) AddUserToChannel(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	channelID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		TargetUserID uint `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.channelService.AddUserToChannel(userID, uint(channelID), req.TargetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User added to channel"})
}

func (h *ChannelHandler) LeaveChannel(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	err := h.channelService.LeaveChannel(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave channel"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Left channel"})
}

func (h *ChannelHandler) RemoveUserFromChannel(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	channelID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		UserID uint `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.channelService.RemoveUserFromChannel(userID, uint(channelID), req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User removed from channel"})
}
