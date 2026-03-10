#!/bin/bash
set -e

echo "=== Terminalis macOS Build Script ==="
echo ""

# Check if running on macOS
if [[ "$(uname)" != "Darwin" ]]; then
    echo "This script must be run on macOS."
    exit 1
fi

# Check for Xcode Command Line Tools
if ! xcode-select -p &>/dev/null; then
    echo "[1/5] Installing Xcode Command Line Tools..."
    xcode-select --install
    echo "    Please complete the Xcode CLI Tools installation and re-run this script."
    exit 1
else
    echo "[1/5] Xcode Command Line Tools already installed."
fi

# Install Go 1.23 if not present
if ! command -v go &>/dev/null || ! go version 2>/dev/null | grep -q "go1.23"; then
    echo "[2/5] Installing Go 1.23.6..."
    ARCH="$(uname -m)"
    if [[ "$ARCH" == "arm64" ]]; then
        GO_PKG="go1.23.6.darwin-arm64.pkg"
    else
        GO_PKG="go1.23.6.darwin-amd64.pkg"
    fi
    curl -fsSL "https://go.dev/dl/$GO_PKG" -o /tmp/go.pkg
    sudo installer -pkg /tmp/go.pkg -target /
    rm /tmp/go.pkg
else
    echo "[2/5] Go 1.23 already installed."
fi

export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
echo "    Go version: $(go version)"

# Install Node.js if not present
if ! command -v node &>/dev/null; then
    echo "[3/5] Installing Node.js..."
    if command -v brew &>/dev/null; then
        brew install node
    else
        echo "    Homebrew not found. Installing Node.js via official installer..."
        ARCH="$(uname -m)"
        if [[ "$ARCH" == "arm64" ]]; then
            NODE_PKG="node-v20.18.0-darwin-arm64.tar.gz"
        else
            NODE_PKG="node-v20.18.0-darwin-x64.tar.gz"
        fi
        curl -fsSL "https://nodejs.org/dist/v20.18.0/$NODE_PKG" -o /tmp/node.tar.gz
        sudo mkdir -p /usr/local/lib/nodejs
        sudo tar -xzf /tmp/node.tar.gz -C /usr/local/lib/nodejs
        rm /tmp/node.tar.gz
        export PATH="/usr/local/lib/nodejs/${NODE_PKG%.tar.gz}/bin:$PATH"
    fi
else
    echo "[3/5] Node.js already installed: $(node --version)"
fi

# Install Wails CLI
if ! command -v wails &>/dev/null; then
    echo "[4/5] Installing Wails CLI..."
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
else
    echo "[4/5] Wails CLI already installed."
fi

# Build
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
echo "[5/5] Building Terminalis for macOS..."
cd "$SCRIPT_DIR"
wails build -platform darwin/universal

echo ""
echo "=== Build complete! ==="
echo "App bundle: $SCRIPT_DIR/build/bin/Terminalis.app"
