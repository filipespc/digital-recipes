package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"digital-recipes/api-service/db"
	"digital-recipes/api-service/handlers"
	"digital-recipes/api-service/middleware"
	"digital-recipes/api-service/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PantrySearchTestSuite contains tests for pantry search endpoints
type PantrySearchTestSuite struct {
	suite.Suite
	db             *db.Database
	router         *gin.Engine
	authRouter     *gin.Engine
	testUserID     int
	otherUserID    int
	authConfig     *middleware.AuthConfig
	testUserToken  string
	otherUserToken string
}

// SetupSuite runs before all tests in the suite
func (suite *PantrySearchTestSuite) SetupSuite() {
	// Set up test database connection
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		suite.T().Skip("TEST_DATABASE_URL not set, skipping pantry search tests")
	}

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Initialize database connection
	database, err := db.NewConnection()
	require.NoError(suite.T(), err, "Failed to connect to test database")
	suite.db = database

	// Run migrations
	err = suite.db.RunMigrations("../db/migrations")
	require.NoError(suite.T(), err, "Failed to run migrations")

	// Create test users
	suite.testUserID = suite.createTestUser("test@example.com", "Test User")
	suite.otherUserID = suite.createTestUser("other@example.com", "Other User")

	// Set up auth configuration
	os.Setenv("JWT_SECRET", "test-secret-key-must-be-at-least-32-characters-long")
	suite.authConfig = middleware.NewAuthConfig()

	// Generate test tokens
	suite.testUserToken, err = middleware.GenerateToken(suite.authConfig, suite.testUserID, "test@example.com", "Test User")
	require.NoError(suite.T(), err)
	suite.otherUserToken, err = middleware.GenerateToken(suite.authConfig, suite.otherUserID, "other@example.com", "Other User")
	require.NoError(suite.T(), err)

	// Set up routers
	suite.setupRouters()

	// Seed test pantry items
	suite.seedPantryItems()
}

// setupRouters creates test routers with and without auth
func (suite *PantrySearchTestSuite) setupRouters() {
	// Create handlers (pass nil for StorageService in tests)
	recipeHandler := handlers.NewRecipeHandler(suite.db, nil)

	// Router without auth (for testing missing auth)
	suite.router = gin.New()
	suite.router.GET("/api/v1/pantry/search", recipeHandler.SearchPantryItems)
	suite.router.GET("/api/v1/pantry/fuzzy-search", recipeHandler.FuzzySearchPantryItems)

	// Router with auth middleware
	suite.authRouter = gin.New()
	protected := suite.authRouter.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(suite.authConfig))
	{
		protected.GET("/pantry/search", recipeHandler.SearchPantryItems)
		protected.GET("/pantry/fuzzy-search", recipeHandler.FuzzySearchPantryItems)
	}
}

// createTestUser creates a test user and returns their ID
func (suite *PantrySearchTestSuite) createTestUser(email, name string) int {
	var userID int
	err := suite.db.DB.QueryRow(`
		INSERT INTO users (email, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (email) DO UPDATE SET name = $2
		RETURNING id
	`, email, name).Scan(&userID)
	require.NoError(suite.T(), err)
	return userID
}

// seedPantryItems adds test pantry items to the database
func (suite *PantrySearchTestSuite) seedPantryItems() {
	// Add items for test user
	items := []string{"tomato", "tomatoes", "potato", "salt", "pepper", "onion"}
	for _, item := range items {
		_, err := suite.db.DB.Exec(`
			INSERT INTO pantry_items (user_id, name, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
			ON CONFLICT (user_id, name) DO NOTHING
		`, suite.testUserID, item)
		require.NoError(suite.T(), err)
	}

	// Add items for other user
	otherItems := []string{"carrot", "celery", "garlic"}
	for _, item := range otherItems {
		_, err := suite.db.DB.Exec(`
			INSERT INTO pantry_items (user_id, name, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
			ON CONFLICT (user_id, name) DO NOTHING
		`, suite.otherUserID, item)
		require.NoError(suite.T(), err)
	}
}

