#!/bin/bash
set -e

echo "=== Terminalis Linux Build Script ==="
echo ""

# Check if running on Linux
if [[ "$(uname)" != "Linux" ]]; then
    echo "This script must be run on Linux (or WSL)."
    exit 1
fi

# Install system dependencies
echo "[1/5] Installing system dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq libgtk-3-dev libwebkit2gtk-4.0-dev build-essential pkg-config curl

# Install Go 1.23 if not present
if ! /usr/local/go/bin/go version 2>/dev/null | grep -q "go1.23"; then
    echo "[2/5] Installing Go 1.23.6..."
    curl -fsSL https://go.dev/dl/go1.23.6.linux-amd64.tar.gz -o /tmp/go.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
else
    echo "[2/5] Go 1.23 already installed."
fi

export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
echo "    Go version: $(/usr/local/go/bin/go version)"

# Install Node.js if not present
if ! command -v node &>/dev/null; then
    echo "[3/5] Installing Node.js 20..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    sudo apt-get install -y -qq nodejs
else
    echo "[3/5] Node.js already installed: $(node --version)"
fi

# Install Wails CLI
if ! command -v wails &>/dev/null; then
    echo "[4/5] Installing Wails CLI..."
    /usr/local/go/bin/go install github.com/wailsapp/wails/v2/cmd/wails@latest
else
    echo "[4/5] Wails CLI already installed."
fi

# Build
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
echo "[5/5] Building Terminalis for Linux..."
cd "$SCRIPT_DIR"
wails build

echo ""
echo "=== Build complete! ==="
echo "Binary: $SCRIPT_DIR/build/bin/Terminalis"
