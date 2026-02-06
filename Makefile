.PHONY: build run dev test clean deps lint

# Binary name
BINARY=lootstash-marketplace

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: deps build

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build the application
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY) main.go

# Run the application
run: build
	./$(BINARY) serve

# Run in development mode
dev:
	$(GOCMD) run main.go serve

# Run with custom port
dev-port:
	$(GOCMD) run main.go serve --port $(PORT)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY)
	rm -f coverage.out coverage.html

# Lint the code
lint:
	golangci-lint run

# Format code
fmt:
	$(GOCMD) fmt ./...

# Generate mocks (if using mockgen)
mocks:
	$(GOCMD) generate ./...

# Docker build
docker-build:
	docker build -t lootstash-marketplace-api .

# Docker run
docker-run:
	docker run -p 8081:8081 --env-file .env lootstash-marketplace-api

# Help
help:
	@echo "Available targets:"
	@echo "  all          - Download dependencies and build"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  build        - Build the application"
	@echo "  run          - Build and run the application"
	@echo "  dev          - Run in development mode"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  clean        - Clean build artifacts"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  help         - Show this help"