// TestFuzzySearchAuthorization tests authorization checks
func (suite *PantrySearchTestSuite) TestFuzzySearchAuthorization() {
	tests := []struct {
		name           string
		userID         int
		token          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid user accessing own pantry",
			userID:         suite.testUserID,
			token:          suite.testUserToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User trying to access another user's pantry",
			userID:         suite.otherUserID,
			token:          suite.testUserToken,
			expectedStatus: http.StatusForbidden,
			expectedError:  "access denied",
		},
		{
			name:           "Missing authentication",
			userID:         suite.testUserID,
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authorization header required",
		},
		{
			name:           "Invalid token",
			userID:         suite.testUserID,
			token:          "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid authorization header format",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/pantry/fuzzy-search?q=tomato&user_id=%d", tt.userID), nil)
			if tt.token != "" {
				if tt.token == "invalid-token" {
					req.Header.Set("Authorization", "Bearer invalid-token")
				} else {
					req.Header.Set("Authorization", "Bearer "+tt.token)
				}
			}

			w := httptest.NewRecorder()
			suite.authRouter.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(suite.T(), err)
				assert.Contains(suite.T(), response["error"], tt.expectedError)
			}
		})
	}
}

// TestFuzzySearchInputValidation tests input validation
func (suite *PantrySearchTestSuite) TestFuzzySearchInputValidation() {
	tests := []struct {
		name           string
		query          string
		threshold      string
		limit          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid query",
			query:          "tomato",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing query",
			query:          "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "query parameter 'q' is required",
		},
		{
			name:           "Query too short",
			query:          "a",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "query must be at least 2 characters",
		},
		{
			name:           "Query too long",
			query:          string(make([]byte, 101)),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "query must not exceed 100 characters",
		},
		{
			name:           "Invalid threshold (too low)",
			query:          "tomato",
			threshold:      "0.2",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "threshold must be between 0.3 and 1.0",
		},
		{
			name:           "Invalid threshold (too high)",
			query:          "tomato",
			threshold:      "1.5",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "threshold must be between 0.3 and 1.0",
		},
		{
			name:           "Invalid limit (too low)",
			query:          "tomato",
			limit:          "0",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "limit must be between 1 and 20",
		},
		{
			name:           "Invalid limit (too high)",
			query:          "tomato",
			limit:          "50",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "limit must be between 1 and 20",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			url := fmt.Sprintf("/api/v1/pantry/fuzzy-search?q=%s&user_id=%d", tt.query, suite.testUserID)
			if tt.threshold != "" {
				url += "&threshold=" + tt.threshold
			}
			if tt.limit != "" {
				url += "&limit=" + tt.limit
			}

			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+suite.testUserToken)

			w := httptest.NewRecorder()
			suite.authRouter.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(suite.T(), err)
				assert.Contains(suite.T(), response["error"], tt.expectedError)
			}
		})
	}
}

// TestSQLInjectionPrevention tests SQL injection prevention
func (suite *PantrySearchTestSuite) TestSQLInjectionPrevention() {
	maliciousQueries := []string{
		"'; DROP TABLE pantry_items; --",
		"1' OR '1'='1",
		"'; SELECT * FROM users; --",
		`tomato"; DELETE FROM pantry_items WHERE "1"="1`,
		"tomato\\'; DROP TABLE users; --",
		"tomato' UNION SELECT * FROM users --",
	}

	for _, malicious := range maliciousQueries {
		suite.Run(fmt.Sprintf("SQL Injection: %s", malicious), func() {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/pantry/fuzzy-search?q=%s&user_id=%d", malicious, suite.testUserID), nil)
			req.Header.Set("Authorization", "Bearer "+suite.testUserToken)

			w := httptest.NewRecorder()
			suite.authRouter.ServeHTTP(w, req)

			// Should return 400 Bad Request for dangerous input
			assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			assert.Contains(suite.T(), response["error"], "invalid characters or SQL keywords")

			// Verify database is still intact
			var count int
			err = suite.db.DB.QueryRow("SELECT COUNT(*) FROM pantry_items").Scan(&count)
			require.NoError(suite.T(), err)
			assert.Greater(suite.T(), count, 0, "pantry_items table should still exist and have data")
		})
	}
}

