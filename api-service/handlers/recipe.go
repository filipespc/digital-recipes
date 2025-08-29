package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"digital-recipes/api-service/db"
	"digital-recipes/api-service/models"
	"github.com/gin-gonic/gin"
)

// RecipeHandler handles recipe-related HTTP requests
type RecipeHandler struct {
	db *db.Database
}

// NewRecipeHandler creates a new recipe handler
func NewRecipeHandler(database *db.Database) *RecipeHandler {
	return &RecipeHandler{db: database}
}

// GetRecipes handles GET /recipes requests
func (h *RecipeHandler) GetRecipes(c *gin.Context) {
	// Parse query parameters
	status := c.Query("status")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	// Validate pagination parameters
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
		return
	}
	
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset parameter"})
		return
	}
	
	// Validate status parameter if provided
	if status != "" {
		validStatuses := []string{"processing", "review_required", "published"}
		isValid := false
		for _, validStatus := range validStatuses {
			if status == validStatus {
				isValid = true
				break
			}
		}
		if !isValid {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid status: %s. Valid statuses are: %s", 
					status, strings.Join(validStatuses, ", ")),
			})
			return
		}
	}

	// Build query with optional status filter
	query := `
		SELECT id, title, servings, instructions, tips, status, user_id, created_at, updated_at
		FROM recipes
	`
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		query += fmt.Sprintf(" WHERE status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := h.db.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve recipes"})
		return
	}
	defer rows.Close()

	// Parse results
	var recipes []models.Recipe
	for rows.Next() {
		var recipe models.Recipe
		err := rows.Scan(
			&recipe.ID,
			&recipe.Title,
			&recipe.Servings,
			&recipe.Instructions,
			&recipe.Tips,
			&recipe.Status,
			&recipe.UserID,
			&recipe.CreatedAt,
			&recipe.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse recipe data"})
			return
		}
		recipes = append(recipes, recipe)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error reading recipe data"})
		return
	}

	// Return empty array if no recipes found
	if recipes == nil {
		recipes = []models.Recipe{}
	}

	c.JSON(http.StatusOK, recipes)
}

// GetRecipe handles GET /recipes/:id requests
func (h *RecipeHandler) GetRecipe(c *gin.Context) {
	// Parse recipe ID from URL parameter
	idStr := c.Param("id")
	recipeID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe ID"})
		return
	}

	// Query for the specific recipe
	query := `
		SELECT id, title, servings, instructions, tips, status, user_id, created_at, updated_at
		FROM recipes
		WHERE id = $1
	`

	var recipe models.Recipe
	err = h.db.DB.QueryRow(query, recipeID).Scan(
		&recipe.ID,
		&recipe.Title,
		&recipe.Servings,
		&recipe.Instructions,
		&recipe.Tips,
		&recipe.Status,
		&recipe.UserID,
		&recipe.CreatedAt,
		&recipe.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve recipe"})
		return
	}

	c.JSON(http.StatusOK, recipe)
}