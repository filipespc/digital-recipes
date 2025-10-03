-- Migration: Rename canonical_ingredients to pantry_items
-- This migration renames the canonical ingredients concept to pantry items

-- Step 1: Rename the table
ALTER TABLE canonical_ingredients RENAME TO pantry_items;

-- Step 2: Rename the constraint
ALTER TABLE pantry_items
RENAME CONSTRAINT canonical_ingredients_user_name_unique TO pantry_items_user_name_unique;

-- Step 3: Update recipe_ingredients table foreign key column name
ALTER TABLE recipe_ingredients
RENAME COLUMN canonical_ingredient_id TO pantry_item_id;

-- Step 4: Rename the foreign key constraint
ALTER TABLE recipe_ingredients
DROP CONSTRAINT IF EXISTS recipe_ingredients_canonical_ingredient_id_fkey;

ALTER TABLE recipe_ingredients
ADD CONSTRAINT recipe_ingredients_pantry_item_id_fkey
FOREIGN KEY (pantry_item_id) REFERENCES pantry_items(id) ON DELETE SET NULL;

-- Step 5: Update indexes
DROP INDEX IF EXISTS idx_canonical_ingredients_user_id;
CREATE INDEX idx_pantry_items_user_id ON pantry_items(user_id);

DROP INDEX IF EXISTS idx_recipe_ingredients_canonical_ingredient_id;
CREATE INDEX idx_recipe_ingredients_pantry_item_id ON recipe_ingredients(pantry_item_id);

-- Step 6: Add category and default_unit columns to pantry_items
ALTER TABLE pantry_items
ADD COLUMN category VARCHAR(50),
ADD COLUMN default_unit VARCHAR(20);

-- Step 7: Update comments
COMMENT ON TABLE pantry_items IS 'User-scoped pantry items. Each user has their own pantry collection.';
COMMENT ON COLUMN pantry_items.user_id IS 'Scopes pantry item to specific user. Enables personal pantry collections.';
COMMENT ON COLUMN pantry_items.category IS 'Optional category for organizing pantry items (e.g., dairy, produce, spices)';
COMMENT ON COLUMN pantry_items.default_unit IS 'Default unit of measurement for this pantry item';
COMMENT ON COLUMN recipe_ingredients.pantry_item_id IS 'NULL = New ingredient (unique to this recipe), NOT NULL = Linked to existing pantry item';