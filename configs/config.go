package configs

import (
	"chat-service/configs/database"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Config holds all configuration values
type Config struct {
	Port       string
	JWTSecret  string
	JWTExpire  time.Duration
	DB         *gorm.DB
	Redis      *redis.Client
	WSUpgrader websocket.Upgrader

	// // Redis
	// Redis struct {
	// 	Host     string
	// 	Port     string
	// 	Password string
	// 	Addr     string
	// }

	// // Postgrest
	// Postgres struct {
	// 	User     string
	// 	Password string
	// 	Host     string
	// 	Port     string
	// 	DbName   string
	// }

	// MinIO
	// MinIO struct {
	// 	Endpoint string
	// 	User     string
	// 	Password string
	// 	UseSSL   bool
	// 	Bucket   string
	// }

	// Kafka
	// Kafka struct {
	// 	Brokers []string
	// 	Topic   string
	// }
}

var (
	ConfigInstance *Config
	once           sync.Once
)

// Load loads configuration from .env file
func Load() *Config {
	once.Do(func() {

		var expire = getEnv("NOTIFY_JWT_EXPIRE", "24h")

		// App
		appPort := getEnv("NOTIFY_PORT", "8080")
		appJWTSecret := getEnv("NOTIFY_JWT_SECRET", "secret")
		appJWTExpire, err := time.ParseDuration(expire)
		if err != nil {
			log.Fatal("Invalid JWT_EXPIRE format")
		}

		// Redis
		redisClient, _ := database.InitRedis()

		// Postgres
		postgresDB, _ := database.NewPostgresConnection()

		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Allow all origin connect to websocket
			// TODO: fix in production
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		ConfigInstance = &Config{
			Port:       appPort,
			JWTSecret:  appJWTSecret,
			JWTExpire:  appJWTExpire,
			DB:         postgresDB,
			Redis:      redisClient,
			WSUpgrader: upgrader,
		}
	})
	return ConfigInstance
}

// Helper function to get environment variable with default value
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
