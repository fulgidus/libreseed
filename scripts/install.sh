#!/bin/bash
# LibreSeed Binary Installation Script
# Automatically downloads and installs the latest LibreSeed binaries (lbs and lbsd)

set -e

# Configuration
REPO="fulgidus/libreseed"
BINARIES=("lbs" "lbsd")  # Install both CLI and daemon
DEFAULT_INSTALL_DIR="$HOME/.local/bin"
SYSTEM_INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse command line arguments
INSTALL_DIR="$DEFAULT_INSTALL_DIR"
USE_SYSTEM=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --system)
            USE_SYSTEM=true
            INSTALL_DIR="$SYSTEM_INSTALL_DIR"
            shift
            ;;
        *)
            echo -e "${RED}Error: Unknown option: $1${NC}"
            echo "Usage: $0 [--system]"
            echo "  --system    Install to $SYSTEM_INSTALL_DIR (may require sudo)"
            exit 1
            ;;
    esac
done

# Helper functions
log_info() {
    echo -e "${BLUE}==>${NC} $1"
}

log_success() {
    echo -e "${GREEN}==>${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}Warning:${NC} $1"
}

log_error() {
    echo -e "${RED}Error:${NC} $1"
}

# Detect platform and architecture
detect_platform() {
    local os arch
    
    # Detect OS
    case "$(uname -s)" in
        Linux*)
            os="linux"
            ;;
        Darwin*)
            os="darwin"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            os="windows"
            ;;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    
    PLATFORM="${os}-${arch}"
    OS_TYPE="$os"
}

# Get latest release version from GitHub
get_latest_version() {
    log_info "Fetching latest release version..."
    
    if command -v curl > /dev/null 2>&1; then
        VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget > /dev/null 2>&1; then
        VERSION=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        log_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi
    
    if [ -z "$VERSION" ]; then
        log_error "Failed to fetch latest version"
        exit 1
    fi
    
    log_info "Latest version: $VERSION"
}

# Download binary and checksum
download_binary() {
    local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_FILE}"
    local checksum_url="${download_url}.sha256"
    local tmp_dir
    
    tmp_dir=$(mktemp -d)
    DOWNLOAD_FILE="${tmp_dir}/${BINARY_FILE}"
    CHECKSUM_FILE="${tmp_dir}/${BINARY_FILE}.sha256"
    
    log_info "Downloading binary from ${download_url}..."
    
    if command -v curl > /dev/null 2>&1; then
        if ! curl -fsSL -o "$DOWNLOAD_FILE" "$download_url"; then
            log_error "Failed to download binary"
            rm -rf "$tmp_dir"
            exit 1
        fi
        if ! curl -fsSL -o "$CHECKSUM_FILE" "$checksum_url"; then
            log_error "Failed to download checksum"
            rm -rf "$tmp_dir"
            exit 1
        fi
    elif command -v wget > /dev/null 2>&1; then
        if ! wget -q -O "$DOWNLOAD_FILE" "$download_url"; then
            log_error "Failed to download binary"
            rm -rf "$tmp_dir"
            exit 1
        fi
        if ! wget -q -O "$CHECKSUM_FILE" "$checksum_url"; then
            log_error "Failed to download checksum"
            rm -rf "$tmp_dir"
            exit 1
        fi
    fi
    
    log_success "Download complete"
}

# Verify checksum
verify_checksum() {
    log_info "Verifying checksum..."
    
    cd "$(dirname "$DOWNLOAD_FILE")"
    
    if command -v sha256sum > /dev/null 2>&1; then
        if ! sha256sum -c "$CHECKSUM_FILE" > /dev/null 2>&1; then
            log_error "Checksum verification failed!"
            log_error "The downloaded binary may be corrupted or tampered with."
            rm -rf "$(dirname "$DOWNLOAD_FILE")"
            exit 1
        fi
    elif command -v shasum > /dev/null 2>&1; then
        if ! shasum -a 256 -c "$CHECKSUM_FILE" > /dev/null 2>&1; then
            log_error "Checksum verification failed!"
            log_error "The downloaded binary may be corrupted or tampered with."
            rm -rf "$(dirname "$DOWNLOAD_FILE")"
            exit 1
        fi
    else
        log_error "Neither sha256sum nor shasum found. Cannot verify checksum."
        exit 1
    fi
    
    log_success "Checksum verified"
}

# Install binary
install_binary() {
    log_info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
    
    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "Creating directory: $INSTALL_DIR"
        if [ "$USE_SYSTEM" = true ]; then
            if ! sudo mkdir -p "$INSTALL_DIR"; then
                log_error "Failed to create directory: $INSTALL_DIR"
                exit 1
            fi
        else
            if ! mkdir -p "$INSTALL_DIR"; then
                log_error "Failed to create directory: $INSTALL_DIR"
                exit 1
            fi
        fi
    fi
    
    # Make binary executable
    chmod +x "$DOWNLOAD_FILE"
    
    # Install binary
    if [ "$USE_SYSTEM" = true ]; then
        if ! sudo mv "$DOWNLOAD_FILE" "${INSTALL_DIR}/${BINARY_NAME}"; then
            log_error "Failed to install binary"
            rm -rf "$(dirname "$DOWNLOAD_FILE")"
            exit 1
        fi
    else
        if ! mv "$DOWNLOAD_FILE" "${INSTALL_DIR}/${BINARY_NAME}"; then
            log_error "Failed to install binary"
            rm -rf "$(dirname "$DOWNLOAD_FILE")"
            exit 1
        fi
    fi
    
    # Clean up
    rm -rf "$(dirname "$DOWNLOAD_FILE")"
    
    log_success "Installation complete!"
}

# Check if install directory is in PATH
check_path() {
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        log_warning "Installation directory is not in your PATH!"
        log_warning "Add the following line to your shell configuration file:"
        echo ""
        echo "    export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
        
        # Detect shell and suggest appropriate config file
        local shell_config=""
        if [ -n "$BASH_VERSION" ]; then
            shell_config="~/.bashrc"
        elif [ -n "$ZSH_VERSION" ]; then
            shell_config="~/.zshrc"
        else
            shell_config="your shell configuration file"
        fi
        
        log_warning "Add this to ${shell_config}, then run: source ${shell_config}"
    else
        log_success "Installation directory is in your PATH"
    fi
}

# Display completion message
display_completion() {
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  LibreSeed ${VERSION} installed successfully!  ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════╝${NC}"
    echo ""
    log_info "Installed to: ${INSTALL_DIR}/${BINARY_NAME}"
    log_info "Verify installation: ${BINARY_NAME} --version"
    echo ""
    log_info "Start the daemon: ${BINARY_NAME} start"
    log_info "View daemon status: ${BINARY_NAME} status"
    log_info "View statistics: ${BINARY_NAME} stats"
    echo ""
    log_info "For more commands, run: ${BINARY_NAME} --help"
    echo ""
}

# Main installation flow
main() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║  LibreSeed Binary Installation Script      ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
    echo ""
    
    if [ "$USE_SYSTEM" = true ]; then
        log_info "Installation mode: System-wide (${INSTALL_DIR})"
    else
        log_info "Installation mode: User (${INSTALL_DIR})"
    fi
    
    detect_platform
    log_info "Detected platform: $PLATFORM"
    
    get_latest_version
    download_binary
    verify_checksum
    install_binary
    check_path
    display_completion
}

# Run main function
main "$@"
