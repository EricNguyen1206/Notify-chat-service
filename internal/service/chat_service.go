package service

import "errors"

// import (
// 	"chat-service/configs/utils/ws"
// 	"chat-service/internal/models"
// 	"chat-service/internal/repository"
// 	"errors"
// )

var (
	ErrChatNotFound    = errors.New("chat not found")
	ErrInvalidType     = errors.New("invalid chat type")
	ErrInvalidProvider = errors.New("invalid provider")
	ErrNotAuthorized   = errors.New("not authorized")
)

// // type ChatService interface {
// // 	CreateChat(ctx context.Context, userID uint, req *models.ChatRequest) (*models.ChatResponse, error)
// // 	GetChat(ctx context.Context, id uint) (*models.ChatResponse, error)
// // 	GetUserChats(ctx context.Context, userID uint) ([]*models.ChatResponse, error)
// // 	GetChannelChats(ctx context.Context, channelID uint) ([]*models.ChatResponse, error)
// // 	GetFriendChats(ctx context.Context, friendID uint) ([]*models.ChatResponse, error)
// // 	DeleteChat(ctx context.Context, id uint, userID uint) error
// // 	BroadcastMessage(hub *ws.Hub, message *models.ChatResponse) error
// // }

// type ChatService struct {
// 	repo       repository.ChatRepository
// 	friendRepo repository.FriendRepository
// }

// func NewChatService(repo repository.ChatRepository, friendRepo repository.FriendRepository) *ChatService {
// 	return &ChatService{repo: repo, friendRepo: friendRepo}
// }

// func (s *ChatService) SendDirectMessage(chat *models.Chat) error {
// 	receiverID := chat.ReceiverID
// 	senderID := chat.SenderID
// 	text := chat.Text

// 	// Handle Friendship
// 	reverse, err := s.friendRepo.GetFriendship(*receiverID, senderID)
// 	if err == nil && reverse.Status == "pending" {
// 		_ = s.friendRepo.UpdateFriendshipStatus(*receiverID, senderID, "accepted")
// 		_ = s.friendRepo.CreateFriendship(senderID, *receiverID, "accepted")
// 	} else if _, err := s.friendRepo.GetFriendship(senderID, *receiverID); err != nil {
// 		_ = s.friendRepo.CreateFriendship(senderID, *receiverID, "pending")
// 	}

// 	// 🔥 Broadcast WebSocket
// 	ws.ChatHub.SendDirectMessage(ws.DirectMessage{
// 		FromUserID: senderID,
// 		ToUserID:   *receiverID,
// 		Content:    *text,
// 		Timestamp:  chat.CreatedAt,
// 	})

// 	return nil
// }

// func (s *ChatService) CreateChat(userID uint, req *models.ChatRequest) (*models.ChatResponse, error) {
// 	// Validate chat type and provider
// 	if req.Type != string(models.ChatTypeChannel) && req.Type != string(models.ChatTypeDirect) {
// 		return nil, ErrInvalidType
// 	}

// 	// Validate required fields based on type and provider
// 	if req.Type == string(models.ChatTypeDirect) && req.ReceiverID == nil {
// 		return nil, errors.New("friendId is required for direct messages")
// 	}

// 	if req.Type == string(models.ChatTypeChannel) && (req.ServerID == nil || req.ChannelID == nil) {
// 		return nil, errors.New("serverId and channelId are required for server messages")
// 	}

// 	chat := &models.Chat{
// 		Type:       req.Type,
// 		SenderID:   userID,
// 		ReceiverID: req.ReceiverID,
// 		ServerID:   req.ServerID,
// 		ChannelID:  req.ChannelID,
// 		Text:       req.Text,
// 		URL:        req.URL,
// 		FileName:   req.FileName,
// 	}

// 	if err := s.repo.Create(chat); err != nil {
// 		return nil, err
// 	}

