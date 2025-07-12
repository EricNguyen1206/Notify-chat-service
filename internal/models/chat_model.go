package models

import (
	"time"

	"gorm.io/gorm"
)

// enum
type ChatType string

const (
	ChatTypeDirect  ChatType = "direct"
	ChatTypeChannel ChatType = "channel"
)

/** --------------------ENTITIES-------------------- */
// Chat represents a chat message
type Chat struct {
	gorm.Model

	SenderID   uint   `gorm:"not null" json:"senderId"`
	ReceiverID *uint  `gorm:"type:uint" json:"receiverId"`                                               // for direct messages
	Type       string `gorm:"not null;type:varchar(20);check:type IN ('direct', 'channel')" json:"type"` // Use consts

	ChannelID uint `gorm:"type:uint" json:"channelId"` // only if type == channel

	Text     *string `json:"text,omitempty"`     // optional
	URL      *string `json:"url,omitempty"`      // optional
	FileName *string `json:"fileName,omitempty"` // optional

	Sender   User     `gorm:"foreignKey:SenderID"`
	Receiver *User    `gorm:"foreignKey:ReceiverID"` // pointer to allow null
	Channel  *Channel `gorm:"foreignKey:ChannelID"`
}

/** -------------------- DTOs -------------------- */
// Request
type ChatRequest struct {
	Type       string  `json:"type" binding:"required,oneof=direct channel"`
	ReceiverID *uint   `json:"receiverId,omitempty"` // for direct
	ChannelID  *uint   `json:"channelId,omitempty"`  // for channel
	Text       *string `json:"text,omitempty"`
	URL        *string `json:"url,omitempty"`
	FileName   *string `json:"fileName,omitempty"`
}

// Response
type ChatResponse struct {
	ID         uint      `json:"id"`
	Type       string    `json:"type"` // "direct" | "channel"
	SenderID   uint      `json:"senderId"`
	SenderName string    `json:"senderName"`
	Text       *string   `json:"text,omitempty"`
	URL        *string   `json:"url,omitempty"`
	FileName   *string   `json:"fileName,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`

	// Relate to type message
	ReceiverID *uint `json:"receiverId,omitempty"` // direct
	ChannelID  *uint `json:"channelId,omitempty"`  // channel
}
