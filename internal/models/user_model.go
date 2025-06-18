package models

import (
	"time"

	"gorm.io/gorm"
)

/** --------------------ENTITIES-------------------- */
// User represents the user entity
type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null" json:"username"`
	Email    string `gorm:"uniqueIndex;not null" json:"email"`
	Password string `json:"-"`

	Friends        []Friend `gorm:"foreignKey:UserID;references:ID"`
	FriendRequests []Friend `gorm:"foreignKey:FriendID;references:ID"`
	// Servers        []JoinServer    `gorm:"foreignKey:UserID;references:ID"`
	// Chats          []Chat          `gorm:"foreignKey:UserID;references:ID"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	// if u.Role == "" {
	// 	u.Role = "user" // Set default role is user
	// }
	return nil
}

/** -------------------- DTOs -------------------- */
// Request
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Response
type UserResponse struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// User Status
type UserStatus string

const (
	UserStatusOnline  UserStatus = "online"
	UserStatusOffline UserStatus = "offline"
)

type StatusUpdate struct {
	UserID uint       `json:"user_id"`
	Status UserStatus `json:"status"`
	Time   time.Time  `json:"time"`
}

type OnlineUser struct {
	UserID   uint      `json:"user_id"`
	LastSeen time.Time `json:"last_seen"`
}
