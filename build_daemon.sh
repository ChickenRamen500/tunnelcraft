#!/bin/bash
# Build script for tunnelcraftd daemon
# Copies the built binary to the bin/ directory for Tauri sidecar

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CORE_DIR="$SCRIPT_DIR/core"
BIN_DIR="$SCRIPT_DIR/bin"

echo "[build] Building tunnelcraftd daemon..."

cd "$CORE_DIR"

# Build the daemon
go build -o "$BIN_DIR/tunnelcraftd.exe" ./cmd/tunnelcraftd

if [ $? -eq 0 ]; then
    echo "[build] Successfully built tunnelcraftd.exe in $BIN_DIR"
else
    echo "[build] Failed to build tunnelcraftd.exe"
    exit 1
fi

# Verify the binary exists
if [ -f "$BIN_DIR/tunnelcraftd.exe" ]; then
    echo "[build] Daemon binary ready: $BIN_DIR/tunnelcraftd.exe"
    ls -la "$BIN_DIR/tunnelcraftd.exe"
else
    echo "[build] Error: Binary not found after build"
    exit 1
fi
