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

// AddFriend godoc
// @Summary Add a friend
// @Description Send a friend request to another user
// @Tags friends
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body map[string]uint true "Friend request data"
// @Success 200 {object} map[string]string "Friend request sent successfully"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid input data"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /friends/ [post]
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

// GetFriends godoc
// @Summary Get user's friends
// @Description Get the list of friends for the current user
// @Tags friends
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string][]models.FriendResponse "List of friends retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /friends/ [get]
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
	}
}
