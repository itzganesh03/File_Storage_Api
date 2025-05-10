package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yourusername/file-storage-api/src/auth"
	"github.com/yourusername/file-storage-api/src/constants"
	"github.com/yourusername/file-storage-api/src/models"
)

// AuthHandler handles authentication related endpoints
type AuthHandler struct{}

// RegisterUser handles user registration
func (h *AuthHandler) RegisterUser(c *gin.Context) {
	var req models.UserRegistration

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: constants.MessageInvalidRequest})
		return
	}

	// Hash the password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to process password"})
		return
	}

	// Get default storage limit
	storageLimit := auth.GetDefaultStorageLimit()

	// Create the user
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

// LoginUser handles user login
func (h *AuthHandler) LoginUser(c *gin.Context) {
	var req models.UserLogin

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: constants.MessageInvalidRequest})
		return
	}

	// Find user by username
	user, err := models.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: constants.MessageInvalidCredentials})
		return
	}

	// Verify password
	if !auth.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: constants.MessageInvalidCredentials})
		return
	}

	// Generate JWT token
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

// AuthMiddleware is a middleware to check if the user is authenticated
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the token from the Authorization header
		authHeader := c.GetHeader(constants.AuthorizationHeader)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Authorization header is required"})
			return
		}

		// Expecting "Bearer <token>"
		if len(authHeader) <= len(constants.BearerPrefix) || authHeader[:len(constants.BearerPrefix)] != constants.BearerPrefix {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Authorization header format must be Bearer <token>"})
			return
		}
		tokenString := authHeader[len(constants.BearerPrefix):]

		// Validate the token
		userID, err := auth.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{Error: constants.MessageInvalidToken})
			return
		}

		// Get user from store
		user, err := models.GetUserByID(userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{Error: constants.MessageUserNotFound})
			return
		}

		// Set the user ID and user in the context for future use
		c.Set("userID", userID)
		c.Set("user", user)

		c.Next()
	}
}

// GetUserInfo returns information about the current user
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	user, _ := c.Get("user")

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
