package tests

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"digital-recipes/api-service/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	_ "github.com/lib/pq"
)

// Database table constants for maintainable references
const (
	TableUsers                  = "users"
	TableRecipes                = "recipes"
	TableCanonicalIngredients   = "canonical_ingredients"
	TableRecipeIngredients      = "recipe_ingredients"
)

// Test constants
const (
	NonExistentID = 999999
	TestTimeout   = 30 * time.Second
)

// DatabaseIntegrationTestSuite contains our database integration tests
type DatabaseIntegrationTestSuite struct {
	suite.Suite
	db *db.Database
}

// SetupSuite runs before all tests in the suite
func (suite *DatabaseIntegrationTestSuite) SetupSuite() {
	// Set up test database connection
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		suite.T().Skip("TEST_DATABASE_URL not set, skipping database integration tests")
	}

	// Initialize database connection
	database, err := db.NewConnection()
	require.NoError(suite.T(), err, "Failed to connect to test database")
	
	suite.db = database
	
	// Run migrations to set up schema
	err = suite.db.RunMigrations("../db/migrations")
	require.NoError(suite.T(), err, "Failed to run migrations on test database")
}

// TearDownSuite runs after all tests in the suite
func (suite *DatabaseIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.cleanupTestData()
		suite.db.Close()
	}
}

// SetupTest runs before each individual test
func (suite *DatabaseIntegrationTestSuite) SetupTest() {
	// Clean up any existing test data before each test
	suite.cleanupTestData()
}

// cleanupTestData removes all test data from tables using efficient TRUNCATE
func (suite *DatabaseIntegrationTestSuite) cleanupTestData() {
	tx, err := suite.db.DB.Begin()
	if err != nil {
		log.Printf("Warning: Failed to start cleanup transaction: %v", err)
		return
	}
	defer tx.Rollback()
	
	// Use TRUNCATE for better performance and automatic CASCADE
	// Order matters for foreign key constraints
	tables := []string{TableRecipeIngredients, TableRecipes, TableCanonicalIngredients, TableUsers}
	
	for _, table := range tables {
		_, err := tx.Exec(fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", table))
		if err != nil {
			log.Printf("Warning: Failed to truncate table %s: %v", table, err)
			// Fall back to DELETE if TRUNCATE fails
			_, fallbackErr := tx.Exec(fmt.Sprintf("DELETE FROM %s", table))
			if fallbackErr != nil {
				log.Printf("Error: Failed to delete from table %s: %v", table, fallbackErr)
				return
			}
		}
	}
	
	if err := tx.Commit(); err != nil {
		log.Printf("Warning: Failed to commit cleanup transaction: %v", err)
	}
}

// TestDatabaseConnection tests basic database connectivity
func (suite *DatabaseIntegrationTestSuite) TestDatabaseConnection() {
	err := suite.db.HealthCheck()
	assert.NoError(suite.T(), err, "Database health check should pass")
}

// TestSchemaValidation tests that all expected tables and constraints exist
func (suite *DatabaseIntegrationTestSuite) TestSchemaValidation() {
	// Test that all expected tables exist
	expectedTables := []string{"users", "recipes", "canonical_ingredients", "recipe_ingredients"}
	
	for _, tableName := range expectedTables {
		var exists bool
		err := suite.db.DB.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_name = $1
			)
		`, tableName).Scan(&exists)
		
		require.NoError(suite.T(), err, "Failed to check if table %s exists", tableName)
		assert.True(suite.T(), exists, "Table %s should exist", tableName)
	}
	
	// Test that foreign key constraints exist
	expectedConstraints := []struct {
		table      string
		constraint string
	}{
		{"recipes", "recipes_user_id_fkey"},
		{"recipe_ingredients", "recipe_ingredients_recipe_id_fkey"},
		{"recipe_ingredients", "recipe_ingredients_canonical_ingredient_id_fkey"},
	}
	
	for _, constraint := range expectedConstraints {
		var exists bool
		err := suite.db.DB.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.table_constraints 
				WHERE table_name = $1 AND constraint_name = $2
			)
		`, constraint.table, constraint.constraint).Scan(&exists)
		
		require.NoError(suite.T(), err, "Failed to check constraint %s", constraint.constraint)
		assert.True(suite.T(), exists, "Constraint %s should exist on table %s", constraint.constraint, constraint.table)
	}
}

