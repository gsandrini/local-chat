#!/usr/bin/env bash
set -e

APP_NAME="local-chat"
GITHUB_REPO="gsandrini/local-chat"
INSTALL_DIR="$HOME/.local/bin"
ICON_DIR="$HOME/.local/share/icons"
DESKTOP_DIR="$HOME/.local/share/applications"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Handle uninstall flag
if [[ "$1" == "--uninstall" ]]; then
    echo -e "${BLUE}Uninstalling ${APP_NAME}...${NC}"
    rm -f "${INSTALL_DIR}/${APP_NAME}"
    rm -f "${ICON_DIR}/${APP_NAME}.png"
    rm -f "${DESKTOP_DIR}/${APP_NAME}.desktop"
    command -v update-desktop-database &> /dev/null && \
        update-desktop-database "$DESKTOP_DIR" 2>/dev/null || true
    echo -e "${GREEN}✓ ${APP_NAME} uninstalled successfully!${NC}"
    exit 0
fi

echo -e "${BLUE}Installing ${APP_NAME}...${NC}"

# Check that curl is installed
if ! command -v curl &> /dev/null; then
    echo -e "${RED}Error: curl is required but not installed.${NC}"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) BINARY="${APP_NAME}" ;;
  *)
    echo -e "${RED}Error: unsupported architecture: $ARCH${NC}"
    exit 1
    ;;
esac

# Fetch the latest available version
LATEST_VERSION=$(curl -fsSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}Error: could not determine latest version.${NC}"
    exit 1
fi

echo -e "Latest version: ${BLUE}${LATEST_VERSION}${NC}"

BASE_URL="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_VERSION}"

# Create required directories
mkdir -p "$INSTALL_DIR"
mkdir -p "$ICON_DIR"
mkdir -p "$DESKTOP_DIR"

# Download the executable
echo "Downloading executable..."
curl -fsSL "${BASE_URL}/${BINARY}" -o "${INSTALL_DIR}/${APP_NAME}"
chmod +x "${INSTALL_DIR}/${APP_NAME}"

# Download the icon
echo "Downloading icon..."
curl -fsSL "${BASE_URL}/appicon.png" -o "${ICON_DIR}/${APP_NAME}.png"

# Create the .desktop entry
echo "Creating .desktop entry..."
cat > "${DESKTOP_DIR}/${APP_NAME}.desktop" <<DESKTOP
[Desktop Entry]
Version=1.0
Type=Application
Name=Local Chat
Comment=Chat with local AI models via Ollama
Exec=${INSTALL_DIR}/${APP_NAME}
Icon=${ICON_DIR}/${APP_NAME}.png
Terminal=false
Categories=Utility
DESKTOP

# Update the application database
if command -v update-desktop-database &> /dev/null; then
    update-desktop-database "$DESKTOP_DIR" 2>/dev/null || true
fi

# Check that ~/.local/bin is in PATH
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo ""
    echo -e "${RED}Warning: $HOME/.local/bin is not in your PATH.${NC}"
    echo "Add the following line to your ~/.bashrc or ~/.zshrc:"
    echo -e "  ${BLUE}export PATH=\"\$HOME/.local/bin:\$PATH\"${NC}"
    echo "Then run: source ~/.bashrc"
fi

echo ""
echo -e "${GREEN}✓ ${APP_NAME} ${LATEST_VERSION} installed successfully!${NC}"
echo -e "  Executable : ${INSTALL_DIR}/${APP_NAME}"
echo -e "  Icon       : ${ICON_DIR}/${APP_NAME}.png"
echo -e "  Desktop    : ${DESKTOP_DIR}/${APP_NAME}.desktop"