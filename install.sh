#!/usr/bin/env bash
# LibreSeed Installation Script
# Builds and installs lbs (CLI) and lbsd (daemon) binaries with checksum verification and rollback

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/usr/local/bin"
DATA_DIR="$HOME/.local/share/libreseed"
BACKUP_DIR="${DATA_DIR}/backup"
MANIFEST_FILE="${DATA_DIR}/.install_manifest"
REQUIRED_GO_VERSION="1.21"

# Global state for rollback
INSTALLATION_STARTED=0
BINARIES_BACKED_UP=0
BINARIES_INSTALLED=0

# Trap for rollback on failure
trap 'rollback_on_failure $?' EXIT

# Helper functions
error() {
    echo -e "${RED}ERROR: $1${NC}" >&2
}

success() {
    echo -e "${GREEN}$1${NC}"
}

warning() {
    echo -e "${YELLOW}WARNING: $1${NC}"
}

info() {
    echo -e "${BLUE}$1${NC}"
}

# T011: Platform detection
detect_platform() {
    local os=""
    local arch=""
    
    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        FreeBSD*)   os="freebsd" ;;
        OpenBSD*)   os="openbsd" ;;
        *)          os="unknown" ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        i386|i686)      arch="386" ;;
        aarch64|arm64)  arch="arm64" ;;
        armv7l|armv6l)  arch="arm" ;;
        *)              arch="unknown" ;;
    esac
    
    echo "${os}/${arch}"
}

# T010: Check prerequisites
check_prerequisites() {
    info "Checking prerequisites..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        error "Go is not installed. Please install Go ${REQUIRED_GO_VERSION} or later."
        exit 1
    fi
    
    local go_version=$(go version | awk '{print $3}' | sed 's/go//')
    local major=$(echo "$go_version" | cut -d. -f1)
    local minor=$(echo "$go_version" | cut -d. -f2)
    local required_major=$(echo "$REQUIRED_GO_VERSION" | cut -d. -f1)
    local required_minor=$(echo "$REQUIRED_GO_VERSION" | cut -d. -f2)
    
    if [ "$major" -lt "$required_major" ] || ([ "$major" -eq "$required_major" ] && [ "$minor" -lt "$required_minor" ]); then
        error "Go version ${go_version} is too old. Minimum required: ${REQUIRED_GO_VERSION}"
        exit 1
    fi
    
    success "  ✓ Go version ${go_version} detected"
    
    # Check Make
    if ! command -v make &> /dev/null; then
        error "Make is not installed. Please install Make."
        exit 1
    fi
    success "  ✓ Make detected"
    
    # Check for checksum tool
    if command -v sha256sum &> /dev/null; then
        success "  ✓ sha256sum detected"
    elif command -v shasum &> /dev/null; then
        success "  ✓ shasum detected"
    else
        error "Neither sha256sum nor shasum found. Please install coreutils or equivalent."
        exit 1
    fi
    
    # Check write permissions
    if [ ! -w "$INSTALL_DIR" ] && ! command -v sudo &> /dev/null; then
        error "No write permissions to ${INSTALL_DIR} and sudo not available."
        exit 1
    fi
    
    success "  ✓ All prerequisites met"
}

# T020: Create data directories
create_data_directories() {
    info "Creating data directories..."
    
    mkdir -p "${DATA_DIR}" || {
        error "Failed to create data directory: ${DATA_DIR}"
        exit 1
    }
    
    mkdir -p "${BACKUP_DIR}" || {
        error "Failed to create backup directory: ${BACKUP_DIR}"
        exit 1
    }
    
    success "  ✓ Data directories created at ${DATA_DIR}"
}

# T012: Backup existing binaries
backup_existing_binaries() {
    info "Checking for existing installation..."
    
    local backup_needed=0
    local timestamp=$(date +%Y%m%d_%H%M%S)
    
    if [ -f "${INSTALL_DIR}/lbs" ]; then
        info "  Found existing lbs binary"
        cp "${INSTALL_DIR}/lbs" "${BACKUP_DIR}/lbs.${timestamp}" 2>/dev/null || {
            # If regular copy fails, try with sudo
            if [ ! -w "$INSTALL_DIR" ]; then
                sudo cp "${INSTALL_DIR}/lbs" "${BACKUP_DIR}/lbs.${timestamp}" || {
                    warning "  Failed to backup lbs binary"
                }
            fi
        }
        backup_needed=1
    fi
    
    if [ -f "${INSTALL_DIR}/lbsd" ]; then
        info "  Found existing lbsd binary"
        cp "${INSTALL_DIR}/lbsd" "${BACKUP_DIR}/lbsd.${timestamp}" 2>/dev/null || {
            # If regular copy fails, try with sudo
            if [ ! -w "$INSTALL_DIR" ]; then
                sudo cp "${INSTALL_DIR}/lbsd" "${BACKUP_DIR}/lbsd.${timestamp}" || {
                    warning "  Failed to backup lbsd binary"
                }
            fi
        }
        backup_needed=1
    fi
    
    if [ $backup_needed -eq 1 ]; then
        BINARIES_BACKED_UP=1
        success "  ✓ Existing binaries backed up to ${BACKUP_DIR}"
        echo "${timestamp}" > "${MANIFEST_FILE}"
    else
        info "  No existing installation found"
    fi
}

