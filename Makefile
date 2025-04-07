# Simple Makefile for Go project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GORUN=$(GOCMD) run
GOLINT=golangci-lint

# Binary name
BINARY_NAME=icecast-ripper
BINARY_PATH=./bin/$(BINARY_NAME)

# Source files entrypoint
MAIN_GO=./cmd/icecast-ripper/main.go

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_PATH) $(MAIN_GO)
	@echo "$(BINARY_NAME) built successfully."

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BINARY_PATH)

# Run linters
lint:
	@echo "Running linters..."
	$(GOLINT) run ./...

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_PATH)
	@echo "Cleaned."

# Get dependencies
deps:
	@echo "Getting dependencies..."
	$(GOGET) ./...

.PHONY: all build run lint test clean deps
