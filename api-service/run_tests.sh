#!/bin/bash

# Database Integration Test Runner for Digital Recipes API
# This script sets up a test database and runs integration tests

set -e

echo "🧪 Setting up database integration tests..."

# Secure test database configuration using environment variables
TEST_DB_USER="${TEST_DB_USER:-testuser}"
TEST_DB_NAME="${TEST_DB_NAME:-digital_recipes_test}"
TEST_DB_PORT="${TEST_DB_PORT:-5433}"

# Generate secure random password if not provided
if [ -z "${TEST_DB_PASS}" ]; then
    TEST_DB_PASS=$(openssl rand -base64 12 2>/dev/null || echo "fallback_test_pass_$(date +%s)")
fi

# Export all variables for use in docker and tests
export TEST_DB_USER
export TEST_DB_NAME  
export TEST_DB_PORT
export TEST_DB_PASS

export TEST_DATABASE_URL="postgres://${TEST_DB_USER}:${TEST_DB_PASS}@localhost:${TEST_DB_PORT}/${TEST_DB_NAME}?sslmode=disable"
export DATABASE_URL="${TEST_DATABASE_URL}"

echo "Using test database: ${TEST_DB_NAME} on port ${TEST_DB_PORT}"
echo "Database URL: ${DATABASE_URL}"

# Always clean up any existing test container to ensure fresh start
echo "🧹 Cleaning up any existing test container..."
docker stop postgres-test 2>/dev/null || true
docker rm postgres-test 2>/dev/null || true

# Check if PostgreSQL test container is running on specified port
if ! nc -z localhost ${TEST_DB_PORT}; then
    echo "📦 Starting test PostgreSQL container..."
    docker run --name postgres-test -d \
        -e POSTGRES_DB="${TEST_DB_NAME}" \
        -e POSTGRES_USER="${TEST_DB_USER}" \
        -e POSTGRES_PASSWORD="${TEST_DB_PASS}" \
        -p "${TEST_DB_PORT}:5432" \
        postgres:15 || echo "Container may already exist"
    
    # Wait for PostgreSQL to be ready
    echo "⏳ Waiting for PostgreSQL to be ready..."
    until docker exec postgres-test pg_isready -U "${TEST_DB_USER}" -d "${TEST_DB_NAME}"; do
        sleep 1
    done
    echo "✅ PostgreSQL test database is ready!"
else
    echo "✅ PostgreSQL test database already running"
fi

# Run the tests
echo "🚀 Running database integration tests..."
cd /home/filipe-carneiro/projects/digital-recipes/api-service
go test -v ./tests -timeout 30s

# Test result
if [ $? -eq 0 ]; then
    echo "✅ All database integration tests passed!"
else
    echo "❌ Some tests failed!"
    exit 1
fi

echo "🧹 Cleaning up test container..."
docker stop postgres-test || true
docker rm postgres-test || true

echo "🎉 Database integration tests completed!"