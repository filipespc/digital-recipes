# Database Schema and Migrations

This directory contains the database schema, migrations, and database utilities for the Digital Recipes API service.

## Schema Overview

The database schema implements the design from the ADR with four main tables:

### Tables

1. **users** - User accounts
   - `id` - Primary key
   - `email` - Unique email address  
   - `name` - User display name
   - Timestamps: `created_at`, `updated_at`

2. **recipes** - Recipe information
   - `id` - Primary key
   - `title` - Recipe name
   - `servings` - Number of servings
   - `instructions` - Cooking instructions
   - `tips` - Additional cooking tips
   - `status` - Processing status (processing, review_required, published)
   - `user_id` - Foreign key to users table
   - Timestamps: `created_at`, `updated_at`

3. **canonical_ingredients** - Master ingredient list
   - `id` - Primary key
   - `name` - Canonical ingredient name (e.g., "Egg", "All-Purpose Flour")
   - `is_approved` - Whether ingredient is approved for use
   - Timestamps: `created_at`, `updated_at`

4. **recipe_ingredients** - Links recipes to ingredients
   - `id` - Primary key
   - `recipe_id` - Foreign key to recipes table
   - `canonical_ingredient_id` - Foreign key to canonical_ingredients table (nullable)
   - `original_text` - Original text from recipe (e.g., "2 large eggs, beaten")
   - `quantity` - Parsed quantity amount
   - `unit` - Parsed unit of measurement
   - Timestamps: `created_at`, `updated_at`

## Migrations

### Migration Files

- **001_initial_schema.up.sql** - Creates all tables, indexes, and triggers
- **001_initial_schema.down.sql** - Drops all tables for rollback
- **002_seed_data.up.sql** - Development seed data with test users and recipes
- **002_seed_data.down.sql** - Removes seed data

### Running Migrations

Migrations are automatically run when the API service starts. You can also run them manually:

```bash
# Build migration tool
make build

# Run migrations manually
make migrate

# Or use the migration CLI directly
cd api-service
./bin/migrate -command=up
```

### Migration Features

- **Automatic timestamps** - All tables have `created_at` and `updated_at` fields
- **Cascade deletes** - Dependent records are cleaned up automatically
- **Indexes** - Performance indexes on foreign keys and search columns
- **Check constraints** - Data validation at database level
- **Migration tracking** - `schema_migrations` table tracks applied migrations

## Development Data

The seed migration includes:
- 2 test users
- 15 common canonical ingredients (eggs, flour, etc.)
- 2 sample recipes with linked ingredients

This data is perfect for development and testing the recipe management workflow.