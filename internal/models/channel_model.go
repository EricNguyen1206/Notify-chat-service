package models

import (
	"time"

	"gorm.io/gorm"
)

// Channel represents a channel within a category
type Channel struct {
	gorm.Model
	Name     string `gorm:"not null" json:"name"`
	OwnerID  uint   `gorm:"not null;type:uint" json:"ownerId"` // userid
	ServerID uint   `gorm:"not null;type:uint" json:"serverId"`

	Members []*User `gorm:"many2many:channel_members"`
	Server  *Server `gorm:"foreignKey:ServerID"`
}

type UpdateChannelRequest struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required,oneof=text voice"`
}

type ChannelResponse struct {
	ID        uint      `json:"id"`
	ServerID  uint      `json:"serverId"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
}
