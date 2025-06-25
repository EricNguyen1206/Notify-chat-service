package repository

import (
	"chat-service/internal/models"
	"context"

	"gorm.io/gorm"
)

type ChatRepository interface {
	Create(chat *models.Chat) error
	FindByID(id uint) (*models.Chat, error)
	FindByUserID(userID uint) ([]*models.Chat, error)
	FindByServerID(serverID uint) ([]*models.Chat, error)
	FindByChannelID(channelID uint) ([]*models.Chat, error)
	FindByFriendID(friendID uint) ([]*models.Chat, error)
	Delete(id uint) error
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) Create(chat *models.Chat) error {
	return r.db.Create(chat).Error
}

func (r *chatRepository) FindByID(id uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.First(&chat, "id = ?", id).Error
	return &chat, err
}

func (r *chatRepository) FindByUserID(userID uint) ([]*models.Chat, error) {
	var chats []*models.Chat
	err := r.db.Where("user_id = ?", userID).Find(&chats).Error
	return chats, err
}

func (r *chatRepository) FindByServerID(serverID uint) ([]*models.Chat, error) {
	var chats []*models.Chat
	err := r.db.Where("server_id = ?", serverID).Find(&chats).Error
	return chats, err
}

func (r *chatRepository) FindByChannelID(channelID uint) ([]*models.Chat, error) {
	var chats []*models.Chat
	err := r.db.Where("channel_id = ?", channelID).Find(&chats).Error
	return chats, err
}

func (r *chatRepository) FindByFriendID(friendID uint) ([]*models.Chat, error) {
	var chats []*models.Chat
	err := r.db.Where("friend_id = ?", friendID).Find(&chats).Error
	return chats, err
}

func (r *chatRepository) Delete(id uint) error {
	return r.db.Delete(&models.Chat{}, "id = ?", id).Error
}