// TestFuzzySearchResults tests actual fuzzy search functionality
func (suite *PantrySearchTestSuite) TestFuzzySearchResults() {
	tests := []struct {
		name            string
		query           string
		threshold       string
		expectedMatches []string
	}{
		{
			name:            "Exact match",
			query:           "tomato",
			threshold:       "1.0",
			expectedMatches: []string{"tomato"},
		},
		{
			name:            "Fuzzy match with default threshold",
			query:           "tomate",
			threshold:       "",
			expectedMatches: []string{"tomato", "tomatoes"},
		},
		{
			name:            "No match with high threshold",
			query:           "carrot",
			threshold:       "0.9",
			expectedMatches: []string{},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			url := fmt.Sprintf("/api/v1/pantry/fuzzy-search?q=%s&user_id=%d", tt.query, suite.testUserID)
			if tt.threshold != "" {
				url += "&threshold=" + tt.threshold
			}

			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+suite.testUserToken)

			w := httptest.NewRecorder()
			suite.authRouter.ServeHTTP(w, req)

			assert.Equal(suite.T(), http.StatusOK, w.Code)

			var response struct {
				Data []models.PantryItemWithSimilarity `json:"data"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			// Check that we got the expected number of matches
			assert.Len(suite.T(), response.Data, len(tt.expectedMatches))

			// Check that all expected items are in the results
			resultNames := make(map[string]bool)
			for _, item := range response.Data {
				resultNames[item.Name] = true
			}
			for _, expected := range tt.expectedMatches {
				assert.True(suite.T(), resultNames[expected], "Expected %s in results", expected)
			}
		})
	}
}

// TestSearchPantryItemsAuthorization tests authorization for regular search
func (suite *PantrySearchTestSuite) TestSearchPantryItemsAuthorization() {
	// Test that users can only search their own pantry items
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/pantry/search?q=&user_id=%d", suite.otherUserID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.testUserToken)

	w := httptest.NewRecorder()
	suite.authRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], "access denied")
}

// TestBatchResolveAuthorization tests authorization for batch resolve
func (suite *PantrySearchTestSuite) TestBatchResolveAuthorization() {
	recipeHandler := handlers.NewRecipeHandler(suite.db, nil)

	// Create a test recipe
	var recipeID int
	err := suite.db.DB.QueryRow(`
		INSERT INTO recipes (title, user_id, status, created_at, updated_at)
		VALUES ('Test Recipe', $1, 'draft', NOW(), NOW())
		RETURNING id
	`, suite.testUserID).Scan(&recipeID)
	require.NoError(suite.T(), err)

	// Add router with batch endpoint
	batchRouter := gin.New()
	protected := batchRouter.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(suite.authConfig))
	{
		protected.POST("/pantry/batch-resolve", recipeHandler.BatchResolvePantryItems)
	}

	// Test unauthorized access
	payload := models.BatchResolveRequest{
		RecipeID: recipeID,
		UserID:   suite.otherUserID, // Trying to access another user's data
		Ingredients: []models.BatchResolveItem{
			{IngredientID: 1, OriginalText: "tomato"},
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/pantry/batch-resolve", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.testUserToken)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	batchRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

// TearDownSuite runs after all tests
func (suite *PantrySearchTestSuite) TearDownSuite() {
	// Clean up test data
	suite.db.DB.Exec("DELETE FROM pantry_items WHERE user_id IN ($1, $2)", suite.testUserID, suite.otherUserID)
	suite.db.DB.Exec("DELETE FROM users WHERE id IN ($1, $2)", suite.testUserID, suite.otherUserID)

	// Close database connection
	if suite.db != nil {
		suite.db.Close()
	}
}

// TestPantrySearchSuite runs the test suite
func TestPantrySearchSuite(t *testing.T) {
	suite.Run(t, new(PantrySearchTestSuite))
}