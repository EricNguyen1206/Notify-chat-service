package router

import (
	"chat-service/configs"
	"chat-service/configs/middleware"
	"chat-service/internal/handler"
	"chat-service/internal/repository"
	"chat-service/internal/service"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type App struct {
	router     *gin.Engine
	postgresDB *gorm.DB
	// mongoDB      *database.MongoDB
	WSUpgrader websocket.Upgrader
}

func NewApp() (*App, error) {
	config := configs.Load()

	// Repository
	userRepo := repository.NewUserRepository(config.DB)
	friendRepo := repository.NewFriendRepository(config.DB, config.Redis)
	channelRepo := repository.NewChannelRepository(config.DB)

	// Service
	userService := service.NewUserService(userRepo, config.JWTSecret, config.Redis)
	friendService := service.NewFriendService(friendRepo)
	channelService := service.NewChannelService(channelRepo, userRepo)

	// Handler
	userHandler := handler.NewUserHandler(userService, config.Redis)
	friendHandler := handler.NewFriendHandler(friendService)
	channelHandler := handler.NewChannelHandler(channelService)

	wsHandler := handler.NewWSHandler(config.WSHub)

	// Setup router
	router := gin.Default()

	// Add middlewares
	router.Use(middleware.CORS())
	router.Use(middleware.LogApi())

	// Register API routes
	api := router.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "UP",
			})
		})

		// WebSocket routes
		wsHandler.RegisterRoutes(api)
		userHandler.RegisterRoutes(api)
		friendHandler.RegisterRoutes(api)
		channelHandler.RegisterRoutes(api)
	}

	return &App{
		router:     router,
		postgresDB: config.DB,
		// mongoDB:      mongoDB,
		WSUpgrader: config.WSUpgrader,
	}, nil
}

func (a *App) Run() error {
	log.Printf("Starting server on port 8080...")
	return a.router.Run(":8080")
}
