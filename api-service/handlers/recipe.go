package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"digital-recipes/api-service/db"
	"digital-recipes/api-service/middleware"
	"digital-recipes/api-service/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RecipeHandler handles recipe-related HTTP requests
type RecipeHandler struct {
	db             *db.Database
	storageService *StorageService
}

// NewRecipeHandler creates a new recipe handler
func NewRecipeHandler(database *db.Database, storageService *StorageService) *RecipeHandler {
	return &RecipeHandler{
		db:             database,
		storageService: storageService,
	}
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
	logrus.WithFields(logrus.Fields{
		"status":   status,
		"page":     pageStr,
		"per_page": perPageStr,
		"ip":       c.ClientIP(),
	}).Debug("GetRecipes request")

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
		logrus.WithError(err).Error("GetRecipes query error")
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
			logrus.WithError(err).Error("GetRecipes scan error")
			InternalServerError(c, "failed to parse recipe data")
			return
		}
		recipes = append(recipes, recipe)
	}

	if err = rows.Err(); err != nil {
		logrus.WithError(err).Error("GetRecipes rows error")
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
		logrus.WithFields(logrus.Fields{"id": idStr, "ip": c.ClientIP()}).Warn("GetRecipe invalid ID")
		BadRequestError(c, "invalid recipe ID")
		return
	}
	
	// Log request
	logrus.WithFields(logrus.Fields{"recipe_id": recipeID, "ip": c.ClientIP()}).Debug("GetRecipe request")

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
		logrus.WithError(err).Error("GetRecipe query error")
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
		logrus.WithError(err).Error("GetRecipe ingredients query error")
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
			logrus.WithError(err).Error("GetRecipe ingredient scan error")
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

// PostUploadRequest handles POST /recipes/upload-request requests with enhanced security
func (h *RecipeHandler) PostUploadRequest(c *gin.Context) {
	logger := middleware.LogWithContext(c)
	
	// Parse and validate request body
	var uploadRequest models.UploadRequest
	if err := c.ShouldBindJSON(&uploadRequest); err != nil {
		logger.WithError(err).Warn("Upload request binding failed")
		ValidationError(c, "Invalid request format. Check image_count field.")
		return
	}

	// Perform additional business logic validation
	if err := uploadRequest.Validate(); err != nil {
		logger.WithError(err).Warn("Upload request validation failed")
		ValidationError(c, err.Error())
		return
	}

	logger.WithFields(logrus.Fields{
		"image_count":       uploadRequest.ImageCount,
		"max_file_size_mb":  uploadRequest.GetMaxFileSizeMB(),
		"allowed_types":     uploadRequest.GetAllowedTypes(),
		"expiration_hours":  uploadRequest.GetExpirationHours(),
	}).Info("Processing upload request")

	// Validate storage service is available
	if h.storageService == nil {
		logger.Error("Storage service not available")
		InternalServerError(c, "File upload service is temporarily unavailable")
		return
	}

	// Get authenticated user ID (set by auth middleware)
	userID := middleware.GetUserID(c)
	if userID == 0 {
		logger.Error("No authenticated user found")
		AuthenticationError(c, "Authentication required for file uploads")
		return
	}

	// Begin transaction for recipe creation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := h.db.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.WithError(err).Error("Failed to begin database transaction")
		InternalServerError(c, "Failed to process upload request")
		return
	}
	defer tx.Rollback()

	// Insert new recipe with processing status
	var recipeID int
	query := `
		INSERT INTO recipes (title, status, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	
	now := time.Now().UTC()
	err = tx.QueryRow(query, "Processing Recipe", "processing", userID, now, now).Scan(&recipeID)
	if err != nil {
		logger.WithError(err).Error("Failed to create recipe record")
		DatabaseError(c, err, "create recipe")
		return
	}

	// Generate pre-signed upload URLs with enhanced security
	uploadURLs, err := h.storageService.GenerateUploadURLs(ctx, recipeID, &uploadRequest, c.ClientIP())
	if err != nil {
		logger.WithError(err).Error("Failed to generate upload URLs")
		StorageError(c, err, "generate upload URLs")
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		logger.WithError(err).Error("Failed to commit transaction")
		DatabaseError(c, err, "commit recipe creation")
		return
	}

	// Create response
	response := models.UploadResponse{
		RecipeID:   recipeID,
		UploadURLs: uploadURLs,
	}

	logger.WithFields(logrus.Fields{
		"recipe_id":    recipeID,
		"upload_count": len(uploadURLs),
	}).Info("Upload request processed successfully")

	// Return standardized response
	SuccessResponse(c, response)
}