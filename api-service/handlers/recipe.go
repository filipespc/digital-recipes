package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"digital-recipes/api-service/db"
	"digital-recipes/api-service/middleware"
	"digital-recipes/api-service/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func init() {
	// Initialize random seed for jitter calculations
	rand.Seed(time.Now().UnixNano())
}

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

// Constants for ingredient quantity validation
const (
	maxQuantity = 999999.0 // Maximum allowed quantity value
	minQuantity = 0.0      // Minimum allowed quantity value
)

// parseAndValidateQuantity safely parses and validates ingredient quantity values
// Returns the parsed quantity as interface{} (float64 or nil) and any validation error
func parseAndValidateQuantity(quantityStr *string) (interface{}, error) {
	// Handle nil or empty quantity
	if quantityStr == nil || strings.TrimSpace(*quantityStr) == "" {
		return nil, nil
	}

	// Parse the quantity string
	parsedQty, err := strconv.ParseFloat(strings.TrimSpace(*quantityStr), 64)
	if err != nil {
		// If parsing fails, return nil to store as NULL in database
		// This preserves the original text while allowing the system to continue
		return nil, nil
	}

	// Validate for special float values
	if math.IsInf(parsedQty, 0) || math.IsNaN(parsedQty) {
		return nil, fmt.Errorf("invalid quantity value: %s", *quantityStr)
	}

	// Validate quantity bounds
	if parsedQty < minQuantity {
		return nil, fmt.Errorf("quantity must be positive, got: %f", parsedQty)
	}

	if parsedQty > maxQuantity {
		return nil, fmt.Errorf("quantity too large, maximum allowed: %f, got: %f", maxQuantity, parsedQty)
	}

	return parsedQty, nil
}

// sanitizeAndValidateIngredientName cleans and validates ingredient names
// Returns the sanitized name and any validation error
func sanitizeAndValidateIngredientName(name string) (string, error) {
	// Trim whitespace
	sanitized := strings.TrimSpace(name)

	// Check minimum length
	if len(sanitized) < 2 {
		return "", fmt.Errorf("ingredient name must be at least 2 characters long")
	}

	// Check maximum length
	if len(sanitized) > 100 {
		return "", fmt.Errorf("ingredient name must be less than 100 characters")
	}

	// Check for potentially dangerous characters
	if strings.ContainsAny(sanitized, "<>\"'&") {
		return "", fmt.Errorf("ingredient name contains invalid characters")
	}

	// HTML escape to prevent XSS (even though we validate above, defense in depth)
	sanitized = html.EscapeString(sanitized)

	// Additional validation for suspicious patterns
	if strings.Contains(strings.ToLower(sanitized), "script") ||
	   strings.Contains(strings.ToLower(sanitized), "javascript") ||
	   strings.Contains(strings.ToLower(sanitized), "onload") {
		return "", fmt.Errorf("ingredient name contains suspicious content")
	}

	return sanitized, nil
}

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

	// Extract image URLs for processing queue (convert signed URLs to GCS URLs)
	var imageURLs []string
	for _, uploadURL := range uploadURLs {
		// Extract the base GCS URL (without query parameters for public access)
		gcsURL := h.storageService.GetPublicImageURL(recipeID, uploadURL.ImageID, uploadURL.Extension)
		imageURLs = append(imageURLs, gcsURL)
	}

	// Enqueue processing job asynchronously
	go func() {
		h.enqueueProcessingJob(recipeID, imageURLs)
	}()

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

