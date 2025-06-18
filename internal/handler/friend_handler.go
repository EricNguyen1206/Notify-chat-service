package handler

import (
	"chat-service/configs/middleware"
	"chat-service/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type FriendHandler struct {
	friendService *service.FriendService
}

func NewFriendHandler(friendService *service.FriendService) *FriendHandler {
	return &FriendHandler{friendService: friendService}
}

func (h *FriendHandler) AddFriend(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var input struct {
		FriendID uint `json:"friendId"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := h.friendService.AddFriend(userID, input.FriendID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Friend request sent"})
}

func (h *FriendHandler) GetFriends(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	friends, err := h.friendService.GetFriends(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"friends": friends,
	})
}

// Register routes
func (h *FriendHandler) RegisterRoutes(r *gin.RouterGroup) {
	friends := r.Group("/friends")
	{
		// Protected routes
		friends.Use(middleware.Auth())
		friends.POST("/", h.AddFriend)
		friends.GET("/", h.GetFriends)
		// friends.GET("/friends/pending", h.GetPendingFriends)
		// friends.POST("/friends/accept/:id", h.AcceptFriendRequest)
		// friends.POST("/friends/reject/:id", h.RejectFriendRequest)
	}
}
