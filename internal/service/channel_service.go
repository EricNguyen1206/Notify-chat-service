package service

type ChannelService struct {
	repo *ChannelRepository
}

func NewChannelService(repo *ChannelRepository) *ChannelService {
	return &ChannelService{repo}
}

func (s *ChannelService) CreateChannel(name string, ownerID, serverID uint) (*Channel, error) {
	channel := &Channel{
		Name:     name,
		OwnerID:  ownerID,
		ServerID: serverID,
		Members:  []*User{{Model: gorm.Model{ID: ownerID}}}, // Auto join
	}
	err := s.repo.Create(channel)
	return channel, err
}

func (s *ChannelService) UpdateChannel(channelID uint, name string) error {
	channel, err := s.repo.GetByID(channelID)
	if err != nil {
		return err
	}
	channel.Name = name
	return s.repo.Update(channel)
}

func (s *ChannelService) DeleteChannel(channelID uint) error {
	return s.repo.Delete(channelID)
}

func (s *ChannelService) GetChannelByID(channelID uint) (*Channel, error) {
	return s.repo.GetByID(channelID)
}

func (s *ChannelService) GetChannelsByUserAndServer(userID, serverID uint) ([]Channel, error) {
	return s.repo.GetListByUserAndServer(userID, serverID)
}

func (s *ChannelService) JoinChannel(channelID, userID uint) error {
	return s.repo.AddUser(channelID, userID)
}

func (s *ChannelService) LeaveChannel(channelID, userID uint) error {
	return s.repo.RemoveUser(channelID, userID)
}

func (s *ChannelService) RemoveUserFromChannel(channelID, targetUserID uint) error {
	return s.repo.RemoveUser(channelID, targetUserID)
}

func (s *ChannelService) GetChatMessagesByChannel(channelID uint) ([]Chat, error) {
	return s.repo.GetChatMessages(channelID)
}
