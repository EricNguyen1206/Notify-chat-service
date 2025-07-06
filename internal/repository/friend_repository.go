package repository

import (
	"chat-service/internal/models"
	"errors"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// type FriendRepository interface {
// 	AddFriend(userID, friendID uint) error
// 	RemoveFriend(userID, friendID uint) error
// 	GetFriendsByUserID(userID uint) ([]models.FriendShip, error)
// 	GetFriendsByFriendID(userID uint) ([]models.FriendShip, error)
// 	IsFriend(userID, friendID uint) (bool, error)
// }

type FriendRepository struct {
	db          *gorm.DB
	redisClient *redis.Client
}

func NewFriendRepository(db *gorm.DB, redisClient *redis.Client) *FriendRepository {
	return &FriendRepository{db: db, redisClient: redisClient}
}

func (r *FriendRepository) AddFriend(userID, friendID uint) error {
	// Create 2 side relationship (user -> friend và friend -> user)
	friendship := models.FriendShip{
		UserID:   userID,
		FriendID: friendID,
		Status:   "accepted",
	}

	return r.db.Create(&friendship).Error
}

func (r *FriendRepository) RemoveFriend(userID, friendID uint) error {
	// Remove friendship in both directions
	// (user -> friend and friend -> user)
	if userID == friendID {
		return errors.New("cannot remove self as a friend")
	}
	// Delete the friendship record from the database
	err := r.db.
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userID, friendID, friendID, userID).
		Delete(&models.FriendShip{}).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("friendship not found")
	}
	return err
}

func (r *FriendRepository) CreateFriendship(userID, friendID uint, status string) error {
	return r.db.Create(&models.FriendShip{
		UserID:   userID,
		FriendID: friendID,
		Status:   status,
	}).Error
}

func (r *FriendRepository) GetFriendship(userID, friendID uint) (*models.FriendShip, error) {
	var f models.FriendShip
	err := r.db.Where("user_id = ? AND friend_id = ?", userID, friendID).First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FriendRepository) UpdateFriendshipStatus(userID, friendID uint, status string) error {
	return r.db.Model(&models.FriendShip{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Update("status", status).Error
}

func (r *FriendRepository) GetFriendsByUserID(userID uint) ([]models.FriendShip, error) {
	var friends []models.FriendShip
	err := r.db.
		Preload("Friend").
		Where("user_id = ? AND status = ?", userID, "accepted").
		Find(&friends).Error
	return friends, err
}

func (r *FriendRepository) GetFriendsByFriendID(friendId uint) ([]models.FriendShip, error) {
	var friends []models.FriendShip
	err := r.db.
		Preload("User").
		Where("friend_id = ? AND status = ?", friendId, "accepted").
		Find(&friends).Error
	return friends, err
}

func (r *FriendRepository) IsFriend(userID, friendID uint) (bool, error) {
	var count int64
	err := r.db.
		Model(&models.FriendShip{}).
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userID, friendID, friendID, userID).
		Count(&count).Error

	return count > 0, err
}
