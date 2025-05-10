package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/file-storage-api/src/auth"
	"github.com/yourusername/file-storage-api/src/helper"
	"github.com/yourusername/file-storage-api/src/models"
)

func TestUserRegistration(t *testing.T) {
	// Setup test environment
	helper.SetupTestDatabase(t)
	defer helper.CleanupTestDatabase(t)

	router := helper.SetupTestRouter(t)

	// Test user registration
	w := httptest.NewRecorder()
	reqBody := map[string]string{
		"username": "testuser",
		"password": "testpassword",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "message")
	assert.Equal(t, "User created successfully", response["message"])

	assert.Contains(t, response, "user")
	user := response["user"].(map[string]interface{})
	assert.Equal(t, "testuser", user["username"])
}

func TestUserLogin(t *testing.T) {
	// Setup test environment
	helper.SetupTestDatabase(t)
	defer helper.CleanupTestDatabase(t)

	router := helper.SetupTestRouter(t)

	// Create a test user
	hashedPassword, _ := auth.HashPassword("testpassword")
	user := &models.User{
		Username:     "testuser",
		Password:     hashedPassword,
		StorageLimit: 1048576,
		StorageUsed:  0,
	}

	err := models.CreateUser(user)
	assert.NoError(t, err)

	// Test user login
	w := httptest.NewRecorder()
	reqBody := map[string]string{
		"username": "testuser",
		"password": "testpassword",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "token")
	assert.NotEmpty(t, response["token"])
}

func TestInvalidLogin(t *testing.T) {
	// Setup test environment
	helper.SetupTestDatabase(t)
	defer helper.CleanupTestDatabase(t)

	router := helper.SetupTestRouter(t)

	// Create a test user
	hashedPassword, _ := auth.HashPassword("testpassword")
	user := &models.User{
		Username:     "testuser",
		Password:     hashedPassword,
		StorageLimit: 1048576,
		StorageUsed:  0,
	}

	err := models.CreateUser(user)
	assert.NoError(t, err)

	// Test invalid password
	w := httptest.NewRecorder()
	reqBody := map[string]string{
		"username": "testuser",
		"password": "wrongpassword",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Equal(t, "Invalid username or password", response["error"])
}

func TestDuplicateUsername(t *testing.T) {
	// Setup test environment
	helper.SetupTestDatabase(t)
	defer helper.CleanupTestDatabase(t)

	router := helper.SetupTestRouter(t)

	// Create a test user
	hashedPassword, _ := auth.HashPassword("testpassword")
	user := &models.User{
		Username:     "testuser",
		Password:     hashedPassword,
		StorageLimit: 1048576,
		StorageUsed:  0,
	}

	err := models.CreateUser(user)
	assert.NoError(t, err)

	// Test duplicate username
	w := httptest.NewRecorder()
	reqBody := map[string]string{
		"username": "testuser",
		"password": "anotherpassword",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Equal(t, "User already exists", response["error"])
}
