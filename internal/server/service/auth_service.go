package service

import (
	"context"
	"errors"
	"time"

	"chat-service/internal/ports/models"
	"chat-service/internal/server/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo      repository.AuthRepository
	jwtSecret string
	jwtExpire time.Duration
}

func NewAuthService(repo repository.AuthRepository, secret string, expire time.Duration) *AuthService {
	return &AuthService{
		repo:      repo,
		jwtSecret: secret,
		jwtExpire: expire,
	}
}

// Register handles user registration
func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		IsActive:  true,
		LastLogin: time.Now().UTC(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login handles user authentication
func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Update last login
	user.LastLogin = time.Now().UTC()
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"exp":   time.Now().Add(s.jwtExpire).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		Token: tokenString,
	}, nil
}