// TestUsersCRUD tests CRUD operations for users table
func (suite *DatabaseIntegrationTestSuite) TestUsersCRUD() {
	// Test INSERT
	var userID int
	err := suite.db.DB.QueryRow(`
		INSERT INTO users (email, name) 
		VALUES ($1, $2) 
		RETURNING id
	`, "test@example.com", "Test User").Scan(&userID)
	
	require.NoError(suite.T(), err, "Failed to insert user")
	assert.Greater(suite.T(), userID, 0, "User ID should be positive")
	
	// Test SELECT
	var email, name string
	var createdAt, updatedAt time.Time
	err = suite.db.DB.QueryRow(`
		SELECT email, name, created_at, updated_at 
		FROM users WHERE id = $1
	`, userID).Scan(&email, &name, &createdAt, &updatedAt)
	
	require.NoError(suite.T(), err, "Failed to select user")
	assert.Equal(suite.T(), "test@example.com", email)
	assert.Equal(suite.T(), "Test User", name)
	assert.False(suite.T(), createdAt.IsZero(), "Created_at should be set")
	assert.False(suite.T(), updatedAt.IsZero(), "Updated_at should be set")
	
	// Test UPDATE
	// Store initial timestamp for comparison
	initialUpdatedAt := updatedAt
	time.Sleep(2 * time.Millisecond) // Minimal sleep to ensure timestamp difference
	_, err = suite.db.DB.Exec(`
		UPDATE users SET name = $1 WHERE id = $2
	`, "Updated User", userID)
	
	require.NoError(suite.T(), err, "Failed to update user")
	
	// Verify update and timestamp trigger
	var updatedName string
	var newUpdatedAt time.Time
	err = suite.db.DB.QueryRow(`
		SELECT name, updated_at FROM users WHERE id = $1
	`, userID).Scan(&updatedName, &newUpdatedAt)
	
	require.NoError(suite.T(), err, "Failed to select updated user")
	assert.Equal(suite.T(), "Updated User", updatedName)
	assert.True(suite.T(), newUpdatedAt.After(initialUpdatedAt), 
		"Updated_at should be newer than %v, got %v", initialUpdatedAt, newUpdatedAt)
	
	// Test DELETE
	result, err := suite.db.DB.Exec(`DELETE FROM users WHERE id = $1`, userID)
	require.NoError(suite.T(), err, "Failed to delete user")
	
	rowsAffected, err := result.RowsAffected()
	require.NoError(suite.T(), err, "Failed to get rows affected")
	assert.Equal(suite.T(), int64(1), rowsAffected, "Should delete exactly one user")
	
	// Verify deletion
	err = suite.db.DB.QueryRow(`SELECT id FROM users WHERE id = $1`, userID).Scan(&userID)
	assert.Equal(suite.T(), sql.ErrNoRows, err, "User should not exist after deletion")
}

