package service

import (
	"chat-service/internal/models"
	"chat-service/internal/repository"
	"context"
	"errors"
	"time"
)

var (
	ErrServerNotFound = errors.New("server not found")
	ErrAlreadyMember  = errors.New("user is already a member")
	ErrNotMember      = errors.New("user is not a member")
)

type ServerService interface {
	CreateServer(ctx context.Context, ownerID uint, req *models.CreateServerRequest) (*models.ServerResponse, error)
	GetServer(ctx context.Context, id uint) (*models.ServerResponse, error)
	GetUserServers(userID uint) ([]*models.ServerResponse, error)
	UpdateServer(ctx context.Context, id uint, ownerID uint, req *models.UpdateServerRequest) (*models.ServerResponse, error)
	DeleteServer(ctx context.Context, id uint, ownerID uint) error
	JoinServer(ctx context.Context, userID uint, req *models.JoinServerRequest) error
	LeaveServer(ctx context.Context, serverID uint, userID uint) error
	GetServerMembers(ctx context.Context, serverID uint) ([]*models.JoinServerResponse, error)
}

type serverService struct {
	repo repository.ServerRepository
}

func NewServerService(repo repository.ServerRepository) ServerService {
	return &serverService{repo: repo}
}

func (s *serverService) CreateServer(ctx context.Context, ownerID uint, req *models.CreateServerRequest) (*models.ServerResponse, error) {
	server := &models.Server{
		Name:    req.Name,
		Avatar:  req.Avatar,
		OwnerId: ownerID,
	}

	if err := s.repo.Create(ctx, server); err != nil {
		return nil, err
	}

	// Auto-join the owner to the server
	join := &models.ServerMembers{
		ServerID:   server.ID,
		UserID:     ownerID,
		JoinedDate: time.Now(),
	}

	if err := s.repo.JoinServer(ctx, join); err != nil {
		return nil, err
	}

	return &models.ServerResponse{
		ID:        server.ID,
		Name:      server.Name,
		Avatar:    server.Avatar,
		CreatedAt: time.Now(),
	}, nil
}

func (s *serverService) GetServer(ctx context.Context, id uint) (*models.ServerResponse, error) {
	server, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrServerNotFound
	}

	// members, err := s.repo.GetServerMembers(ctx, id)
	// if err != nil {
	// 	return nil, err
	// }

	// memberResponses := make([]models.JoinServerResponse, len(members))
	// for i, member := range members {
	// 	memberResponses[i] = models.JoinServerResponse{
	// 		ID:         member.ID,
	// 		ServerID:   member.ServerID,
	// 		UserID:     member.UserID,
	// 		JoinedDate: member.JoinedDate,
	// 	}
	// }

	return &models.ServerResponse{
		ID:        server.ID,
		Name:      server.Name,
		OwnerId:   server.OwnerId,
		Avatar:    server.Avatar,
		CreatedAt: server.CreatedAt,
	}, nil
}

func (s *serverService) GetUserServers(userID uint) ([]*models.ServerResponse, error) {
	joins, err := s.repo.GetUserServers(userID)
	if err != nil {
		return nil, err
	}

	var servers []*models.ServerResponse
	for _, join := range joins {
		server, err := s.repo.FindByID(join.ServerID)
		if err != nil {
			continue
		}

		servers = append(servers, &models.ServerResponse{
			ID:        server.ID,
			Name:      server.Name,
			OwnerId:   server.OwnerId,
			Avatar:    server.Avatar,
			CreatedAt: server.CreatedAt,
		})
	}

	return servers, nil
}

func (s *serverService) UpdateServer(ctx context.Context, id uint, ownerID uint, req *models.UpdateServerRequest) (*models.ServerResponse, error) {
	server, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrServerNotFound
	}

	if server.OwnerId != ownerID {
		return nil, ErrNotAuthorized
	}

	server.Name = req.Name
	server.Avatar = req.Avatar
	server.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, server); err != nil {
		return nil, err
	}

	return &models.ServerResponse{
		ID:        server.ID,
		Name:      server.Name,
		OwnerId:   server.OwnerId,
		Avatar:    server.Avatar,
		CreatedAt: server.CreatedAt,
	}, nil
}

func (s *serverService) DeleteServer(ctx context.Context, id uint, ownerID uint) error {
	server, err := s.repo.FindByID(id)
	if err != nil {
		return ErrServerNotFound
	}

	if server.OwnerId != ownerID {
		return ErrNotAuthorized
	}

	return s.repo.Delete(ctx, id)
}

func (s *serverService) JoinServer(ctx context.Context, userID uint, req *models.JoinServerRequest) error {
	// Check if server exists
	_, err := s.repo.FindByID(req.ServerID)
	if err != nil {
		return ErrServerNotFound
	}

	// Check if already a member
	isMember, err := s.repo.IsMember(ctx, req.ServerID, userID)
	if err != nil {
		return err
	}
	if isMember {
		return ErrAlreadyMember
	}

	join := &models.ServerMembers{
		ServerID:   req.ServerID,
		UserID:     userID,
		JoinedDate: time.Now(),
	}

	return s.repo.JoinServer(ctx, join)
}

func (s *serverService) LeaveServer(ctx context.Context, serverID uint, userID uint) error {
	// Check if server exists
	server, err := s.repo.FindByID(serverID)
	if err != nil {
		return ErrServerNotFound
	}

	// Owner cannot leave the server
	if server.OwnerId == userID {
		return ErrNotAuthorized
	}

	// Check if member
	isMember, err := s.repo.IsMember(ctx, serverID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotMember
	}

	return s.repo.LeaveServer(ctx, serverID, userID)
}

func (s *serverService) GetServerMembers(ctx context.Context, serverID uint) ([]*models.JoinServerResponse, error) {
	members, err := s.repo.GetServerMembers(ctx, serverID)
	if err != nil {
		return nil, err
	}

	var responses []*models.JoinServerResponse
	for _, member := range members {
		responses = append(responses, &models.JoinServerResponse{
			ServerID:   member.ServerID,
			UserID:     member.UserID,
			JoinedDate: member.JoinedDate,
		})
	}

	return responses, nil
}
