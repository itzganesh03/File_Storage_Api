package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"file-storage-api/src/config"
	"file-storage-api/src/constants"
	"file-storage-api/src/models"
)

type FileService struct {
	storagePath string
}

func NewFileService() (*FileService, error) {
	storagePath := config.GetStoragePath()

	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("could not create storage directory: %w", err)
	}

	return &FileService{
		storagePath: storagePath,
	}, nil
}

func (s *FileService) createUserDirectory(username string) (string, error) {
	userDir := filepath.Join(s.storagePath, username)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", fmt.Errorf("could not create user directory: %w", err)
	}
	return userDir, nil
}

// UploadFile stores a file for a user
func (s *FileService) UploadFile(userID primitive.ObjectID, fileName string, fileData io.Reader) (*models.FileMetadata, error) {
	user, err := models.GetUserByID(userID)
	if err != nil {
		return nil, errors.New(constants.MessageUserNotFound)
	}

	exists, err := models.FileExistsByName(userID, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to check for duplicate files: %w", err)
	}
	if exists {
		return nil, errors.New(constants.MessageFileDuplicate)
	}
	userDir, err := s.createUserDirectory(user.Username)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(userDir, fileName)

	tmpFile, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	size, err := io.Copy(tmpFile, fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}
	if err := models.UpdateStorageUsed(userID, size); err != nil {
		return nil, err
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		_ = models.UpdateStorageUsed(userID, -size)
		return nil, fmt.Errorf("failed to reset file position: %w", err)
	}

	destFile, err := os.Create(filePath)
	if err != nil {
		_ = models.UpdateStorageUsed(userID, -size)
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, tmpFile); err != nil {
		_ = models.UpdateStorageUsed(userID, -size)
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}
	metadata := &models.FileMetadata{
		UserID:   userID,
		FileName: fileName,
		FilePath: filePath,
		Size:     size,
	}

	if err := models.CreateFileMetadata(metadata); err != nil {
		_ = models.UpdateStorageUsed(userID, -size)
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file metadata: %w", err)
	}

	return metadata, nil
}

func (s *FileService) DeleteFile(fileID, userID primitive.ObjectID) error {
	file, err := models.GetFileByID(fileID, userID)
	if err != nil {
		return err
	}

	if err := os.Remove(file.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	if err := models.UpdateStorageUsed(userID, -file.Size); err != nil {
		return err
	}

	return models.DeleteFileMetadata(fileID, userID)
}

func (s *FileService) DownloadFile(fileID, userID primitive.ObjectID) (string, *os.File, error) {
	file, err := models.GetFileByID(fileID, userID)
	if err != nil {
		return "", nil, err
	}

	f, err := os.Open(file.FilePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file.FileName, f, nil
}
