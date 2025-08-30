package models

import (
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