-- Rollback: Rename pantry_items back to canonical_ingredients

-- Step 1: Remove new columns
ALTER TABLE pantry_items
DROP COLUMN IF EXISTS category,
DROP COLUMN IF EXISTS default_unit;

-- Step 2: Update indexes
DROP INDEX IF EXISTS idx_pantry_items_user_id;
CREATE INDEX idx_canonical_ingredients_user_id ON pantry_items(user_id);

DROP INDEX IF EXISTS idx_recipe_ingredients_pantry_item_id;
CREATE INDEX idx_recipe_ingredients_canonical_ingredient_id ON recipe_ingredients(pantry_item_id);

-- Step 3: Rename the foreign key constraint
ALTER TABLE recipe_ingredients
DROP CONSTRAINT IF EXISTS recipe_ingredients_pantry_item_id_fkey;

ALTER TABLE recipe_ingredients
ADD CONSTRAINT recipe_ingredients_canonical_ingredient_id_fkey
FOREIGN KEY (pantry_item_id) REFERENCES pantry_items(id) ON DELETE SET NULL;

-- Step 4: Rename the column in recipe_ingredients
ALTER TABLE recipe_ingredients
RENAME COLUMN pantry_item_id TO canonical_ingredient_id;

-- Step 5: Rename the constraint
ALTER TABLE pantry_items
RENAME CONSTRAINT pantry_items_user_name_unique TO canonical_ingredients_user_name_unique;

-- Step 6: Rename the table
ALTER TABLE pantry_items RENAME TO canonical_ingredients;

-- Step 7: Restore comments
COMMENT ON TABLE canonical_ingredients IS 'User-scoped ingredient collection. Each user has their own ingredient vocabulary.';
COMMENT ON COLUMN canonical_ingredients.user_id IS 'Scopes ingredient to specific user. Enables personal ingredient collections.';
COMMENT ON COLUMN recipe_ingredients.canonical_ingredient_id IS 'NULL = New Ingredient (unique to this recipe), NOT NULL = Linked to Existing (in users collection)';