// UpdateRecipeStatus handles PUT /recipes/:id/status requests
func (h *RecipeHandler) UpdateRecipeStatus(c *gin.Context) {
	logger := middleware.LogWithContext(c)

	// Parse recipe ID from URL parameter
	idStr := c.Param("id")
	recipeID, err := strconv.Atoi(idStr)
	if err != nil {
		logger.WithFields(logrus.Fields{"id": idStr}).Warn("UpdateRecipeStatus invalid ID")
		BadRequestError(c, "invalid recipe ID")
		return
	}

	// Parse request body
	var statusUpdate struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
		logger.WithError(err).Warn("Status update binding failed")
		ValidationError(c, "Invalid request format. Status field is required.")
		return
	}

	// Validate status value
	validStatuses := []string{"processing", "review_required", "published", "failed"}
	isValid := false
	for _, validStatus := range validStatuses {
		if statusUpdate.Status == validStatus {
			isValid = true
			break
		}
	}
	if !isValid {
		BadRequestError(c, fmt.Sprintf("invalid status: %s. Valid statuses are: %s",
			statusUpdate.Status, strings.Join(validStatuses, ", ")))
		return
	}

	logger.WithFields(logrus.Fields{
		"recipe_id": recipeID,
		"status":    statusUpdate.Status,
	}).Info("Updating recipe status")

	// Update recipe status in database
	query := `
		UPDATE recipes
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := h.db.DB.Exec(query, statusUpdate.Status, time.Now().UTC(), recipeID)
	if err != nil {
		logger.WithError(err).Error("Failed to update recipe status")
		InternalServerError(c, "failed to update recipe status")
		return
	}

	// Check if recipe was found and updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.WithError(err).Error("Failed to get rows affected")
		InternalServerError(c, "failed to verify status update")
		return
	}

	if rowsAffected == 0 {
		NotFoundError(c, "recipe not found")
		return
	}

	logger.WithFields(logrus.Fields{
		"recipe_id": recipeID,
		"status":    statusUpdate.Status,
	}).Info("Recipe status updated successfully")

	// Return success response
	SuccessResponse(c, gin.H{
		"message":   "Recipe status updated successfully",
		"recipe_id": recipeID,
		"status":    statusUpdate.Status,
	})
}

// UpdateRecipe handles PUT /recipes/:id requests
func (h *RecipeHandler) UpdateRecipe(c *gin.Context) {
	logger := middleware.LogWithContext(c)

	// Parse recipe ID from URL parameter
	idStr := c.Param("id")
	recipeID, err := strconv.Atoi(idStr)
	if err != nil {
		logger.WithFields(logrus.Fields{"id": idStr}).Warn("UpdateRecipe invalid ID")
		BadRequestError(c, "invalid recipe ID")
		return
	}

	// Verify recipe ownership
	if !h.verifyRecipeOwnership(c, recipeID) {
		return
	}

	// Parse request body
	var updateRequest models.RecipeUpdateRequest
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		logger.WithError(err).Warn("Recipe update binding failed")
		ValidationError(c, "Invalid request format")
		return
	}

	logger.WithFields(logrus.Fields{
		"recipe_id": recipeID,
		"title":     updateRequest.Title,
	}).Info("Updating recipe data")

	// Begin transaction
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := h.db.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.WithError(err).Error("Failed to begin transaction")
		InternalServerError(c, "Failed to update recipe")
		return
	}
	defer tx.Rollback()

	// Convert instructions and tips arrays to strings if provided
	var instructionsStr *string
	var tipsStr *string

	if updateRequest.Instructions != nil && len(*updateRequest.Instructions) > 0 {
		joined := strings.Join(*updateRequest.Instructions, "\n")
		instructionsStr = &joined
	}

	if updateRequest.Tips != nil && len(*updateRequest.Tips) > 0 {
		joined := strings.Join(*updateRequest.Tips, "\n")
		tipsStr = &joined
	}

	// Update recipe fields
	updateQuery := `
		UPDATE recipes
		SET title = $1, servings = $2, instructions = $3, tips = $4,
		    prep_time = $5, cook_time = $6, total_time = $7, notes = $8, updated_at = $9
		WHERE id = $10
	`

	result, err := tx.Exec(updateQuery,
		updateRequest.Title,
		updateRequest.Servings,
		instructionsStr,
		tipsStr,
		updateRequest.PrepTime,
		updateRequest.CookTime,
		updateRequest.TotalTime,
		updateRequest.Notes,
		time.Now().UTC(),
		recipeID,
	)
	if err != nil {
		logger.WithError(err).Error("Failed to update recipe")
		InternalServerError(c, "failed to update recipe")
		return
	}

	// Check if recipe was found and updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.WithError(err).Error("Failed to get rows affected")
		InternalServerError(c, "failed to verify recipe update")
		return
	}

	if rowsAffected == 0 {
		NotFoundError(c, "recipe not found")
		return
	}

	// Update ingredients if provided
	if updateRequest.Ingredients != nil {
		// Delete existing ingredients
		deleteQuery := `DELETE FROM recipe_ingredients WHERE recipe_id = $1`
		_, err = tx.Exec(deleteQuery, recipeID)
		if err != nil {
			logger.WithError(err).Error("Failed to delete existing ingredients")
			InternalServerError(c, "failed to update ingredients")
			return
		}

		// Insert new ingredients
		for _, ingredient := range updateRequest.Ingredients {
			// Sanitize and validate ingredient name
			sanitizedName, err := sanitizeAndValidateIngredientName(ingredient.Name)
			if err != nil {
				logger.WithError(err).WithField("raw_name", ingredient.Name).Warn("Invalid ingredient name in UpdateRecipe")
				ValidationError(c, fmt.Sprintf("Invalid ingredient name: %s", err.Error()))
				return
			}
			ingredient.Name = sanitizedName

			insertQuery := `
				INSERT INTO recipe_ingredients (recipe_id, original_text, quantity, unit, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6)
			`

			// Parse and validate quantity
			quantityValue, err := parseAndValidateQuantity(ingredient.Quantity)
			if err != nil {
				logger.WithError(err).WithFields(logrus.Fields{
					"ingredient_name": ingredient.Name,
					"quantity":        ingredient.Quantity,
				}).Error("Invalid quantity value in ingredient")
				ValidationError(c, fmt.Sprintf("Invalid quantity for ingredient '%s': %s", ingredient.Name, err.Error()))
				return
			}

			// Create original text from quantity, unit, and name
			originalText := ingredient.Name
			if ingredient.Quantity != nil && *ingredient.Quantity != "" {
				originalText = *ingredient.Quantity + " " + ingredient.Name
				if ingredient.Unit != nil && *ingredient.Unit != "" {
					originalText = *ingredient.Quantity + " " + *ingredient.Unit + " " + ingredient.Name
				}
			}

			_, err = tx.Exec(insertQuery,
				recipeID,
				originalText,
				quantityValue,
				ingredient.Unit,
				time.Now().UTC(),
				time.Now().UTC(),
			)
			if err != nil {
				logger.WithError(err).Error("Failed to insert ingredient")
				InternalServerError(c, "failed to update ingredients")
				return
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		logger.WithError(err).Error("Failed to commit transaction")
		InternalServerError(c, "failed to update recipe")
		return
	}

	logger.WithFields(logrus.Fields{
		"recipe_id": recipeID,
	}).Info("Recipe updated successfully")

	// Return success response
	SuccessResponse(c, gin.H{
		"message":   "Recipe updated successfully",
		"recipe_id": recipeID,
	})
}

// enqueueProcessingJob sends a processing request to the parser service
func (h *RecipeHandler) enqueueProcessingJob(recipeID int, imageURLs []string) {
	logger := logrus.WithFields(logrus.Fields{
		"recipe_id":   recipeID,
		"image_count": len(imageURLs),
	})

	// Verify all images are uploaded before processing
	if !h.waitForImagesUpload(recipeID, imageURLs, logger) {
		logger.Error("Images not available after waiting, skipping processing")
		return
	}

	logger.Info("All images verified as uploaded, proceeding with processing")

	// Call parser service to enqueue processing
	parserURL := "http://parser-service:8081/process" // Use Docker service name

	payload := map[string]interface{}{
		"recipe_id":  fmt.Sprintf("%d", recipeID),
		"image_urls": imageURLs,
	}

	// Create HTTP request
	reqBody, err := json.Marshal(payload)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal processing request")
		return
	}

	// Make request to parser service
	resp, err := http.Post(parserURL, "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		logger.WithError(err).Error("Failed to enqueue processing job")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.WithField("status_code", resp.StatusCode).Error("Parser service returned error")
		return
	}

	logger.Info("Successfully enqueued recipe processing job")
}

// waitForImagesUpload verifies that all images are uploaded to GCS before processing
// Uses exponential backoff with jitter to prevent thundering herd problems
func (h *RecipeHandler) waitForImagesUpload(recipeID int, imageURLs []string, logger *logrus.Entry) bool {
	maxRetries := 6
	baseDelay := 1 * time.Second
	maxTimeout := 45 * time.Second // Total maximum wait time

	// Create context with timeout to prevent infinite waiting
	ctx, cancel := context.WithTimeout(context.Background(), maxTimeout)
	defer cancel()

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			logger.WithError(ctx.Err()).Warn("Upload verification cancelled due to timeout")
			return false
		default:
		}

		logger.WithField("attempt", attempt).Info("Checking if images are uploaded")

		allUploaded := true
		for i, imageURL := range imageURLs {
			// Check context before each image verification
			select {
			case <-ctx.Done():
				logger.WithError(ctx.Err()).Warn("Upload verification cancelled during image check")
				return false
			default:
			}

			if !h.verifyImageExists(imageURL, logger.WithField("image_index", i)) {
				allUploaded = false
				break
			}
		}

		if allUploaded {
			logger.Info("All images verified as uploaded")
			return true
		}

		if attempt < maxRetries {
			// Exponential backoff: delay = baseDelay * 2^(attempt-1)
			exponentialDelay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))

			// Add jitter: random value between 0 and exponentialDelay/2
			jitter := time.Duration(rand.Int63n(int64(exponentialDelay / 2)))
			totalDelay := exponentialDelay + jitter

			// Cap the delay to prevent extremely long waits
			maxSingleDelay := 10 * time.Second
			if totalDelay > maxSingleDelay {
				totalDelay = maxSingleDelay
			}

			logger.WithFields(logrus.Fields{
				"retry_in_seconds":    totalDelay.Seconds(),
				"exponential_delay":   exponentialDelay.Seconds(),
				"jitter_seconds":      jitter.Seconds(),
				"attempt":             attempt,
				"max_retries":         maxRetries,
			}).Info("Some images not uploaded yet, retrying with exponential backoff")

			// Use context-aware sleep
			select {
			case <-ctx.Done():
				logger.WithError(ctx.Err()).Warn("Upload verification cancelled during retry delay")
				return false
			case <-time.After(totalDelay):
				// Continue to next retry
			}
		}
	}

	logger.WithFields(logrus.Fields{
		"max_retries":     maxRetries,
		"total_wait_time": maxTimeout.Seconds(),
	}).Error("Images not uploaded after maximum retries")
	return false
}

// verifyImageExists checks if an image exists in GCS
func (h *RecipeHandler) verifyImageExists(imageURL string, logger *logrus.Entry) bool {
	// Parse the GCS URL to extract bucket and object path
	// Expected format: https://storage.googleapis.com/bucket-name/path/to/object
	if !strings.HasPrefix(imageURL, "https://storage.googleapis.com/") {
		logger.WithField("url", imageURL).Warn("Invalid GCS URL format")
		return false
	}

	// Extract bucket and object path from URL
	urlParts := strings.TrimPrefix(imageURL, "https://storage.googleapis.com/")
	pathParts := strings.SplitN(urlParts, "/", 2)
	if len(pathParts) != 2 {
		logger.WithField("url", imageURL).Warn("Could not parse GCS URL")
		return false
	}

	bucketName := pathParts[0]
	objectPath := pathParts[1]

	// Use the storage service to check if the object exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := h.storageService.ObjectExists(ctx, objectPath)
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"bucket": bucketName,
			"object": objectPath,
		}).Error("Error checking if image exists in GCS")
		return false
	}

	if !exists {
		logger.WithFields(logrus.Fields{
			"bucket": bucketName,
			"object": objectPath,
		}).Debug("Image not found in GCS")
		return false
	}

	logger.WithFields(logrus.Fields{
		"bucket": bucketName,
		"object": objectPath,
	}).Debug("Image verified in GCS")
	return true
}

// verifyRecipeOwnership checks if the authenticated user owns the specified recipe
func (h *RecipeHandler) verifyRecipeOwnership(c *gin.Context, recipeID int) bool {
	logger := middleware.LogWithContext(c)

	userID := middleware.GetUserID(c)
	if userID == 0 {
		logger.Warn("No authenticated user found for recipe ownership check")
		BadRequestError(c, "authentication required")
		return false
	}

	var ownerID int
	query := `SELECT user_id FROM recipes WHERE id = $1`
	err := h.db.DB.QueryRow(query, recipeID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.WithFields(logrus.Fields{
				"recipe_id": recipeID,
				"user_id":   userID,
			}).Warn("Recipe not found during ownership check")
			NotFoundError(c, "recipe not found")
		} else {
			logger.WithError(err).Error("Failed to verify recipe ownership")
			InternalServerError(c, "failed to verify recipe ownership")
		}
		return false
	}

	if ownerID != userID {
		logger.WithFields(logrus.Fields{
			"recipe_id":    recipeID,
			"owner_id":     ownerID,
			"requesting_user_id": userID,
		}).Warn("Access denied: user does not own recipe")
		BadRequestError(c, "access to this recipe is forbidden")
		return false
	}

	logger.WithFields(logrus.Fields{
		"recipe_id": recipeID,
		"user_id":   userID,
	}).Debug("Recipe ownership verified")

	return true
}

// AddIngredient handles POST /recipes/:id/ingredients requests
func (h *RecipeHandler) AddIngredient(c *gin.Context) {
	logger := middleware.LogWithContext(c)

	// Parse recipe ID from URL parameter
	idStr := c.Param("id")
	recipeID, err := strconv.Atoi(idStr)
	if err != nil {
		logger.WithFields(logrus.Fields{"id": idStr}).Warn("AddIngredient invalid recipe ID")
		BadRequestError(c, "invalid recipe ID")
		return
	}

	// Verify recipe ownership
	if !h.verifyRecipeOwnership(c, recipeID) {
		return
	}

	// Parse request body
	var ingredient models.ProcessedIngredient
	if err := c.ShouldBindJSON(&ingredient); err != nil {
		logger.WithError(err).Warn("Add ingredient binding failed")
		ValidationError(c, "Invalid ingredient format")
		return
	}

	// Sanitize and validate ingredient name
	sanitizedName, err := sanitizeAndValidateIngredientName(ingredient.Name)
	if err != nil {
		logger.WithError(err).WithField("raw_name", ingredient.Name).Warn("Invalid ingredient name in AddIngredient")
		ValidationError(c, fmt.Sprintf("Invalid ingredient name: %s", err.Error()))
		return
	}
	ingredient.Name = sanitizedName

	logger.WithFields(logrus.Fields{
		"recipe_id": recipeID,
		"name":      ingredient.Name,
	}).Info("Adding ingredient to recipe")

	// Check if recipe exists
	var recipeExists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM recipes WHERE id = $1)`
	err = h.db.DB.QueryRow(checkQuery, recipeID).Scan(&recipeExists)
	if err != nil {
		logger.WithError(err).Error("Failed to check recipe existence")
		InternalServerError(c, "failed to verify recipe")
		return
	}
	if !recipeExists {
		NotFoundError(c, "recipe not found")
		return
	}

	// Parse and validate quantity
	quantityValue, err := parseAndValidateQuantity(ingredient.Quantity)
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"ingredient_name": ingredient.Name,
			"quantity":        ingredient.Quantity,
		}).Error("Invalid quantity value in AddIngredient")
		ValidationError(c, fmt.Sprintf("Invalid quantity for ingredient '%s': %s", ingredient.Name, err.Error()))
		return
	}

	// Create original text from quantity, unit, and name
	originalText := ingredient.Name
	if ingredient.Quantity != nil && *ingredient.Quantity != "" {
		originalText = *ingredient.Quantity + " " + ingredient.Name
		if ingredient.Unit != nil && *ingredient.Unit != "" {
			originalText = *ingredient.Quantity + " " + *ingredient.Unit + " " + ingredient.Name
		}
	}

	// Insert ingredient
	insertQuery := `
		INSERT INTO recipe_ingredients (recipe_id, original_text, quantity, unit, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var ingredientID int
	now := time.Now().UTC()
	err = h.db.DB.QueryRow(insertQuery,
		recipeID,
		originalText,
		quantityValue,
		ingredient.Unit,
		now,
		now,
	).Scan(&ingredientID)

	if err != nil {
		logger.WithError(err).Error("Failed to insert ingredient")
		InternalServerError(c, "failed to add ingredient")
		return
	}

	logger.WithFields(logrus.Fields{
		"recipe_id":     recipeID,
		"ingredient_id": ingredientID,
	}).Info("Ingredient added successfully")

	// Return success response with the new ingredient ID
	SuccessResponse(c, gin.H{
		"message":       "Ingredient added successfully",
		"recipe_id":     recipeID,
		"ingredient_id": ingredientID,
	})
}

// UpdateIngredient handles PUT /recipes/:id/ingredients/:ingredient_id requests
func (h *RecipeHandler) UpdateIngredient(c *gin.Context) {
	logger := middleware.LogWithContext(c)

	// Parse recipe ID and ingredient ID from URL parameters
	recipeIDStr := c.Param("id")
	ingredientIDStr := c.Param("ingredient_id")

	recipeID, err := strconv.Atoi(recipeIDStr)
	if err != nil {
		logger.WithFields(logrus.Fields{"recipe_id": recipeIDStr}).Warn("UpdateIngredient invalid recipe ID")
		BadRequestError(c, "invalid recipe ID")
		return
	}

	ingredientID, err := strconv.Atoi(ingredientIDStr)
	if err != nil {
		logger.WithFields(logrus.Fields{"ingredient_id": ingredientIDStr}).Warn("UpdateIngredient invalid ingredient ID")
		BadRequestError(c, "invalid ingredient ID")
		return
	}

	// Verify recipe ownership
	if !h.verifyRecipeOwnership(c, recipeID) {
		return
	}

	// Parse request body
	var ingredient models.ProcessedIngredient
	if err := c.ShouldBindJSON(&ingredient); err != nil {
		logger.WithError(err).Warn("Update ingredient binding failed")
		ValidationError(c, "Invalid ingredient format")
		return
	}

	// Sanitize and validate ingredient name
	sanitizedName, err := sanitizeAndValidateIngredientName(ingredient.Name)
	if err != nil {
		logger.WithError(err).WithField("raw_name", ingredient.Name).Warn("Invalid ingredient name in UpdateIngredient")
		ValidationError(c, fmt.Sprintf("Invalid ingredient name: %s", err.Error()))
		return
	}
	ingredient.Name = sanitizedName

	logger.WithFields(logrus.Fields{
		"recipe_id":     recipeID,
		"ingredient_id": ingredientID,
		"name":          ingredient.Name,
	}).Info("Updating ingredient")

	// Check if ingredient exists and belongs to the recipe
	var ingredientExists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM recipe_ingredients WHERE id = $1 AND recipe_id = $2)`
	err = h.db.DB.QueryRow(checkQuery, ingredientID, recipeID).Scan(&ingredientExists)
	if err != nil {
		logger.WithError(err).Error("Failed to check ingredient existence")
		InternalServerError(c, "failed to verify ingredient")
		return
	}
	if !ingredientExists {
		NotFoundError(c, "ingredient not found")
		return
	}

	// Parse and validate quantity
	quantityValue, err := parseAndValidateQuantity(ingredient.Quantity)
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"ingredient_name": ingredient.Name,
			"quantity":        ingredient.Quantity,
		}).Error("Invalid quantity value in UpdateIngredient")
		ValidationError(c, fmt.Sprintf("Invalid quantity for ingredient '%s': %s", ingredient.Name, err.Error()))
		return
	}

	// Create original text from quantity, unit, and name
	originalText := ingredient.Name
	if ingredient.Quantity != nil && *ingredient.Quantity != "" {
		originalText = *ingredient.Quantity + " " + ingredient.Name
		if ingredient.Unit != nil && *ingredient.Unit != "" {
			originalText = *ingredient.Quantity + " " + *ingredient.Unit + " " + ingredient.Name
		}
	}

	// Update ingredient
	updateQuery := `
		UPDATE recipe_ingredients
		SET original_text = $1, quantity = $2, unit = $3, updated_at = $4
		WHERE id = $5 AND recipe_id = $6
	`

	result, err := h.db.DB.Exec(updateQuery,
		originalText,
		quantityValue,
		ingredient.Unit,
		time.Now().UTC(),
		ingredientID,
		recipeID,
	)

	if err != nil {
		logger.WithError(err).Error("Failed to update ingredient")
		InternalServerError(c, "failed to update ingredient")
		return
	}

	// Check if ingredient was updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.WithError(err).Error("Failed to get rows affected")
		InternalServerError(c, "failed to verify ingredient update")
		return
	}

	if rowsAffected == 0 {
		NotFoundError(c, "ingredient not found")
		return
	}

	logger.WithFields(logrus.Fields{
		"recipe_id":     recipeID,
		"ingredient_id": ingredientID,
	}).Info("Ingredient updated successfully")

	// Return success response
	SuccessResponse(c, gin.H{
		"message":       "Ingredient updated successfully",
		"recipe_id":     recipeID,
		"ingredient_id": ingredientID,
	})
}

