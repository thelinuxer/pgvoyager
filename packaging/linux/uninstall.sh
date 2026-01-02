#!/bin/bash
# PgVoyager Linux Uninstaller

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
WARN="${YELLOW}⚠${NC}"

# Print functions
print_banner() {
    echo ""
    echo -e "${RED}"
    echo "  ╔══════════════════════════════════════════════════════════════════════════════════╗"
    echo "  ║                                                                                  ║"
    echo "  ║   ██████╗  ██████╗ ██╗   ██╗ ██████╗ ██╗   ██╗ █████╗  ██████╗ ███████╗██████╗   ║"
    echo "  ║   ██╔══██╗██╔════╝ ██║   ██║██╔═══██╗╚██╗ ██╔╝██╔══██╗██╔════╝ ██╔════╝██╔══██╗  ║"
    echo "  ║   ██████╔╝██║  ███╗██║   ██║██║   ██║ ╚████╔╝ ███████║██║  ███╗█████╗  ██████╔╝  ║"
    echo "  ║   ██╔═══╝ ██║   ██║╚██╗ ██╔╝██║   ██║  ╚██╔╝  ██╔══██║██║   ██║██╔══╝  ██╔══██╗  ║"
    echo "  ║   ██║     ╚██████╔╝ ╚████╔╝ ╚██████╔╝   ██║   ██║  ██║╚██████╔╝███████╗██║  ██║  ║"
    echo "  ║   ╚═╝      ╚═════╝   ╚═══╝   ╚═════╝    ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝  ║"
    echo "  ║                                                                                  ║"
    echo "  ║                               UNINSTALLER                                        ║"
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

print_warning() {
    echo -e "  ${WARN} ${YELLOW}$1${NC}"
}

print_progress() {
    echo -ne "  ${ARROW} $1..."
}

print_done() {
    echo -e " ${GREEN}done${NC}"
}

# Configuration
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
ICON_DIR="${HOME}/.local/share/icons/hicolor"
DESKTOP_DIR="${HOME}/.local/share/applications"
CONFIG_DIR="${HOME}/.config/pgvoyager"

# Print banner
print_banner

# Show what will be removed
print_header "Uninstall Configuration"
echo ""
echo -e "  ${DIM}Install directory:${NC}  ${WHITE}${INSTALL_DIR}${NC}"
echo -e "  ${DIM}Config directory:${NC}   ${WHITE}${CONFIG_DIR}${NC}"
echo -e "  ${DIM}Desktop directory:${NC}  ${WHITE}${DESKTOP_DIR}${NC}"
echo ""

# Stop running instance
print_header "Stopping Running Instances"
echo ""
if pgrep -f "${INSTALL_DIR}/pgvoyager" > /dev/null 2>&1; then
    print_step "Stopping running PgVoyager instance"
    sudo pkill -f "${INSTALL_DIR}/pgvoyager" 2>/dev/null || true
    sleep 1
    print_success "Instance stopped"
else
    print_info "No running instance found"
fi
echo ""

# Remove binaries
print_header "Removing Binaries"
echo ""
BINARIES_REMOVED=0

if [ -f "${INSTALL_DIR}/pgvoyager" ]; then
    print_progress "Removing PgVoyager binary"
    sudo rm -f "${INSTALL_DIR}/pgvoyager" 2>/dev/null || rm -f "${INSTALL_DIR}/pgvoyager"
    print_done
    ((BINARIES_REMOVED++))
else
    print_info "PgVoyager binary not found"
fi

if [ -f "${INSTALL_DIR}/pgvoyager-launcher" ]; then
    print_progress "Removing launcher script"
    sudo rm -f "${INSTALL_DIR}/pgvoyager-launcher" 2>/dev/null || rm -f "${INSTALL_DIR}/pgvoyager-launcher"
    print_done
    ((BINARIES_REMOVED++))
else
    print_info "Launcher script not found"
fi

if [ -f "${INSTALL_DIR}/pgvoyager-mcp" ]; then
    print_progress "Removing MCP server"
    sudo rm -f "${INSTALL_DIR}/pgvoyager-mcp" 2>/dev/null || rm -f "${INSTALL_DIR}/pgvoyager-mcp"
    print_done
    ((BINARIES_REMOVED++))
else
    print_info "MCP server not found"
fi

if [ $BINARIES_REMOVED -gt 0 ]; then
    print_success "Removed ${BINARIES_REMOVED} binary file(s)"
fi
echo ""

# Remove desktop integration
print_header "Removing Desktop Integration"
echo ""

if [ -f "${DESKTOP_DIR}/pgvoyager.desktop" ]; then
    print_progress "Removing desktop entry"
    rm -f "${DESKTOP_DIR}/pgvoyager.desktop"
    print_done
else
    print_info "Desktop entry not found"
fi

ICONS_REMOVED=0
for size in 256 128 64 48; do
    if [ -f "${ICON_DIR}/${size}x${size}/apps/pgvoyager.png" ]; then
        rm -f "${ICON_DIR}/${size}x${size}/apps/pgvoyager.png"
        ((ICONS_REMOVED++))
    fi
done

if [ $ICONS_REMOVED -gt 0 ]; then
    print_success "Removed ${ICONS_REMOVED} icon file(s)"
else
    print_info "No icon files found"
fi

print_progress "Updating system caches"
gtk-update-icon-cache "${ICON_DIR}" 2>/dev/null || true
update-desktop-database "${DESKTOP_DIR}" 2>/dev/null || true
print_done
echo ""

# Ask about config removal
print_header "Configuration Data"
echo ""
if [ -d "${CONFIG_DIR}" ]; then
    print_warning "Configuration directory exists: ${CONFIG_DIR}"
    echo ""
    echo -e "  ${DIM}This contains your saved connections and preferences.${NC}"
    echo ""
    read -p "  Remove configuration data? [y/N] " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_progress "Removing configuration"
        rm -rf "${CONFIG_DIR}"
        print_done
        print_success "Configuration removed"
    else
        print_info "Configuration preserved at ${CONFIG_DIR}"
    fi
else
    print_info "No configuration directory found"
fi
echo ""

# Print success message
echo -e "${GREEN}"
echo "  ╔═══════════════════════════════════════════════════════════════╗"
echo "  ║                                                               ║"
echo "  ║            PgVoyager uninstalled successfully!                ║"
echo "  ║                                                               ║"
echo "  ╚═══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

echo -e "  ${DIM}Thank you for using PgVoyager!${NC}"
echo -e "  ${DIM}Reinstall anytime from: ${WHITE}https://github.com/thelinuxer/pgvoyager${NC}"
echo ""
