package main

import (
	"chat-service/internal/config"
	"chat-service/internal/database"
	"chat-service/internal/models"
	"chat-service/internal/repositories/postgres"
	"chat-service/internal/services"
	"context"
	"log"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	slog.Info("Starting database seeding...")

	// Connect to database
	db, err := database.NewPostgresConnection(
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Connect to Redis
	redisClient, err := database.NewRedisConnection(&cfg.Redis)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisClient.Close()

	slog.Info("Database and Redis connections established")

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	channelRepo := postgres.NewChannelRepository(db)

	// Initialize services
	channelService := services.NewChannelService(channelRepo, userRepo)

	ctx := context.Background()

	// Seed initial users
	slog.Info("Creating initial users...")

	// Create admin user
	adminPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	adminUser := &models.User{
		Username: "admin",
		Email:    "admin@notify.com",
		Password: string(adminPassword),
	}

	if err := userRepo.Create(ctx, adminUser); err != nil {
		slog.Warn("Admin user might already exist", "error", err)
	} else {
		slog.Info("Created admin user", "id", adminUser.ID)
	}

	// Create test user
	testPassword, _ := bcrypt.GenerateFromPassword([]byte("test123"), bcrypt.DefaultCost)
	testUser := &models.User{
		Username: "testuser",
		Email:    "test@notify.com",
		Password: string(testPassword),
	}

	if err := userRepo.Create(ctx, testUser); err != nil {
		slog.Warn("Test user might already exist", "error", err)
	} else {
		slog.Info("Created test user", "id", testUser.ID)
	}

	// Seed initial channels
	slog.Info("Creating initial channels...")

	// Get admin user for channel creation
	admin, err := userRepo.FindByEmail(ctx, "admin@notify.com")
	if err != nil {
		slog.Warn("Could not find admin user for channel creation", "error", err)
	} else {
		// Create general channel
		generalChannel, err := channelService.CreateChannel("general", admin.ID)
		if err != nil {
			slog.Warn("General channel might already exist", "error", err)
		} else {
			slog.Info("Created general channel", "id", generalChannel.ID)
		}

		// Create random channel
		randomChannel, err := channelService.CreateChannel("random", admin.ID)
		if err != nil {
			slog.Warn("Random channel might already exist", "error", err)
		} else {
			slog.Info("Created random channel", "id", randomChannel.ID)
		}
	}

	slog.Info("Database seeding completed successfully!")
}