// TestRecipesCRUD tests CRUD operations for recipes table
func (suite *DatabaseIntegrationTestSuite) TestRecipesCRUD() {
	// First create a user for foreign key constraint
	var userID int
	err := suite.db.DB.QueryRow(`
		INSERT INTO users (email, name) 
		VALUES ($1, $2) 
		RETURNING id
	`, "chef@example.com", "Chef User").Scan(&userID)
	require.NoError(suite.T(), err, "Failed to create test user")
	
	// Test INSERT
	var recipeID int
	err = suite.db.DB.QueryRow(`
		INSERT INTO recipes (title, servings, instructions, tips, user_id) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id
	`, "Test Recipe", "4", "Mix ingredients", "Use fresh ingredients", userID).Scan(&recipeID)
	
	require.NoError(suite.T(), err, "Failed to insert recipe")
	assert.Greater(suite.T(), recipeID, 0, "Recipe ID should be positive")
	
	// Test SELECT with default status
	var title, servings, instructions, tips, status string
	var recipeUserID int
	err = suite.db.DB.QueryRow(`
		SELECT title, servings, instructions, tips, status, user_id 
		FROM recipes WHERE id = $1
	`, recipeID).Scan(&title, &servings, &instructions, &tips, &status, &recipeUserID)
	
	require.NoError(suite.T(), err, "Failed to select recipe")
	assert.Equal(suite.T(), "Test Recipe", title)
	assert.Equal(suite.T(), "4", servings)
	assert.Equal(suite.T(), "Mix ingredients", instructions)
	assert.Equal(suite.T(), "Use fresh ingredients", tips)
	assert.Equal(suite.T(), "processing", status) // Default status
	assert.Equal(suite.T(), userID, recipeUserID)
	
	// Test UPDATE with status change
	_, err = suite.db.DB.Exec(`
		UPDATE recipes SET status = $1, title = $2 WHERE id = $3
	`, "published", "Updated Recipe", recipeID)
	
	require.NoError(suite.T(), err, "Failed to update recipe")
	
	// Verify update
	err = suite.db.DB.QueryRow(`
		SELECT title, status FROM recipes WHERE id = $1
	`, recipeID).Scan(&title, &status)
	
	require.NoError(suite.T(), err, "Failed to select updated recipe")
	assert.Equal(suite.T(), "Updated Recipe", title)
	assert.Equal(suite.T(), "published", status)
	
	// Test DELETE (should cascade delete related recipe_ingredients)
	result, err := suite.db.DB.Exec(`DELETE FROM recipes WHERE id = $1`, recipeID)
	require.NoError(suite.T(), err, "Failed to delete recipe")
	
	rowsAffected, err := result.RowsAffected()
	require.NoError(suite.T(), err, "Failed to get rows affected")
	assert.Equal(suite.T(), int64(1), rowsAffected, "Should delete exactly one recipe")
}

// TestCanonicalIngredientsCRUD tests CRUD operations for canonical_ingredients table
func (suite *DatabaseIntegrationTestSuite) TestCanonicalIngredientsCRUD() {
	// Test INSERT
	var ingredientID int
	err := suite.db.DB.QueryRow(`
		INSERT INTO canonical_ingredients (name, is_approved) 
		VALUES ($1, $2) 
		RETURNING id
	`, "eggs", true).Scan(&ingredientID)
	
	require.NoError(suite.T(), err, "Failed to insert canonical ingredient")
	assert.Greater(suite.T(), ingredientID, 0, "Ingredient ID should be positive")
	
	// Test SELECT
	var name string
	var isApproved bool
	err = suite.db.DB.QueryRow(`
		SELECT name, is_approved FROM canonical_ingredients WHERE id = $1
	`, ingredientID).Scan(&name, &isApproved)
	
	require.NoError(suite.T(), err, "Failed to select canonical ingredient")
	assert.Equal(suite.T(), "eggs", name)
	assert.True(suite.T(), isApproved)
	
	// Test UPDATE
	_, err = suite.db.DB.Exec(`
		UPDATE canonical_ingredients SET is_approved = $1 WHERE id = $2
	`, false, ingredientID)
	
	require.NoError(suite.T(), err, "Failed to update canonical ingredient")
	
	// Verify update
	err = suite.db.DB.QueryRow(`
		SELECT is_approved FROM canonical_ingredients WHERE id = $1
	`, ingredientID).Scan(&isApproved)
	
	require.NoError(suite.T(), err, "Failed to select updated canonical ingredient")
	assert.False(suite.T(), isApproved)
	
	// Test unique constraint
	_, err = suite.db.DB.Exec(`
		INSERT INTO canonical_ingredients (name) VALUES ($1)
	`, "eggs")
	
	assert.Error(suite.T(), err, "Should fail to insert duplicate ingredient name")
	assert.Contains(suite.T(), strings.ToLower(err.Error()), "unique", 
		"Should be a unique constraint violation")
	
	// Test DELETE
	result, err := suite.db.DB.Exec(`DELETE FROM canonical_ingredients WHERE id = $1`, ingredientID)
	require.NoError(suite.T(), err, "Failed to delete canonical ingredient")
	
	rowsAffected, err := result.RowsAffected()
	require.NoError(suite.T(), err, "Failed to get rows affected")
	assert.Equal(suite.T(), int64(1), rowsAffected, "Should delete exactly one ingredient")
}

