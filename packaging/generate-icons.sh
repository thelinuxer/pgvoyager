#!/bin/bash
# Generate icons for all platforms from the SVG logo

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

# macOS icon (icns format)
echo "Generating macOS icon..."
mkdir -p "${SCRIPT_DIR}/macos/PgVoyager.app/Contents/Resources"
ICONSET_DIR="${TEMP_DIR}/pgvoyager.iconset"
mkdir -p "$ICONSET_DIR"
convert "${TEMP_DIR}/icon-1024.png" -resize 16x16 "${ICONSET_DIR}/icon_16x16.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 32x32 "${ICONSET_DIR}/icon_16x16@2x.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 32x32 "${ICONSET_DIR}/icon_32x32.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 64x64 "${ICONSET_DIR}/icon_32x32@2x.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 128x128 "${ICONSET_DIR}/icon_128x128.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 256x256 "${ICONSET_DIR}/icon_128x128@2x.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 256x256 "${ICONSET_DIR}/icon_256x256.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 512x512 "${ICONSET_DIR}/icon_256x256@2x.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 512x512 "${ICONSET_DIR}/icon_512x512.png"
convert "${TEMP_DIR}/icon-1024.png" -resize 1024x1024 "${ICONSET_DIR}/icon_512x512@2x.png"

# Create icns file (requires iconutil on macOS, or we use png2icns/icnsify if available)
if command -v iconutil &> /dev/null; then
    iconutil -c icns -o "${SCRIPT_DIR}/macos/PgVoyager.app/Contents/Resources/pgvoyager.icns" "$ICONSET_DIR"
elif command -v png2icns &> /dev/null; then
    png2icns "${SCRIPT_DIR}/macos/PgVoyager.app/Contents/Resources/pgvoyager.icns" \
        "${ICONSET_DIR}/icon_16x16.png" \
        "${ICONSET_DIR}/icon_32x32.png" \
        "${ICONSET_DIR}/icon_128x128.png" \
        "${ICONSET_DIR}/icon_256x256.png" \
        "${ICONSET_DIR}/icon_512x512.png"
else
    echo "Warning: No icns generator available. Saving iconset for later conversion."
    cp -r "$ICONSET_DIR" "${SCRIPT_DIR}/macos/pgvoyager.iconset"
fi

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
echo "  macOS: ${SCRIPT_DIR}/macos/PgVoyager.app/Contents/Resources/pgvoyager.icns"
echo "  Windows: ${SCRIPT_DIR}/windows/pgvoyager.ico"
