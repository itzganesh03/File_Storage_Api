package models

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"file-storage-api/src/config"
	"file-storage-api/src/constants"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username     string             `bson:"username" json:"username"`
	Password     string             `bson:"password" json:"-"`
	StorageLimit int64              `bson:"storage_limit" json:"storage_limit"`
	StorageUsed  int64              `bson:"storage_used" json:"storage_used"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

type UserRegistration struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserLogin struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

var (
	client         *mongo.Client
	userCollection *mongo.Collection
)

func InitMongoDB() error {
	var err error

	clientOptions := options.Client().ApplyURI(config.GetMongoDBURI())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	userCollection = client.Database(config.GetMongoDBName()).Collection(config.GetMongoDBUsersCollection())

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err = userCollection.Indexes().CreateOne(ctx, indexModel)
	return err
}

func CloseMongoDB() {
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}
}

func CreateUser(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := userCollection.CountDocuments(ctx, bson.M{"username": user.Username})
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New(constants.MessageUserExists)
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	result, err := userCollection.InsertOne(ctx, user)
	if err != nil {
		return err
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

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

func UpdateStorageUsed(userID primitive.ObjectID, sizeChange int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := GetUserByID(userID)
	if err != nil {
		return err
	}

	newSize := user.StorageUsed + sizeChange
	if newSize < 0 {
		newSize = 0
	}

	if sizeChange > 0 && newSize > user.StorageLimit {
		return errors.New(constants.MessageStorageLimitExceeded)
	}
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
