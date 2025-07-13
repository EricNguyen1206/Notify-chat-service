package router

import (
	"chat-service/configs"
	"chat-service/configs/middleware"
	"chat-service/internal/handler"
	"chat-service/internal/repository"
	"chat-service/internal/service"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// @title Chat Service API
// @version 1.0
// @description A real-time chat service API with WebSocket support for instant messaging, user management, friend system, and channel management.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

type App struct {
	router     *gin.Engine
	postgresDB *gorm.DB
	WSUpgrader websocket.Upgrader
}

func NewApp() (*App, error) {
	config := configs.Load()

	// Repository
	userRepo := repository.NewUserRepository(config.DB)
	channelRepo := repository.NewChannelRepository(config.DB)
	chatRepo := repository.NewChatRepository(config.DB)

	// Service
	userService := service.NewUserService(userRepo, config.JWTSecret, config.Redis)
	channelService := service.NewChannelService(channelRepo, userRepo)

	// Handler
	userHandler := handler.NewUserHandler(userService, config.Redis)
	channelHandler := handler.NewChannelHandler(channelService)
	chatHandler := handler.NewChatHandler(channelService, chatRepo, config.WSHub)

	wsHandler := handler.NewWSHandler(config.WSHub)

	// Setup router
	router := gin.Default()

	// Add middlewares
	router.Use(middleware.CORS())
	router.Use(middleware.LogApi())

	// Register API routes
	api := router.Group("/api")
	{
		// Health check endpoint
		api.GET("/health", healthCheck)
		// WebSocket routes
		wsHandler.RegisterRoutes(api)
		userHandler.RegisterRoutes(api)
		channelHandler.RegisterRoutes(api)
		chatHandler.RegisterRoutes(api)
	}

	// Swagger documentation (only in development)
	if os.Getenv("GIN_MODE") != "release" {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		log.Println("📚 Swagger UI available at: http://localhost:8080/swagger/index.html")
	}

	return &App{
		router:     router,
		postgresDB: config.DB,
		WSUpgrader: config.WSUpgrader,
	}, nil
}

// healthCheck godoc
// @Summary Health check
// @Description Check if the API is running
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "API is healthy"
// @Router /health [get]
func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "UP",
	})
}

func (a *App) Run() error {
	log.Printf("Starting server on port 8080...")
	if os.Getenv("GIN_MODE") != "release" {
		log.Printf("Swagger UI available at: http://localhost:8080/swagger/index.html")
	}
	return a.router.Run(":8080")
}
