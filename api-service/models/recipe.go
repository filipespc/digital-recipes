package models

import (
	"fmt"
	"time"
)

// Recipe represents a recipe in the system
type Recipe struct {
	ID           int       `json:"id" db:"id"`
	Title        string    `json:"title" db:"title"`
	Servings     *string   `json:"servings,omitempty" db:"servings"`
	Instructions *string   `json:"instructions,omitempty" db:"instructions"`
	Tips         *string   `json:"tips,omitempty" db:"tips"`
	Status       string    `json:"status" db:"status"`
	UserID       int       `json:"user_id" db:"user_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// RecipeWithIngredients represents a recipe with its ingredients
type RecipeWithIngredients struct {
	Recipe
	Ingredients []RecipeIngredient `json:"ingredients,omitempty"`
}

// RecipeIngredient represents an ingredient in a recipe
type RecipeIngredient struct {
	ID                     int      `json:"id" db:"id"`
	RecipeID               int      `json:"recipe_id" db:"recipe_id"`
	CanonicalIngredientID  *int     `json:"canonical_ingredient_id,omitempty" db:"canonical_ingredient_id"`
	OriginalText           string   `json:"original_text" db:"original_text"`
	Quantity               *float64 `json:"quantity,omitempty" db:"quantity"`
	Unit                   *string  `json:"unit,omitempty" db:"unit"`
	CanonicalName          *string  `json:"canonical_name,omitempty" db:"canonical_name"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// UploadRequest represents a request to upload recipe images
type UploadRequest struct {
	ImageCount      int      `json:"image_count" binding:"required,min=1,max=10"`
	MaxFileSizeMB   int      `json:"max_file_size_mb,omitempty" binding:"omitempty,min=1,max=50"`
	AllowedTypes    []string `json:"allowed_types,omitempty" binding:"omitempty,dive,oneof=image/jpeg image/jpg image/png image/webp"`
	ExpirationHours int      `json:"expiration_hours,omitempty" binding:"omitempty,min=1,max=24"`
}

// GetMaxFileSizeMB returns the max file size with a default
func (ur *UploadRequest) GetMaxFileSizeMB() int {
	if ur.MaxFileSizeMB <= 0 {
		return 10 // Default 10MB
	}
	return ur.MaxFileSizeMB
}

// GetAllowedTypes returns allowed file types with defaults
func (ur *UploadRequest) GetAllowedTypes() []string {
	if len(ur.AllowedTypes) == 0 {
		return []string{"image/jpeg", "image/png", "image/webp"}
	}
	return ur.AllowedTypes
}

// GetExpirationHours returns expiration hours with default
func (ur *UploadRequest) GetExpirationHours() int {
	if ur.ExpirationHours <= 0 {
		return 1 // Default 1 hour
	}
	return ur.ExpirationHours
}

// Validate performs additional business logic validation
func (ur *UploadRequest) Validate() error {
	// Enhanced security validation
	
	// 1. Strict size and count limits to prevent resource abuse
	if ur.ImageCount > 5 {
		return fmt.Errorf("maximum 5 images allowed per upload request")
	}
	
	if ur.GetMaxFileSizeMB() > 25 {
		return fmt.Errorf("maximum file size is 25MB per image")
	}
	
	// 2. Prevent resource exhaustion attacks
	totalSizeLimit := ur.ImageCount * ur.GetMaxFileSizeMB()
	if totalSizeLimit > 100 { // Maximum 100MB total per request
		return fmt.Errorf("total upload size cannot exceed 100MB (currently %dMB)", totalSizeLimit)
	}
	
	// 3. Validate file types with strict whitelist
	validTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,  // Allow both JPEG variants
		"image/png":  true,
		"image/webp": true,
	}
	
	allowedTypes := ur.GetAllowedTypes()
	if len(allowedTypes) == 0 {
		return fmt.Errorf("at least one allowed file type must be specified")
	}
	
	for _, fileType := range allowedTypes {
		if !validTypes[fileType] {
			return fmt.Errorf("unsupported file type: %s. Allowed types: image/jpeg, image/png, image/webp", fileType)
		}
	}
	
	// 4. Validate expiration time is reasonable
	expirationHours := ur.GetExpirationHours()
	if expirationHours > 24 {
		return fmt.Errorf("expiration time cannot exceed 24 hours")
	}
	
	// 5. Additional security checks for suspicious patterns
	if ur.ImageCount <= 0 {
		return fmt.Errorf("image count must be positive")
	}
	
	return nil
}

// UploadResponse represents the response for an upload request
type UploadResponse struct {
	RecipeID   int                   `json:"recipe_id"`
	UploadURLs []ImageUploadURL      `json:"upload_urls"`
}

// ImageUploadURL represents a pre-signed URL for image upload
type ImageUploadURL struct {
	ImageID   string `json:"image_id"`
	UploadURL string `json:"upload_url"`
	Fields    map[string]string `json:"fields,omitempty"`
}