#!/bin/bash
# Generate icons for Linux and Windows from the SVG logo

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SVG_SOURCE="${PROJECT_ROOT}/frontend/static/logo.svg"
TEMP_DIR=$(mktemp -d)

echo "Generating icons from ${SVG_SOURCE}..."

# Generate high-res PNG first
echo "Creating base PNG..."
inkscape -w 1024 -h 1024 "$SVG_SOURCE" -o "${TEMP_DIR}/icon-1024.png" 2>/dev/null || \
    convert -background none -resize 1024x1024 "$SVG_SOURCE" "${TEMP_DIR}/icon-1024.png"

# Linux icons
echo "Generating Linux icons..."
mkdir -p "${SCRIPT_DIR}/linux"
convert "${TEMP_DIR}/icon-1024.png" -resize 256x256 "${SCRIPT_DIR}/linux/pgvoyager-256.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 128x128 "${SCRIPT_DIR}/linux/pgvoyager-128.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 64x64 "${SCRIPT_DIR}/linux/pgvoyager-64.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 48x48 "${SCRIPT_DIR}/linux/pgvoyager-48.png"

# Windows icon (ico format)
echo "Generating Windows icon..."
mkdir -p "${SCRIPT_DIR}/windows"
convert "${TEMP_DIR}/icon-1024.png" \
    \( -clone 0 -resize 16x16 \) \
    \( -clone 0 -resize 32x32 \) \
    \( -clone 0 -resize 48x48 \) \
    \( -clone 0 -resize 64x64 \) \
    \( -clone 0 -resize 128x128 \) \
    \( -clone 0 -resize 256x256 \) \
    -delete 0 "${SCRIPT_DIR}/windows/pgvoyager.ico"

# Cleanup
rm -rf "$TEMP_DIR"

echo "Icons generated successfully!"
echo "  Linux: ${SCRIPT_DIR}/linux/pgvoyager-*.png"
echo "  Windows: ${SCRIPT_DIR}/windows/pgvoyager.ico"
