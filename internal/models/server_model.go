package models

import (
	"time"

	"gorm.io/gorm"
)

/** --------------------ENTITIES-------------------- */
// Server represents a server entity
type Server struct {
	gorm.Model
	Name    string `gorm:"not null;type:text" json:"name"`
	Avatar  string `gorm:"nullable;type:varchar(255)" json:"avatar"`
	OwnerId uint   `gorm:"type:uint" json:"ownerId"`

	Channels []Channel
	Members  []*User `gorm:"many2many:server_members"`
}

func (s *Server) BeforeCreate(tx *gorm.DB) error {
	// if s.ID == "" {
	// 	s.ID = uuid.New().String()
	// }
	return nil
}

// JoinServer represents server membership
type ServerMembers struct {
	ServerID   uint           `gorm:"not null" json:"serverId"`
	UserID     uint           `gorm:"not null" json:"userId"`
	JoinedDate time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"joinedDate"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

/** -------------------- DTOs -------------------- */
// Request
type CreateServerRequest struct {
	Name   string `json:"name" binding:"required"`
	Avatar string `json:"avatar"`
}

type UpdateServerRequest struct {
	Name   string `json:"name" binding:"required"`
	Avatar string `json:"avatar"`
}

type JoinServerRequest struct {
	ServerID uint `json:"serverId" binding:"required"`
}

// Response
type ServerResponse struct {
	ID        uint                 `json:"id"`
	Name      string               `json:"name"`
	OwnerId   uint                 `json:"owner"`
	Avatar    string               `json:"avatar"`
	CreatedAt time.Time            `json:"created"`
	Members   []JoinServerResponse `json:"members,omitempty"`
}

type JoinServerResponse struct {
	ServerID   uint      `json:"serverId"`
	UserID     uint      `json:"userId"`
	JoinedDate time.Time `json:"joinedDate"`
}
