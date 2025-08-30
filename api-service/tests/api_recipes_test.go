package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"digital-recipes/api-service/db"
	"digital-recipes/api-service/handlers"
	"digital-recipes/api-service/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// RecipeAPITestSuite contains our recipe API integration tests
type RecipeAPITestSuite struct {
	suite.Suite
	db     *db.Database
	router *gin.Engine
	testUserID int
}

// SetupSuite runs before all tests in the suite
func (suite *RecipeAPITestSuite) SetupSuite() {
	// Set up test database connection
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		suite.T().Skip("TEST_DATABASE_URL not set, skipping recipe API integration tests")
	}

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Initialize database connection
	database, err := db.NewConnection()
	require.NoError(suite.T(), err, "Failed to connect to test database")
	
	suite.db = database
	
	// Run migrations to set up schema
	err = suite.db.RunMigrations("../db/migrations")
	require.NoError(suite.T(), err, "Failed to run migrations on test database")

	// Set up the router with handlers
	suite.router = gin.New()
	// Storage service not needed for recipe GET tests
	recipeHandler := handlers.NewRecipeHandler(suite.db, nil)
	
	// Register routes
	v1 := suite.router.Group("/api/v1")
	{
		v1.GET("/recipes", recipeHandler.GetRecipes)
		v1.GET("/recipes/:id", recipeHandler.GetRecipe)
	}
}

// TearDownSuite runs after all tests in the suite
func (suite *RecipeAPITestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.cleanupTestData()
		suite.db.Close()
	}
}

// SetupTest runs before each individual test
func (suite *RecipeAPITestSuite) SetupTest() {
	// Clean up any existing test data before each test
	suite.cleanupTestData()
	
	// Create a test user for all recipe tests
	err := suite.db.DB.QueryRow(`
		INSERT INTO users (email, name) 
		VALUES ($1, $2) 
		RETURNING id
	`, "testuser@example.com", "Test User").Scan(&suite.testUserID)
	require.NoError(suite.T(), err, "Failed to create test user")
}

// cleanupTestData removes all test data from tables
func (suite *RecipeAPITestSuite) cleanupTestData() {
	tx, err := suite.db.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	
	// Order matters for foreign key constraints
	tables := []string{"recipe_ingredients", "recipes", "canonical_ingredients", "users"}
	
	for _, table := range tables {
		_, err := tx.Exec(fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", table))
		if err != nil {
			// Fall back to DELETE if TRUNCATE fails
			tx.Exec(fmt.Sprintf("DELETE FROM %s", table))
		}
	}
	
	tx.Commit()
}

// createTestRecipe creates a recipe for testing
func (suite *RecipeAPITestSuite) createTestRecipe(title string, status string) int {
	var recipeID int
	err := suite.db.DB.QueryRow(`
		INSERT INTO recipes (title, servings, instructions, tips, status, user_id) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id
	`, title, "4", "Test instructions", "Test tips", status, suite.testUserID).Scan(&recipeID)
	require.NoError(suite.T(), err, "Failed to create test recipe")
	return recipeID
}

// TestGetRecipesEmpty tests GET /recipes endpoint with no recipes
func (suite *RecipeAPITestSuite) TestGetRecipesEmpty() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recipes", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response handlers.StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Failed to unmarshal response")
	
	// Convert data to recipes array
	dataBytes, _ := json.Marshal(response.Data)
	var recipes []models.Recipe
	err = json.Unmarshal(dataBytes, &recipes)
	require.NoError(suite.T(), err, "Failed to unmarshal recipes data")
	
	assert.Empty(suite.T(), recipes, "Should return empty array when no recipes exist")
	assert.Equal(suite.T(), 0, response.Pagination.Total, "Total should be 0")
}

