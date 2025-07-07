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
	// First, clear the many-to-many association to ensure cascade deletion
	err := r.db.Model(&models.Channel{Model: gorm.Model{ID: channelID}}).Association("Members").Clear()
	if err != nil {
		return err
	}

	// Then delete the channel
	return r.db.Delete(&models.Channel{}, channelID).Error
}

func (r *ChannelRepository) GetAllChannels() ([]models.Channel, error) {
	var c []models.Channel
	err := r.db.Preload("Members", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, email, created_at, updated_at, deleted_at")
	}).Find(&c).Error
	return c, err
}

func (r *ChannelRepository) GetAllUserChannels(userID uint) ([]models.Channel, error) {
	var c []models.Channel
	err := r.db.
		Preload("Members", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, username, email, created_at, updated_at, deleted_at")
		}).
		Joins("JOIN channel_members ON channels.id = channel_members.channel_id").
		Where("channel_members.user_id = ?", userID).
		Find(&c).Error
	return c, err
}

func (r *ChannelRepository) GetByID(channelID uint) (*models.Channel, error) {
	var c models.Channel
	err := r.db.Preload("Members", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, email, created_at, updated_at, deleted_at")
	}).First(&c, channelID).Error
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

// GetChatMessagesWithPagination returns chat messages for a channel with pagination and time-based infinite scroll
func (r *ChannelRepository) GetChatMessagesWithPagination(channelID uint, limit int, before *int64) ([]models.Chat, error) {
	var messages []models.Chat
	db := r.db.Where("channel_id = ?", channelID)
	if before != nil {
		db = db.Where("created_at < to_timestamp(?)", *before)
	}
	if limit <= 0 || limit > 100 {
		limit = 20 // default limit
	}
	db = db.Order("created_at DESC").Limit(limit)
	err := db.Find(&messages).Error
	return messages, err
}
