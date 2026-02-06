.PHONY: help build run test test-coverage clean deps mocks migrate-up migrate-down docker-up docker-down

help: ## Display this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

deps: ## Install dependencies
	go mod download
	go mod tidy

mocks: ## Generate mocks using mockery
	@echo "Generating mocks..."
	mockery

build: ## Build the application
	@echo "Building application..."
	go build -o bin/api cmd/api/main.go

run: ## Run the application
	@echo "Running application..."
	go run cmd/api/main.go

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	go test -race ./...

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

migrate-up: ## Run database migrations
	@echo "Running migrations..."
	psql -d $(DB_NAME) -f migrations/001_initial_schema.sql

migrate-down: ## Rollback database migrations
	@echo "Rolling back migrations..."
	# Add rollback SQL here

docker-up: ## Start Docker containers
	docker-compose up -d

docker-down: ## Stop Docker containers
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f

install-tools: ## Install development tools
	go install github.com/vektra/mockery/v2@latest
	go install golang.org/x/tools/cmd/goimports@latest

.DEFAULT_GOAL := help