// DeleteIngredient handles DELETE /recipes/:id/ingredients/:ingredient_id requests
func (h *RecipeHandler) DeleteIngredient(c *gin.Context) {
	logger := middleware.LogWithContext(c)

	// Parse recipe ID and ingredient ID from URL parameters
	recipeIDStr := c.Param("id")
	ingredientIDStr := c.Param("ingredient_id")

	recipeID, err := strconv.Atoi(recipeIDStr)
	if err != nil {
		logger.WithFields(logrus.Fields{"recipe_id": recipeIDStr}).Warn("DeleteIngredient invalid recipe ID")
		BadRequestError(c, "invalid recipe ID")
		return
	}

	ingredientID, err := strconv.Atoi(ingredientIDStr)
	if err != nil {
		logger.WithFields(logrus.Fields{"ingredient_id": ingredientIDStr}).Warn("DeleteIngredient invalid ingredient ID")
		BadRequestError(c, "invalid ingredient ID")
		return
	}

	// Verify recipe ownership
	if !h.verifyRecipeOwnership(c, recipeID) {
		return
	}

	logger.WithFields(logrus.Fields{
		"recipe_id":     recipeID,
		"ingredient_id": ingredientID,
	}).Info("Deleting ingredient")

	// Delete ingredient (with recipe_id check to ensure it belongs to the recipe)
	deleteQuery := `DELETE FROM recipe_ingredients WHERE id = $1 AND recipe_id = $2`

	result, err := h.db.DB.Exec(deleteQuery, ingredientID, recipeID)
	if err != nil {
		logger.WithError(err).Error("Failed to delete ingredient")
		InternalServerError(c, "failed to delete ingredient")
		return
	}

	// Check if ingredient was deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.WithError(err).Error("Failed to get rows affected")
		InternalServerError(c, "failed to verify ingredient deletion")
		return
	}

	if rowsAffected == 0 {
		NotFoundError(c, "ingredient not found")
		return
	}

	logger.WithFields(logrus.Fields{
		"recipe_id":     recipeID,
		"ingredient_id": ingredientID,
	}).Info("Ingredient deleted successfully")

	// Return success response
	SuccessResponse(c, gin.H{
		"message":       "Ingredient deleted successfully",
		"recipe_id":     recipeID,
		"ingredient_id": ingredientID,
	})
}

