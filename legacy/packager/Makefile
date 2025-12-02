.PHONY: build clean test install deps keygen help

# Binary name
BINARY_NAME=packager
BUILD_DIR=build
INSTALL_DIR=/usr/local/bin

# Build the packager binary
build: deps
	@echo "Building packager..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/packager
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies ready"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	@echo "✓ Clean complete"

# Install to system
install: build
	@echo "Installing packager to $(INSTALL_DIR)..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@echo "✓ Installed: $(INSTALL_DIR)/$(BINARY_NAME)"

# Generate a keypair (helper target)
keygen:
	@echo "Generating Ed25519 keypair..."
	@mkdir -p keys
	@$(BUILD_DIR)/$(BINARY_NAME) keygen keys/private.key
	@echo "✓ Keys generated in keys/ directory"

# Show help
help:
	@echo "LibreSeed Packager - Makefile targets:"
	@echo ""
	@echo "  make build      - Build the packager binary"
	@echo "  make deps       - Download and tidy Go dependencies"
	@echo "  make test       - Run all tests"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make install    - Install packager to system (requires sudo)"
	@echo "  make keygen     - Generate an Ed25519 keypair (requires build first)"
	@echo "  make help       - Show this help message"
	@echo ""
