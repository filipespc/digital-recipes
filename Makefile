.PHONY: help build test run clean

help:
	@echo "Available commands:"
	@echo "  build        - Build both services"
	@echo "  test         - Run tests for both services"
	@echo "  run          - Start services with docker-compose"
	@echo "  clean        - Stop and clean docker containers"

build:
	@echo "Building API service..."
	cd api-service && go build -o bin/api-service ./main.go
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