// TestRecipeIngredientsCRUD tests CRUD operations for recipe_ingredients table
func (suite *DatabaseIntegrationTestSuite) TestRecipeIngredientsCRUD() {
	// Set up test data: user, recipe, and canonical ingredient
	var userID int
	err := suite.db.DB.QueryRow(`
		INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id
	`, "test@example.com", "Test User").Scan(&userID)
	require.NoError(suite.T(), err, "Failed to create test user")
	
	var recipeID int
	err = suite.db.DB.QueryRow(`
		INSERT INTO recipes (title, user_id) VALUES ($1, $2) RETURNING id
	`, "Test Recipe", userID).Scan(&recipeID)
	require.NoError(suite.T(), err, "Failed to create test recipe")
	
	var ingredientID int
	err = suite.db.DB.QueryRow(`
		INSERT INTO canonical_ingredients (name) VALUES ($1) RETURNING id
	`, "flour").Scan(&ingredientID)
	require.NoError(suite.T(), err, "Failed to create test ingredient")
	
	// Test INSERT
	var recipeIngredientID int
	err = suite.db.DB.QueryRow(`
		INSERT INTO recipe_ingredients (recipe_id, canonical_ingredient_id, original_text, quantity, unit) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id
	`, recipeID, ingredientID, "2 cups all-purpose flour", 2.0, "cups").Scan(&recipeIngredientID)
	
	require.NoError(suite.T(), err, "Failed to insert recipe ingredient")
	assert.Greater(suite.T(), recipeIngredientID, 0, "Recipe ingredient ID should be positive")
	
	// Test SELECT
	var originalText, unit string
	var quantity float64
	var linkedRecipeID, linkedIngredientID int
	err = suite.db.DB.QueryRow(`
		SELECT recipe_id, canonical_ingredient_id, original_text, quantity, unit 
		FROM recipe_ingredients WHERE id = $1
	`, recipeIngredientID).Scan(&linkedRecipeID, &linkedIngredientID, &originalText, &quantity, &unit)
	
	require.NoError(suite.T(), err, "Failed to select recipe ingredient")
	assert.Equal(suite.T(), recipeID, linkedRecipeID)
	assert.Equal(suite.T(), ingredientID, linkedIngredientID)
	assert.Equal(suite.T(), "2 cups all-purpose flour", originalText)
	assert.Equal(suite.T(), 2.0, quantity)
	assert.Equal(suite.T(), "cups", unit)
	
	// Test UPDATE
	_, err = suite.db.DB.Exec(`
		UPDATE recipe_ingredients SET quantity = $1, unit = $2 WHERE id = $3
	`, 2.5, "cups", recipeIngredientID)
	
	require.NoError(suite.T(), err, "Failed to update recipe ingredient")
	
	// Verify update
	err = suite.db.DB.QueryRow(`
		SELECT quantity, unit FROM recipe_ingredients WHERE id = $1
	`, recipeIngredientID).Scan(&quantity, &unit)
	
	require.NoError(suite.T(), err, "Failed to select updated recipe ingredient")
	assert.Equal(suite.T(), 2.5, quantity)
	assert.Equal(suite.T(), "cups", unit)
	
	// Test CASCADE DELETE when recipe is deleted
	_, err = suite.db.DB.Exec(`DELETE FROM recipes WHERE id = $1`, recipeID)
	require.NoError(suite.T(), err, "Failed to delete recipe")
	
	// Verify recipe ingredient was cascade deleted
	err = suite.db.DB.QueryRow(`SELECT id FROM recipe_ingredients WHERE id = $1`, recipeIngredientID).Scan(&recipeIngredientID)
	assert.Equal(suite.T(), sql.ErrNoRows, err, "Recipe ingredient should be cascade deleted")
}

