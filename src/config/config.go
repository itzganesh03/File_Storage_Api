package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration
type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`

	JWT struct {
		Secret          string `yaml:"secret"`
		ExpirationHours int    `yaml:"expiration_hours"`
	} `yaml:"jwt"`
	Storage struct {
		Path        string `yaml:"path"`
		MaxPerUser  int64  `yaml:"max_per_user"`
		DisplayInMB bool   `yaml:"display_in_mb"`
	} `yaml:"storage"`

	MongoDB struct {
		URI         string `yaml:"uri"`
		Database    string `yaml:"database"`
		Collections struct {
			Users string `yaml:"users"`
			Files string `yaml:"files"`
		} `yaml:"collections"`
	} `yaml:"mongodb"`
}

// Global configuration
var AppConfig Config

// LoadConfig loads application configuration from YAML file
func LoadConfig(configPath string) error {
	// Use default config path if not provided
	if configPath == "" {
		configPath = "config.yml"
	}

	// Read config file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, &AppConfig); err != nil {
		return fmt.Errorf("error parsing config: %v", err)
	}

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(AppConfig.Storage.Path, 0755); err != nil {
		return fmt.Errorf("could not create storage directory: %v", err)
	}

	// Set defaults if values are missing
	if AppConfig.Server.Port == 0 {
		AppConfig.Server.Port = 8080
	}

	if AppConfig.Server.Host == "" {
		AppConfig.Server.Host = "localhost"
	}

	if AppConfig.JWT.Secret == "" {
		log.Println("Warning: JWT_SECRET not set. Using a default value. This is not secure for production.")
		AppConfig.JWT.Secret = "default-insecure-jwt-secret-change-in-production"
	}

	if AppConfig.JWT.ExpirationHours == 0 {
		AppConfig.JWT.ExpirationHours = 24
	}

	if AppConfig.Storage.MaxPerUser == 0 {
		AppConfig.Storage.MaxPerUser = 104857600 // 100MB default
	}

	if AppConfig.Storage.Path == "" {
		// Default to ./storage
		AppConfig.Storage.Path = filepath.Join(".", "storage")
	}

	// MongoDB defaults
	if AppConfig.MongoDB.URI == "" {
		AppConfig.MongoDB.URI = "mongodb://localhost:27017"
	}

	if AppConfig.MongoDB.Database == "" {
		AppConfig.MongoDB.Database = "file_storage_api"
	}

	if AppConfig.MongoDB.Collections.Users == "" {
		AppConfig.MongoDB.Collections.Users = "users"
	}

	if AppConfig.MongoDB.Collections.Files == "" {
		AppConfig.MongoDB.Collections.Files = "files"
	}

	return nil
}

// GetServerAddress returns the formatted server address (host:port)
func GetServerAddress() string {
	return fmt.Sprintf("%s:%d", AppConfig.Server.Host, AppConfig.Server.Port)
}

// GetStoragePath returns the configured storage path
func GetStoragePath() string {
	return AppConfig.Storage.Path
}

// GetJWTSecret returns the configured JWT secret
func GetJWTSecret() string {
	return AppConfig.JWT.Secret
}

// GetMaxStoragePerUser returns the maximum storage allowed per user
func GetMaxStoragePerUser() int64 {
	return AppConfig.Storage.MaxPerUser
}

// GetMongoDBURI returns the MongoDB connection URI
func GetMongoDBURI() string {
	return AppConfig.MongoDB.URI
}

// GetMongoDBName returns the MongoDB database name
func GetMongoDBName() string {
	return AppConfig.MongoDB.Database
}

// GetMongoDBUsersCollection returns the MongoDB users collection name
func GetMongoDBUsersCollection() string {
	return AppConfig.MongoDB.Collections.Users
}

// GetMongoDBFilesCollection returns the MongoDB files collection name
func GetMongoDBFilesCollection() string {
	return AppConfig.MongoDB.Collections.Files
}

// GetDisplayInMB returns whether storage should be displayed in MB
func GetDisplayInMB() bool {
	return AppConfig.Storage.DisplayInMB
}

// FormatStorageSize formats storage size according to configuration (MB or bytes)
func FormatStorageSize(sizeInBytes int64) float64 {
	if AppConfig.Storage.DisplayInMB {
		// Convert bytes to MB and round to 2 decimal places
		mbValue := float64(sizeInBytes) / 1024 / 1024
		return math.Round(mbValue*100) / 100 // Round to 2 decimal places
	}
	return float64(sizeInBytes)
}
