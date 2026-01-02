#!/bin/bash
# PgVoyager Linux Installer

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Symbols
CHECK="${GREEN}✓${NC}"
CROSS="${RED}✗${NC}"
ARROW="${CYAN}➜${NC}"
INFO="${BLUE}ℹ${NC}"

# Print functions
print_banner() {
    echo ""
    echo -e "${MAGENTA}"
    echo "  ╔══════════════════════════════════════════════════════════════════════════════════╗"
    echo "  ║                                                                                  ║"
    echo "  ║   ██████╗  ██████╗ ██╗   ██╗ ██████╗ ██╗   ██╗ █████╗  ██████╗ ███████╗██████╗   ║"
    echo "  ║   ██╔══██╗██╔════╝ ██║   ██║██╔═══██╗╚██╗ ██╔╝██╔══██╗██╔════╝ ██╔════╝██╔══██╗  ║"
    echo "  ║   ██████╔╝██║  ███╗██║   ██║██║   ██║ ╚████╔╝ ███████║██║  ███╗█████╗  ██████╔╝  ║"
    echo "  ║   ██╔═══╝ ██║   ██║╚██╗ ██╔╝██║   ██║  ╚██╔╝  ██╔══██║██║   ██║██╔══╝  ██╔══██╗  ║"
    echo "  ║   ██║     ╚██████╔╝ ╚████╔╝ ╚██████╔╝   ██║   ██║  ██║╚██████╔╝███████╗██║  ██║  ║"
    echo "  ║   ╚═╝      ╚═════╝   ╚═══╝   ╚═════╝    ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝  ║"
    echo "  ║                                                                                  ║"
    echo "  ║               PostgreSQL Database Explorer with Claude AI                        ║"
    echo "  ║                                                                                  ║"
    echo "  ╚══════════════════════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

print_header() {
    echo ""
    echo -e "${BOLD}${WHITE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${WHITE}  $1${NC}"
    echo -e "${BOLD}${WHITE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_step() {
    echo -e "  ${ARROW} $1"
}

print_success() {
    echo -e "  ${CHECK} $1"
}

print_error() {
    echo -e "  ${CROSS} ${RED}$1${NC}"
}

print_info() {
    echo -e "  ${INFO} ${DIM}$1${NC}"
}

print_progress() {
    echo -ne "  ${ARROW} $1..."
}

print_done() {
    echo -e " ${GREEN}done${NC}"
}

# Parse command line arguments
PGVOYAGER_PORT="${1:-5137}"

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
ICON_DIR="${HOME}/.local/share/icons/hicolor"
DESKTOP_DIR="${HOME}/.local/share/applications"
CONFIG_DIR="${HOME}/.config/pgvoyager"

# Get the directory where the script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Print banner
print_banner

# Show configuration
print_header "Installation Configuration"
echo ""
echo -e "  ${DIM}Install directory:${NC}  ${WHITE}${INSTALL_DIR}${NC}"
echo -e "  ${DIM}Config directory:${NC}   ${WHITE}${CONFIG_DIR}${NC}"
echo -e "  ${DIM}Server port:${NC}        ${WHITE}${PGVOYAGER_PORT}${NC}"
echo ""

# Detect architecture
print_header "System Detection"
echo ""
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        BINARY="pgvoyager-linux-amd64"
        print_success "Architecture: ${WHITE}x86_64 (64-bit)${NC}"
        ;;
    aarch64)
        BINARY="pgvoyager-linux-arm64"
        print_success "Architecture: ${WHITE}ARM64${NC}"
        ;;
    *)
        print_error "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

OS=$(uname -s)
print_success "Operating System: ${WHITE}${OS}${NC}"
echo ""

# Stop running instance
print_header "Preparing Installation"
echo ""
if pgrep -f "/usr/local/bin/pgvoyager" > /dev/null 2>&1; then
    print_step "Stopping running PgVoyager instance"
    sudo pkill -f "/usr/local/bin/pgvoyager" 2>/dev/null || true
    sleep 1
    print_success "Previous instance stopped"
else
    print_info "No running instance found"
fi
echo ""

# Download binary if needed
print_header "Installing Binaries"
echo ""
if [ ! -f "${SCRIPT_DIR}/pgvoyager" ] && [ ! -f "${SCRIPT_DIR}/${BINARY}" ]; then
    print_step "Downloading ${BINARY}..."
    if curl -fsSL "https://github.com/thelinuxer/pgvoyager/releases/latest/download/${BINARY}" -o "${SCRIPT_DIR}/pgvoyager"; then
        chmod +x "${SCRIPT_DIR}/pgvoyager"
        print_success "Downloaded successfully"
    else
        print_error "Failed to download binary"
        exit 1
    fi
