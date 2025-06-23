package repository

type ChannelRepository struct {
	db *gorm.DB
}

func NewChannelRepository(db *gorm.DB) *ChannelRepository {
	return &ChannelRepository{db}
}

func (r *ChannelRepository) Create(channel *Channel) error {
	return r.db.Create(channel).Error
}

func (r *ChannelRepository) Update(channel *Channel) error {
	return r.db.Save(channel).Error
}

func (r *ChannelRepository) Delete(channelID uint) error {
	return r.db.Delete(&Channel{}, channelID).Error
}

func (r *ChannelRepository) GetByID(channelID uint) (*Channel, error) {
	var c Channel
	err := r.db.Preload("Members").Preload("Server").First(&c, channelID).Error
	return &c, err
}

func (r *ChannelRepository) GetListByUserAndServer(userID uint, serverID uint) ([]Channel, error) {
	var channels []Channel
	err := r.db.
		Joins("JOIN channel_members cm ON cm.channel_id = channels.id").
		Where("cm.user_id = ? AND channels.server_id = ?", userID, serverID).
		Find(&channels).Error
	return channels, err
}

func (r *ChannelRepository) AddUser(channelID uint, userID uint) error {
	return r.db.Model(&Channel{Model: gorm.Model{ID: channelID}}).Association("Members").Append(&User{Model: gorm.Model{ID: userID}})
}

func (r *ChannelRepository) RemoveUser(channelID uint, userID uint) error {
	return r.db.Model(&Channel{Model: gorm.Model{ID: channelID}}).Association("Members").Delete(&User{Model: gorm.Model{ID: userID}})
}

func (r *ChannelRepository) GetChatMessages(channelID uint) ([]Chat, error) {
	var messages []Chat
	err := r.db.
		Where("channel_id = ?", channelID).
		Order("created_at ASC").
		Find(&messages).Error
	return messages, err
}
