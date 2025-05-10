package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/yourusername/file-storage-api/src/auth"
	"github.com/yourusername/file-storage-api/src/helper"
	"github.com/yourusername/file-storage-api/src/models"
)

func createTestUser(t *testing.T) (string, primitive.ObjectID) {
	// Create a test user
	hashedPassword, _ := auth.HashPassword("testpassword")
	user := &models.User{
		Username:     "testuser",
		Password:     hashedPassword,
		StorageLimit: 1048576, // 1MB
		StorageUsed:  0,
	}

	err := models.CreateUser(user)
	assert.NoError(t, err)

	// Generate token
	token, err := auth.GenerateToken(*user)
	assert.NoError(t, err)

	return token, user.ID
}

func createTestFile(t *testing.T, filename string, size int) string {
	// Create a temporary test file
	filePath := filepath.Join(os.TempDir(), filename)
	file, err := os.Create(filePath)
	assert.NoError(t, err)
	defer file.Close()

	// Write random data to the file
	data := make([]byte, size)
	_, err = file.Write(data)
	assert.NoError(t, err)

	return filePath
}

func TestFileUploadAndList(t *testing.T) {
	// Setup test environment
	helper.SetupTestDatabase(t)
	defer helper.CleanupTestDatabase(t)

	router := helper.SetupTestRouter(t)

	// Get auth token
	token, _ := createTestUser(t)

	// Create a test file
	testFilePath := createTestFile(t, "test-file.txt", 1024) // 1KB file
	defer os.Remove(testFilePath)

	// Prepare file upload request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test-file.txt")
	assert.NoError(t, err)

	file, err := os.Open(testFilePath)
	assert.NoError(t, err)
	defer file.Close()

	_, err = io.Copy(part, file)
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	// Upload file
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// List files
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/files", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	files := response["files"].([]interface{})
	assert.Equal(t, 1, len(files))

	file1 := files[0].(map[string]interface{})
	assert.Equal(t, "test-file.txt", file1["file_name"])
}

func TestDownloadAndDeleteFile(t *testing.T) {
	// Setup test environment
	helper.SetupTestDatabase(t)
	defer helper.CleanupTestDatabase(t)

	router := helper.SetupTestRouter(t)

	// Get auth token
	token, _ := createTestUser(t)

	// Create and upload a test file
	testFilePath := createTestFile(t, "test-file.txt", 1024) // 1KB file
	defer os.Remove(testFilePath)

	// Prepare file upload request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test-file.txt")
	assert.NoError(t, err)

	file, err := os.Open(testFilePath)
	assert.NoError(t, err)
	defer file.Close()

	_, err = io.Copy(part, file)
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	// Upload file
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	var uploadResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &uploadResponse)
	assert.NoError(t, err)

	fileData := uploadResponse["file"].(map[string]interface{})
	fileID := fileData["id"].(string)

	// Download file
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/files/"+fileID+"/download", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "attachment; filename=test-file.txt", w.Header().Get("Content-Disposition"))

	// Delete file
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/files/"+fileID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var deleteResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &deleteResponse)
	assert.NoError(t, err)

	assert.Equal(t, "File deleted successfully", deleteResponse["message"])

	// Verify file is no longer in the list
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/files", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	var listResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	assert.NoError(t, err)

	files := listResponse["files"].([]interface{})
	assert.Equal(t, 0, len(files))
}

func TestStorageLimitExceeded(t *testing.T) {
	// Setup test environment
	helper.SetupTestDatabase(t)
	defer helper.CleanupTestDatabase(t)

	router := helper.SetupTestRouter(t)

	// Get auth token and create a user with very low storage limit
	hashedPassword, _ := auth.HashPassword("testpassword")
	user := &models.User{
		Username:     "limiteduser",
		Password:     hashedPassword,
		StorageLimit: 512, // Only 512 bytes
		StorageUsed:  0,
	}

	err := models.CreateUser(user)
	assert.NoError(t, err)

	token, err := auth.GenerateToken(*user)
	assert.NoError(t, err)

	// Create a test file larger than the limit
	testFilePath := createTestFile(t, "large-file.txt", 1024) // 1KB file
	defer os.Remove(testFilePath)

	// Prepare file upload request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "large-file.txt")
	assert.NoError(t, err)

	file, err := os.Open(testFilePath)
	assert.NoError(t, err)
	defer file.Close()

	_, err = io.Copy(part, file)
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	// Upload file - should fail due to storage limit
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Equal(t, "Storage limit exceeded", response["error"])
}
