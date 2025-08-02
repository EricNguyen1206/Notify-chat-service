package handlers

import (
	"chat-service/internal/models"
	"chat-service/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type UserHandler struct {
	userService services.UserService
	redisClient *redis.Client
}

func NewUserHandler(userService services.UserService, redisClient *redis.Client) *UserHandler {
	return &UserHandler{userService: userService, redisClient: redisClient}
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with username, email, and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "User registration data"
// @Success 201 {object} models.UserResponse "User created successfully"
// @Failure 400 {object} models.ErrorResponse "Bad request - invalid input data"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid input data",
			Details: err.Error(),
		})
		return
	}

	user, err := h.userService.Register(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Register failed",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// Login godoc
// @Summary User login
// @Description Authenticate user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "User login credentials"
// @Success 200 {object} models.LoginResponse "Login successful - returns JWT token and user data"
// @Failure 400 {object} models.ErrorResponse "Bad request - invalid input data"
// @Failure 401 {object} models.ErrorResponse "Unauthorized - invalid credentials"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid input data",
			Details: err.Error(),
		})
		return
	}

	loginResponse, err := h.userService.Login(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, loginResponse)
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get the current user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserResponse "User profile retrieved successfully"
// @Failure 401 {object} models.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	getError := c.GetString("error")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized",
			Details: getError,
		})
		return
	}
	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Invalid user ID type",
			Details: "user_id in context is not of type uint",
		})
		return
	}
	profile, err := h.userService.GetProfile(c.Request.Context(), userIDUint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Get profile failed",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}
