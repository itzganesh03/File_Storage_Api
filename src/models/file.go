package models

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/yourusername/file-storage-api/src/config"
	"github.com/yourusername/file-storage-api/src/constants"
)

// FileMetadata represents metadata for a stored file
type FileMetadata struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	FileName  string             `bson:"file_name" json:"file_name"`
	FilePath  string             `bson:"file_path" json:"-"` // Internal path, not exposed in API
	Size      int64              `bson:"size" json:"size"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

var (
	fileCollection *mongo.Collection
)

// InitFileCollection initializes the file collection in MongoDB
func InitFileCollection() {
	if client != nil {
		fileCollection = client.Database(config.GetMongoDBName()).Collection(config.GetMongoDBFilesCollection())

		// Create index on user_id for faster queries
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		indexModel := mongo.IndexModel{
			Keys: bson.D{{Key: "user_id", Value: 1}},
		}

		_, _ = fileCollection.Indexes().CreateOne(ctx, indexModel)
	}
}

// CreateFileMetadata saves file metadata to MongoDB
func CreateFileMetadata(metadata *FileMetadata) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set creation and update times
	now := time.Now()
	metadata.CreatedAt = now
	metadata.UpdatedAt = now

	// Insert file metadata
	result, err := fileCollection.InsertOne(ctx, metadata)
	if err != nil {
		return err
	}

	// Set the ID from the inserted document
	metadata.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetFilesByUserID lists all files for a given user
func GetFilesByUserID(userID primitive.ObjectID) ([]*FileMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := fileCollection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var files []*FileMetadata
	if err = cursor.All(ctx, &files); err != nil {
		return nil, err
	}

	return files, nil
}

// GetFileByID retrieves a file by its ID
func GetFileByID(fileID, userID primitive.ObjectID) (*FileMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var file FileMetadata
	err := fileCollection.FindOne(ctx, bson.M{
		"_id":     fileID,
		"user_id": userID,
	}).Decode(&file)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New(constants.MessageFileNotFound)
		}
		return nil, err
	}

	return &file, nil
}

// DeleteFileMetadata removes file metadata from MongoDB
func DeleteFileMetadata(fileID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := fileCollection.DeleteOne(ctx, bson.M{
		"_id":     fileID,
		"user_id": userID,
	})

	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New(constants.MessageFileNotFound)
	}

	return nil
}
