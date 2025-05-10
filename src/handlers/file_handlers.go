package handlers

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"file-storage-api/src/config"
	"file-storage-api/src/constants"
	"file-storage-api/src/models"
	"file-storage-api/src/storage"
)

// FileHandler handles file operations
type FileHandler struct {
	FileService *storage.FileService
}

// NewFileHandler creates a new FileHandler
func NewFileHandler(fileService *storage.FileService) *FileHandler {
	return &FileHandler{
		FileService: fileService,
	}
}

// UploadFile handles file uploads
func (h *FileHandler) UploadFile(c *gin.Context) {
	userID, _ := c.Get("userID")
	userIDObj := userID.(primitive.ObjectID)

	// Parse form
	if err := c.Request.ParseMultipartForm(constants.MaxMultipartMemory); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Failed to parse form"})
		return
	}

	// Get file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "No file provided"})
		return
	}
	defer file.Close()
	// Upload file
	metadata, err := h.FileService.UploadFile(userIDObj, header.Filename, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Format file size if needed
	response := gin.H{
		"message": constants.MessageFileUploaded,
		"file":    metadata,
	}
	// Add unit information if configured to display in MB
	if config.GetDisplayInMB() {
		response["unit"] = "MB"
		// Create a copy of metadata with size in MB
		formattedMeta := *metadata
		// Set the size directly as the formatted float value
		size := config.FormatStorageSize(metadata.Size)
		response["file"] = gin.H{
			"id":         formattedMeta.ID,
			"user_id":    formattedMeta.UserID,
			"file_name":  formattedMeta.FileName,
			"size":       size,
			"created_at": formattedMeta.CreatedAt,
			"updated_at": formattedMeta.UpdatedAt,
		}
	}

	c.JSON(http.StatusCreated, response)
}

// ListFiles lists all files for the current user with pagination support
func (h *FileHandler) ListFiles(c *gin.Context) {
	userID, _ := c.Get("userID")
	userIDObj := userID.(primitive.ObjectID)

	// Parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10 // Default page size
	}

	// Get files with pagination
	files, totalCount, err := models.GetFilesByUserIDPaginated(userIDObj, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list files"})
		return
	}

	// Calculate pagination metadata
	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	hasNextPage := page < totalPages
	hasPrevPage := page > 1
	response := gin.H{
		"files": files,
		"pagination": gin.H{
			"total_items":  totalCount,
			"total_pages":  totalPages,
			"current_page": page,
			"page_size":    pageSize,
			"has_next":     hasNextPage,
			"has_prev":     hasPrevPage,
		},
	}
	// Add unit information if configured to display in MB
	if config.GetDisplayInMB() {
		response["unit"] = "MB"
		// Format file sizes if display in MB is enabled
		formattedFiles := make([]gin.H, len(files))
		for i, file := range files {
			// Use a map to represent the file with properly formatted size
			size := config.FormatStorageSize(file.Size)
			formattedFiles[i] = gin.H{
				"id":         file.ID,
				"user_id":    file.UserID,
				"file_name":  file.FileName,
				"size":       size,
				"created_at": file.CreatedAt,
				"updated_at": file.UpdatedAt,
			}
		}
		response["files"] = formattedFiles
	}

	c.JSON(http.StatusOK, response)
}

// GetFile gets a specific file by ID
func (h *FileHandler) GetFile(c *gin.Context) {
	userID, _ := c.Get("userID")
	userIDObj := userID.(primitive.ObjectID)
	fileIDStr := c.Param("id")

	fileID, err := primitive.ObjectIDFromHex(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid file ID"})
		return
	}
	file, err := models.GetFileByID(fileID, userIDObj)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: constants.MessageFileNotFound})
		return
	}

	response := gin.H{
		"file": file,
	}
	// Add unit information if configured to display in MB
	if config.GetDisplayInMB() {
		response["unit"] = "MB"
		// Format file size using a clean map representation
		size := config.FormatStorageSize(file.Size)
		response["file"] = gin.H{
			"id":         file.ID,
			"user_id":    file.UserID,
			"file_name":  file.FileName,
			"size":       size,
			"created_at": file.CreatedAt,
			"updated_at": file.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, response)
}

// DownloadFile downloads a specific file by ID
func (h *FileHandler) DownloadFile(c *gin.Context) {
	userID, _ := c.Get("userID")
	userIDObj := userID.(primitive.ObjectID)
	fileIDStr := c.Param("id")

	fileID, err := primitive.ObjectIDFromHex(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid file ID"})
		return
	}

	fileName, file, err := h.FileService.DownloadFile(fileID, userIDObj)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: constants.MessageFileNotFound})
		return
	}
	defer file.Close()

	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/octet-stream")
	c.File(file.Name())
}

// DeleteFile deletes a specific file by ID
func (h *FileHandler) DeleteFile(c *gin.Context) {
	userID, _ := c.Get("userID")
	userIDObj := userID.(primitive.ObjectID)
	fileIDStr := c.Param("id")

	fileID, err := primitive.ObjectIDFromHex(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid file ID"})
		return
	}

	if err := h.FileService.DeleteFile(fileID, userIDObj); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: constants.MessageFileNotFound})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": constants.MessageFileDeleted,
	})
}

// GetRemainingStorage gets the remaining storage space for the current user
func (h *FileHandler) GetRemainingStorage(c *gin.Context) {
	userObj, _ := c.Get("user")
	user := userObj.(*models.User)

	// Calculate remaining storage
	remainingStorage := user.StorageLimit - user.StorageUsed

	// Format sizes based on configuration
	displayInMB := config.GetDisplayInMB()
	totalStorage := config.FormatStorageSize(user.StorageLimit)
	storageUsed := config.FormatStorageSize(user.StorageUsed)
	remainingStorageFormatted := config.FormatStorageSize(remainingStorage)

	// Prepare response
	response := gin.H{
		"total_storage":     totalStorage,
		"storage_used":      storageUsed,
		"remaining_storage": remainingStorageFormatted,
	}

	// Add unit information
	if displayInMB {
		response["unit"] = "MB"
	} else {
		response["unit"] = "bytes"
	}

	c.JSON(http.StatusOK, response)
}
