#!/usr/bin/env bash

# UiGraph CLI Installation Script
# Usage: curl -sSL https://cli.uigraph.app/install.sh | sh

set -e

# Configuration
REPO="uigraph/uigraph-cli"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="uigraph"

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux*)   OS="linux" ;;
  Darwin*)  OS="darwin" ;;
  *)        echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
  x86_64)   ARCH="amd64" ;;
  arm64)    ARCH="arm64" ;;
  aarch64)  ARCH="arm64" ;;
  *)        echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest release
echo "Detecting latest version..."
LATEST_VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
  echo "Error: Could not detect latest version"
  exit 1
fi

echo "Latest version: $LATEST_VERSION"

# Download URL
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_VERSION}/${BINARY_NAME}-${OS}-${ARCH}"

if [ "$OS" = "windows" ]; then
  DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
  BINARY_NAME="${BINARY_NAME}.exe"
fi

echo "Downloading from: $DOWNLOAD_URL"

# Download binary
TMP_FILE=$(mktemp)
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
  echo "Error: Failed to download $DOWNLOAD_URL"
  rm -f "$TMP_FILE"
  exit 1
fi

# Make binary executable
chmod +x "$TMP_FILE"

# Install binary
echo "Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
else
  echo "Note: Requires sudo to install to $INSTALL_DIR"
  sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
fi

echo "✓ UiGraph CLI installed successfully!"
echo ""
echo "Usage:"
echo "  uigraph sync"
echo ""
echo "Documentation:"
echo "  https://docs.uigraph.app/cli"