fi

# Install main binary
if [ -f "${SCRIPT_DIR}/pgvoyager" ]; then
    print_progress "Installing PgVoyager binary"
    sudo cp "${SCRIPT_DIR}/pgvoyager" "${INSTALL_DIR}/pgvoyager"
    sudo chmod 755 "${INSTALL_DIR}/pgvoyager"
    print_done
    print_success "Installed: ${DIM}${INSTALL_DIR}/pgvoyager${NC}"
fi

# Install MCP server
if [ -f "${SCRIPT_DIR}/pgvoyager-mcp" ]; then
    print_progress "Installing MCP server"
    sudo cp "${SCRIPT_DIR}/pgvoyager-mcp" "${INSTALL_DIR}/pgvoyager-mcp"
    sudo chmod 755 "${INSTALL_DIR}/pgvoyager-mcp"
    print_done
    print_success "Installed: ${DIM}${INSTALL_DIR}/pgvoyager-mcp${NC}"
fi

# Install launcher
if [ -f "${SCRIPT_DIR}/pgvoyager-launcher" ]; then
    print_progress "Installing launcher script"
    sudo cp "${SCRIPT_DIR}/pgvoyager-launcher" "${INSTALL_DIR}/pgvoyager-launcher"
    sudo chmod 755 "${INSTALL_DIR}/pgvoyager-launcher"
    print_done
    print_success "Installed: ${DIM}${INSTALL_DIR}/pgvoyager-launcher${NC}"
fi
echo ""

# Install icons
print_header "Installing Desktop Integration"
echo ""
print_step "Setting up application icons"
mkdir -p "${ICON_DIR}/256x256/apps"
mkdir -p "${ICON_DIR}/128x128/apps"
mkdir -p "${ICON_DIR}/64x64/apps"
mkdir -p "${ICON_DIR}/48x48/apps"

ICONS_INSTALLED=0
for size in 256 128 64 48; do
    if [ -f "${SCRIPT_DIR}/pgvoyager-${size}.png" ]; then
        cp "${SCRIPT_DIR}/pgvoyager-${size}.png" "${ICON_DIR}/${size}x${size}/apps/pgvoyager.png"
        ICONS_INSTALLED=$((ICONS_INSTALLED + 1))
    fi
done
print_success "Installed ${ICONS_INSTALLED} icon sizes"

# Install desktop entry
if [ -f "${SCRIPT_DIR}/pgvoyager.desktop" ]; then
    print_progress "Installing desktop entry"
    mkdir -p "${DESKTOP_DIR}"
    cp "${SCRIPT_DIR}/pgvoyager.desktop" "${DESKTOP_DIR}/pgvoyager.desktop"
    print_done
    print_success "Desktop entry installed"
fi

# Update caches
print_progress "Updating system caches"
gtk-update-icon-cache "${ICON_DIR}" 2>/dev/null || true
update-desktop-database "${DESKTOP_DIR}" 2>/dev/null || true
print_done
echo ""

# Save configuration
print_header "Finalizing"
echo ""
mkdir -p "${CONFIG_DIR}"
echo "PGVOYAGER_PORT=${PGVOYAGER_PORT}" > "${CONFIG_DIR}/config"
print_success "Configuration saved to ${DIM}${CONFIG_DIR}/config${NC}"
echo ""

# Print success message
echo -e "${GREEN}"
echo "  ╔═══════════════════════════════════════════════════════════════╗"
echo "  ║                                                               ║"
echo "  ║              Installation completed successfully!             ║"
echo "  ║                                                               ║"
echo "  ╚═══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

echo -e "  ${BOLD}Quick Start:${NC}"
echo ""
echo -e "    ${ARROW} Launch from application menu: ${WHITE}PgVoyager${NC}"
echo -e "    ${ARROW} Or run from terminal:         ${WHITE}pgvoyager-launcher${NC}"
echo ""
echo -e "  ┌─────────────────────────────────────────────────────────────┐"
echo -e "  │  ${BOLD}${CYAN}Open in browser:${NC}  ${BOLD}${WHITE}http://localhost:${PGVOYAGER_PORT}${NC}                    │"
echo -e "  └─────────────────────────────────────────────────────────────┘"
echo ""
echo -e "  ${DIM}To use a different port, reinstall with: ./install.sh <port>${NC}"
echo ""
