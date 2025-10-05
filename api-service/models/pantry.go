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