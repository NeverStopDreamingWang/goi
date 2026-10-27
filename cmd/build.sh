#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$SCRIPT_DIR"

WINDOWS_OUTPUT="$SCRIPT_DIR/dist/windows-amd64/goi.exe"
LINUX_OUTPUT="$SCRIPT_DIR/dist/linux-amd64/goi"

mkdir -p "$(dirname "$WINDOWS_OUTPUT")" "$(dirname "$LINUX_OUTPUT")"

echo "Building goi for windows/amd64..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$WINDOWS_OUTPUT" ./goi

echo "Building goi for linux/amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$LINUX_OUTPUT" ./goi

echo "Build completed:"
echo "  Windows: $WINDOWS_OUTPUT"
echo "  Linux:   $LINUX_OUTPUT"
