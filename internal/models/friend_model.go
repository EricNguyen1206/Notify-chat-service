package models

import (
	"gorm.io/gorm"
)

type Friendship struct {
	gorm.Model
	UserID   uint   `gorm:"not null" json:"userId"`
	FriendID uint   `gorm:"not null" json:"friendId"`
	Status    string    `gorm:"type:enum('pending','accepted','blocked');not null"`

	User   User `gorm:"foreignKey:UserID;references:ID" json:"user"`
	Friend User `gorm:"foreignKey:FriendID;references:ID" json:"friend"`
}

// FriendResponse represents the friend data returned to the client
type FriendResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}
