#!/bin/bash
# PgVoyager Linux Installer

set -e

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
ICON_DIR="${HOME}/.local/share/icons/hicolor"
DESKTOP_DIR="${HOME}/.local/share/applications"

echo "Installing PgVoyager..."

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64) BINARY="pgvoyager-linux-amd64" ;;
    aarch64) BINARY="pgvoyager-linux-arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Download binary if not present
if [ ! -f "pgvoyager" ] && [ ! -f "${BINARY}" ]; then
    echo "Downloading ${BINARY}..."
    curl -L "https://github.com/thelinuxer/pgvoyager/releases/latest/download/${BINARY}" -o pgvoyager
    chmod +x pgvoyager
fi

# Install binary
echo "Installing binary to ${INSTALL_DIR}..."
sudo cp pgvoyager "${INSTALL_DIR}/pgvoyager" 2>/dev/null || cp pgvoyager "${INSTALL_DIR}/pgvoyager"
sudo chmod +x "${INSTALL_DIR}/pgvoyager" 2>/dev/null || chmod +x "${INSTALL_DIR}/pgvoyager"

# Install launcher script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "${SCRIPT_DIR}/pgvoyager-launcher" ]; then
    sudo cp "${SCRIPT_DIR}/pgvoyager-launcher" "${INSTALL_DIR}/pgvoyager-launcher" 2>/dev/null || cp "${SCRIPT_DIR}/pgvoyager-launcher" "${INSTALL_DIR}/pgvoyager-launcher"
    sudo chmod +x "${INSTALL_DIR}/pgvoyager-launcher" 2>/dev/null || chmod +x "${INSTALL_DIR}/pgvoyager-launcher"
fi

# Install icon
echo "Installing icon..."
mkdir -p "${ICON_DIR}/256x256/apps"
mkdir -p "${ICON_DIR}/128x128/apps"
mkdir -p "${ICON_DIR}/64x64/apps"
mkdir -p "${ICON_DIR}/48x48/apps"

if [ -f "${SCRIPT_DIR}/pgvoyager-256.png" ]; then
    cp "${SCRIPT_DIR}/pgvoyager-256.png" "${ICON_DIR}/256x256/apps/pgvoyager.png"
    cp "${SCRIPT_DIR}/pgvoyager-128.png" "${ICON_DIR}/128x128/apps/pgvoyager.png"
    cp "${SCRIPT_DIR}/pgvoyager-64.png" "${ICON_DIR}/64x64/apps/pgvoyager.png"
    cp "${SCRIPT_DIR}/pgvoyager-48.png" "${ICON_DIR}/48x48/apps/pgvoyager.png"
fi

# Install desktop entry
echo "Installing desktop entry..."
mkdir -p "${DESKTOP_DIR}"
if [ -f "${SCRIPT_DIR}/pgvoyager.desktop" ]; then
    cp "${SCRIPT_DIR}/pgvoyager.desktop" "${DESKTOP_DIR}/pgvoyager.desktop"
fi

# Update icon cache
gtk-update-icon-cache "${ICON_DIR}" 2>/dev/null || true
update-desktop-database "${DESKTOP_DIR}" 2>/dev/null || true

echo "PgVoyager installed successfully!"
echo "You can now launch it from your application menu or run: pgvoyager-launcher"
