package models

import (
	"time"

	"gorm.io/gorm"
)

// Channel type constants
const (
	ChannelTypeDirect  = "direct"
	ChannelTypeChannel = "channel"
)

// Channel represents a channel within a category
type Channel struct {
	gorm.Model
	Name    string `gorm:"not null" json:"name"`
	OwnerID uint   `gorm:"not null;type:uint" json:"ownerId"`                                         // userid
	Type    string `gorm:"not null;type:varchar(20);check:type IN ('direct', 'channel')" json:"type"` // Use consts

	Members []*User `gorm:"many2many:channel_members" json:"members"`
}

type UpdateChannelRequest struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required,oneof=text voice"`
}

type ChannelResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	OwnerID   uint      `json:"ownerId"`
	Members   []User    `json:"members"` // List of members in the channel
}

type ChannelListResponse struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	OwnerID uint   `json:"ownerId"`
}
