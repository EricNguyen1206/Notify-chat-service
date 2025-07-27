package main

import (
	"chat-service/internal/config"
	"chat-service/internal/database"
	"log"
	"log/slog"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	slog.Info("Starting database migration...")

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

	slog.Info("Database connection established")

	// Auto migrate the schema
	slog.Info("Running GORM auto-migration...")

	// Get the underlying *sql.DB for better control
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	slog.Info("Database migration completed successfully!")
}
