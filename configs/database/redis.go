package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis(redisURL string) (*redis.Client, error) {
	if redisURL == "" {
		redisURL = "redis://:mypassword@127.0.0.1:6379/0"
	}
	opt, _ := redis.ParseURL(redisURL)
	RedisClient := redis.NewClient(opt)

	// Health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := RedisClient.Ping(ctx).Result()
	return RedisClient, err
}
