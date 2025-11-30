#!/bin/bash
# End-to-End Package Management Test Script
# Tests the complete workflow: start daemon → add package → list packages → stop daemon

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
LBS_BIN="./lbs"
LBSD_BIN="./lbsd"
TEST_PKG_NAME="test-package"
TEST_PKG_VERSION="1.0.0"
TEST_PKG_DESC="Test package for end-to-end testing"
TEST_FILE="/tmp/libreseed-test-package.tar.gz"

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

cleanup() {
    log_info "Cleaning up..."
    if [ -f "$TEST_FILE" ]; then
        rm -f "$TEST_FILE"
        log_info "Removed test file: $TEST_FILE"
    fi
    
    # Stop daemon if running
    if $LBS_BIN status &>/dev/null; then
        log_info "Stopping daemon..."
        $LBS_BIN stop || true
        sleep 1
    fi
}

# Set up cleanup trap
trap cleanup EXIT

# Main test flow
main() {
    echo "=========================================="
    echo "LibreSeed Package Management E2E Test"
    echo "=========================================="
    echo ""

    # Step 1: Build binaries
    log_info "Building binaries..."
    if ! go build -o "$LBS_BIN" ./cmd/lbs; then
        log_error "Failed to build lbs CLI"
        exit 1
    fi
    if ! go build -o "$LBSD_BIN" ./cmd/lbsd; then
        log_error "Failed to build lbsd daemon"
        exit 1
    fi
    log_success "Binaries built successfully"
    echo ""

    # Step 2: Check version
    log_info "Checking version..."
    VERSION=$($LBS_BIN version)
    echo "  $VERSION"
    log_success "Version check passed"
    echo ""

    # Step 3: Create test package file
    log_info "Creating test package file..."
    echo "Test package content" | gzip > "$TEST_FILE"
    if [ ! -f "$TEST_FILE" ]; then
        log_error "Failed to create test file"
        exit 1
    fi
    log_success "Test package created: $TEST_FILE"
    echo ""

    # Step 4: Start daemon
    log_info "Starting daemon..."
    if $LBS_BIN status &>/dev/null; then
        log_warning "Daemon already running, stopping it first..."
        $LBS_BIN stop
        sleep 2
    fi
    
    $LBS_BIN start
    sleep 3  # Give daemon time to start
    
    if ! $LBS_BIN status; then
        log_error "Daemon failed to start"
        exit 1
    fi
    log_success "Daemon started successfully"
    echo ""

    # Step 5: Add package
    log_info "Adding package..."
    if ! $LBS_BIN add "$TEST_FILE" "$TEST_PKG_NAME" "$TEST_PKG_VERSION" "$TEST_PKG_DESC"; then
        log_error "Failed to add package"
        exit 1
    fi
    log_success "Package added successfully"
    echo ""

    # Step 6: List packages
    log_info "Listing packages..."
    if ! $LBS_BIN list; then
        log_error "Failed to list packages"
        exit 1
    fi
    log_success "Package listing successful"
    echo ""

    # Step 7: Verify package appears in list
    log_info "Verifying package in list..."
    if ! $LBS_BIN list | grep -q "$TEST_PKG_NAME"; then
        log_error "Package not found in list"
        exit 1
    fi
    log_success "Package verified in list"
    echo ""

    # Step 8: Check daemon stats
    log_info "Checking daemon statistics..."
    if ! $LBS_BIN stats; then
        log_error "Failed to get daemon stats"
        exit 1
    fi
    log_success "Statistics retrieved successfully"
    echo ""

    # Step 9: Stop daemon
    log_info "Stopping daemon..."
    if ! $LBS_BIN stop; then
        log_error "Failed to stop daemon"
        exit 1
    fi
    sleep 1
    
    if $LBS_BIN status &>/dev/null; then
        log_error "Daemon still running after stop command"
        exit 1
    fi
    log_success "Daemon stopped successfully"
    echo ""

    # Final summary
    echo "=========================================="
    log_success "ALL TESTS PASSED! ✓"
    echo "=========================================="
    echo ""
    echo "Summary:"
    echo "  ✓ Binaries built"
    echo "  ✓ Version check"
    echo "  ✓ Test file created"
    echo "  ✓ Daemon started"
    echo "  ✓ Package added"
    echo "  ✓ Package listed"
    echo "  ✓ Package verified"
    echo "  ✓ Stats retrieved"
    echo "  ✓ Daemon stopped"
    echo ""
}

# Run main function
main "$@"