# T014: Build binaries using Makefile
build_with_make() {
    info "Building binaries with Make..."
    
    if [ ! -f "Makefile" ]; then
        error "Makefile not found. Are you running this from the project root?"
        exit 1
    fi
    
    # Clean previous builds
    make clean &> /dev/null || true
    
    # Build binaries
    if ! make build; then
        error "Build failed. Check the output above for details."
        exit 1
    fi
    
    success "  ✓ Binaries built successfully"
}

# T015: Generate checksums
generate_checksums() {
    info "Generating checksums..."
    
    if ! make checksums; then
        error "Checksum generation failed."
        exit 1
    fi
    
    if [ ! -f "bin/SHA256SUMS" ]; then
        error "SHA256SUMS file not found after generation."
        exit 1
    fi
    
    success "  ✓ Checksums generated"
}

# T016: Verify checksums
verify_checksums() {
    info "Verifying checksums..."
    
    if ! make verify; then
        error "Checksum verification failed. Binaries may be corrupted."
        exit 1
    fi
    
    success "  ✓ Checksums verified successfully"
}

# T017: Install binaries
install_binaries() {
    info "Installing binaries to ${INSTALL_DIR}..."
    
    # Check if we need sudo
    local SUDO=""
    if [ ! -w "$INSTALL_DIR" ]; then
        info "  Administrator privileges required for installation."
        SUDO="sudo"
    fi
    
    # Use Makefile install target
    if ! ${SUDO} make install INSTALL_DIR="${INSTALL_DIR}"; then
        error "Installation failed."
        exit 1
    fi
    
    BINARIES_INSTALLED=1
    success "  ✓ Binaries installed to ${INSTALL_DIR}"
    
    # Record installation in manifest
    echo "installed=$(date -Iseconds)" >> "${MANIFEST_FILE}"
    echo "version=$(cat VERSION)" >> "${MANIFEST_FILE}"
}

# T013: Rollback on failure
rollback_on_failure() {
    local exit_code=$1
    
    # If exit code is 0 and installation completed, skip rollback
    if [ $exit_code -eq 0 ] && [ $INSTALLATION_STARTED -eq 1 ]; then
        trap - EXIT
        return 0
    fi
    
    # If we haven't started installation, nothing to rollback
    if [ $INSTALLATION_STARTED -eq 0 ]; then
        trap - EXIT
        return 0
    fi
    
    # Only rollback if we had an error and actually installed something
    if [ $exit_code -ne 0 ] && [ $BINARIES_INSTALLED -eq 1 ]; then
        echo ""
        warning "Installation failed. Rolling back..."
        
        local SUDO=""
        if [ ! -w "$INSTALL_DIR" ]; then
            SUDO="sudo"
        fi
        
        # Remove newly installed binaries
        if [ -f "${INSTALL_DIR}/lbs" ]; then
            ${SUDO} rm -f "${INSTALL_DIR}/lbs" 2>/dev/null || true
            info "  Removed lbs"
        fi
        
        if [ -f "${INSTALL_DIR}/lbsd" ]; then
            ${SUDO} rm -f "${INSTALL_DIR}/lbsd" 2>/dev/null || true
            info "  Removed lbsd"
        fi
        
        # Restore backups if they exist
        if [ $BINARIES_BACKED_UP -eq 1 ] && [ -f "${MANIFEST_FILE}" ]; then
            local timestamp=$(head -n1 "${MANIFEST_FILE}")
            
            if [ -f "${BACKUP_DIR}/lbs.${timestamp}" ]; then
                ${SUDO} cp "${BACKUP_DIR}/lbs.${timestamp}" "${INSTALL_DIR}/lbs" 2>/dev/null || true
                ${SUDO} chmod +x "${INSTALL_DIR}/lbs" 2>/dev/null || true
                info "  Restored previous lbs binary"
            fi
            
            if [ -f "${BACKUP_DIR}/lbsd.${timestamp}" ]; then
                ${SUDO} cp "${BACKUP_DIR}/lbsd.${timestamp}" "${INSTALL_DIR}/lbsd" 2>/dev/null || true
                ${SUDO} chmod +x "${INSTALL_DIR}/lbsd" 2>/dev/null || true
                info "  Restored previous lbsd binary"
            fi
        fi
        
        error "Installation rolled back. Previous state restored."
        trap - EXIT
        exit $exit_code
    fi
    
    trap - EXIT
}

