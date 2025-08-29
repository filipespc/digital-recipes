.PHONY: help build test run clean db-up db-down db-reset migrate

help:
	@echo "Available commands:"
	@echo "  build        - Build both services"
	@echo "  test         - Run tests for both services"
	@echo "  run          - Start services with docker-compose"
	@echo "  clean        - Stop and clean docker containers"
	@echo "  db-up        - Start database only"
	@echo "  db-down      - Stop database"
	@echo "  db-reset     - Reset database (clean + up)"
	@echo "  migrate      - Run database migrations"

build:
	@echo "Building API service..."
	cd api-service && go build -o bin/api-service ./main.go
	cd api-service && go build -o bin/migrate ./cmd/migrate/main.go
	@echo "Installing parser service dependencies..."
	cd parser-service && pip install -r requirements.txt

test:
	@echo "Running API service tests..."
	cd api-service && go test ./...
	@echo "Running parser service tests..."
	cd parser-service && python -m pytest

run:
	docker-compose up --build

clean:
	docker-compose down -v

db-up:
	docker-compose up -d postgres

db-down:
	docker-compose stop postgres

db-reset: clean db-up
	@echo "Database reset complete"

migrate:
	@echo "Running database migrations..."
	cd api-service && ./bin/migrate -command=up