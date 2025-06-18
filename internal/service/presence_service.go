package service

import (
	"chat-service/configs/utils/ws"
	"chat-service/internal/models"
	"chat-service/internal/repository"
	"context"
)

type PresenceService struct {
	presenceRepo repository.PresenceRepository
	friendRepo   repository.FriendRepository
	hub          *ws.Hub
}

func NewPresenceService(
	presenceRepo repository.PresenceRepository,
	friendRepo repository.FriendRepository,
	hub *ws.Hub,
) *PresenceService {
	return &PresenceService{
		presenceRepo: presenceRepo,
		friendRepo:   friendRepo,
		hub:          hub,
	}
}

func (s *PresenceService) SetOnline(userID uint) error {
	if err := s.presenceRepo.SetOnline(userID); err != nil {
		return err
	}
	// Broadcast cho bạn bè biết user này online
	s.hub.BroadcastFriendStatus(userID, "online")
	return nil
}

func (s *PresenceService) SetOffline(userID uint) error {
	if err := s.presenceRepo.SetOffline(userID); err != nil {
		return err
	}
	// Broadcast cho bạn bè biết user này offline
	s.hub.BroadcastFriendStatus(userID, "offline")
	return nil
}

func (s *PresenceService) GetOnlineFriends(userIDs []uint) ([]uint, error) {
	return s.presenceRepo.GetOnlineFriends(userIDs)
}

func (s *PresenceService) SubscribeToStatusUpdates(ctx context.Context) (<-chan *models.StatusUpdate, error) {
	return s.presenceRepo.SubscribeToStatusUpdates(ctx)
}
