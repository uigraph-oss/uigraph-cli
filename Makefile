.PHONY: build test clean install run-example

# Build configuration
BINARY_NAME=uigraph
BUILD_DIR=bin
GO=go
MAIN_PATH=./main.go

# Build the CLI
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@GOOS=linux GOARCH=arm64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	@GOOS=darwin GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	@GOOS=darwin GOARCH=arm64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@GOOS=windows GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "✓ Cross-platform builds complete"

# Run tests
test:
	@echo "Running tests..."
	@$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests complete"

# Run tests with coverage report
test-coverage: test
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

# Install the CLI locally
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "✓ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Run example with dry-run
run-example: build
	@echo "Running example (dry-run)..."
	@cd examples && ../$(BUILD_DIR)/$(BINARY_NAME) sync --dry-run

# Format code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...
	@echo "✓ Format complete"

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run ./...
	@echo "✓ Lint complete"

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@$(GO) mod tidy
	@echo "✓ Tidy complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GO) mod download
	@echo "✓ Dependencies downloaded"

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the CLI binary"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Remove build artifacts"
	@echo "  install       - Install CLI locally"
	@echo "  run-example   - Run example with dry-run"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  tidy          - Tidy dependencies"
	@echo "  deps          - Download dependencies"
	@echo "  help          - Show this help message"