// SearchIngredients handles GET /ingredients/search requests
func (h *RecipeHandler) SearchIngredients(c *gin.Context) {
	logger := middleware.LogWithContext(c)

	// Parse query parameter
	query := c.Query("q")
	if query == "" {
		BadRequestError(c, "query parameter 'q' is required")
		return
	}

	// Parse limit parameter (default to 20, max 100)
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		BadRequestError(c, "invalid limit parameter. Must be between 1 and 100")
		return
	}

	logger.WithFields(logrus.Fields{
		"query": query,
		"limit": limit,
	}).Debug("Searching canonical ingredients")

	// Search canonical ingredients using fuzzy matching (ILIKE for PostgreSQL)
	// This supports partial matches and is case-insensitive
	searchQuery := `
		SELECT id, name, is_approved, created_at, updated_at
		FROM canonical_ingredients
		WHERE name ILIKE '%' || $1 || '%'
		ORDER BY
			-- Exact matches first
			CASE WHEN LOWER(name) = LOWER($1) THEN 1 ELSE 2 END,
			-- Then by length (shorter names first for closer matches)
			LENGTH(name),
			-- Finally alphabetically
			name
		LIMIT $2
	`

	rows, err := h.db.DB.Query(searchQuery, query, limit)
	if err != nil {
		logger.WithError(err).Error("Failed to search ingredients")
		InternalServerError(c, "failed to search ingredients")
		return
	}
	defer rows.Close()

	var ingredients []models.CanonicalIngredient
	for rows.Next() {
		var ingredient models.CanonicalIngredient
		err := rows.Scan(
			&ingredient.ID,
			&ingredient.Name,
			&ingredient.IsApproved,
			&ingredient.CreatedAt,
			&ingredient.UpdatedAt,
		)
		if err != nil {
			logger.WithError(err).Error("Failed to scan ingredient")
			InternalServerError(c, "failed to parse ingredient data")
			return
		}
		ingredients = append(ingredients, ingredient)
	}

	if err = rows.Err(); err != nil {
		logger.WithError(err).Error("Rows iteration error")
		InternalServerError(c, "error reading ingredient data")
		return
	}

	// Return empty array if no ingredients found
	if ingredients == nil {
		ingredients = []models.CanonicalIngredient{}
	}

	logger.WithFields(logrus.Fields{
		"query":        query,
		"result_count": len(ingredients),
	}).Debug("Ingredient search completed")

	// Return standardized response
	SuccessResponse(c, ingredients)
}

