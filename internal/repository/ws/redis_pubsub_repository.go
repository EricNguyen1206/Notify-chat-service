package ws

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisPubSubRepository handles Redis pub/sub operations for WebSocket messaging
type RedisPubSubRepository struct {
	redisClient *redis.Client
	ctx         context.Context
}

// NewRedisPubSubRepository creates a new Redis pub/sub repository
func NewRedisPubSubRepository(redisClient *redis.Client) *RedisPubSubRepository {
	return &RedisPubSubRepository{
		redisClient: redisClient,
		ctx:         context.Background(),
	}
}

// PublishMessage publishes a message to a Redis channel
func (r *RedisPubSubRepository) PublishMessage(channelID uint, message []byte) error {
	startTime := time.Now()

	channelKey := getChannelKey(channelID)
	err := r.redisClient.Publish(r.ctx, channelKey, message).Err()

	if err != nil {
		log.Printf("Error publishing to Redis channel %s: %v", channelKey, err)
		return err
	}

	log.Printf("Published message to Redis channel %s in %v", channelKey, time.Since(startTime))
	return nil
}

// SubscribeToChannel subscribes to a Redis channel and processes incoming messages
func (r *RedisPubSubRepository) SubscribeToChannel(channelID uint, messageHandler func([]byte)) error {
	channelKey := getChannelKey(channelID)

	pubsub := r.redisClient.Subscribe(r.ctx, channelKey)
	defer pubsub.Close()

	// Start a goroutine to process messages
	go func() {
		channel := pubsub.Channel()
		for msg := range channel {
			messageHandler([]byte(msg.Payload))
		}
	}()

	return nil
}

// PublishUserPresence publishes a user presence update to Redis
func (r *RedisPubSubRepository) PublishUserPresence(userID uint, channelID uint, action string) error {
	presenceData := map[string]interface{}{
		"userId":    userID,
		"channelId": channelID,
		"action":    action,
		"timestamp": time.Now().Unix(),
	}

	jsonData, err := json.Marshal(presenceData)
	if err != nil {
		return err
	}

	presenceChannel := "presence_updates"
	return r.redisClient.Publish(r.ctx, presenceChannel, jsonData).Err()
}

// Helper function to get Redis channel key from channel ID
func getChannelKey(channelID uint) string {
	return "channel:" + string(channelID)
}
