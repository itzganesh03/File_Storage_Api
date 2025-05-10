package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/yourusername/file-storage-api/src/config"
	"github.com/yourusername/file-storage-api/src/constants"
	"github.com/yourusername/file-storage-api/src/models"
)

// JWT claims structure
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// HashPassword creates a hashed password from plain text
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash compares a password with a hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken generates a JWT token for a user
func GenerateToken(user models.User) (string, error) {
	// Get JWT secret
	jwtSecret := config.GetJWTSecret()
	if jwtSecret == "" {
		return "", errors.New("JWT secret not configured")
	}

	// Calculate expiration time
	expirationTime := time.Now().Add(time.Duration(config.AppConfig.JWT.ExpirationHours) * time.Hour)

	// Create the Claims
	claims := &Claims{
		UserID: user.ID.Hex(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the user ID
func ValidateToken(tokenString string) (primitive.ObjectID, error) {
	// Get JWT secret
	jwtSecret := config.GetJWTSecret()
	if jwtSecret == "" {
		return primitive.NilObjectID, errors.New("JWT secret not configured")
	}

	// Parse the token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return primitive.NilObjectID, err
	}

	// Validate the token and extract the claims
	if !token.Valid {
		return primitive.NilObjectID, errors.New(constants.MessageInvalidToken)
	}

	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return primitive.NilObjectID, errors.New(constants.MessageInvalidToken)
	}

	return objectID, nil
}

// GetDefaultStorageLimit returns the default storage limit for new users
func GetDefaultStorageLimit() int64 {
	return config.GetMaxStoragePerUser()
}
