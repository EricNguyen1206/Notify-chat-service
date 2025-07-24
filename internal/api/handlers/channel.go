package handlers

import (
	"net/http"
	"strconv"

	"chat-service/internal/api/middleware"
	"chat-service/internal/services"

	"github.com/gin-gonic/gin"
)

type ChannelHandler struct {
	channelService *services.ChannelService
}

func NewChannelHandler(channelService *services.ChannelService) *ChannelHandler {
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

// GetUserChannels godoc
// @Summary Get user's channels
// @Description Get all channels that the current user is a member of
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.ChannelListResponse "List of user's channels"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /channels/ [get]
func (h *ChannelHandler) GetUserChannels(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	channels, err := h.channelService.GetUserChannels(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get channel"})
		return
	}
	c.JSON(http.StatusOK, channels)
}

// CreateChannel godoc
// @Summary Create a new channel
// @Description Create a new channel with the specified name
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body map[string]string true "Channel creation data"
// @Success 200 {object} models.ChannelResponse "Channel created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid input data"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /channels/ [post]
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

// UpdateChannel godoc
// @Summary Update channel
// @Description Update the name of an existing channel
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Param request body map[string]string true "Channel update data"
// @Success 200 {object} map[string]string "Channel updated successfully"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid input data"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /channels/{id} [put]
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

// DeleteChannel godoc
// @Summary Delete channel
// @Description Delete a channel (only channel owner can delete)
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Success 200 {object} map[string]string "Channel deleted successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /channels/{id} [delete]
func (h *ChannelHandler) DeleteChannel(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.channelService.DeleteChannel(userID, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Channel deleted"})
}

// GetChannelByID godoc
// @Summary Get channel by ID
// @Description Get detailed information about a specific channel
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Success 200 {object} models.ChannelResponse "Channel details retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 404 {object} map[string]interface{} "Channel not found"
// @Router /channels/{id} [get]
func (h *ChannelHandler) GetChannelByID(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	channel, err := h.channelService.GetChannelByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	c.JSON(http.StatusOK, channel)
}

// AddUserToChannel godoc
// @Summary Add user to channel
// @Description Add a user to a channel (only channel owner can add users)
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Param request body map[string]uint true "User addition data"
// @Success 200 {object} map[string]string "User added to channel successfully"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid input data"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /channels/{id}/user [post]
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

// LeaveChannel godoc
// @Summary Leave channel
// @Description Remove the current user from a channel
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Success 200 {object} map[string]string "User left channel successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /channels/{id}/user [put]
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

// RemoveUserFromChannel godoc
// @Summary Remove user from channel
// @Description Remove a user from a channel (only channel owner can remove users)
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Param request body map[string]uint true "User removal data"
// @Success 200 {object} map[string]string "User removed from channel successfully"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid input data"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /channels/{id}/user [delete]
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
