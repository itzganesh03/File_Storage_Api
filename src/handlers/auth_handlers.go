package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"file-storage-api/src/auth"
	"file-storage-api/src/constants"
	"file-storage-api/src/models"
)

type AuthHandler struct{}

func (h *AuthHandler) RegisterUser(c *gin.Context) {
	var req models.UserRegistration

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: constants.MessageInvalidRequest})
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to process password"})
		return
	}

	storageLimit := auth.GetDefaultStorageLimit()

	user := &models.User{
		Username:     req.Username,
		Password:     hashedPassword,
		StorageLimit: storageLimit,
		StorageUsed:  0,
	}

	if err := models.CreateUser(user); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": constants.MessageUserCreated,
		"user": gin.H{
			"id":            user.ID.Hex(),
			"username":      user.Username,
			"storage_limit": user.StorageLimit,
			"storage_used":  user.StorageUsed,
			"created_at":    user.CreatedAt,
		},
	})
}

func (h *AuthHandler) LoginUser(c *gin.Context) {
	var req models.UserLogin

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: constants.MessageInvalidRequest})
		return
	}

	user, err := models.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: constants.MessageInvalidCredentials})
		return
	}

	if !auth.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: constants.MessageInvalidCredentials})
		return
	}

	token, err := auth.GenerateToken(*user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to generate authentication token"})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		Token: token,
		User: models.User{
			ID:           user.ID,
			Username:     user.Username,
			StorageLimit: user.StorageLimit,
			StorageUsed:  user.StorageUsed,
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
		},
	})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(constants.AuthorizationHeader)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Authorization header is required"})
			return
		}

		if len(authHeader) <= len(constants.BearerPrefix) || authHeader[:len(constants.BearerPrefix)] != constants.BearerPrefix {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Authorization header format must be Bearer <token>"})
			return
		}
		tokenString := authHeader[len(constants.BearerPrefix):]

		userID, err := auth.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{Error: constants.MessageInvalidToken})
			return
		}

		user, err := models.GetUserByID(userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{Error: constants.MessageUserNotFound})
			return
		}

		c.Set("userID", userID)
		c.Set("user", user)

		c.Next()
	}
}

func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	user, _ := c.Get("user")

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
