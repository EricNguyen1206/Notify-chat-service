package configs

import (
	"chat-service/configs/database"
	"chat-service/configs/utils/ws"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)
// maskPassword masks the password in a database URL for safe logging
func maskPassword(dsn string) string {
	if dsn == "" {
		return ""
	}
	// Simple masking - replace password with ***
	// This is a basic implementation, in production you might want more sophisticated masking
	if len(dsn) > 20 {
		return dsn[:20] + "***"
	}
	return "***"
}

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
		viper.SetConfigName(".env")
		viper.SetConfigType("env")
		viper.AddConfigPath(".")

		// Set defaults
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

		// Read the .env file
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Warning: Error reading .env file: %v", err)
			log.Printf("Using environment variables and defaults")
		}

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

		// Postgres - try DATABASE_URL first, then fallback to individual components
		databaseURL := viper.GetString("DATABASE_URL")
		var postgresDB *gorm.DB

		if databaseURL != "" {
			log.Printf("Using DATABASE_URL: %s", maskPassword(databaseURL))
			postgresDB, err = database.NewPostgresConnectionWithURL(databaseURL)
		} else {
			log.Printf("Using individual database configuration")
			pgUser := viper.GetString("POSTGRES_USER")
			pgPassword := viper.GetString("POSTGRES_PASSWORD")
			pgHost := viper.GetString("POSTGRES_HOST")
			pgPort := viper.GetString("POSTGRES_PORT")
			pgDB := viper.GetString("POSTGRES_DB")
			log.Printf("DB Config - Host: %s, User: %s, Port: %s, DB: %s", pgHost, pgUser, pgPort, pgDB)
			postgresDB, err = database.NewPostgresConnection(pgUser, pgPassword, pgHost, pgPort, pgDB)
		}
		if err != nil {
			log.Fatalf("❌ Failed to connect to database: %v", err)
		}
		log.Printf("✅ Database connection established successfully")

		// Use the centralized WebSocket upgrader with proper origin checking
		wsHub := ws.WsNewHub(redisClient)

		ConfigInstance = &Config{
			Port:       appPort,
			JWTSecret:  appJWTSecret,
			JWTExpire:  appJWTExpire,
			DB:         postgresDB,
			Redis:      redisClient,
			RedisURL:   redisURL,
			WSUpgrader: ws.Upgrader,
			WSHub:      wsHub,
		}
	})
	return ConfigInstance
}
