package router

import (
	"github.com/gin-gonic/gin"

	"file-storage-api/src/config"
	"file-storage-api/src/handlers"
	"file-storage-api/src/storage"
)

func SetupRouter(fileService *storage.FileService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	authHandler := &handlers.AuthHandler{}
	fileHandler := handlers.NewFileHandler(fileService)
	api := router.Group("/api")

	api.POST(config.GetRegisterEndpoint(), authHandler.RegisterUser)
	api.POST(config.GetLoginEndpoint(), authHandler.LoginUser)

	protected := api.Group("")
	protected.Use(handlers.AuthMiddleware())
	{
		protected.GET(config.GetMeEndpoint(), authHandler.GetUserInfo)

		protected.POST(config.GetFilesUploadEndpoint(), fileHandler.UploadFile)
		protected.GET(config.GetFileListEndpoint(), fileHandler.ListFiles)
		protected.GET(config.GetRemainingStorageEndpoint(), fileHandler.GetRemainingStorage)
	}

	return router
}
