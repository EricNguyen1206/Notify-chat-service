package service

import (
	"chat-service/internal/models"
	"chat-service/internal/repository"
	"errors"
	"log"
)

type FriendService struct {
	friendRepo *repository.FriendRepository
}

func NewFriendService(friendRepo *repository.FriendRepository) *FriendService {
	return &FriendService{friendRepo}
}

func (s *FriendService) AddFriend(userID, friendID uint) error {
	if ok, _ := s.friendRepo.IsFriend(userID, friendID); ok {
		return errors.New("already friends")
	}
	return s.friendRepo.AddFriend(userID, friendID)
}

func (s *FriendService) GetFriends(userID uint) ([]models.FriendResponse, error) {
	friends, err := s.friendRepo.GetFriendsByUserID(userID)
	friends2, err2 := s.friendRepo.GetFriendsByFriendID(userID)
	if err != nil || err2 != nil {
		return nil, err
	}

	// Transform the data into FriendResponse objects
	responses := make([]models.FriendResponse, len(friends)+len(friends2))
	log.Println("friends", friends)
	log.Println("friends2", friends2)
	for i, friend := range friends {
		responses[i] = models.FriendResponse{
			ID:       friend.Friend.ID,
			Username: friend.Friend.Username,
			Email:    friend.Friend.Email,
			Status:   friend.Status,
		}
	}
	for i, friend := range friends2 {
		responses[i+len(friends)] = models.FriendResponse{
			ID:       friend.User.ID,
			Username: friend.User.Username,
			Email:    friend.User.Email,
			Status:   friend.Status,
		}
	}
	return responses, nil
}

func (s *FriendService) RemoveFriend(userID, friendID uint) error {
	return s.friendRepo.RemoveFriend(userID, friendID)
}
