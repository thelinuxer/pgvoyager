#!/bin/bash
# PgVoyager Linux Uninstaller

set -e

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
ICON_DIR="${HOME}/.local/share/icons/hicolor"
DESKTOP_DIR="${HOME}/.local/share/applications"

echo "Uninstalling PgVoyager..."

# Remove binaries
echo "Removing binaries..."
sudo rm -f "${INSTALL_DIR}/pgvoyager" 2>/dev/null || rm -f "${INSTALL_DIR}/pgvoyager"
sudo rm -f "${INSTALL_DIR}/pgvoyager-launcher" 2>/dev/null || rm -f "${INSTALL_DIR}/pgvoyager-launcher"
sudo rm -f "${INSTALL_DIR}/pgvoyager-mcp" 2>/dev/null || rm -f "${INSTALL_DIR}/pgvoyager-mcp"

# Remove desktop entry
echo "Removing desktop entry..."
rm -f "${DESKTOP_DIR}/pgvoyager.desktop"

# Remove icons
echo "Removing icons..."
rm -f "${ICON_DIR}/256x256/apps/pgvoyager.png"
rm -f "${ICON_DIR}/128x128/apps/pgvoyager.png"
rm -f "${ICON_DIR}/64x64/apps/pgvoyager.png"
rm -f "${ICON_DIR}/48x48/apps/pgvoyager.png"

# Update caches
gtk-update-icon-cache "${ICON_DIR}" 2>/dev/null || true
update-desktop-database "${DESKTOP_DIR}" 2>/dev/null || true

echo "PgVoyager uninstalled successfully!"
