package repository

import (
	"chat-service/internal/models"
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type FriendRepository interface {
	AddFriend(userID, friendID uint) error
	RemoveFriend(userID, friendID uint) error
	GetFriendsByUserID(userID uint) ([]models.Friend, error)
	GetFriendsByFriendID(userID uint) ([]models.Friend, error)
	IsFriend(userID, friendID uint) (bool, error)
}

type friendRepository struct {
	db          *gorm.DB
	redisClient *redis.Client
}

func NewFriendRepository(db *gorm.DB, redisClient *redis.Client) FriendRepository {
	return &friendRepository{db: db, redisClient: redisClient}
}

func (r *friendRepository) AddFriend(userID, friendID uint) error {
	// Create 2 side relationship (user -> friend và friend -> user)
	friendship := models.Friend{
		UserID:   userID,
		FriendID: friendID,
		Status:   "accepted",
	}

	return r.db.Create(&friendship).Error
}

func (r *friendRepository) RemoveFriend(userID, friendID uint) error {
	// Remove friendship in both directions
	// (user -> friend and friend -> user)
	if userID == friendID {
		return errors.New("cannot remove self as a friend")
	}
	// Delete the friendship record from the database
	err := r.db.
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userID, friendID, friendID, userID).
		Delete(&models.Friend{}).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("friendship not found")
	}
	return err
}

func (r *friendRepository) CreateFriendship(userID, friendID uint, status string) error {
	return r.db.Create(&Friendship{
		UserID:   userID,
		FriendID: friendID,
		Status:   status,
	}).Error
}

func (r *friendRepository) GetFriendship(userID, friendID uint) (*Friendship, error) {
	var f Friendship
	err := r.db.Where("user_id = ? AND friend_id = ?", userID, friendID).First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *friendRepository) UpdateFriendshipStatus(userID, friendID uint, status string) error {
	return r.db.Model(&Friendship{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Update("status", status).Error
}

func (r *friendRepository) GetFriendsByUserID(userID uint) ([]models.Friend, error) {
	var friends []models.Friend
	err := r.db.
		Preload("Friend").
		Where("user_id = ? AND status = ?", userID, "accepted").
		Find(&friends).Error
	return friends, err
}

func (r *friendRepository) GetFriendsByFriendID(friendId uint) ([]models.Friend, error) {
	var friends []models.Friend
	err := r.db.
		Preload("User").
		Where("friend_id = ? AND status = ?", friendId, "accepted").
		Find(&friends).Error
	return friends, err
}

func (r *friendRepository) IsFriend(userID, friendID uint) (bool, error) {
	var count int64
	err := r.db.
		Model(&models.Friend{}).
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userID, friendID, friendID, userID).
		Count(&count).Error

	return count > 0, err
}
