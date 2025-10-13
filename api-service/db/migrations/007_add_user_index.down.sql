-- Rollback migration: 007_add_user_index
-- Remove indexes added for user_id filtering

-- Drop composite index
DROP INDEX IF EXISTS idx_pantry_items_user_name;

-- Drop single column index
DROP INDEX IF EXISTS idx_pantry_items_user_id;