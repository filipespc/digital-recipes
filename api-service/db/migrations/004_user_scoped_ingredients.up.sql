-- Migration: User-Scoped Ingredients
-- This migration implements the user-centered ingredient approach
-- where each user has their own ingredient collection

-- Step 1: Add user_id to canonical_ingredients table
ALTER TABLE canonical_ingredients
ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;

-- Step 2: Drop the global unique constraint on name
ALTER TABLE canonical_ingredients
DROP CONSTRAINT canonical_ingredients_name_key;

-- Step 3: Create a user-scoped unique constraint
-- This allows multiple users to have ingredients with the same name
-- but prevents duplicate ingredients within a single user's collection
ALTER TABLE canonical_ingredients
ADD CONSTRAINT canonical_ingredients_user_name_unique
UNIQUE (user_id, name);

-- Step 4: Add index for user_id for performance
CREATE INDEX idx_canonical_ingredients_user_id ON canonical_ingredients(user_id);

-- Step 5: Update existing data (if any exists)
-- For existing canonical ingredients without user_id, we'll need to handle them
-- Option A: Delete them (if no critical data)
-- Option B: Assign to a system user (if preserving data)
-- For this example, we'll delete orphaned canonical ingredients
DELETE FROM canonical_ingredients WHERE user_id IS NULL;

-- Step 6: Make user_id NOT NULL after cleanup
ALTER TABLE canonical_ingredients
ALTER COLUMN user_id SET NOT NULL;

-- Step 7: Add a comment to clarify the new design
COMMENT ON TABLE canonical_ingredients IS 'User-scoped ingredient collection. Each user has their own ingredient vocabulary.';
COMMENT ON COLUMN canonical_ingredients.user_id IS 'Scopes ingredient to specific user. Enables personal ingredient collections.';
COMMENT ON COLUMN recipe_ingredients.canonical_ingredient_id IS 'NULL = New Ingredient (unique to this recipe), NOT NULL = Linked to Existing (in users collection)';