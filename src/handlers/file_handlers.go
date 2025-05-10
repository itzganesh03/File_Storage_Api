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

type FileHandler struct {
	FileService *storage.FileService
}

func NewFileHandler(fileService *storage.FileService) *FileHandler {
	return &FileHandler{
		FileService: fileService,
	}
}

func (h *FileHandler) UploadFile(c *gin.Context) {
	userID, _ := c.Get("userID")
	userIDObj := userID.(primitive.ObjectID)

	if err := c.Request.ParseMultipartForm(constants.MaxMultipartMemory); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Failed to parse form"})
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "No file provided"})
		return
	}
	defer file.Close()

	metadata, err := h.FileService.UploadFile(userIDObj, header.Filename, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	response := gin.H{
		"message": constants.MessageFileUploaded,
		"file":    metadata,
	}

	if config.GetDisplayInMB() {
		response["unit"] = "MB"
		formattedMeta := *metadata
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

func (h *FileHandler) ListFiles(c *gin.Context) {
	userID, _ := c.Get("userID")
	userIDObj := userID.(primitive.ObjectID)

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	files, totalCount, err := models.GetFilesByUserIDPaginated(userIDObj, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list files"})
		return
	}

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

	if config.GetDisplayInMB() {
		response["unit"] = "MB"
		formattedFiles := make([]gin.H, len(files))
		for i, file := range files {
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
