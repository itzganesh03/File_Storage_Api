package auth

import (
	"errors"
	"time"

	"file-storage-api/src/config"
	"file-storage-api/src/constants"
	"file-storage-api/src/models"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// HashPassword as requested on assignment using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateToken(user models.User) (string, error) {
	jwtSecret := config.GetJWTSecret()
	if jwtSecret == "" {
		return "", errors.New("JWT secret not configured")
	}

	expirationTime := time.Now().Add(time.Duration(config.AppConfig.JWT.ExpirationHours) * time.Hour)

	claims := &Claims{
		UserID: user.ID.Hex(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateToken(tokenString string) (primitive.ObjectID, error) {
	jwtSecret := config.GetJWTSecret()
	if jwtSecret == "" {
		return primitive.NilObjectID, errors.New("JWT secret not configured")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return primitive.NilObjectID, err
	}

	if !token.Valid {
		return primitive.NilObjectID, errors.New(constants.MessageInvalidToken)
	}

	objectID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return primitive.NilObjectID, errors.New(constants.MessageInvalidToken)
	}

	return objectID, nil
}

func GetDefaultStorageLimit() int64 {
	return config.GetMaxStoragePerUser()
}