// LinkIngredientToCanonical handles PUT /recipes/:id/ingredients/:ingredient_id/link requests
func (h *RecipeHandler) LinkIngredientToCanonical(c *gin.Context) {
	logger := middleware.LogWithContext(c)

	// Parse recipe ID and ingredient ID from URL parameters
	recipeIDStr := c.Param("id")
	ingredientIDStr := c.Param("ingredient_id")

	recipeID, err := strconv.Atoi(recipeIDStr)
	if err != nil {
		logger.WithFields(logrus.Fields{"recipe_id": recipeIDStr}).Warn("LinkIngredientToCanonical invalid recipe ID")
		BadRequestError(c, "invalid recipe ID")
		return
	}

	ingredientID, err := strconv.Atoi(ingredientIDStr)
	if err != nil {
		logger.WithFields(logrus.Fields{"ingredient_id": ingredientIDStr}).Warn("LinkIngredientToCanonical invalid ingredient ID")
		BadRequestError(c, "invalid ingredient ID")
		return
	}

	// Verify recipe ownership
	if !h.verifyRecipeOwnership(c, recipeID) {
		return
	}

	// Parse request body
	var linkRequest struct {
		CanonicalIngredientID int `json:"canonical_ingredient_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&linkRequest); err != nil {
		logger.WithError(err).Warn("Link ingredient binding failed")
		ValidationError(c, "Invalid request format. canonical_ingredient_id is required.")
		return
	}

	logger.WithFields(logrus.Fields{
		"recipe_id":               recipeID,
		"ingredient_id":           ingredientID,
		"canonical_ingredient_id": linkRequest.CanonicalIngredientID,
	}).Info("Linking ingredient to canonical ingredient")

	// Check if ingredient exists and belongs to the recipe
	var ingredientExists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM recipe_ingredients WHERE id = $1 AND recipe_id = $2)`
	err = h.db.DB.QueryRow(checkQuery, ingredientID, recipeID).Scan(&ingredientExists)
	if err != nil {
		logger.WithError(err).Error("Failed to check ingredient existence")
		InternalServerError(c, "failed to verify ingredient")
		return
	}
	if !ingredientExists {
		NotFoundError(c, "ingredient not found")
		return
	}

	// Check if canonical ingredient exists
	var canonicalExists bool
	checkCanonicalQuery := `SELECT EXISTS(SELECT 1 FROM canonical_ingredients WHERE id = $1)`
	err = h.db.DB.QueryRow(checkCanonicalQuery, linkRequest.CanonicalIngredientID).Scan(&canonicalExists)
	if err != nil {
		logger.WithError(err).Error("Failed to check canonical ingredient existence")
		InternalServerError(c, "failed to verify canonical ingredient")
		return
	}
	if !canonicalExists {
		NotFoundError(c, "canonical ingredient not found")
		return
	}

	// Update the ingredient to link it to the canonical ingredient
	updateQuery := `
		UPDATE recipe_ingredients
		SET canonical_ingredient_id = $1, updated_at = $2
		WHERE id = $3 AND recipe_id = $4
	`

	result, err := h.db.DB.Exec(updateQuery,
		linkRequest.CanonicalIngredientID,
		time.Now().UTC(),
		ingredientID,
		recipeID,
	)

	if err != nil {
		logger.WithError(err).Error("Failed to link ingredient")
		InternalServerError(c, "failed to link ingredient")
		return
	}

	// Check if ingredient was updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.WithError(err).Error("Failed to get rows affected")
		InternalServerError(c, "failed to verify ingredient link")
		return
	}

	if rowsAffected == 0 {
		NotFoundError(c, "ingredient not found")
		return
	}

	logger.WithFields(logrus.Fields{
		"recipe_id":               recipeID,
		"ingredient_id":           ingredientID,
		"canonical_ingredient_id": linkRequest.CanonicalIngredientID,
	}).Info("Ingredient linked to canonical ingredient successfully")

	// Return success response
	SuccessResponse(c, gin.H{
		"message":                 "Ingredient linked successfully",
		"recipe_id":               recipeID,
		"ingredient_id":           ingredientID,
		"canonical_ingredient_id": linkRequest.CanonicalIngredientID,
	})
}

