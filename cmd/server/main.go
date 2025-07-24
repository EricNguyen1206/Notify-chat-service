package main

import (
	"chat-service/internal/api/routes"
	"log"
)

func main() {
	application, err := routes.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
