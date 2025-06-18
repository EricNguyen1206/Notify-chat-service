package repository

import (
	"chat-service/internal/models"
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type PresenceRepository interface {
	SetOnline(userID uint) error
	SetOffline(userID uint) error
	GetStatus(userID uint) (string, error)
	GetOnlineFriends(userIDs []uint) ([]uint, error)
	SubscribeToStatusUpdates(ctx context.Context) (<-chan *models.StatusUpdate, error)
	PublishStatusUpdate(ctx context.Context, update *models.StatusUpdate) error
	Close() error
}

type presenceRepository struct {
	client *redis.Client
	pubsub *redis.PubSub
}

func NewPresenceRepository(client *redis.Client) *presenceRepository {
	return &presenceRepository{client: client}
}

// SetOnline - Key: "presence:{userID} - online", TTL: 5 minutes
func (r *presenceRepository) SetOnline(userID uint) error {
	ctx := context.Background()
	return r.client.Set(ctx, "presence:"+strconv.Itoa(int(userID)), "online", 5*time.Minute).Err()
}

// SetOffline - Key: "presence:{userID} - offline", TTL: 1 minute (avoid flicker)
func (r *presenceRepository) SetOffline(userID uint) error {
	ctx := context.Background()
	return r.client.Set(ctx, "presence:"+strconv.Itoa(int(userID)), "offline", 1*time.Minute).Err()
}

// GetStatus - Check user status
func (r *presenceRepository) GetStatus(userID uint) (string, error) {
	ctx := context.Background()
	return r.client.Get(ctx, "presence:"+strconv.Itoa(int(userID))).Result()
}

// GetOnlineFriends - Get list of online friends
func (r *presenceRepository) GetOnlineFriends(userIDs []uint) ([]uint, error) {
	ctx := context.Background()
	keys := make([]string, len(userIDs))
	for i, id := range userIDs {
		keys[i] = "presence:" + strconv.Itoa(int(id))
	}

	// Pipeline to reduce roundtrip
	cmds, err := r.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, key := range keys {
			pipe.Get(ctx, key)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	onlineUsers := make([]uint, 0)
	for i, cmd := range cmds {
		if val, _ := cmd.(*redis.StringCmd).Result(); val == "online" {
			onlineUsers = append(onlineUsers, userIDs[i])
		}
	}
	return onlineUsers, nil
}

func (r *presenceRepository) SubscribeToStatusUpdates(ctx context.Context) (<-chan *models.StatusUpdate, error) {
	if r.pubsub == nil {
		r.pubsub = r.client.Subscribe(ctx, "user_status")
	}

	ch := make(chan *models.StatusUpdate)
	go func() {
		defer close(ch)
		redisCh := r.pubsub.Channel()

		for msg := range redisCh {
			var update models.StatusUpdate
			if err := json.Unmarshal([]byte(msg.Payload), &update); err != nil {
				log.Printf("Failed to unmarshal status update: %v", err)
				continue
			}
			ch <- &update
		}
	}()

	return ch, nil
}

func (r *presenceRepository) PublishStatusUpdate(ctx context.Context, update *models.StatusUpdate) error {
	updateJSON, err := json.Marshal(update)
	if err != nil {
		return err
	}
	return r.client.Publish(ctx, "user_status", updateJSON).Err()
}

func (r *presenceRepository) Close() error {
	if r.pubsub != nil {
		return r.pubsub.Close()
	}
	return nil
}
