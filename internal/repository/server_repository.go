package repository

import (
	"chat-service/internal/models"
	"context"

	"gorm.io/gorm"
)

type ServerRepository interface {
	Create(ctx context.Context, server *models.Server) error
	FindByID(ctx context.Context, id uint) (*models.Server, error)
	FindByOwner(ctx context.Context, ownerID uint) ([]*models.Server, error)
	Update(ctx context.Context, server *models.Server) error
	Delete(ctx context.Context, id uint) error
	JoinServer(ctx context.Context, join *models.ServerMembers) error
	LeaveServer(ctx context.Context, serverID, userID uint) error
	GetServerMembers(ctx context.Context, serverID uint) ([]*models.ServerMembers, error)
	GetUserServers(ctx context.Context, userID uint) ([]*models.ServerMembers, error)
	IsMember(ctx context.Context, serverID, userID uint) (bool, error)
}

type serverRepository struct {
	db *gorm.DB
}

func NewServerRepository(db *gorm.DB) ServerRepository {
	return &serverRepository{db: db}
}

func (r *serverRepository) Create(ctx context.Context, server *models.Server) error {
	return r.db.WithContext(ctx).Create(server).Error
}

func (r *serverRepository) FindByID(ctx context.Context, id uint) (*models.Server, error) {
	var server models.Server
	err := r.db.WithContext(ctx).First(&server, "id = ?", id).Error
	return &server, err
}

func (r *serverRepository) FindByOwner(ctx context.Context, ownerID uint) ([]*models.Server, error) {
	var servers []*models.Server
	err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Find(&servers).Error
	return servers, err
}

func (r *serverRepository) Update(ctx context.Context, server *models.Server) error {
	return r.db.WithContext(ctx).Save(server).Error
}

func (r *serverRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Server{}, "id = ?", id).Error
}

func (r *serverRepository) JoinServer(ctx context.Context, join *models.ServerMembers) error {
	return r.db.WithContext(ctx).Create(join).Error
}

func (r *serverRepository) LeaveServer(ctx context.Context, serverID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("server_id = ? AND user_id = ?", serverID, userID).
		Delete(&models.ServerMembers{}).Error
}

func (r *serverRepository) GetServerMembers(ctx context.Context, serverID uint) ([]*models.ServerMembers, error) {
	var members []*models.ServerMembers
	err := r.db.WithContext(ctx).
		Where("server_id = ?", serverID).
		Find(&members).Error
	return members, err
}

func (r *serverRepository) GetUserServers(ctx context.Context, userID uint) ([]*models.ServerMembers, error) {
	var servers []*models.ServerMembers
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&servers).Error
	return servers, err
}

func (r *serverRepository) IsMember(ctx context.Context, serverID, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ServerMembers{}).
		Where("server_id = ? AND user_id = ?", serverID, userID).
		Count(&count).Error
	return count > 0, err
}
