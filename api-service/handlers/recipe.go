package handlers

import (
	"database/sql"
	"fmt"
	"log"
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

// Constants for pagination limits
const (
	maxPerPage     = 100
	defaultPerPage = 10
	maxPage        = 10000 // Prevent excessive offset calculations
)

// GetRecipes handles GET /recipes requests
func (h *RecipeHandler) GetRecipes(c *gin.Context) {
	// Parse query parameters
	status := c.Query("status")
	pageStr := c.DefaultQuery("page", "1")
	perPageStr := c.DefaultQuery("per_page", strconv.Itoa(defaultPerPage))
	
	// Log request parameters
	log.Printf("GetRecipes request - status: %s, page: %s, per_page: %s, IP: %s", 
		status, pageStr, perPageStr, c.ClientIP())

	// Validate pagination parameters with proper bounds
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 || page > maxPage {
		BadRequestError(c, fmt.Sprintf("invalid page parameter. Must be between 1 and %d", maxPage))
		return
	}
	
	perPage, err := strconv.Atoi(perPageStr)
	if err != nil || perPage < 1 || perPage > maxPerPage {
		BadRequestError(c, fmt.Sprintf("invalid per_page parameter. Must be between 1 and %d", maxPerPage))
		return
	}

	// Convert to limit and offset
	limit := perPage
	offset := (page - 1) * perPage
	
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
			BadRequestError(c, fmt.Sprintf("invalid status: %s. Valid statuses are: %s", 
				status, strings.Join(validStatuses, ", ")))
			return
		}
	}

	// Build secure query using query builder
	queryBuilder := NewRecipesQueryBuilder()
	
	// Add status filter if provided
	if status != "" {
		queryBuilder.WithStatus(status)
	}
	
	// Add pagination
	queryBuilder.WithPagination(limit, offset)
	
	// Build final query
	query, args := queryBuilder.Build()

	// Execute single query for both data and count
	rows, err := h.db.DB.Query(query, args...)
	if err != nil {
		log.Printf("GetRecipes query error: %v", err)
		InternalServerError(c, "failed to retrieve recipes")
		return
	}
	defer rows.Close()

	// Parse results and get total count from first row
	var recipes []models.Recipe
	var total int
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
			&total, // Total count from window function
		)
		if err != nil {
			log.Printf("GetRecipes scan error: %v", err)
			InternalServerError(c, "failed to parse recipe data")
			return
		}
		recipes = append(recipes, recipe)
	}

	if err = rows.Err(); err != nil {
		log.Printf("GetRecipes rows error: %v", err)
		InternalServerError(c, "error reading recipe data")
		return
	}

	// Return empty array if no recipes found
	if recipes == nil {
		recipes = []models.Recipe{}
		total = 0
	}

	// Calculate pagination metadata
	totalPages := (total + perPage - 1) / perPage

	// Create pagination metadata
	pagination := &Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	// Return standardized paginated response
	SuccessResponseWithPagination(c, recipes, pagination)
}

// GetRecipe handles GET /recipes/:id requests
func (h *RecipeHandler) GetRecipe(c *gin.Context) {
	// Parse recipe ID from URL parameter
	idStr := c.Param("id")
	recipeID, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("GetRecipe invalid ID - id: %s, IP: %s", idStr, c.ClientIP())
		BadRequestError(c, "invalid recipe ID")
		return
	}
	
	// Log request
	log.Printf("GetRecipe request - id: %d, IP: %s", recipeID, c.ClientIP())

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
			NotFoundError(c, "recipe not found")
			return
		}
		log.Printf("GetRecipe query error: %v", err)
		InternalServerError(c, "failed to retrieve recipe")
		return
	}

	// Query for ingredients
	ingredientsQuery := `
		SELECT 
			ri.id,
			ri.recipe_id,
			ri.canonical_ingredient_id,
			ri.original_text,
			ri.quantity,
			ri.unit,
			ri.created_at,
			ri.updated_at,
			ci.name as canonical_name
		FROM recipe_ingredients ri
		LEFT JOIN canonical_ingredients ci ON ri.canonical_ingredient_id = ci.id
		WHERE ri.recipe_id = $1
		ORDER BY ri.id
	`

	ingredientRows, err := h.db.DB.Query(ingredientsQuery, recipeID)
	if err != nil {
		log.Printf("GetRecipe ingredients query error: %v", err)
		InternalServerError(c, "failed to retrieve ingredients")
		return
	}
	defer ingredientRows.Close()

	var ingredients []models.RecipeIngredient
	for ingredientRows.Next() {
		var ingredient models.RecipeIngredient
		var canonicalName sql.NullString
		
		err := ingredientRows.Scan(
			&ingredient.ID,
			&ingredient.RecipeID,
			&ingredient.CanonicalIngredientID,
			&ingredient.OriginalText,
			&ingredient.Quantity,
			&ingredient.Unit,
			&ingredient.CreatedAt,
			&ingredient.UpdatedAt,
			&canonicalName,
		)
		if err != nil {
			log.Printf("GetRecipe ingredient scan error: %v", err)
			InternalServerError(c, "failed to parse ingredient data")
			return
		}
		
		// Set canonical name if available
		if canonicalName.Valid {
			ingredient.CanonicalName = &canonicalName.String
		}
		
		ingredients = append(ingredients, ingredient)
	}

	// Create response with ingredients using RecipeWithIngredients model
	recipeWithIngredients := models.RecipeWithIngredients{
		Recipe:      recipe,
		Ingredients: ingredients,
	}

	// Return standardized response
	SuccessResponse(c, recipeWithIngredients)
}