// TestForeignKeyConstraints tests foreign key constraint enforcement
func (suite *DatabaseIntegrationTestSuite) TestForeignKeyConstraints() {
	// Test recipe foreign key constraint (should fail with non-existent user)
	_, err := suite.db.DB.Exec(`
		INSERT INTO recipes (title, user_id) VALUES ($1, $2)
	`, "Test Recipe", NonExistentID) // Non-existent user ID
	
	assert.Error(suite.T(), err, "Should fail to insert recipe with non-existent user_id")
	assert.Contains(suite.T(), err.Error(), "violates foreign key constraint", 
		"Should be a foreign key constraint violation")
	
	// Test recipe_ingredients foreign key constraints
	_, err = suite.db.DB.Exec(`
		INSERT INTO recipe_ingredients (recipe_id, original_text) VALUES ($1, $2)
	`, NonExistentID, "test ingredient") // Non-existent recipe ID
	
	assert.Error(suite.T(), err, "Should fail to insert recipe ingredient with non-existent recipe_id")
	assert.Contains(suite.T(), err.Error(), "violates foreign key constraint", 
		"Should be a foreign key constraint violation")
}

// TestTimestampTriggers tests that timestamp triggers work correctly
func (suite *DatabaseIntegrationTestSuite) TestTimestampTriggers() {
	// Create test user
	var userID int
	err := suite.db.DB.QueryRow(`
		INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id
	`, "timestamp@example.com", "Timestamp User").Scan(&userID)
	require.NoError(suite.T(), err, "Failed to create test user")
	
	// Get initial timestamps
	var initialCreatedAt, initialUpdatedAt time.Time
	err = suite.db.DB.QueryRow(`
		SELECT created_at, updated_at FROM users WHERE id = $1
	`, userID).Scan(&initialCreatedAt, &initialUpdatedAt)
	require.NoError(suite.T(), err, "Failed to get initial timestamps")
	
	// Store initial timestamps for comparison
	time.Sleep(2 * time.Millisecond) // Minimal sleep to ensure timestamp difference
	
	// Update the user
	_, err = suite.db.DB.Exec(`
		UPDATE users SET name = $1 WHERE id = $2
	`, "Updated Timestamp User", userID)
	require.NoError(suite.T(), err, "Failed to update user")
	
	// Check that updated_at changed but created_at didn't
	var newCreatedAt, newUpdatedAt time.Time
	err = suite.db.DB.QueryRow(`
		SELECT created_at, updated_at FROM users WHERE id = $1
	`, userID).Scan(&newCreatedAt, &newUpdatedAt)
	require.NoError(suite.T(), err, "Failed to get new timestamps")
	
	assert.Equal(suite.T(), initialCreatedAt, newCreatedAt, "Created_at should not change on update")
	assert.True(suite.T(), newUpdatedAt.After(initialUpdatedAt), 
		"Updated_at should be newer than %v, got %v", initialUpdatedAt, newUpdatedAt)
	// Ensure the difference is reasonable (within a few seconds)
	assert.WithinDuration(suite.T(), newUpdatedAt, time.Now(), 5*time.Second, 
		"Updated timestamp should be recent")
}

// TestIndexesExist tests that expected indexes exist for performance
func (suite *DatabaseIntegrationTestSuite) TestIndexesExist() {
	expectedIndexes := []string{
		"idx_recipes_user_id",
		"idx_recipes_status", 
		"idx_recipe_ingredients_recipe_id",
		"idx_recipe_ingredients_canonical_id",
		"idx_canonical_ingredients_name",
	}
	
	for _, indexName := range expectedIndexes {
		var exists bool
		err := suite.db.DB.QueryRow(`
			SELECT EXISTS (
				SELECT FROM pg_indexes 
				WHERE indexname = $1
			)
		`, indexName).Scan(&exists)
		
		require.NoError(suite.T(), err, "Failed to check if index %s exists", indexName)
		assert.True(suite.T(), exists, "Index %s should exist", indexName)
	}
}

// Run the test suite
func TestDatabaseIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseIntegrationTestSuite))
}