package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/yourusername/file-storage-api/src/config"
	"github.com/yourusername/file-storage-api/src/constants"
	"github.com/yourusername/file-storage-api/src/models"
)

// FileService manages file storage operations
type FileService struct {
	storagePath string
}

// NewFileService creates a new FileService
func NewFileService() (*FileService, error) {
	storagePath := config.GetStoragePath()

	// Ensure storage path exists
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("could not create storage directory: %w", err)
	}

	return &FileService{
		storagePath: storagePath,
	}, nil
}

// createUserDirectory creates a directory for a user if it doesn't exist
func (s *FileService) createUserDirectory(username string) (string, error) {
	userDir := filepath.Join(s.storagePath, username)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", fmt.Errorf("could not create user directory: %w", err)
	}
	return userDir, nil
}

// UploadFile stores a file for a user
func (s *FileService) UploadFile(userID primitive.ObjectID, fileName string, fileData io.Reader) (*models.FileMetadata, error) {
	// Check if user exists
	user, err := models.GetUserByID(userID)
	if err != nil {
		return nil, errors.New(constants.MessageUserNotFound)
	}

	// Create user directory if it doesn't exist
	userDir, err := s.createUserDirectory(user.Username)
	if err != nil {
		return nil, err
	}

	// Create the file path
	filePath := filepath.Join(userDir, fileName)

	// Create a temporary file to calculate size before adding to quota
	tmpFile, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy file data to temporary file to get size
	size, err := io.Copy(tmpFile, fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	// Check if this would exceed user's storage limit
	if err := models.UpdateStorageUsed(userID, size); err != nil {
		return nil, err
	}

	// Reset file position
	if _, err := tmpFile.Seek(0, 0); err != nil {
		// Rollback storage usage update
		_ = models.UpdateStorageUsed(userID, -size)
		return nil, fmt.Errorf("failed to reset file position: %w", err)
	}

	// Create the destination file
	destFile, err := os.Create(filePath)
	if err != nil {
		// Rollback storage usage update
		_ = models.UpdateStorageUsed(userID, -size)
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer destFile.Close()

	// Copy from temp file to destination
	if _, err := io.Copy(destFile, tmpFile); err != nil {
		// Rollback storage usage update
		_ = models.UpdateStorageUsed(userID, -size)
		// Try to remove the failed file
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Create file metadata in MongoDB
	metadata := &models.FileMetadata{
		UserID:   userID,
		FileName: fileName,
		FilePath: filePath,
		Size:     size,
	}

	if err := models.CreateFileMetadata(metadata); err != nil {
		// Rollback storage usage update
		_ = models.UpdateStorageUsed(userID, -size)
		// Try to remove the failed file
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file metadata: %w", err)
	}

	return metadata, nil
}

// DeleteFile removes a file
func (s *FileService) DeleteFile(fileID, userID primitive.ObjectID) error {
	// Get file metadata
	file, err := models.GetFileByID(fileID, userID)
	if err != nil {
		return err
	}

	// Delete the file from storage
	if err := os.Remove(file.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Update user's storage usage
	if err := models.UpdateStorageUsed(userID, -file.Size); err != nil {
		return err
	}

	// Remove file metadata from MongoDB
	return models.DeleteFileMetadata(fileID, userID)
}

// DownloadFile opens a file for download
func (s *FileService) DownloadFile(fileID, userID primitive.ObjectID) (string, *os.File, error) {
	// Get file metadata
	file, err := models.GetFileByID(fileID, userID)
	if err != nil {
		return "", nil, err
	}

	// Open the file
	f, err := os.Open(file.FilePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file.FileName, f, nil
}
