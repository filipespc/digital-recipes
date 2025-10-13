-- Migration: 007_add_user_index
-- Purpose: Add index on user_id for improved query performance on pantry_items
-- Date: 2025-10-13
-- Issue: Fuzzy search queries filter by user_id but no index exists, causing full table scans

-- Add composite index for user_id + name for optimal query performance
-- This index helps with:
-- 1. Filtering pantry items by user_id (WHERE user_id = ?)
-- 2. Combined filtering and sorting (WHERE user_id = ? ORDER BY name)
CREATE INDEX IF NOT EXISTS idx_pantry_items_user_name
ON pantry_items (user_id, name);

-- Add comment to explain the index
COMMENT ON INDEX idx_pantry_items_user_name IS
'Composite index for efficient filtering by user_id and ordering by name. Used by /api/v1/pantry/search and /api/v1/pantry/fuzzy-search endpoints.';

-- Also add a single column index on user_id for queries that only filter by user
-- This is useful for counting items per user or other user-specific queries
CREATE INDEX IF NOT EXISTS idx_pantry_items_user_id
ON pantry_items (user_id);

-- Add comment to explain the single column index
COMMENT ON INDEX idx_pantry_items_user_id IS
'Single column index for queries filtering only by user_id. Improves performance for user-specific pantry item counts and listings.';