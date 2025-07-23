package config

import (
	"sync"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
}

var (
	ConfigInstance *Config
	once           sync.Once
)

type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	URI          string
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
}

type JWTConfig struct {
	Secret         string
	ExpirationTime time.Duration
}

func LoadConfig() (*Config, error) {
	// Viper setup
	once.Do(func() {
		viper.SetDefault("NOTIFY_PORT", "8080")
		viper.SetDefault("NOTIFY_READ_TIMEOUT", 30*time.Second)
		viper.SetDefault("NOTIFY_WRITE_TIMEOUT", 30*time.Second)
		viper.SetDefault("NOTIFY_JWT_SECRET", "secret")
		viper.SetDefault("NOTIFY_JWT_EXPIRE", "24h")
		viper.SetDefault("REDIS_URL", "redis://:mypassword@127.0.0.1:6379/0")
		viper.SetDefault("REDIS_MAX_RETRIES", 3)
		viper.SetDefault("REDIS_POOL_SIZE", 100)
		viper.SetDefault("REDIS_MIN_IDLE_CONNS", 10)
		viper.SetDefault("REDIS_DIAL_TIMEOUT", 5*time.Second)
		viper.SetDefault("REDIS_READ_TIMEOUT", 3*time.Second)
		viper.SetDefault("REDIS_WRITE_TIMEOUT", 3*time.Second)
		viper.SetDefault("POSTGRES_USER", "postgres")
		viper.SetDefault("POSTGRES_PASSWORD", "password")
		viper.SetDefault("POSTGRES_HOST", "localhost")
		viper.SetDefault("POSTGRES_PORT", "5432")
		viper.SetDefault("POSTGRES_DB", "postgres")
		viper.AutomaticEnv()
		// Move ConfigInstance to package level to avoid "declared and not used" error
		ConfigInstance = &Config{
			Server: ServerConfig{
				Host:         viper.GetString("NOTIFY_HOST"),
				Port:         viper.GetString("NOTIFY_PORT"),
				ReadTimeout:  viper.GetDuration("NOTIFY_READ_TIMEOUT"),
				WriteTimeout: viper.GetDuration("NOTIFY_WRITE_TIMEOUT"),
				IdleTimeout:  viper.GetDuration("NOTIFY_IDLE_TIMEOUT"),
			},
			Database: DatabaseConfig{
				Host:     viper.GetString("POSTGRES_HOST"),
				Port:     viper.GetString("POSTGRES_PORT"),
				User:     viper.GetString("POSTGRES_USER"),
				Password: viper.GetString("POSTGRES_PASSWORD"),
				DBName:   viper.GetString("POSTGRES_DB"),
				// SSLMode:  viper.GetString("DB_SSLMODE", "disable"),
			},
			Redis: RedisConfig{
				URI:          viper.GetString("REDIS_HOST"),
				MaxRetries:   viper.GetInt("REDIS_MAX_RETRIES"),
				DialTimeout:  viper.GetDuration("REDIS_DIAL_TIMEOUT"),
				ReadTimeout:  viper.GetDuration("REDIS_READ_TIMEOUT"),
				WriteTimeout: viper.GetDuration("REDIS_WRITE_TIMEOUT"),
				PoolSize:     viper.GetInt("REDIS_POOL_SIZE"),
				MinIdleConns: viper.GetInt("REDIS_MIN_IDLE_CONNS"),
			},
			JWT: JWTConfig{
				Secret:         viper.GetString("NOTIFY_JWT_SECRET"),
				ExpirationTime: viper.GetDuration("NOTIFY_JWT_EXPIRE"),
			},
		}
	})

	return ConfigInstance, nil
}
