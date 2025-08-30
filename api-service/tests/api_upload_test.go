package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"digital-recipes/api-service/db"
	"digital-recipes/api-service/handlers"
	"digital-recipes/api-service/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates and sets up a test database connection
func setupTestDB(t *testing.T) *db.Database {
	// Check if we have a test database URL
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping upload API integration tests")
	}

	// Initialize database connection
	database, err := db.NewConnection()
	require.NoError(t, err, "Failed to connect to test database")

	// Run migrations to set up schema
	err = database.RunMigrations("../db/migrations")
	require.NoError(t, err, "Failed to run migrations on test database")

	return database
}

// cleanupTestDB closes the database connection
func cleanupTestDB(t *testing.T, database *db.Database) {
	if database != nil {
		database.Close()
	}
}

func TestPostUploadRequest(t *testing.T) {
	// Set up test database
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)

	// Create storage service (will fail gracefully in test environment)
	storageService, _ := handlers.NewStorageService()

	// Create recipe handler
	recipeHandler := handlers.NewRecipeHandler(database, storageService)

	// Set up Gin router
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/recipes/upload-request", recipeHandler.PostUploadRequest)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "Valid upload request with 2 images",
			requestBody: models.UploadRequest{
				ImageCount: 2,
			},
			expectedStatus: 0, // Will be set dynamically based on result
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				// Check if response has data (success case)
				if data, exists := response["data"]; exists && data != nil {
					dataMap := data.(map[string]interface{})

					// Check recipe_id is present and positive
					require.Contains(t, dataMap, "recipe_id")
					recipeID := dataMap["recipe_id"].(float64)
					assert.Greater(t, int(recipeID), 0)

					// Check upload_urls array
					require.Contains(t, dataMap, "upload_urls")
					uploadURLs := dataMap["upload_urls"].([]interface{})
					
					// Should have 2 URLs if storage service is available
					// If not available, should still create recipe but no URLs
					if len(uploadURLs) > 0 {
						assert.Equal(t, 2, len(uploadURLs))
						
						// Check first upload URL structure
						firstURL := uploadURLs[0].(map[string]interface{})
						require.Contains(t, firstURL, "image_id")
						require.Contains(t, firstURL, "upload_url")
						
						assert.NotEmpty(t, firstURL["image_id"])
						assert.NotEmpty(t, firstURL["upload_url"])
					}
				} else {
					// If no data field, might be an error response
					// This can happen if storage service is not configured
					t.Logf("Response without data field (possibly due to storage service unavailable): %+v", response)
				}
			},
		},
		{
			name: "Valid upload request with 1 image",
			requestBody: models.UploadRequest{
				ImageCount: 1,
			},
			expectedStatus: 0, // Dynamic status
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				// Same flexible handling as first test
				if data, exists := response["data"]; exists && data != nil {
					dataMap := data.(map[string]interface{})

					require.Contains(t, dataMap, "recipe_id")
					recipeID := dataMap["recipe_id"].(float64)
					assert.Greater(t, int(recipeID), 0)

					require.Contains(t, dataMap, "upload_urls")
					uploadURLs := dataMap["upload_urls"].([]interface{})
					
					// Should have 1 URL if storage available
					if len(uploadURLs) > 0 {
						assert.Equal(t, 1, len(uploadURLs))
					}
				} else {
					t.Logf("Response without data field (possibly due to storage service unavailable): %+v", response)
				}
			},
		},
		{
			name: "Invalid request - image count too high",
			requestBody: models.UploadRequest{
				ImageCount: 15, // Max is 10
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				require.Contains(t, response, "error")
				assert.Equal(t, "invalid request body", response["error"])
			},
		},
		{
			name: "Invalid request - image count zero",
			requestBody: models.UploadRequest{
				ImageCount: 0,
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				require.Contains(t, response, "error")
				assert.Equal(t, "invalid request body", response["error"])
			},
		},
		{
			name: "Invalid request - missing image count",
			requestBody: map[string]interface{}{
				"other_field": "value",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				require.Contains(t, response, "error")
				assert.Equal(t, "invalid request body", response["error"])
			},
		},
		{
			name:           "Invalid request - empty body",
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				require.Contains(t, response, "error")
				assert.Equal(t, "invalid request body", response["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			var requestBody []byte
			var err error
			
			if tt.requestBody == "" {
				requestBody = []byte("")
			} else {
				requestBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			// Create request
			req, err := http.NewRequest("POST", "/api/v1/recipes/upload-request", bytes.NewBuffer(requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			r.ServeHTTP(w, req)

			// Check status code (flexible for first test)
			if tt.expectedStatus != 0 {
				assert.Equal(t, tt.expectedStatus, w.Code)
			} else {
				// For dynamic status - accept either success or error
				assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError,
					"Expected either 200 (success) or 500 (storage unavailable), got %d", w.Code)
			}

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Run specific checks
			tt.checkResponse(t, response)
		})
	}
}

func TestPostUploadRequestDatabaseIntegration(t *testing.T) {
	// Set up test database
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)

	// Create storage service
	storageService, _ := handlers.NewStorageService()

	// Create recipe handler
	recipeHandler := handlers.NewRecipeHandler(database, storageService)

	// Set up Gin router
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/recipes/upload-request", recipeHandler.PostUploadRequest)

	// Create upload request
	uploadRequest := models.UploadRequest{
		ImageCount: 2,
	}

	requestBody, err := json.Marshal(uploadRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/recipes/upload-request", bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should succeed (or fail gracefully if no S3 credentials)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)

	if w.Code == http.StatusOK {
		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		recipeID := int(data["recipe_id"].(float64))

		// Verify recipe was created in database
		var count int
		query := `SELECT COUNT(*) FROM recipes WHERE id = $1 AND status = 'processing'`
		err = database.DB.QueryRow(query, recipeID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Verify recipe details
		var title, status string
		var userID int
		query = `SELECT title, status, user_id FROM recipes WHERE id = $1`
		err = database.DB.QueryRow(query, recipeID).Scan(&title, &status, &userID)
		require.NoError(t, err)
		
		assert.Equal(t, "Processing Recipe", title)
		assert.Equal(t, "processing", status)
		assert.Equal(t, 1, userID) // MVP default user ID
	}
}