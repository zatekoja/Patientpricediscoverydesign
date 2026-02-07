.PHONY: help build run test test-coverage clean deps mocks migrate-up migrate-down docker-up docker-down test-provider-integration test-provider-unit

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

test-db-up: ## Start test database
	@echo "Starting test database..."
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for database to be ready..."
	@sleep 3

test-db-down: ## Stop test database
	@echo "Stopping test database..."
	docker-compose -f docker-compose.test.yml down -v

test-integration: ## Run integration tests (requires live database)
	@echo "Running integration tests..."
	@echo "Make sure test database is running (make test-db-up)..."
	TEST_DB_HOST=localhost TEST_DB_PORT=5440 TEST_DB_USER=postgres TEST_DB_PASSWORD=postgres TEST_DB_NAME=patient_price_discovery_test TEST_DB_SSLMODE=disable go test -v -tags=integration ./tests/integration/...

test-integration-full: test-db-up test-integration test-db-down ## Start test DB, run integration tests, stop test DB
test-provider-integration: ## Run provider API integration test (requires docker-compose.test.yml)
	@echo "Running provider API integration test..."
	PROVIDER_API_BASE_URL=http://localhost:3002/api/v1 PROVIDER_ID=file_price_list node tests/integration/provider_api_integration_test.mjs

test-provider-graphql-integration: ## Run GraphQL provider integration test (requires provider API)
	@echo "Running GraphQL provider integration test..."
	PROVIDER_API_BASE_URL=http://localhost:3002/api/v1 PROVIDER_ID=file_price_list go test -v -tags=integration ./tests/integration -run TestProviderPriceCurrentGraphQLIntegration

test-provider-unit: ## Run provider unit tests (parser + file provider)
	@echo "Running provider unit tests..."
	./node_modules/.bin/ts-node tests/unit/price_list_parser.test.ts
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
	go install github.com/99designs/gqlgen@latest

# GraphQL targets
graphql-generate: ## Generate GraphQL code from schema
	@echo "Generating GraphQL code..."
	go run github.com/99designs/gqlgen generate
	@echo "GraphQL code generated successfully!"

graphql-init: ## Initialize GraphQL project (first time only)
	go run github.com/99designs/gqlgen init

run-graphql: ## Run GraphQL server (Query Service)
	@echo "Running GraphQL server..."
	go run cmd/graphql/main.go

test-query: ## Run query service tests
	@echo "Running query service tests..."
	go test -v ./internal/query/...

test-graphql: ## Run GraphQL resolver tests
	@echo "Running GraphQL tests..."
	go test -v ./internal/graphql/...

build-graphql: ## Build GraphQL server binary
	@echo "Building GraphQL server..."
	go build -o bin/graphql cmd/graphql/main.go

build-indexer: ## Build indexer binary
	@echo "Building indexer..."
	go build -o bin/indexer cmd/indexer/main.go

index-data: ## Index all data from PostgreSQL to Typesense
	@echo "Indexing data to Typesense..."
	go run cmd/indexer/main.go --action=reindex

index-data-dry-run: ## Dry run of data indexing
	@echo "Dry run indexing..."
	go run cmd/indexer/main.go --action=reindex --dry-run

docker-logs-graphql: ## View GraphQL server logs
	docker-compose logs -f graphql

docker-up-graphql: ## Start GraphQL service
	docker-compose up -d graphql

.DEFAULT_GOAL := help
# GraphQL Commands
graphql-generate: ## Generate GraphQL code from schema
	@echo "Generating GraphQL code..."
	go run github.com/99designs/gqlgen generate
graphql-build: ## Build GraphQL server
	@echo "Building GraphQL server..."
	go build -o bin/graphql-server ./cmd/graphql/
graphql-run: ## Run GraphQL server
	@echo "Running GraphQL server..."
	go run ./cmd/graphql/
test-graphql: ## Run GraphQL resolver tests
	@echo "Running GraphQL tests..."
	go test -v ./internal/graphql/resolvers/...
test-query: ## Run query service tests
	@echo "Running query service tests..."
	go test -v ./internal/query/services/...