# Get version from VERSION file
get_version() {
    if [ ! -f "VERSION" ]; then
        error "VERSION file not found. Are you running this from the project root?"
        exit 1
    fi
    cat VERSION
}

# Uninstall binaries
uninstall_binaries() {
    info "Uninstalling LibreSeed binaries from ${INSTALL_DIR}..."
    
    local SUDO=""
    if [ ! -w "$INSTALL_DIR" ]; then
        info "  Administrator privileges required for uninstallation."
        SUDO="sudo"
    fi
    
    local removed=0
    
    # Remove lbs
    if [ -f "${INSTALL_DIR}/lbs" ]; then
        ${SUDO} rm "${INSTALL_DIR}/lbs" || {
            error "Failed to remove lbs"
            exit 1
        }
        success "  ✓ lbs removed from ${INSTALL_DIR}"
        removed=1
    fi
    
    # Remove lbsd
    if [ -f "${INSTALL_DIR}/lbsd" ]; then
        ${SUDO} rm "${INSTALL_DIR}/lbsd" || {
            error "Failed to remove lbsd"
            exit 1
        }
        success "  ✓ lbsd removed from ${INSTALL_DIR}"
        removed=1
    fi
    
    if [ $removed -eq 0 ]; then
        warning "No LibreSeed binaries found in ${INSTALL_DIR}"
    else
        success "LibreSeed binaries successfully uninstalled."
        echo ""
        info "Note: Configuration files in ${DATA_DIR} have been preserved."
        info "To remove them, run: rm -rf ${DATA_DIR}"
    fi
}

# Show usage
usage() {
    cat << EOF
LibreSeed Installation Script

Usage: $0 [OPTIONS]

OPTIONS:
    --uninstall     Remove LibreSeed binaries from ${INSTALL_DIR}
    --help, -h      Show this help message

EXAMPLES:
    # Install or update LibreSeed
    $0

    # Uninstall LibreSeed
    $0 --uninstall

NOTES:
    - Requires Go ${REQUIRED_GO_VERSION} or later
    - Requires Make for building
    - Configuration files in ${DATA_DIR} are preserved during uninstall
    - Binaries are installed to ${INSTALL_DIR}
    - Automatic rollback on failure

EOF
}

# Main installation flow
install() {
    INSTALLATION_STARTED=1
    
    info "=== LibreSeed Installation ==="
    echo ""
    
    # T011: Detect platform
    local platform=$(detect_platform)
    info "Platform: ${platform}"
    
    # T010: Check prerequisites
    check_prerequisites
    echo ""
    
    # Get version
    local version=$(get_version)
    info "Version: ${version}"
    echo ""
    
    # T020: Create data directories
    create_data_directories
    echo ""
    
    # T012: Backup existing binaries
    backup_existing_binaries
    echo ""
    
    # T014: Build with Make
    build_with_make
    echo ""
    
    # T015: Generate checksums
    generate_checksums
    echo ""
    
    # T016: Verify checksums
    verify_checksums
    echo ""
    
    # T017: Install binaries
    install_binaries
    echo ""
    
    success "=== Installation Complete ==="
    echo ""
    info "LibreSeed ${version} has been installed successfully!"
    echo ""
    info "Quick start:"
    info "  lbs start       # Start the daemon"
    info "  lbs status      # Check daemon status"
    info "  lbs stats       # View detailed statistics"
    info "  lbs restart     # Restart the daemon"
    info "  lbs stop        # Stop the daemon"
    echo ""
    info "Data directory: ${DATA_DIR}"
    info "Configuration: ${DATA_DIR}/config.yaml (will be created on first run)"
    info "Backups: ${BACKUP_DIR}"
    echo ""
}

# Main execution
main() {
    # Parse arguments
    case "${1:-}" in
        --uninstall)
            uninstall_binaries
            exit 0
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        "")
            # Normal installation
            ;;
        *)
            error "Unknown option: $1"
            echo ""
            usage
            exit 1
            ;;
    esac
    
    # Check if running from project root
    if [ ! -f "go.mod" ] || [ ! -d "cmd/lbs" ] || [ ! -d "cmd/lbsd" ]; then
        error "This script must be run from the LibreSeed project root directory."
        exit 1
    fi
    
    # Check if Makefile exists
    if [ ! -f "Makefile" ]; then
        error "Makefile not found. This installation requires the build system to be present."
        exit 1
    fi
    
    # Run installation
    install
}

main "$@"
