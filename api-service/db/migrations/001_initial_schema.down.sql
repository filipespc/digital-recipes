-- Rollback script for initial schema

-- Drop triggers first
DROP TRIGGER IF EXISTS update_recipe_ingredients_updated_at ON recipe_ingredients;
DROP TRIGGER IF EXISTS update_canonical_ingredients_updated_at ON canonical_ingredients;
DROP TRIGGER IF EXISTS update_recipes_updated_at ON recipes;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_canonical_ingredients_name;
DROP INDEX IF EXISTS idx_recipe_ingredients_canonical_id;
DROP INDEX IF EXISTS idx_recipe_ingredients_recipe_id;
DROP INDEX IF EXISTS idx_recipes_status;
DROP INDEX IF EXISTS idx_recipes_user_id;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS recipe_ingredients;
DROP TABLE IF EXISTS canonical_ingredients;
DROP TABLE IF EXISTS recipes;
DROP TABLE IF EXISTS users;