// TestGetRecipesSingle tests GET /recipes endpoint with one recipe
func (suite *RecipeAPITestSuite) TestGetRecipesSingle() {
	// Create a test recipe
	recipeID := suite.createTestRecipe("Test Recipe", "published")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recipes", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response handlers.StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Failed to unmarshal response")
	
	// Convert data to recipes array
	dataBytes, _ := json.Marshal(response.Data)
	var recipes []models.Recipe
	err = json.Unmarshal(dataBytes, &recipes)
	require.NoError(suite.T(), err, "Failed to unmarshal recipes data")
	
	require.Len(suite.T(), recipes, 1, "Should return exactly one recipe")
	assert.Equal(suite.T(), 1, response.Pagination.Total, "Total should be 1")
	
	recipe := recipes[0]
	assert.Equal(suite.T(), recipeID, recipe.ID)
	assert.Equal(suite.T(), "Test Recipe", recipe.Title)
	assert.Equal(suite.T(), "4", *recipe.Servings)
	assert.Equal(suite.T(), "Test instructions", *recipe.Instructions)
	assert.Equal(suite.T(), "Test tips", *recipe.Tips)
	assert.Equal(suite.T(), "published", recipe.Status)
	assert.Equal(suite.T(), suite.testUserID, recipe.UserID)
	assert.False(suite.T(), recipe.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(suite.T(), recipe.UpdatedAt.IsZero(), "UpdatedAt should be set")
}

// TestGetRecipesMultiple tests GET /recipes endpoint with multiple recipes
func (suite *RecipeAPITestSuite) TestGetRecipesMultiple() {
	// Create multiple test recipes
	recipe1ID := suite.createTestRecipe("First Recipe", "published")
	recipe2ID := suite.createTestRecipe("Second Recipe", "review_required") 
	recipe3ID := suite.createTestRecipe("Third Recipe", "processing")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recipes", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response handlers.StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Failed to unmarshal response")
	
	// Convert data to recipes array
	dataBytes, _ := json.Marshal(response.Data)
	var recipes []models.Recipe
	err = json.Unmarshal(dataBytes, &recipes)
	require.NoError(suite.T(), err, "Failed to unmarshal recipes data")
	
	require.Len(suite.T(), recipes, 3, "Should return all three recipes")
	assert.Equal(suite.T(), 3, response.Pagination.Total, "Total should be 3")
	
	// Recipes should be ordered by created_at DESC (newest first)
	expectedIDs := []int{recipe3ID, recipe2ID, recipe1ID}
	for i, recipe := range recipes {
		assert.Equal(suite.T(), expectedIDs[i], recipe.ID, "Recipes should be ordered by creation time (newest first)")
	}
}

// TestGetRecipesFiltering tests GET /recipes endpoint with status filtering
func (suite *RecipeAPITestSuite) TestGetRecipesFiltering() {
	// Create recipes with different statuses
	suite.createTestRecipe("Published Recipe", "published")
	suite.createTestRecipe("Review Recipe", "review_required") 
	suite.createTestRecipe("Processing Recipe", "processing")

	// Test filtering by published status
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recipes?status=published", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response handlers.StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Failed to unmarshal response")
	
	// Convert data to recipes array
	dataBytes, _ := json.Marshal(response.Data)
	var recipes []models.Recipe
	err = json.Unmarshal(dataBytes, &recipes)
	require.NoError(suite.T(), err, "Failed to unmarshal recipes data")
	
	require.Len(suite.T(), recipes, 1, "Should return only published recipes")
	assert.Equal(suite.T(), 1, response.Pagination.Total, "Total should be 1")
	assert.Equal(suite.T(), "published", recipes[0].Status)
	assert.Equal(suite.T(), "Published Recipe", recipes[0].Title)
}

// TestGetRecipesInvalidStatus tests GET /recipes endpoint with invalid status filter
func (suite *RecipeAPITestSuite) TestGetRecipesInvalidStatus() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recipes?status=invalid_status", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Failed to unmarshal error response")
	
	assert.Contains(suite.T(), response["error"], "invalid status")
}

