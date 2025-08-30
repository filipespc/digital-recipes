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
	// Check for reasonable combinations
	if ur.ImageCount > 5 && ur.GetMaxFileSizeMB() > 20 {
		return fmt.Errorf("cannot upload more than 5 images when max file size exceeds 20MB")
	}
	
	// Validate file types
	validTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}
	
	for _, fileType := range ur.GetAllowedTypes() {
		if !validTypes[fileType] {
			return fmt.Errorf("unsupported file type: %s", fileType)
		}
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