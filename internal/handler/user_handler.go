package handler

import (
	"chat-service/configs/middleware"
	"chat-service/internal/models"
	"chat-service/internal/service"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type UserHandler struct {
	userService service.UserService
	redisClient *redis.Client
}

func NewUserHandler(userService service.UserService, redisClient *redis.Client) *UserHandler {
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
// @Failure 400 {object} map[string]interface{} "Bad request - invalid input data"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Handler: Registration validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data", "details": err.Error()})
		return
	}

	log.Printf("🔄 Handler: Processing registration request for email: %s", req.Email)

	user, err := h.userService.Register(c.Request.Context(), &req)
	if err != nil {
		log.Printf("❌ Handler: Registration failed for email %s: %v", req.Email, err)

		// Handle specific error types
		switch err.Error() {
		case "user already exists":
			c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		case "invalid request":
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		}
		return
	}

	log.Printf("✅ Handler: Registration successful for user ID: %d, Email: %s", user.ID, user.Email)
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
// @Failure 400 {object} map[string]interface{} "Bad request - invalid input data"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid credentials"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.userService.Login(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get the current user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserResponse "User profile retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	getError := c.GetString("error")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": getError})
		return
	}
	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "invalid user ID type",
			"details": "user_id in context is not of type uint",
		})
		return
	}
	profile, err := h.userService.GetProfile(c.Request.Context(), userIDUint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// Register routes
func (h *UserHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
	}
	user := r.Group("/users")
	{
		// Protected routes
		user.Use(middleware.Auth())
		user.GET("/profile", h.GetProfile)
	}
}
