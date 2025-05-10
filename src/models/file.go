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

// GetFilesByUserIDPaginated lists files for a given user with pagination
func GetFilesByUserIDPaginated(userID primitive.ObjectID, page, pageSize int) ([]*FileMetadata, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Calculate skip value based on page and pageSize
	skip := (page - 1) * pageSize

	// Get total count of files for pagination metadata
	totalCount, err := fileCollection.CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, 0, err
	}

	// Find files with pagination
	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(pageSize))
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}}) // Sort by newest first

	cursor, err := fileCollection.Find(ctx, bson.M{"user_id": userID}, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var files []*FileMetadata
	if err = cursor.All(ctx, &files); err != nil {
		return nil, 0, err
	}

	return files, totalCount, nil
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

// FileExistsByName checks if a file with the given name already exists for the user
func FileExistsByName(userID primitive.ObjectID, fileName string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Count files with the same name for this user
	count, err := fileCollection.CountDocuments(ctx, bson.M{
		"user_id":   userID,
		"file_name": fileName,
	})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
