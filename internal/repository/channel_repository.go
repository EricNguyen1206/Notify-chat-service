package repository

import (
	"chat-service/internal/models"

	"gorm.io/gorm"
)

type ChannelRepository struct {
	db *gorm.DB
}

func NewChannelRepository(db *gorm.DB) *ChannelRepository {
	return &ChannelRepository{db}
}

func (r *ChannelRepository) Create(channel *models.Channel) error {
	return r.db.Create(channel).Error
}

func (r *ChannelRepository) Update(channel *models.Channel) error {
	return r.db.Save(channel).Error
}

func (r *ChannelRepository) Delete(channelID uint) error {
	return r.db.Delete(&models.Channel{}, channelID).Error
}

func (r *ChannelRepository) GetAllChannels() ([]models.Channel, error) {
	var c []models.Channel
	err := r.db.Find(&c).Error
	return c, err
}

func (r *ChannelRepository) GetByID(channelID uint) (*models.Channel, error) {
	var c models.Channel
	err := r.db.Preload("Members").First(&c, channelID).Error
	return &c, err
}

func (r *ChannelRepository) AddUser(channelID uint, userID uint) error {
	return r.db.Model(&models.Channel{Model: gorm.Model{ID: channelID}}).Association("Members").Append(&models.User{Model: gorm.Model{ID: userID}})
}

func (r *ChannelRepository) RemoveUser(channelID uint, userID uint) error {
	return r.db.Model(&models.Channel{Model: gorm.Model{ID: channelID}}).Association("Members").Delete(&models.User{Model: gorm.Model{ID: userID}})
}

func (r *ChannelRepository) GetChatMessages(channelID uint) ([]models.Chat, error) {
	var messages []models.Chat
	err := r.db.
		Where("channel_id = ?", channelID).
		Order("created_at ASC").
		Find(&messages).Error
	return messages, err
}
