package service

import (
	"chat-service/configs/utils/ws"
	"chat-service/internal/models"
	"chat-service/internal/repository"
	"context"
	"errors"
)

var (
	ErrChatNotFound    = errors.New("chat not found")
	ErrInvalidType     = errors.New("invalid chat type")
	ErrInvalidProvider = errors.New("invalid provider")
	ErrNotAuthorized   = errors.New("not authorized")
)

type ChatService interface {
	CreateChat(ctx context.Context, userID uint, req *models.ChatRequest) (*models.ChatResponse, error)
	GetChat(ctx context.Context, id uint) (*models.ChatResponse, error)
	GetUserChats(ctx context.Context, userID uint) ([]*models.ChatResponse, error)
	GetServerChats(ctx context.Context, serverID uint) ([]*models.ChatResponse, error)
	GetChannelChats(ctx context.Context, channelID uint) ([]*models.ChatResponse, error)
	GetFriendChats(ctx context.Context, friendID uint) ([]*models.ChatResponse, error)
	DeleteChat(ctx context.Context, id uint, userID uint) error
	BroadcastMessage(hub *ws.Hub, message *models.ChatResponse) error
}

type chatService struct {
	repo repository.ChatRepository
}

func NewChatService(repo repository.ChatRepository) ChatService {
	return &chatService{repo: repo}
}

func (s *chatService) CreateChat(ctx context.Context, userID uint, req *models.ChatRequest) (*models.ChatResponse, error) {
	// Validate chat type and provider
	if req.Type != string(models.ChatTypeChannel) && req.Type != string(models.ChatTypeDirect) {
		return nil, ErrInvalidType
	}

	// Validate required fields based on type and provider
	if req.Type == string(models.ChatTypeDirect) && req.ReceiverID == nil {
		return nil, errors.New("friendId is required for direct messages")
	}

	if req.Type == string(models.ChatTypeChannel) && (req.ServerID == nil || req.ChannelID == nil) {
		return nil, errors.New("serverId and channelId are required for server messages")
	}

	chat := &models.Chat{
		Type:       req.Type,
		SenderID:   userID,
		ReceiverID: req.ReceiverID,
		ServerID:   req.ServerID,
		ChannelID:  req.ChannelID,
		Text:       req.Text,
		URL:        req.URL,
		FileName:   req.FileName,
	}

	if err := s.repo.Create(ctx, chat); err != nil {
		return nil, err
	}

	return &models.ChatResponse{
		ID:         chat.ID,
		SenderID:   chat.SenderID,
		Type:       chat.Type,
		ReceiverID: chat.ReceiverID,
		ServerID:   chat.ServerID,
		ChannelID:  chat.ChannelID,
		Text:       chat.Text,
		URL:        chat.URL,
		FileName:   chat.FileName,
		CreatedAt:  chat.CreatedAt,
	}, nil
}

func (s *chatService) GetChat(ctx context.Context, id uint) (*models.ChatResponse, error) {
	chat, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrChatNotFound
	}

	return &models.ChatResponse{
		ID:         chat.ID,
		Type:       chat.Type,
		SenderID:   chat.SenderID,
		ReceiverID: chat.ReceiverID,
		ServerID:   chat.ServerID,
		ChannelID:  chat.ChannelID,
		Text:       chat.Text,
		URL:        chat.URL,
		FileName:   chat.FileName,
		CreatedAt:  chat.CreatedAt,
	}, nil
}

func (s *chatService) GetUserChats(ctx context.Context, userID uint) ([]*models.ChatResponse, error) {
	chats, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var responses []*models.ChatResponse
	for _, chat := range chats {
		responses = append(responses, &models.ChatResponse{
			ID:         chat.ID,
			Type:       chat.Type,
			SenderID:   chat.SenderID,
			ReceiverID: chat.ReceiverID,
			ServerID:   chat.ServerID,
			ChannelID:  chat.ChannelID,
			Text:       chat.Text,
			URL:        chat.URL,
			FileName:   chat.FileName,
			CreatedAt:  chat.CreatedAt,
		})
	}

	return responses, nil
}

func (s *chatService) GetServerChats(ctx context.Context, serverID uint) ([]*models.ChatResponse, error) {
	chats, err := s.repo.FindByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	var responses []*models.ChatResponse
	for _, chat := range chats {
		responses = append(responses, &models.ChatResponse{
			ID:         chat.ID,
			Type:       chat.Type,
			SenderID:   chat.SenderID,
			ReceiverID: chat.ReceiverID,
			ServerID:   chat.ServerID,
			ChannelID:  chat.ChannelID,
			Text:       chat.Text,
			URL:        chat.URL,
			FileName:   chat.FileName,
			CreatedAt:  chat.CreatedAt,
		})
	}

	return responses, nil
}

func (s *chatService) GetChannelChats(ctx context.Context, channelID uint) ([]*models.ChatResponse, error) {
	chats, err := s.repo.FindByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	var responses []*models.ChatResponse
	for _, chat := range chats {
		responses = append(responses, &models.ChatResponse{
			ID:         chat.ID,
			Type:       chat.Type,
			SenderID:   chat.SenderID,
			ReceiverID: chat.ReceiverID,
			ServerID:   chat.ServerID,
			ChannelID:  chat.ChannelID,
			Text:       chat.Text,
			URL:        chat.URL,
			FileName:   chat.FileName,
			CreatedAt:  chat.CreatedAt,
		})
	}

	return responses, nil
}

func (s *chatService) GetFriendChats(ctx context.Context, receiverId uint) ([]*models.ChatResponse, error) {
	chats, err := s.repo.FindByFriendID(ctx, receiverId)
	if err != nil {
		return nil, err
	}

	var responses []*models.ChatResponse
	for _, chat := range chats {
		responses = append(responses, &models.ChatResponse{
			ID:         chat.ID,
			Type:       chat.Type,
			SenderID:   chat.SenderID,
			ReceiverID: chat.ReceiverID,
			ServerID:   chat.ServerID,
			ChannelID:  chat.ChannelID,
			Text:       chat.Text,
			URL:        chat.URL,
			FileName:   chat.FileName,
			CreatedAt:  chat.CreatedAt,
		})
	}

	return responses, nil
}

func (s *chatService) DeleteChat(ctx context.Context, id uint, userID uint) error {
	chat, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return ErrChatNotFound
	}

	if chat.Sender.ID != userID {
		return ErrNotAuthorized
	}

	return s.repo.Delete(ctx, id)
}

func (s *chatService) BroadcastMessage(hub *ws.Hub, message *models.ChatResponse) error {
	// Convert message to JSON
	// data, err := json.Marshal(message)
	// if err != nil {
	// 	return err
	// }

	// Broadcast to all clients
	// hub.Broadcast <- data
	return nil
}
