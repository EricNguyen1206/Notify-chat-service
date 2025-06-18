package router

import (
	"chat-service/configs"
	"chat-service/configs/database"
	"chat-service/configs/middleware"
	"chat-service/configs/utils/ws"
	"chat-service/internal/handler"
	"chat-service/internal/repository"
	"chat-service/internal/service"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type App struct {
	router     *gin.Engine
	postgresDB *gorm.DB
	// mongoDB      *database.MongoDB
	websocketHub *ws.Hub
}

func NewApp() (*App, error) {
	config := configs.Load()
	// Initialize databases
	postgresDB, err := database.NewPostgresConnection()
	if err != nil {
		return nil, err
	}
	redisClient, _ := database.InitRedis()

	// mongoDB, err := database.NewMongoConnection()
	// if err != nil {
	// 	return nil, err
	// }

	// Initialize WebSocket hub
	hub := ws.NewHub()

	// Repository
	userRepo := repository.NewUserRepository(postgresDB)
	friendRepo := repository.NewFriendRepository(postgresDB, redisClient)
	presenceRepo := repository.NewPresenceRepository(redisClient)

	// Service
	userService := service.NewUserService(userRepo, config.App.JWTSecret, redisClient)
	presenceService := service.NewPresenceService(presenceRepo, friendRepo, hub)
	friendService := service.NewFriendService(friendRepo)

	// Handler
	userHandler := handler.NewUserHandler(userService, redisClient)
	presenceHandler := handler.NewPresenceHandler(presenceService, friendService, hub)
	friendHandler := handler.NewFriendHandler(friendService)

	// Setup router
	router := gin.Default()

	// Add CORS middleware
	router.Use(middleware.CORS())

	// Add logging middleware
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] | %s | %d | %s | %s | %s | %s | %s | %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.ClientIP,
			param.StatusCode,
			param.Method,
			param.Path,
			param.Request.UserAgent(),
			param.ErrorMessage,
			param.Latency,
			param.Request.Proto,
		)
	}))

	// Register API routes
	api := router.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "UP",
			})
		})

		// WebSocket routes
		wsGroup := api.Group("/ws")
		{
			presenceHandler.RegisterRoutes(wsGroup)
		}

		userHandler.RegisterRoutes(api)
		friendHandler.RegisterRoutes(api)
		// categoryHandler.RegisterRoutes(api)
		// serverHandler.RegisterRoutes(api)
	}

	return &App{
		router:     router,
		postgresDB: postgresDB,
		// mongoDB:      mongoDB,
		websocketHub: hub,
	}, nil
}

func (a *App) Run() error {
	log.Printf("Starting server on port 8080...")
	return a.router.Run(":8080")
}
