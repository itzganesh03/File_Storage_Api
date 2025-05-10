package main

import (
	"log"

	"github.com/yourusername/file-storage-api/src/config"
	"github.com/yourusername/file-storage-api/src/models"
	"github.com/yourusername/file-storage-api/src/router"
	"github.com/yourusername/file-storage-api/src/storage"
)

func main() {
	// Load YAML configuration
	if err := config.LoadConfig("config.yml"); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize MongoDB connection
	if err := models.InitMongoDB(); err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer models.CloseMongoDB()

	// Initialize file collection
	models.InitFileCollection()
	// Initialize file collection
	models.InitFileCollection()

	// Create file service
	fileService, err := storage.NewFileService()
	if err != nil {
		log.Fatalf("Failed to create file service: %v", err)
	}

	// Setup router
	r := router.SetupRouter(fileService)

	// Start server
	serverAddr := config.GetServerAddress()
	log.Printf("Server starting on %s...", serverAddr)
	if err := r.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