// TestGetRecipeByID tests GET /recipes/:id endpoint with valid ID
func (suite *RecipeAPITestSuite) TestGetRecipeByID() {
	// Create a test recipe
	recipeID := suite.createTestRecipe("Detailed Recipe", "published")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recipes/"+strconv.Itoa(recipeID), nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response handlers.StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Failed to unmarshal response")
	
	// Convert data to RecipeWithIngredients
	dataBytes, _ := json.Marshal(response.Data)
	var recipe models.RecipeWithIngredients
	err = json.Unmarshal(dataBytes, &recipe)
	require.NoError(suite.T(), err, "Failed to unmarshal recipe data")
	
	assert.Equal(suite.T(), recipeID, recipe.ID)
	assert.Equal(suite.T(), "Detailed Recipe", recipe.Title)
	assert.Equal(suite.T(), "published", recipe.Status)
	assert.Equal(suite.T(), suite.testUserID, recipe.UserID)
	assert.NotNil(suite.T(), recipe.Ingredients, "Ingredients should be included")
}

// TestGetRecipeByIDNotFound tests GET /recipes/:id endpoint with non-existent ID
func (suite *RecipeAPITestSuite) TestGetRecipeByIDNotFound() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recipes/999999", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Failed to unmarshal error response")
	
	assert.Contains(suite.T(), response["error"], "not found")
}

// TestGetRecipeByIDInvalidID tests GET /recipes/:id endpoint with invalid ID format
func (suite *RecipeAPITestSuite) TestGetRecipeByIDInvalidID() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recipes/invalid", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Failed to unmarshal error response")
	
	assert.Contains(suite.T(), response["error"], "invalid")
}

// TestGetRecipesPagination tests GET /recipes endpoint with pagination parameters
func (suite *RecipeAPITestSuite) TestGetRecipesPagination() {
	// Create 5 test recipes
	for i := 1; i <= 5; i++ {
		suite.createTestRecipe(fmt.Sprintf("Recipe %d", i), "published")
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// Test first page with per_page 2
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recipes?page=1&per_page=2", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response handlers.StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Failed to unmarshal response")
	
	// Check response structure
	assert.NotNil(suite.T(), response.Data, "Response should have data")
	assert.NotNil(suite.T(), response.Pagination, "Response should have pagination")
	
	// Convert data to recipes array
	dataBytes, _ := json.Marshal(response.Data)
	var recipes []models.Recipe
	err = json.Unmarshal(dataBytes, &recipes)
	require.NoError(suite.T(), err, "Failed to unmarshal recipes data")
	
	assert.Len(suite.T(), recipes, 2, "Should return exactly 2 recipes on first page")
	assert.Equal(suite.T(), 1, response.Pagination.Page)
	assert.Equal(suite.T(), 2, response.Pagination.PerPage)
	assert.Equal(suite.T(), 5, response.Pagination.Total)
	assert.Equal(suite.T(), 3, response.Pagination.TotalPages)
	
	// Test second page
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/recipes?page=2&per_page=2", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Failed to unmarshal response")
	
	dataBytes, _ = json.Marshal(response.Data)
	err = json.Unmarshal(dataBytes, &recipes)
	require.NoError(suite.T(), err, "Failed to unmarshal recipes data")
	
	assert.Len(suite.T(), recipes, 2, "Should return exactly 2 recipes on second page")
	assert.Equal(suite.T(), 2, response.Pagination.Page)
}

// TestGetRecipesInvalidPagination tests GET /recipes endpoint with invalid pagination parameters
func (suite *RecipeAPITestSuite) TestGetRecipesInvalidPagination() {
	// Test negative page
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recipes?page=-1", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	
	// Test page too large
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/recipes?page=20000", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	
	// Test per_page too large
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/recipes?per_page=200", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	
	// Test non-numeric page
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/recipes?page=abc", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// Run the test suite
func TestRecipeAPITestSuite(t *testing.T) {
	suite.Run(t, new(RecipeAPITestSuite))
}