# Database Integration Tests

## Overview
Comprehensive database integration tests that validate schema, CRUD operations, constraints, and triggers for the Digital Recipes MVP.

## Test Coverage

### ✅ Schema Validation Tests
- All expected tables exist (users, recipes, canonical_ingredients, recipe_ingredients)
- Foreign key constraints are properly configured
- Indexes exist for performance optimization

### ✅ CRUD Operation Tests
- **Users table**: Create, Read, Update, Delete operations
- **Recipes table**: Full CRUD with status management and foreign keys
- **Canonical Ingredients table**: CRUD with uniqueness constraints
- **Recipe Ingredients table**: CRUD with complex relationships

### ✅ Data Integrity Tests
- Foreign key constraint enforcement
- Cascade deletions work properly
- Unique constraints prevent duplicates
- Default values are applied correctly

### ✅ Timestamp Trigger Tests
- `created_at` timestamps set automatically
- `updated_at` timestamps updated on modifications
- Triggers work across all tables

### ✅ Performance Tests
- Required indexes exist for efficient queries
- Index coverage for foreign key relationships

## Running the Tests

### Prerequisites
1. **Docker**: Required for test database container
2. **Go**: Version 1.21 or higher
3. **PostgreSQL Client**: For database connectivity

### Setup & Execution

#### Option 1: Using Automated Test Script (Recommended)
```bash
# Run the complete test suite with automated setup
cd /home/filipe-carneiro/projects/digital-recipes/api-service
sudo ./run_tests.sh

# Or with custom configuration
TEST_DB_USER=myuser TEST_DB_PASS=mypass sudo ./run_tests.sh
```

#### Option 2: Manual Docker Setup
```bash
# Set secure credentials (optional - script generates random password if not set)
export TEST_DB_USER="testuser"
export TEST_DB_PASS="$(openssl rand -base64 12)"
export TEST_DB_NAME="digital_recipes_test"
export TEST_DB_PORT="5433"

# Start test PostgreSQL container
sudo docker run --name postgres-test -d \
    -e POSTGRES_DB="${TEST_DB_NAME}" \
    -e POSTGRES_USER="${TEST_DB_USER}" \
    -e POSTGRES_PASSWORD="${TEST_DB_PASS}" \
    -p "${TEST_DB_PORT}:5432" \
    postgres:15

# Set test database URL
export TEST_DATABASE_URL="postgres://${TEST_DB_USER}:${TEST_DB_PASS}@localhost:${TEST_DB_PORT}/${TEST_DB_NAME}?sslmode=disable"
export DATABASE_URL="${TEST_DATABASE_URL}"

# Wait for database to be ready
until sudo docker exec postgres-test pg_isready -U "${TEST_DB_USER}" -d "${TEST_DB_NAME}"; do
    sleep 1
done

# Run tests
cd /home/filipe-carneiro/projects/digital-recipes/api-service
go test -v ./tests -timeout 30s

# Cleanup
sudo docker stop postgres-test
sudo docker rm postgres-test
```

#### Option 3: Using Existing PostgreSQL
```bash
# Set up test database in existing PostgreSQL
createdb digital_recipes_test
export TEST_DATABASE_URL="postgres://username:password@localhost:5432/digital_recipes_test?sslmode=disable"
export DATABASE_URL="${TEST_DATABASE_URL}"

# Run tests
cd /home/filipe-carneiro/projects/digital-recipes/api-service
go test -v ./tests -timeout 30s
```

#### Option 3: Skip Tests
If no test database is available, tests will be skipped with message:
```
TEST_DATABASE_URL not set, skipping database integration tests
```

## Recent Improvements

### Security Enhancements
- **Secure Credential Management**: Test script uses environment variables and generates random passwords
- **No Hardcoded Secrets**: All credentials are configurable via environment variables
- **Comprehensive Error Checking**: Specific assertion messages for different error types

### Performance Optimizations
- **Connection Pool Configuration**: Database connections use optimized pool settings (25 max open, 10 max idle, 5min lifetime)
- **Efficient Table Cleanup**: Uses `TRUNCATE RESTART IDENTITY CASCADE` for faster test data cleanup
- **Transaction-Based Operations**: Cleanup operations use transactions for atomicity

### Code Quality Improvements
- **Table Name Constants**: Centralized table name management reduces typos and improves maintainability
- **Reliable Timestamp Testing**: Eliminates arbitrary sleep statements with proper timestamp comparisons
- **Specific Error Assertions**: Tests validate exact error types (foreign key violations, unique constraints)
- **Magic Number Elimination**: Uses named constants instead of hardcoded values

## Test Structure

### Test Suite Organization
- **DatabaseIntegrationTestSuite**: Main test suite using testify/suite
- **SetupSuite**: Runs once before all tests (database connection & migrations)
- **SetupTest**: Runs before each test (cleanup test data)
- **TearDownSuite**: Runs once after all tests (cleanup resources)

### Key Test Functions
1. `TestDatabaseConnection` - Basic connectivity
2. `TestSchemaValidation` - Table and constraint validation
3. `TestUsersCRUD` - Complete user operations
4. `TestRecipesCRUD` - Recipe management with foreign keys
5. `TestCanonicalIngredientsCRUD` - Ingredient master data
6. `TestRecipeIngredientsCRUD` - Complex relationship management
7. `TestForeignKeyConstraints` - Data integrity enforcement
8. `TestTimestampTriggers` - Automatic timestamp management
9. `TestIndexesExist` - Performance optimization validation

## Test Data Management
- **Automatic Cleanup**: Each test starts with clean state
- **Transactional Isolation**: Tests don't interfere with each other
- **Realistic Test Data**: Uses representative recipe data
- **Edge Case Coverage**: Tests constraint violations and error conditions

## Expected Test Results
All tests should pass with output similar to:
```
=== RUN   TestDatabaseIntegrationTestSuite
=== RUN   TestDatabaseIntegrationTestSuite/TestDatabaseConnection
=== RUN   TestDatabaseIntegrationTestSuite/TestSchemaValidation
=== RUN   TestDatabaseIntegrationTestSuite/TestUsersCRUD
=== RUN   TestDatabaseIntegrationTestSuite/TestRecipesCRUD
=== RUN   TestDatabaseIntegrationTestSuite/TestCanonicalIngredientsCRUD
=== RUN   TestDatabaseIntegrationTestSuite/TestRecipeIngredientsCRUD
=== RUN   TestDatabaseIntegrationTestSuite/TestForeignKeyConstraints
=== RUN   TestDatabaseIntegrationTestSuite/TestTimestampTriggers
=== RUN   TestDatabaseIntegrationTestSuite/TestIndexesExist
--- PASS: TestDatabaseIntegrationTestSuite (X.XXs)
PASS
```

## Integration with TODO List
✅ **High Priority Foundation**
- [x] Write database integration tests for schema validation and basic CRUD operations

**Next Steps**: Move on to API endpoint tests after database tests pass.