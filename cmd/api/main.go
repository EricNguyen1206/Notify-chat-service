package main

import (
	"chat-service/configs"
	"chat-service/internal/router"

	"log"
)

func main() {
	// Load configuration and start the WebSocket hub
	config := configs.Load()

	// Start the WebSocket hub in a goroutine
	go config.WSHub.WsRun()
	log.Println("🚀 WebSocket hub started")

	application, err := router.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
