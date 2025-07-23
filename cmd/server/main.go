package main

import (
	"chat-service/internal/api/routes"
	"chat-service/internal/config"
	"log"
)

func main() {
	// Load configuration and start the WebSocket hub
	config := config.Load()

	// Start the WebSocket hub in a goroutine
	go config.WSHub.WsRun()
	log.Println("🚀 WebSocket hub started")

	application, err := routes.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