// 	return &models.ChatResponse{
// 		ID:         chat.ID,
// 		SenderID:   chat.SenderID,
// 		Type:       chat.Type,
// 		ReceiverID: chat.ReceiverID,
// 		ServerID:   chat.ServerID,
// 		ChannelID:  chat.ChannelID,
// 		Text:       chat.Text,
// 		URL:        chat.URL,
// 		FileName:   chat.FileName,
// 		CreatedAt:  chat.CreatedAt,
// 	}, nil
// }

// func (s *ChatService) GetChat(id uint) (*models.ChatResponse, error) {
// 	chat, err := s.repo.FindByID(id)
// 	if err != nil {
// 		return nil, ErrChatNotFound
// 	}

// 	return &models.ChatResponse{
// 		ID:         chat.ID,
// 		Type:       chat.Type,
// 		SenderID:   chat.SenderID,
// 		ReceiverID: chat.ReceiverID,
// 		ServerID:   chat.ServerID,
// 		ChannelID:  chat.ChannelID,
// 		Text:       chat.Text,
// 		URL:        chat.URL,
// 		FileName:   chat.FileName,
// 		CreatedAt:  chat.CreatedAt,
// 	}, nil
// }

// func (s *ChatService) GetUserChats(userID uint) ([]*models.ChatResponse, error) {
// 	chats, err := s.repo.FindByUserID(userID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var responses []*models.ChatResponse
// 	for _, chat := range chats {
// 		responses = append(responses, &models.ChatResponse{
// 			ID:         chat.ID,
// 			Type:       chat.Type,
// 			SenderID:   chat.SenderID,
// 			ReceiverID: chat.ReceiverID,
// 			ServerID:   chat.ServerID,
// 			ChannelID:  chat.ChannelID,
// 			Text:       chat.Text,
// 			URL:        chat.URL,
// 			FileName:   chat.FileName,
// 			CreatedAt:  chat.CreatedAt,
// 		})
// 	}

// 	return responses, nil
// }

// func (s *ChatService) GetChannelChats(channelID uint) ([]*models.ChatResponse, error) {
// 	chats, err := s.repo.FindByChannelID(channelID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var responses []*models.ChatResponse
// 	for _, chat := range chats {
// 		responses = append(responses, &models.ChatResponse{
// 			ID:         chat.ID,
// 			Type:       chat.Type,
// 			SenderID:   chat.SenderID,
// 			ReceiverID: chat.ReceiverID,
// 			ServerID:   chat.ServerID,
// 			ChannelID:  chat.ChannelID,
// 			Text:       chat.Text,
// 			URL:        chat.URL,
// 			FileName:   chat.FileName,
// 			CreatedAt:  chat.CreatedAt,
// 		})
// 	}

// 	return responses, nil
// }

// func (s *ChatService) GetFriendChats(receiverId uint) ([]*models.ChatResponse, error) {
// 	chats, err := s.repo.FindByFriendID(receiverId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var responses []*models.ChatResponse
// 	for _, chat := range chats {
// 		responses = append(responses, &models.ChatResponse{
// 			ID:         chat.ID,
// 			Type:       chat.Type,
// 			SenderID:   chat.SenderID,
// 			ReceiverID: chat.ReceiverID,
// 			ServerID:   chat.ServerID,
// 			ChannelID:  chat.ChannelID,
// 			Text:       chat.Text,
// 			URL:        chat.URL,
// 			FileName:   chat.FileName,
// 			CreatedAt:  chat.CreatedAt,
// 		})
// 	}

// 	return responses, nil
// }

// func (s *ChatService) DeleteChat(id uint, userID uint) error {
// 	chat, err := s.repo.FindByID(id)
// 	if err != nil {
// 		return ErrChatNotFound
// 	}

// 	if chat.Sender.ID != userID {
// 		return ErrNotAuthorized
// 	}

// 	return s.repo.Delete(id)
// }

// func (s *ChatService) BroadcastMessage(hub *ws.Hub, message *models.ChatResponse) error {
// 	// Convert message to JSON
// 	// data, err := json.Marshal(message)
// 	// if err != nil {
// 	// 	return err
// 	// }

// 	// Broadcast to all clients
// 	// hub.Broadcast <- data
// 	return nil
// }
