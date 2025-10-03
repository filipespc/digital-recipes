-- Rollback migration: User-Scoped Ingredients
-- This reverts the user-scoped ingredient changes

-- Step 1: Remove user_id NOT NULL constraint
ALTER TABLE canonical_ingredients
ALTER COLUMN user_id DROP NOT NULL;

-- Step 2: Remove comments
COMMENT ON TABLE canonical_ingredients IS NULL;
COMMENT ON COLUMN canonical_ingredients.user_id IS NULL;
COMMENT ON COLUMN recipe_ingredients.canonical_ingredient_id IS NULL;

-- Step 3: Drop user-scoped unique constraint
ALTER TABLE canonical_ingredients
DROP CONSTRAINT canonical_ingredients_user_name_unique;

-- Step 4: Drop user_id index
DROP INDEX idx_canonical_ingredients_user_id;

-- Step 5: Remove user_id column
ALTER TABLE canonical_ingredients
DROP COLUMN user_id;

-- Step 6: Restore global unique constraint on name
ALTER TABLE canonical_ingredients
ADD CONSTRAINT canonical_ingredients_name_key UNIQUE (name);