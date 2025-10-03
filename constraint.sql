ALTER TABLE recipe_ingredients
ADD CONSTRAINT check_published_ingredients_linked
CHECK (
  (SELECT status FROM recipes WHERE id = recipe_id) != 'published'
  OR canonical_ingredient_id IS NOT NULL
);
