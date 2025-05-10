package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"

	"file-storage-api/src/constants"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`

	API struct {
		Register          string `yaml:"Register"`
		Login             string `yaml:"Login"`
		Me                string `yaml:"Me"`
		Files_Upload      string `yaml:"Files_Upload"`
		FileList          string `yaml:"FileList"`
		RemmainingStorage string `yaml:"RemmainingStorage"`
	} `yaml:"api"`

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

func LoadConfig(configPath string) error {
	if configPath == "" {
		configPath = constants.ConfigFilePath
	}

	configPath = filepath.FromSlash(configPath)

	data, err := ioutil.ReadFile(configPath)
	fmt.Println("Config file path:", configPath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	if err := yaml.Unmarshal(data, &AppConfig); err != nil {
		return fmt.Errorf("error parsing config: %v", err)
	}

	if err := os.MkdirAll(AppConfig.Storage.Path, 0755); err != nil {
		return fmt.Errorf("could not create storage directory: %v", err)
	}

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
		AppConfig.Storage.Path = filepath.Join(".", "storage")
	}

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

func GetServerAddress() string {
	return fmt.Sprintf("%s:%d", AppConfig.Server.Host, AppConfig.Server.Port)
}

func GetStoragePath() string {
	return AppConfig.Storage.Path
}

func GetJWTSecret() string {
	return AppConfig.JWT.Secret
}

func GetMaxStoragePerUser() int64 {
	return AppConfig.Storage.MaxPerUser
}

func GetMongoDBURI() string {
	return AppConfig.MongoDB.URI
}

func GetMongoDBName() string {
	return AppConfig.MongoDB.Database
}

func GetMongoDBUsersCollection() string {
	return AppConfig.MongoDB.Collections.Users
}

func GetMongoDBFilesCollection() string {
	return AppConfig.MongoDB.Collections.Files
}

func GetDisplayInMB() bool {
	return AppConfig.Storage.DisplayInMB
}

func FormatStorageSize(sizeInBytes int64) float64 {
	if AppConfig.Storage.DisplayInMB {
		mbValue := float64(sizeInBytes) / 1024 / 1024
		return math.Round(mbValue*100) / 100
	}
	return float64(sizeInBytes)
}

// API endpoint getters
func GetRegisterEndpoint() string {
	return AppConfig.API.Register
}

func GetLoginEndpoint() string {
	return AppConfig.API.Login
}

func GetMeEndpoint() string {
	return AppConfig.API.Me
}

func GetFilesUploadEndpoint() string {
	return AppConfig.API.Files_Upload
}

func GetFileListEndpoint() string {
	return AppConfig.API.FileList
}

func GetRemainingStorageEndpoint() string {
	return AppConfig.API.RemmainingStorage
}
