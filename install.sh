#!/bin/bash
#
# Cortex installer script
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/kareemaly/cortex/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/kareemaly/cortex/main/install.sh | bash -s -- -v v1.0.0
#
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

REPO="kareemaly/cortex"
VERSION=""
INSTALL_DIR=""

# Print colored messages
info() {
    printf "${BLUE}==> ${NC}%s\n" "$1"
}

success() {
    printf "${GREEN}==> ${NC}%s\n" "$1"
}

warn() {
    printf "${YELLOW}==> Warning: ${NC}%s\n" "$1"
}

error() {
    printf "${RED}==> Error: ${NC}%s\n" "$1" >&2
    exit 1
}

# Show usage
usage() {
    cat <<EOF
Cortex Installer

Usage:
    install.sh [options]

Options:
    -v, --version VERSION    Install specific version (e.g., v1.0.0)
    -d, --dir DIRECTORY      Install to specific directory
    -h, --help               Show this help message

Examples:
    # Install latest version
    curl -fsSL https://raw.githubusercontent.com/kareemaly/cortex/main/install.sh | bash

    # Install specific version
    curl -fsSL https://raw.githubusercontent.com/kareemaly/cortex/main/install.sh | bash -s -- -v v1.0.0

    # Install to custom directory
    curl -fsSL https://raw.githubusercontent.com/kareemaly/cortex/main/install.sh | bash -s -- -d ~/bin
EOF
    exit 0
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -d|--dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            -h|--help)
                usage
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
}

# Detect OS
detect_os() {
    local os
    os="$(uname -s)"
    case "$os" in
        Linux)
            echo "linux"
            ;;
        Darwin)
            echo "darwin"
            ;;
        *)
            error "Unsupported operating system: $os"
            ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            error "Unsupported architecture: $arch"
            ;;
    esac
}

