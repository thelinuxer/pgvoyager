#!/bin/bash
# PgVoyager Linux Installer

set -e

# Parse command line arguments
PGVOYAGER_PORT="${1:-5137}"

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
ICON_DIR="${HOME}/.local/share/icons/hicolor"
DESKTOP_DIR="${HOME}/.local/share/applications"
CONFIG_DIR="${HOME}/.config/pgvoyager"

# Get the directory where the script is located (resolve before sudo changes things)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Installing PgVoyager..."
echo "Script directory: ${SCRIPT_DIR}"
echo "Port: ${PGVOYAGER_PORT}"

# Stop any running pgvoyager instance
if pgrep -f "/usr/local/bin/pgvoyager" > /dev/null 2>&1; then
    echo "Stopping running PgVoyager instance..."
    sudo pkill -f "/usr/local/bin/pgvoyager" 2>/dev/null || true
    sleep 1
fi

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64) BINARY="pgvoyager-linux-amd64" ;;
    aarch64) BINARY="pgvoyager-linux-arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Download binary if not present
if [ ! -f "${SCRIPT_DIR}/pgvoyager" ] && [ ! -f "${SCRIPT_DIR}/${BINARY}" ]; then
    echo "Downloading ${BINARY}..."
    curl -L "https://github.com/thelinuxer/pgvoyager/releases/latest/download/${BINARY}" -o "${SCRIPT_DIR}/pgvoyager"
    chmod +x "${SCRIPT_DIR}/pgvoyager"
fi

# Install binary
echo "Installing binary to ${INSTALL_DIR}..."
if [ -f "${SCRIPT_DIR}/pgvoyager" ]; then
    sudo cp "${SCRIPT_DIR}/pgvoyager" "${INSTALL_DIR}/pgvoyager"
    sudo chmod 755 "${INSTALL_DIR}/pgvoyager"
fi

# Install MCP server
if [ -f "${SCRIPT_DIR}/pgvoyager-mcp" ]; then
    echo "Installing MCP server..."
    sudo cp "${SCRIPT_DIR}/pgvoyager-mcp" "${INSTALL_DIR}/pgvoyager-mcp"
    sudo chmod 755 "${INSTALL_DIR}/pgvoyager-mcp"
fi

# Install launcher script
if [ -f "${SCRIPT_DIR}/pgvoyager-launcher" ]; then
    echo "Installing launcher..."
    sudo cp "${SCRIPT_DIR}/pgvoyager-launcher" "${INSTALL_DIR}/pgvoyager-launcher"
    sudo chmod 755 "${INSTALL_DIR}/pgvoyager-launcher"
fi

# Install icons (as current user, not sudo)
echo "Installing icons..."
mkdir -p "${ICON_DIR}/256x256/apps"
mkdir -p "${ICON_DIR}/128x128/apps"
mkdir -p "${ICON_DIR}/64x64/apps"
mkdir -p "${ICON_DIR}/48x48/apps"

if [ -f "${SCRIPT_DIR}/pgvoyager-256.png" ]; then
    cp "${SCRIPT_DIR}/pgvoyager-256.png" "${ICON_DIR}/256x256/apps/pgvoyager.png"
    echo "  Installed 256x256 icon"
fi
if [ -f "${SCRIPT_DIR}/pgvoyager-128.png" ]; then
    cp "${SCRIPT_DIR}/pgvoyager-128.png" "${ICON_DIR}/128x128/apps/pgvoyager.png"
    echo "  Installed 128x128 icon"
fi
if [ -f "${SCRIPT_DIR}/pgvoyager-64.png" ]; then
    cp "${SCRIPT_DIR}/pgvoyager-64.png" "${ICON_DIR}/64x64/apps/pgvoyager.png"
    echo "  Installed 64x64 icon"
fi
if [ -f "${SCRIPT_DIR}/pgvoyager-48.png" ]; then
    cp "${SCRIPT_DIR}/pgvoyager-48.png" "${ICON_DIR}/48x48/apps/pgvoyager.png"
    echo "  Installed 48x48 icon"
fi

# Install desktop entry (as current user)
echo "Installing desktop entry..."
mkdir -p "${DESKTOP_DIR}"
if [ -f "${SCRIPT_DIR}/pgvoyager.desktop" ]; then
    cp "${SCRIPT_DIR}/pgvoyager.desktop" "${DESKTOP_DIR}/pgvoyager.desktop"
    echo "  Installed desktop entry"
fi

# Update caches
gtk-update-icon-cache "${ICON_DIR}" 2>/dev/null || true
update-desktop-database "${DESKTOP_DIR}" 2>/dev/null || true

# Save port configuration
mkdir -p "${CONFIG_DIR}"
echo "PGVOYAGER_PORT=${PGVOYAGER_PORT}" > "${CONFIG_DIR}/config"
echo "  Saved port configuration to ${CONFIG_DIR}/config"

echo ""
echo "PgVoyager installed successfully!"
echo "Server will run on port: ${PGVOYAGER_PORT}"
echo "You can now launch it from your application menu or run: pgvoyager-launcher"
echo ""
echo "To use a different port, reinstall with: ./install.sh <port>"
