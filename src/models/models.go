package models

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/yourusername/file-storage-api/src/config"
	"github.com/yourusername/file-storage-api/src/constants"
)

// User represents a user in the system
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username     string             `bson:"username" json:"username"`
	Password     string             `bson:"password" json:"-"`                  // Password is not included in JSON responses
	StorageLimit int64              `bson:"storage_limit" json:"storage_limit"` // Storage limit in bytes
	StorageUsed  int64              `bson:"storage_used" json:"storage_used"`   // Current storage usage in bytes
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// UserRegistration is used for user registration requests
type UserRegistration struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserLogin is used for user login requests
type UserLogin struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse is the response for authentication operations
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// MongoDB client and collections
var (
	client         *mongo.Client
	userCollection *mongo.Collection
)

// InitMongoDB initializes the MongoDB connection and collections
func InitMongoDB() error {
	var err error

	// Set client options
	clientOptions := options.Client().ApplyURI(config.GetMongoDBURI())

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}

	// Check the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	// Get collection references
	userCollection = client.Database(config.GetMongoDBName()).Collection(config.GetMongoDBUsersCollection())

	// Create indexes
	// Unique index for username
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err = userCollection.Indexes().CreateOne(ctx, indexModel)
	return err
}

// CloseMongoDB closes the MongoDB connection
func CloseMongoDB() {
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}
}

// CreateUser creates a new user in MongoDB
func CreateUser(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if username already exists
	count, err := userCollection.CountDocuments(ctx, bson.M{"username": user.Username})
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New(constants.MessageUserExists)
	}

	// Set creation and update times
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Insert user
	result, err := userCollection.InsertOne(ctx, user)
	if err != nil {
		return err
	}

	// Set the ID from the inserted document
	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetUserByID finds a user by ID
func GetUserByID(id primitive.ObjectID) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := userCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New(constants.MessageUserNotFound)
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByUsername finds a user by username
func GetUserByUsername(username string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := userCollection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New(constants.MessageUserNotFound)
		}
		return nil, err
	}

	return &user, nil
}

// UpdateStorageUsed updates the storage used by a user
func UpdateStorageUsed(userID primitive.ObjectID, sizeChange int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get current user
	user, err := GetUserByID(userID)
	if err != nil {
		return err
	}

	// Calculate new storage used
	newSize := user.StorageUsed + sizeChange
	if newSize < 0 {
		newSize = 0
	}

	// Check if exceeding limit
	if sizeChange > 0 && newSize > user.StorageLimit {
		return errors.New(constants.MessageStorageLimitExceeded)
	}

	// Update storage used and update time
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"storage_used": newSize,
			"updated_at":   time.Now(),
		},
	}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	return err
}
