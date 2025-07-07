package configs

import (
	"chat-service/configs/database"
	"chat-service/configs/utils/ws"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Config holds all configuration values
type Config struct {
	Port             string
	JWTSecret        string
	JWTExpire        time.Duration
	DB               *gorm.DB
	Redis            *redis.Client
	RedisURL         string
	PostgresUser     string
	PostgresPassword string
	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	WSUpgrader       websocket.Upgrader
	WSHub            *ws.Hub
}

var (
	ConfigInstance *Config
	once           sync.Once
)

// Load loads configuration from .env file
func Load() *Config {
	once.Do(func() {
		// Viper setup
		viper.SetDefault("NOTIFY_PORT", "8080")
		viper.SetDefault("NOTIFY_JWT_SECRET", "secret")
		viper.SetDefault("NOTIFY_JWT_EXPIRE", "24h")
		viper.SetDefault("REDIS_URL", "redis://:mypassword@127.0.0.1:6379/0")
		viper.SetDefault("POSTGRES_USER", "postgres")
		viper.SetDefault("POSTGRES_PASSWORD", "password")
		viper.SetDefault("POSTGRES_HOST", "localhost")
		viper.SetDefault("POSTGRES_PORT", "5432")
		viper.SetDefault("POSTGRES_DB", "postgres")
		viper.AutomaticEnv()

		// App
		appPort := viper.GetString("NOTIFY_PORT")
		appJWTSecret := viper.GetString("NOTIFY_JWT_SECRET")
		expire := viper.GetString("NOTIFY_JWT_EXPIRE")
		appJWTExpire, err := time.ParseDuration(expire)
		if err != nil {
			log.Fatal("Invalid JWT_EXPIRE format")
		}

		// Redis
		redisURL := viper.GetString("REDIS_URL")
		redisClient, _ := database.InitRedis(redisURL)

		// Postgres
		pgUser := viper.GetString("POSTGRES_USER")
		pgPassword := viper.GetString("POSTGRES_PASSWORD")
		pgHost := viper.GetString("POSTGRES_HOST")
		pgPort := viper.GetString("POSTGRES_PORT")
		pgDB := viper.GetString("POSTGRES_DB")
		postgresDB, _ := database.NewPostgresConnection(pgUser, pgPassword, pgHost, pgPort, pgDB)

		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Allow all origin connect to websocket
			// TODO: fix in production
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		wsHub := ws.WsNewHub(redisClient)

		ConfigInstance = &Config{
			Port:             appPort,
			JWTSecret:        appJWTSecret,
			JWTExpire:        appJWTExpire,
			DB:               postgresDB,
			Redis:            redisClient,
			RedisURL:         redisURL,
			PostgresUser:     pgUser,
			PostgresPassword: pgPassword,
			PostgresHost:     pgHost,
			PostgresPort:     pgPort,
			PostgresDB:       pgDB,
			WSUpgrader:       upgrader,
			WSHub:            wsHub,
		}
	})
	return ConfigInstance
}
