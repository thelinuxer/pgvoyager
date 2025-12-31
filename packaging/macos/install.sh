#!/bin/bash
# PgVoyager macOS Installer

set -e

INSTALL_DIR="/Applications"
APP_NAME="PgVoyager.app"

echo "Installing PgVoyager..."

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64) BINARY="pgvoyager-darwin-amd64" ;;
    arm64) BINARY="pgvoyager-darwin-arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Check if we have the app bundle template
if [ ! -d "${SCRIPT_DIR}/${APP_NAME}" ]; then
    echo "Error: ${APP_NAME} bundle not found in ${SCRIPT_DIR}"
    exit 1
fi

# Download binary if not present
if [ ! -f "${SCRIPT_DIR}/pgvoyager" ] && [ ! -f "${SCRIPT_DIR}/${BINARY}" ]; then
    echo "Downloading ${BINARY}..."
    curl -L "https://github.com/thelinuxer/pgvoyager/releases/latest/download/${BINARY}" -o "${SCRIPT_DIR}/pgvoyager"
    chmod +x "${SCRIPT_DIR}/pgvoyager"
elif [ -f "${SCRIPT_DIR}/${BINARY}" ]; then
    mv "${SCRIPT_DIR}/${BINARY}" "${SCRIPT_DIR}/pgvoyager"
    chmod +x "${SCRIPT_DIR}/pgvoyager"
fi

# Copy app bundle to Applications
echo "Installing to ${INSTALL_DIR}..."
if [ -d "${INSTALL_DIR}/${APP_NAME}" ]; then
    echo "Removing existing installation..."
    rm -rf "${INSTALL_DIR}/${APP_NAME}"
fi

cp -r "${SCRIPT_DIR}/${APP_NAME}" "${INSTALL_DIR}/"

# Copy binary into app bundle
cp "${SCRIPT_DIR}/pgvoyager" "${INSTALL_DIR}/${APP_NAME}/Contents/Resources/pgvoyager"
chmod +x "${INSTALL_DIR}/${APP_NAME}/Contents/Resources/pgvoyager"
chmod +x "${INSTALL_DIR}/${APP_NAME}/Contents/MacOS/pgvoyager-launcher"

echo "PgVoyager installed successfully!"
echo "You can now launch it from your Applications folder or Spotlight."
