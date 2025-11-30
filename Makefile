# LibreSeed Makefile
# Feature 002: CLI Rename & Install Script (T003)

# Version from VERSION file
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")

# Build configuration
GO := go
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Directories
BIN_DIR := bin
CMD_DIR := cmd
INSTALL_DIR ?= /usr/local/bin

# Binaries
CLI_BINARY := lbs
DAEMON_BINARY := lbsd
BINARIES := $(CLI_BINARY) $(DAEMON_BINARY)

# Module
MODULE := github.com/libreseed/libreseed

# Default target
.DEFAULT_GOAL := build

# Help target
.PHONY: help
help:
	@echo "LibreSeed Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build        Build binaries (lbs, lbsd)"
	@echo "  make clean        Remove build artifacts"
	@echo "  make test         Run test suite"
	@echo "  make lint         Run golangci-lint"
	@echo "  make checksums    Generate SHA256SUMS for binaries"
	@echo "  make install      Install binaries to $(INSTALL_DIR)"
	@echo "  make uninstall    Remove installed binaries"
	@echo "  make version      Show version information"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  GOOS=$(GOOS)"
	@echo "  GOARCH=$(GOARCH)"
	@echo "  INSTALL_DIR=$(INSTALL_DIR)"
	@echo ""

# Build all binaries
.PHONY: build
build: $(BIN_DIR)/$(CLI_BINARY) $(BIN_DIR)/$(DAEMON_BINARY)
	@echo "Build complete: $(BINARIES)"

# Build CLI binary
$(BIN_DIR)/$(CLI_BINARY): $(wildcard $(CMD_DIR)/lbs/*.go) $(wildcard pkg/**/*.go)
	@mkdir -p $(BIN_DIR)
	@echo "Building $(CLI_BINARY) ($(VERSION))..."
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(CLI_BINARY) ./$(CMD_DIR)/lbs
	@echo "✓ $(BIN_DIR)/$(CLI_BINARY)"

# Build daemon binary
$(BIN_DIR)/$(DAEMON_BINARY): $(wildcard $(CMD_DIR)/lbsd/*.go) $(wildcard pkg/**/*.go)
	@mkdir -p $(BIN_DIR)
	@echo "Building $(DAEMON_BINARY) ($(VERSION))..."
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(DAEMON_BINARY) ./$(CMD_DIR)/lbsd
	@echo "✓ $(BIN_DIR)/$(DAEMON_BINARY)"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -f SHA256SUMS
	@echo "✓ Clean complete"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v -race -timeout 30s ./...
	@echo "✓ Tests passed"

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

# Run linter
.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "✓ Lint complete"; \
	else \
		echo "✗ golangci-lint not installed"; \
		echo "  Install: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

# Generate SHA256 checksums
.PHONY: checksums
checksums: build
	@echo "Generating SHA256SUMS..."
	@cd $(BIN_DIR) && sha256sum $(BINARIES) > SHA256SUMS
	@echo "✓ $(BIN_DIR)/SHA256SUMS"
	@cat $(BIN_DIR)/SHA256SUMS

# Verify checksums
.PHONY: verify
verify:
	@if [ ! -f $(BIN_DIR)/SHA256SUMS ]; then \
		echo "✗ SHA256SUMS not found. Run 'make checksums' first."; \
		exit 1; \
	fi
	@echo "Verifying checksums..."
	@cd $(BIN_DIR) && sha256sum -c SHA256SUMS
	@echo "✓ Checksums verified"

# Install binaries (requires checksums)
.PHONY: install
install: checksums
	@echo "Installing binaries to $(INSTALL_DIR)..."
	@install -m 755 $(BIN_DIR)/$(CLI_BINARY) $(INSTALL_DIR)/$(CLI_BINARY)
	@echo "✓ Installed $(INSTALL_DIR)/$(CLI_BINARY)"
	@install -m 755 $(BIN_DIR)/$(DAEMON_BINARY) $(INSTALL_DIR)/$(DAEMON_BINARY)
	@echo "✓ Installed $(INSTALL_DIR)/$(DAEMON_BINARY)"
	@echo ""
	@echo "Installation complete!"
	@echo "  CLI:    $(INSTALL_DIR)/$(CLI_BINARY)"
	@echo "  Daemon: $(INSTALL_DIR)/$(DAEMON_BINARY)"
	@echo ""
	@echo "Run '$(CLI_BINARY) help' to get started."

# Uninstall binaries
.PHONY: uninstall
uninstall:
	@echo "Uninstalling binaries from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/$(CLI_BINARY)
	@echo "✓ Removed $(INSTALL_DIR)/$(CLI_BINARY)"
	@rm -f $(INSTALL_DIR)/$(DAEMON_BINARY)
	@echo "✓ Removed $(INSTALL_DIR)/$(DAEMON_BINARY)"
	@echo "✓ Uninstall complete"

# Show version
.PHONY: version
version:
	@echo "LibreSeed version $(VERSION)"
	@echo "Go version: $(shell $(GO) version)"
	@echo "Platform: $(GOOS)/$(GOARCH)"

# Development targets
.PHONY: dev
dev: clean build test
	@echo "✓ Development build complete"

# CI target (lint + test + build)
.PHONY: ci
ci: lint test build checksums
	@echo "✓ CI pipeline complete"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "✓ Format complete"

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy
	@echo "✓ Dependencies tidied"

# Vendor dependencies
.PHONY: vendor
vendor:
	@echo "Vendoring dependencies..."
	$(GO) mod vendor
	@echo "✓ Dependencies vendored"

# Show module info
.PHONY: modinfo
modinfo:
	@echo "Module: $(MODULE)"
	@echo "Go version: $(shell $(GO) version)"
	@echo ""
	@$(GO) list -m all