# Get latest version from GitHub
get_latest_version() {
    local url="https://api.github.com/repos/${REPO}/releases/latest"
    local version

    if command -v curl &>/dev/null; then
        version=$(curl -fsSL "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    elif command -v wget &>/dev/null; then
        version=$(wget -qO- "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    if [[ -z "$version" ]]; then
        error "Failed to get latest version from GitHub"
    fi

    echo "$version"
}

# Download file
download() {
    local url="$1"
    local dest="$2"

    if command -v curl &>/dev/null; then
        curl -fsSL "$url" -o "$dest"
    elif command -v wget &>/dev/null; then
        wget -q "$url" -O "$dest"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# Verify checksum
verify_checksum() {
    local file="$1"
    local expected="$2"
    local actual

    if command -v sha256sum &>/dev/null; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &>/dev/null; then
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        warn "Neither sha256sum nor shasum found. Skipping checksum verification."
        return 0
    fi

    if [[ "$actual" != "$expected" ]]; then
        error "Checksum verification failed for $file\nExpected: $expected\nActual:   $actual"
    fi
}

# Determine install directory
get_install_dir() {
    if [[ -n "$INSTALL_DIR" ]]; then
        echo "$INSTALL_DIR"
        return
    fi

    # Try /usr/local/bin first (requires sudo), fall back to ~/.local/bin
    if [[ -w "/usr/local/bin" ]]; then
        echo "/usr/local/bin"
    elif [[ -d "/usr/local/bin" ]] && command -v sudo &>/dev/null; then
        echo "/usr/local/bin"
    else
        echo "${HOME}/.local/bin"
    fi
}

# Check if sudo is needed
needs_sudo() {
    local dir="$1"
    if [[ -w "$dir" ]]; then
        return 1
    fi
    return 0
}

# Run command with sudo if needed
run_maybe_sudo() {
    local dir="$1"
    shift

    if needs_sudo "$dir"; then
        sudo "$@"
    else
        "$@"
    fi
}

main() {
    parse_args "$@"

    info "Detecting system..."
    local os arch
    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected: ${os}/${arch}"

    # Get version
    if [[ -z "$VERSION" ]]; then
        info "Getting latest version..."
        VERSION=$(get_latest_version)
    fi
    info "Installing version: ${VERSION}"

    # Determine install directory
    local install_dir
    install_dir=$(get_install_dir)
    info "Install directory: ${install_dir}"

    # Create temp directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf '$tmp_dir'" EXIT

    # Download checksums
    local base_url="https://github.com/${REPO}/releases/download/${VERSION}"
    info "Downloading checksums..."
    download "${base_url}/checksums.txt" "${tmp_dir}/checksums.txt"

    # Download and verify cortex
    local cortex_binary="cortex-${os}-${arch}"
    info "Downloading ${cortex_binary}..."
    download "${base_url}/${cortex_binary}" "${tmp_dir}/cortex"

    local cortex_checksum
    cortex_checksum=$(grep "${cortex_binary}$" "${tmp_dir}/checksums.txt" | awk '{print $1}')
    if [[ -n "$cortex_checksum" ]]; then
        info "Verifying checksum for cortex..."
        verify_checksum "${tmp_dir}/cortex" "$cortex_checksum"
    else
        warn "Checksum not found for ${cortex_binary}"
    fi

    # Download and verify cortexd
    local cortexd_binary="cortexd-${os}-${arch}"
    info "Downloading ${cortexd_binary}..."
    download "${base_url}/${cortexd_binary}" "${tmp_dir}/cortexd"

    local cortexd_checksum
    cortexd_checksum=$(grep "${cortexd_binary}$" "${tmp_dir}/checksums.txt" | awk '{print $1}')
    if [[ -n "$cortexd_checksum" ]]; then
        info "Verifying checksum for cortexd..."
        verify_checksum "${tmp_dir}/cortexd" "$cortexd_checksum"
    else
        warn "Checksum not found for ${cortexd_binary}"
    fi

    # Create install directory if needed
    if [[ ! -d "$install_dir" ]]; then
        info "Creating directory: ${install_dir}"
        mkdir -p "$install_dir"
    fi

    # Install binaries
    info "Installing binaries..."
    chmod +x "${tmp_dir}/cortex" "${tmp_dir}/cortexd"

    if needs_sudo "$install_dir"; then
        info "Installing to ${install_dir} (requires sudo)..."
        sudo cp "${tmp_dir}/cortex" "${install_dir}/cortex"
        sudo cp "${tmp_dir}/cortexd" "${install_dir}/cortexd"
        sudo chmod +x "${install_dir}/cortex" "${install_dir}/cortexd"
    else
        cp "${tmp_dir}/cortex" "${install_dir}/cortex"
        cp "${tmp_dir}/cortexd" "${install_dir}/cortexd"
        chmod +x "${install_dir}/cortex" "${install_dir}/cortexd"
    fi

    # Code sign on macOS
    if [[ "$os" == "darwin" ]]; then
        info "Code signing binaries (macOS)..."
        if needs_sudo "$install_dir"; then
            sudo codesign --force --sign - "${install_dir}/cortex" 2>/dev/null || true
            sudo codesign --force --sign - "${install_dir}/cortexd" 2>/dev/null || true
        else
            codesign --force --sign - "${install_dir}/cortex" 2>/dev/null || true
            codesign --force --sign - "${install_dir}/cortexd" 2>/dev/null || true
        fi
    fi

    # Verify installation
    success "Installation complete!"
    echo ""
    info "Verifying installation..."
    "${install_dir}/cortex" version
    echo ""
    "${install_dir}/cortexd" version
    echo ""

    # Check if install directory is in PATH
    if [[ ":$PATH:" != *":${install_dir}:"* ]]; then
        warn "${install_dir} is not in your PATH"
        echo ""
        echo "Add it to your PATH by adding this line to your shell profile:"
        echo ""
        echo "    export PATH=\"\$PATH:${install_dir}\""
        echo ""
    fi

    success "Done! Run 'cortex --help' to get started."
}

main "$@"
