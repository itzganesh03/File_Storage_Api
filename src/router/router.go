package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yourusername/file-storage-api/src/handlers"
	"github.com/yourusername/file-storage-api/src/storage"
)

// SetupRouter sets up the API routes
func SetupRouter(fileService *storage.FileService) *gin.Engine {
	router := gin.Default()

	// Create handlers
	authHandler := &handlers.AuthHandler{}
	fileHandler := handlers.NewFileHandler(fileService)

	// API group
	api := router.Group("/api")

	// Public routes
	api.POST("/register", authHandler.RegisterUser)
	api.POST("/login", authHandler.LoginUser)

	// Protected routes
	protected := api.Group("")
	protected.Use(handlers.AuthMiddleware())
	{
		protected.GET("/me", authHandler.GetUserInfo)

		// File routes
		protected.POST("/files", fileHandler.UploadFile)
		protected.GET("/files", fileHandler.ListFiles)
		protected.GET("/files/:id", fileHandler.GetFile)
		protected.GET("/files/:id/download", fileHandler.DownloadFile)
		protected.DELETE("/files/:id", fileHandler.DeleteFile)

		// Storage routes
		protected.GET("/storage/remaining", fileHandler.GetRemainingStorage)
	}

	return router
}
