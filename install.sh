#!/bin/sh
#
# APKless CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/apkless-com/apkless-cli/main/install.sh | sh
#
set -e

REPO="apkless-com/apkless-cli"
INSTALL_DIR="/usr/local/bin"
BINARY="apkless"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  darwin) OS="darwin" ;;
  linux)  OS="linux" ;;
  *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)             echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

SUFFIX="${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  SUFFIX="${SUFFIX}.exe"
fi

# Get latest release
echo "Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Failed to fetch latest release"
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${LATEST}/${BINARY}-${SUFFIX}"

echo "Downloading ${BINARY} ${LATEST} (${OS}/${ARCH})..."
curl -fsSL -o "/tmp/${BINARY}" "$URL"
chmod +x "/tmp/${BINARY}"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo ""
echo "✓ ${BINARY} ${LATEST} installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Get started:"
echo "  export APKLESS_KEY=apkless_xxxxxxxx"
echo "  ${BINARY} create"
echo ""
