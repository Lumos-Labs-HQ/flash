# Graft CLI Makefile

# Variables
BINARY_NAME=graft
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe
BUILD_DIR=build
LDFLAGS=-s -w -extldflags "-static"

# Default target now builds for all platforms
.PHONY: all
all: clean build-all

# Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)

	# Linux AMD64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/graft-linux-x64 .

	# Linux ARM64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/graft-linux-arm64 .

	# Windows AMD64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/graft-win32-x64.exe .

	# macOS AMD64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/graft-darwin-x64 .

	# macOS ARM64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/graft-darwin-arm64 .

	@echo "Cross-platform build complete in $(BUILD_DIR)/"

# Compress binaries with UPX (optional - requires UPX installed)
.PHONY: compress
compress: build-all
	@echo "Compressing binaries with UPX..."
	@if command -v upx &> /dev/null; then \
		upx --best --lzma $(BUILD_DIR)/graft-linux-x64; \
		upx --best --lzma $(BUILD_DIR)/graft-linux-arm64; \
		upx --best --lzma $(BUILD_DIR)/graft-win32-x64.exe; \
		upx --best --lzma $(BUILD_DIR)/graft-darwin-x64; \
		upx --best --lzma $(BUILD_DIR)/graft-darwin-arm64; \
		echo "Compression complete!"; \
	else \
		echo "UPX not found. Install from https://upx.github.io/"; \
		echo "Skipping compression..."; \
	fi

# Install the binary to GOPATH/bin (Linux build used by default)
.PHONY: install
install: build-all
	@echo "Installing $(BINARY_NAME) (Linux version) to $(GOPATH)/bin..."
	cp $(BUILD_DIR)/$(BINARY_UNIX) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installation complete"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BINARY_NAME) $(BINARY_UNIX) $(BINARY_WINDOWS) $(BUILD_DIR) release
	@echo "Clean complete"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run

# Run the CLI with help (Linux binary used by default)
.PHONY: run
run: build-all
	./$(BUILD_DIR)/$(BINARY_UNIX) --help

# Development setup
.PHONY: dev-setup
dev-setup: deps
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@echo "Development setup complete"

# Create a release
.PHONY: release
release: clean build-all
	@echo "Creating release..."
	@mkdir -p release
	@cp $(BUILD_DIR)/* release/
	@echo "Release created in release/"

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all         - Clean and build for all platforms"
	@echo "  build-all   - Build for multiple platforms"
	@echo "  compress    - Compress binaries with UPX (requires UPX)"
	@echo "  install     - Install Linux binary to GOPATH/bin"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  deps        - Download dependencies"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  run         - Build and run Linux binary with --help"
	@echo "  dev-setup   - Setup development environment"
	@echo "  release     - Create release build"
	@echo "  help        - Show this help"
