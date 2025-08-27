# Professional Makefile for TODO API

.PHONY: help dev test test-watch clean logs build

# Default target
help:
	@echo "Available commands:"
	@echo "  dev          - Start development environment"
	@echo "  test         - Run all tests"
	@echo "  test-watch   - Run tests with file watching"
	@echo "  clean        - Clean up all containers and volumes"
	@echo "  logs         - Show application logs"
	@echo "  build        - Build the application"

# Development environment
dev:
	@echo "🚀 Starting development environment..."
	docker-compose --env-file .env up --build -d
	@echo "✅ Development server running at http://localhost:8080"
	@echo "✅ Database available at localhost:5432"

# Run tests
test:
	@echo "🧪 Running tests..."
	docker-compose --env-file .env.test -f docker-compose.test.yml up --build --abort-on-container-exit
	@echo "🧹 Cleaning up test containers..."
	docker-compose --env-file .env.test -f docker-compose.test.yml down

# Run tests with watching (for development)
test-watch:
	@echo "🔍 Running tests in watch mode..."
	docker-compose --env-file .env.test -f docker-compose.test.yml up --build db-test -d
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Running tests..."
	docker-compose --env-file .env.test -f docker-compose.test.yml run --rm app-test go test -v ./...

# Show logs
logs:
	docker-compose logs -f app-dev

# Build application
build:
	@echo "🔨 Building application..."
	docker-compose build

# Clean everything
clean:
	@echo "🧹 Cleaning up..."
	docker-compose --env-file .env down -v
	docker-compose --env-file .env.test -f docker-compose.test.yml down -v
	docker system prune -f
	@echo "✅ Cleanup complete"

# Stop development environment
stop:
	@echo "🛑 Stopping development environment..."
	docker-compose --env-file .env down
	@echo "✅ Development environment stopped"