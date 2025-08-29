-- Rollback seed data

-- Delete in reverse dependency order
DELETE FROM recipe_ingredients;
DELETE FROM recipes;
DELETE FROM canonical_ingredients;
DELETE FROM users;