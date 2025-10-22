# SAP Adaptor Makefile

.PHONY: build run test clean docker-build docker-run docker-compose-up docker-compose-down lint fmt

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=sap-adaptor
BINARY_UNIX=$(BINARY_NAME)_unix

# Build the application
build: swagger
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/server

# Build for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./cmd/server

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/server
	./$(BINARY_NAME)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Test simulator
test-simulator:
	$(GOBUILD) -o test-simulator -v ./cmd/test
	./test-simulator
	rm -f test-simulator

# Demo polling
demo-polling:
	$(GOBUILD) -o demo-polling -v ./cmd/demo
	./demo-polling
	rm -f demo-polling

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f coverage.out

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	$(GOCMD) fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate Swagger documentation
swagger:
	swag init -g cmd/server/main.go

# Docker build
docker-build:
	docker build -t $(BINARY_NAME) .

# Docker run
docker-run:
	docker run -p 8080:8080 --env-file env.example $(BINARY_NAME)

# Docker Compose up
docker-compose-up:
	docker-compose up -d

# Docker Compose down
docker-compose-down:
	docker-compose down

# Docker Compose logs
docker-compose-logs:
	docker-compose logs -f

# Development setup
dev-setup:
	cp env.example .env
	$(GOMOD) download
	$(GOMOD) tidy

# Production build
prod-build: build-linux
	docker build -t $(BINARY_NAME):latest .

# Help
help:
	@echo "Available targets:"
	@echo "  build              - Build the application"
	@echo "  build-linux        - Build for Linux"
	@echo "  run                - Build and run the application"
	@echo "  test               - Run tests"
	@echo "  test-simulator     - Test simulator mode functionality"
	@echo "  demo-polling       - Demo polling-based TECO detection"
	@echo "  clean              - Clean build artifacts"
	@echo "  deps               - Install dependencies"
	@echo "  fmt                - Format code"
	@echo "  lint               - Lint code"
	@echo "  swagger            - Generate Swagger documentation"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-run         - Run Docker container"
	@echo "  docker-compose-up  - Start with Docker Compose"
	@echo "  docker-compose-down- Stop Docker Compose"
	@echo "  docker-compose-logs- Show Docker Compose logs"
	@echo "  dev-setup          - Setup development environment"
	@echo "  prod-build         - Production build"
	@echo "  help               - Show this help"
