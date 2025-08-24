# Lab Instrument Gateway Makefile

.PHONY: help build test clean proto docker-build docker-up docker-down deps lint fmt vet

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build the application
build: ## Build the application binary
	@echo "Building lab-gateway..."
	@go build -o bin/lab-gateway cmd/server/main.go

# Run tests
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/ tmp/ coverage.out coverage.html

# Generate protobuf files
proto: ## Generate protobuf Go files
	@echo "Generating protobuf files..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/lab_instrument.proto

# Install dependencies
deps: ## Install/update dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Lint code
lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run

# Format code
fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...

# Vet code
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

# Docker build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -f Dockerfile.dev -t lab-gateway:dev .

# Start development environment
docker-up: ## Start development environment with Docker Compose
	@echo "Starting development environment..."
	@docker-compose up -d

# Stop development environment
docker-down: ## Stop development environment
	@echo "Stopping development environment..."
	@docker-compose down

# View logs
logs: ## View application logs
	@docker-compose logs -f lab-gateway

# Database migration
migrate-up: ## Run database migrations
	@echo "Running database migrations..."
	@go run cmd/migrate/main.go -action=up

# Migration status
migrate-status: ## Check migration status
	@echo "Checking migration status..."
	@go run cmd/migrate/main.go -action=status

# Validate migrations
migrate-validate: ## Validate migration integrity
	@echo "Validating migrations..."
	@go run cmd/migrate/main.go -action=validate

# Connect to database
db-connect: ## Connect to PostgreSQL database
	@docker-compose exec postgres psql -U user -d lab_instruments

# Run development server
dev: ## Run development server with hot reloading
	@echo "Starting development server..."
	@air -c .air.toml

# Install development tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/air-verse/air@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Load test
load-test: ## Run load tests (requires ghz)
	@echo "Running load tests..."
	@ghz --insecure --proto proto/lab_instrument.proto \
		--call lab_instrument.LabInstrumentGateway.HealthCheck \
		--data '{"service": "lab-gateway"}' \
		--connections 100 --concurrency 1000 --duration 30s \
		localhost:9090

# Security scan
security-scan: ## Run security scan with gosec
	@echo "Running security scan..."
	@gosec ./...

# Generate mocks
mocks: ## Generate mocks for testing
	@echo "Generating mocks..."
	@mockgen -source=pkg/db/interfaces.go -destination=test/mocks/db_mocks.go

# Check dependencies for vulnerabilities
vuln-check: ## Check for known vulnerabilities
	@echo "Checking for vulnerabilities..."
	@go list -json -m all | nancy sleuth

# Full CI pipeline
ci: deps fmt vet lint test ## Run full CI pipeline

# Production build
build-prod: ## Build production binary
	@echo "Building production binary..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bin/lab-gateway cmd/server/main.go