package service

import (
	"chat-service/internal/models"
	"chat-service/internal/repository"
	"errors"

	"gorm.io/gorm"
)

type ChannelService struct {
	repo     *repository.ChannelRepository
	userRepo *repository.UserRepository
}

func NewChannelService(repo *repository.ChannelRepository, userRepo *repository.UserRepository) *ChannelService {
	return &ChannelService{repo, userRepo}
}

func (s *ChannelService) GetAllChannel() ([]models.Channel, error) {
	return s.repo.GetAllChannels()
}

func (s *ChannelService) CreateChannel(name string, ownerID uint) (*models.Channel, error) {
	owner, err := s.userRepo.FindByID(ownerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("owner not found")
		}
		return nil, errors.New("failed to find owner: " + err.Error())
	}
	channel := &models.Channel{
		Name:    name,
		OwnerID: ownerID,
		Members: []*models.User{owner},
	}
	err = s.repo.Create(channel)
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

func (s *ChannelService) GetChannelByID(channelID uint) (*models.Channel, error) {
	return s.repo.GetByID(channelID)
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

func (s *ChannelService) GetChatMessagesByChannel(channelID uint) ([]models.Chat, error) {
	return s.repo.GetChatMessages(channelID)
}
