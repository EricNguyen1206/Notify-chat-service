package repository

import (
	"chat-service/internal/models"
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type FriendRepository interface {
	AddFriend(ctx context.Context, userID, friendID uint) error
	RemoveFriend(ctx context.Context, userID, friendID uint) error
	GetFriendsByUserID(ctx context.Context, userID uint) ([]models.Friendship, error)
	GetFriendsByFriendID(ctx context.Context, userID uint) ([]models.Friendship, error)
	IsFriend(ctx context.Context, userID, friendID uint) (bool, error)
}

type friendRepository struct {
	db          *gorm.DB
	redisClient *redis.Client
}

func NewFriendRepository(db *gorm.DB, redisClient *redis.Client) FriendRepository {
	return &friendRepository{db: db, redisClient: redisClient}
}

func (r *friendRepository) AddFriend(ctx context.Context, userID, friendID uint) error {
	// Tạo quan hệ hai chiều (user -> friend và friend -> user)
	friendship := models.Friendship{
		UserID:   userID,
		FriendID: friendID,
		Status:   "accepted", // Hoặc "accepted" tùy logic
	}

	return r.db.WithContext(ctx).Create(&friendship).Error
}

func (r *friendRepository) RemoveFriend(ctx context.Context, userID, friendID uint) error {
	// Xóa quan hệ hai chiều
	err := r.db.WithContext(ctx).
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userID, friendID, friendID, userID).
		Delete(&models.Friendship{}).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("friendship not found")
	}
	return err
}

func (r *friendRepository) GetFriendsByUserID(ctx context.Context, userID uint) ([]models.Friendship, error) {
	var friends []models.Friendship
	err := r.db.WithContext(ctx).
		Preload("Friend").
		Where("user_id = ? AND status = ?", userID, "accepted").
		Find(&friends).Error
	return friends, err
}

func (r *friendRepository) GetFriendsByFriendID(ctx context.Context, friendId uint) ([]models.Friendship, error) {
	var friends []models.Friendship
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("friend_id = ? AND status = ?", friendId, "accepted").
		Find(&friends).Error
	return friends, err
}

func (r *friendRepository) IsFriend(ctx context.Context, userID, friendID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Friendship{}).
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userID, friendID, friendID, userID).
		Count(&count).Error

	return count > 0, err
}