// CreateCanonicalIngredient handles POST /ingredients requests
func (h *RecipeHandler) CreateCanonicalIngredient(c *gin.Context) {
	logger := middleware.LogWithContext(c)

	// Parse request body
	var ingredientRequest struct {
		Name       string `json:"name" binding:"required"`
		IsApproved *bool  `json:"is_approved,omitempty"`
	}
	if err := c.ShouldBindJSON(&ingredientRequest); err != nil {
		logger.WithError(err).Warn("Create canonical ingredient binding failed")
		ValidationError(c, "Invalid request format. name is required.")
		return
	}

	// Sanitize and validate ingredient name
	sanitizedName, err := sanitizeAndValidateIngredientName(ingredientRequest.Name)
	if err != nil {
		logger.WithError(err).WithField("raw_name", ingredientRequest.Name).Warn("Invalid ingredient name")
		ValidationError(c, err.Error())
		return
	}
	ingredientRequest.Name = sanitizedName

	// Set default value for IsApproved if not provided
	isApproved := false
	if ingredientRequest.IsApproved != nil {
		isApproved = *ingredientRequest.IsApproved
	}

	logger.WithFields(logrus.Fields{
		"name":        ingredientRequest.Name,
		"is_approved": isApproved,
	}).Info("Creating canonical ingredient")

	// Check if ingredient already exists (case-insensitive)
	var existingID int
	checkQuery := `SELECT id FROM canonical_ingredients WHERE LOWER(name) = LOWER($1)`
	err = h.db.DB.QueryRow(checkQuery, ingredientRequest.Name).Scan(&existingID)
	if err == nil {
		// Ingredient already exists, return the existing one
		logger.WithFields(logrus.Fields{
			"existing_id": existingID,
			"name":        ingredientRequest.Name,
		}).Info("Canonical ingredient already exists")

		SuccessResponse(c, gin.H{
			"message": "Canonical ingredient already exists",
			"id":      existingID,
			"name":    ingredientRequest.Name,
		})
		return
	} else if err != sql.ErrNoRows {
		logger.WithError(err).Error("Failed to check existing ingredient")
		InternalServerError(c, "failed to check existing ingredient")
		return
	}

	// Insert new canonical ingredient
	insertQuery := `
		INSERT INTO canonical_ingredients (name, is_approved, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var ingredientID int
	now := time.Now().UTC()
	err = h.db.DB.QueryRow(insertQuery,
		ingredientRequest.Name,
		isApproved,
		now,
		now,
	).Scan(&ingredientID)

	if err != nil {
		logger.WithError(err).Error("Failed to create canonical ingredient")
		InternalServerError(c, "failed to create canonical ingredient")
		return
	}

	logger.WithFields(logrus.Fields{
		"ingredient_id": ingredientID,
		"name":          ingredientRequest.Name,
	}).Info("Canonical ingredient created successfully")

	// Return success response
	SuccessResponse(c, gin.H{
		"message":     "Canonical ingredient created successfully",
		"id":          ingredientID,
		"name":        ingredientRequest.Name,
		"is_approved": isApproved,
	})
}