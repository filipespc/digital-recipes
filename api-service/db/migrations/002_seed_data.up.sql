-- Seed data for development and testing

-- Insert test user
INSERT INTO users (email, name) VALUES 
    ('test@example.com', 'Test User'),
    ('chef@example.com', 'Chef Testington');

-- Insert common canonical ingredients
INSERT INTO canonical_ingredients (name, is_approved) VALUES 
    ('Egg', true),
    ('All-Purpose Flour', true),
    ('Sugar', true),
    ('Salt', true),
    ('Butter', true),
    ('Milk', true),
    ('Olive Oil', true),
    ('Onion', true),
    ('Garlic', true),
    ('Black Pepper', true),
    ('Vanilla Extract', true),
    ('Baking Powder', true),
    ('Tomato', true),
    ('Chicken Breast', true),
    ('Rice', true);

-- Insert test recipe
INSERT INTO recipes (title, servings, instructions, tips, status, user_id) VALUES 
    (
        'Classic Scrambled Eggs',
        '2 servings',
        '1. Crack eggs into a bowl and whisk with salt and pepper
2. Heat butter in non-stick pan over medium-low heat
3. Add eggs and gently stir with spatula
4. Cook slowly, stirring frequently, until just set
5. Remove from heat and serve immediately',
        'Low heat is key for creamy eggs. Don''t rush the process.',
        'published',
        1
    ),
    (
        'Simple Pasta',
        '4 servings', 
        '1. Bring large pot of salted water to boil
2. Add pasta and cook according to package directions
3. Meanwhile, heat olive oil in large pan
4. Add garlic and cook until fragrant
5. Drain pasta and toss with oil and garlic
6. Season with salt and pepper',
        'Save some pasta water to adjust consistency if needed.',
        'review_required',
        1
    );

-- Link ingredients to the scrambled eggs recipe
INSERT INTO recipe_ingredients (recipe_id, canonical_ingredient_id, original_text, quantity, unit) VALUES 
    (1, 1, '4 large eggs', 4, 'whole'),
    (1, 4, '1/4 teaspoon salt', 0.25, 'teaspoon'),
    (1, 10, 'pinch of black pepper', 1, 'pinch'),
    (1, 5, '2 tablespoons butter', 2, 'tablespoon');

-- Link ingredients to the pasta recipe  
INSERT INTO recipe_ingredients (recipe_id, canonical_ingredient_id, original_text, quantity, unit) VALUES 
    (2, 7, '3 tablespoons olive oil', 3, 'tablespoon'),
    (2, 9, '3 cloves garlic, minced', 3, 'clove'),
    (2, 4, 'salt to taste', NULL, NULL),
    (2, 10, 'black pepper to taste', NULL, NULL);