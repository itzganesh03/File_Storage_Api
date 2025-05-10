package helper

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/yourusername/file-storage-api/src/config"
	"github.com/yourusername/file-storage-api/src/models"
	"github.com/yourusername/file-storage-api/src/router"
	"github.com/yourusername/file-storage-api/src/storage"
)

var (
	testClient *mongo.Client
)

// SetupTestDatabase creates a test MongoDB connection
func SetupTestDatabase(t *testing.T) {
	// Use a separate test database
	testConfig := config.Config{
		Server: struct {
			Port int    `yaml:"port"`
			Host string `yaml:"host"`
		}{
			Port: 8080,
			Host: "localhost",
		}, JWT: struct {
			Secret          string `yaml:"secret"`
			ExpirationHours int    `yaml:"expiration_hours"`
		}{
			Secret:          "test-secret-key",
			ExpirationHours: 24,
		},
		Storage: struct {
			Path        string `yaml:"path"`
			MaxPerUser  int64  `yaml:"max_per_user"`
			DisplayInMB bool   `yaml:"display_in_mb"`
		}{
			Path:        "./test-storage",
			MaxPerUser:  1048576, // 1MB
			DisplayInMB: false,   // Use bytes for tests
		},
		MongoDB: struct {
			URI         string `yaml:"uri"`
			Database    string `yaml:"database"`
			Collections struct {
				Users string `yaml:"users"`
				Files string `yaml:"files"`
			} `yaml:"collections"`
		}{
			URI:      "mongodb://localhost:27017",
			Database: "file_storage_api_test",
			Collections: struct {
				Users string `yaml:"users"`
				Files string `yaml:"files"`
			}{
				Users: "users",
				Files: "files",
			},
		},
	}

	config.AppConfig = testConfig

	// Create test storage directory
	err := os.MkdirAll(config.GetStoragePath(), 0755)
	if err != nil {
		t.Fatalf("Failed to create test storage directory: %v", err)
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testClient, err = mongo.Connect(ctx, options.Client().ApplyURI(config.GetMongoDBURI()))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Check the connection
	err = testClient.Ping(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to ping MongoDB: %v", err)
	}

	// Drop existing test database to ensure clean state
	err = testClient.Database(config.GetMongoDBName()).Drop(ctx)
	if err != nil {
		t.Fatalf("Failed to drop test database: %v", err)
	}

	// Initialize models
	if err := models.InitMongoDB(); err != nil {
		t.Fatalf("Failed to initialize MongoDB: %v", err)
	}
	models.InitFileCollection()
}

// CleanupTestDatabase cleans up the test database and storage
func CleanupTestDatabase(t *testing.T) {
	// Clean up test storage
	err := os.RemoveAll(config.GetStoragePath())
	if err != nil {
		t.Logf("Warning: Failed to clean up test storage: %v", err)
	}

	// Drop test database
	if testClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = testClient.Database(config.GetMongoDBName()).Drop(ctx)
		if err != nil {
			t.Logf("Warning: Failed to drop test database: %v", err)
		}

		testClient.Disconnect(ctx)
	}

	models.CloseMongoDB()
}

// SetupTestRouter creates a test router with necessary services
func SetupTestRouter(t *testing.T) *gin.Engine {
	// Create file service
	fileService, err := storage.NewFileService()
	if err != nil {
		t.Fatalf("Failed to create file service: %v", err)
	}

	// Setup router
	gin.SetMode(gin.TestMode)
	return router.SetupRouter(fileService)
}
