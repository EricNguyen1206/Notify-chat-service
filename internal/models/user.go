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

	Channels []*Channel `gorm:"many2many:channel_members" json:"channels"`
}

/** -------------------- DTOs -------------------- */
// Request
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
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

// LoginResponse represents the response for a successful login
// swagger:model
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
