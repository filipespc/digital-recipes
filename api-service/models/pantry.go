package models

import (
	"time"
)

// PantryItem represents a user's pantry item (ingredient in their collection)
type PantryItem struct {
	ID          int       `json:"id" db:"id"`
	UserID      int       `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	Category    *string   `json:"category,omitempty" db:"category"`
	DefaultUnit *string   `json:"default_unit,omitempty" db:"default_unit"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// PantryItemInput represents the input for creating or updating a pantry item
type PantryItemInput struct {
	Name        string  `json:"name" binding:"required,min=1,max=100"`
	Category    *string `json:"category,omitempty" binding:"omitempty,max=50"`
	DefaultUnit *string `json:"default_unit,omitempty" binding:"omitempty,max=20"`
}

// PantryItemWithSimilarity represents a pantry item with its similarity score
type PantryItemWithSimilarity struct {
	PantryItem
	Similarity float64 `json:"similarity" db:"similarity"`
}

// BatchResolveRequest represents a request to resolve multiple ingredients to pantry items
type BatchResolveRequest struct {
	RecipeID    int                  `json:"recipe_id" binding:"required"`
	UserID      int                  `json:"user_id" binding:"required"`
	Ingredients []BatchResolveItem   `json:"ingredients" binding:"required,min=1,max=50"`
}

// BatchResolveItem represents a single ingredient to resolve
type BatchResolveItem struct {
	IngredientID int    `json:"ingredient_id" binding:"required"`
	OriginalText string `json:"original_text" binding:"required,min=1,max=500"`
}

// BatchResolveResponse represents the response for batch resolve operation
type BatchResolveResponse struct {
	Resolved   []ResolvedPantryItem `json:"resolved"`
	Errors     []ResolveError       `json:"errors,omitempty"`
}

// ResolvedPantryItem represents a successfully resolved pantry item
type ResolvedPantryItem struct {
	IngredientID int         `json:"ingredient_id"`
	Action       string      `json:"action"` // "linked" or "created"
	PantryItem   PantryItem  `json:"pantry_item"`
}

// ResolveError represents an error resolving a specific ingredient
type ResolveError struct {
	IngredientID int    `json:"ingredient_id"`
	Error        string `json:"error"`
}